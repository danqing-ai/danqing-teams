package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

const (
	defaultMaxTokens     = 128000
	defaultTriggerRatio  = 0.85
	defaultCutTokens     = 16000
	defaultTurnInterval  = 6
	defaultSubInterval   = 4
	defaultToolTruncate  = 2000
	tokenEstimateDivisor = 4
)

type CompactionManager struct {
	mu          sync.RWMutex
	checkpoints map[string]*domain.CompactionCheckpoint
	llm         port.LLMProvider
	configStore port.ConfigStore
	stream      port.EventStream
	store       CompactionCheckpointStore
	modelLimits *ModelConfigRegistry
}

type CompactionCheckpointStore interface {
	Load(sessionID string) (*domain.CompactionCheckpoint, error)
	Save(sessionID string, cp *domain.CompactionCheckpoint) error
}

type compactionCfg struct {
	enabled       bool
	maxTokens     int
	triggerRatio  float64
	cutTokens     int
	turnInterval  int
	subInterval   int
	toolTruncate  int
}

func NewCompactionManager(llm port.LLMProvider, stream port.EventStream, configStore port.ConfigStore, store CompactionCheckpointStore, modelLimits *ModelConfigRegistry) *CompactionManager {
	if modelLimits == nil {
		modelLimits = NewModelConfigRegistry()
	}
	return &CompactionManager{
		checkpoints: make(map[string]*domain.CompactionCheckpoint),
		llm:         llm,
		configStore: configStore,
		stream:      stream,
		store:       store,
		modelLimits: modelLimits,
	}
}

func (m *CompactionManager) loadCfg(ctx context.Context) compactionCfg {
	cfg := compactionCfg{
		maxTokens:    0, // 0 means “use model context window”
		triggerRatio: defaultTriggerRatio,
		cutTokens:    defaultCutTokens,
		turnInterval: defaultTurnInterval,
		subInterval:  defaultSubInterval,
		toolTruncate: defaultToolTruncate,
	}
	if m.configStore != nil {
		if c, err := m.configStore.Load(ctx); err == nil {
			rt := c.Runtime.Compaction
			cfg.enabled = rt.Enabled
			if rt.MaxTokens > 0 {
				cfg.maxTokens = rt.MaxTokens
			}
			if rt.TriggerRatio > 0 {
				cfg.triggerRatio = rt.TriggerRatio
			}
			if rt.CutTokens > 0 {
				cfg.cutTokens = rt.CutTokens
			}
			if rt.TurnInterval > 0 {
				cfg.turnInterval = rt.TurnInterval
			}
			if rt.SubInterval > 0 {
				cfg.subInterval = rt.SubInterval
			}
			if rt.ToolTruncate > 0 {
				cfg.toolTruncate = rt.ToolTruncate
			}
			// Reload model limits from config into the registry.
			m.modelLimits.SetModels(c.LLM.Models)
		}
	}
	return cfg
}

func (m *CompactionManager) ShouldCompact(sessionID string, turnCount int, tokenEstimate int, maxPromptTokens int, model string) bool {
	cfg := m.loadCfg(context.Background())
	if !cfg.enabled {
		return false
	}

	// Determine the effective context limit and actual token usage.
	// Priority: actual API usage > char-based estimation.
	contextLimit := cfg.maxTokens
	actualTokens := tokenEstimate

	// If we have actual prompt tokens from the LLM API, use them + model's real context window.
	if maxPromptTokens > 0 {
		actualTokens = maxPromptTokens
		if modelLimit := m.modelLimits.ContextWindow(model); modelLimit > 0 {
			// Reserve space for output tokens so we trigger before hitting the hard limit.
			maxOutput := m.modelLimits.MaxOutputTokens(model)
			usable := modelLimit - maxOutput
			if usable > 0 {
				// Use the smaller of config override and model-derived usable space.
				if contextLimit == 0 || usable < contextLimit {
					contextLimit = usable
				}
			}
		}
	}

	// Token-based trigger: actual usage exceeds triggerRatio of context limit.
	if contextLimit > 0 && actualTokens > int(float64(contextLimit)*cfg.triggerRatio) {
		return true
	}

	// If no context limit is known, try model context window as fallback.
	if contextLimit == 0 && model != "" {
		if modelLimit := m.modelLimits.ContextWindow(model); modelLimit > 0 {
			maxOutput := m.modelLimits.MaxOutputTokens(model)
			usable := modelLimit - maxOutput
			if usable > 0 && tokenEstimate > int(float64(usable)*cfg.triggerRatio) {
				return true
			}
		}
	}

	// Turn-count-based trigger (fallback when token data is unreliable).
	if cfg.turnInterval > 0 {
		cp := m.getCheckpoint(sessionID)
		if cp == nil {
			return turnCount >= cfg.turnInterval
		}
		interval := cfg.subInterval
		if interval <= 0 {
			interval = defaultSubInterval
		}
		return turnCount-cp.TurnCount >= interval
	}
	return false
}

func (m *CompactionManager) Checkpoint(sessionID string) *domain.CompactionCheckpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checkpoints[sessionID]
}

func (m *CompactionManager) Recover(ctx context.Context, sessionID string) *domain.CompactionCheckpoint {
	return m.getCheckpoint(sessionID)
}

func (m *CompactionManager) Compact(ctx context.Context, sessionID, turnID string, messages []Message, turnCount int, model string) int {
	cfg := m.loadCfg(ctx)
	if len(messages) <= 1 {
		return 0
	}

	tokensBefore := estimateTokenCount(messages)

	prevCP := m.getCheckpoint(sessionID)
	cutIdx := findCutPoint(messages, cfg.cutTokens)
	if cutIdx <= 0 {
		return 0
	}

	oldMessages := messages[:cutIdx]
	conversation := serializeConversation(oldMessages, cfg.toolTruncate)

	summary, err := m.summarize(ctx, conversation, prevCP, model)
	if err != nil || summary == "" {
		return 0
	}

	todos := extractLatestTodos(messages)
	if len(todos) == 0 && prevCP != nil {
		todos = prevCP.Todos
	}

	cp := &domain.CompactionCheckpoint{
		SessionID:     sessionID,
		TurnID:        turnID,
		Summary:       summary,
		Todos:         todos,
		TurnCount:     turnCount,
		TokenEstimate: tokensBefore,
	}
	m.setCheckpoint(sessionID, cp)

	if m.stream != nil {
		m.stream.Publish(ctx, sessionID, turnID, domain.EventContextCompacted, domain.ContextCompactedPayload{
			FilePath:       fmt.Sprintf("checkpoint_%s.json", turnID),
			TurnsCompacted: len(oldMessages),
			TokensBefore:   tokensBefore,
			TokensAfter:    estimateTokenCount(messages[cutIdx:]),
		})
	}

	return cutIdx
}

// CompactToRetain summarizes oldMessages and records retainFromTurnID so later
// LoadSessionMessages skips compacted turns.
func (m *CompactionManager) CompactToRetain(ctx context.Context, sessionID, turnID string, oldMessages []Message, turnCount int, model, retainFromTurnID string, tokensBefore int) bool {
	if len(oldMessages) == 0 || retainFromTurnID == "" {
		return false
	}
	cfg := m.loadCfg(ctx)
	prevCP := m.getCheckpoint(sessionID)
	conversation := serializeConversation(oldMessages, cfg.toolTruncate)
	summary, err := m.summarize(ctx, conversation, prevCP, model)
	if err != nil || summary == "" {
		return false
	}
	todos := extractLatestTodos(oldMessages)
	if len(todos) == 0 && prevCP != nil {
		todos = prevCP.Todos
	}
	cp := &domain.CompactionCheckpoint{
		SessionID:        sessionID,
		TurnID:           turnID,
		Summary:          summary,
		Todos:            todos,
		TurnCount:        turnCount,
		TokenEstimate:    tokensBefore,
		RetainFromTurnID: retainFromTurnID,
	}
	m.setCheckpoint(sessionID, cp)
	if m.stream != nil {
		m.stream.Publish(ctx, sessionID, turnID, domain.EventContextCompacted, domain.ContextCompactedPayload{
			FilePath:       fmt.Sprintf("checkpoint_%s.json", turnID),
			TurnsCompacted: len(oldMessages),
			TokensBefore:   tokensBefore,
			TokensAfter:    0,
		})
	}
	return true
}

func (m *CompactionManager) summarize(ctx context.Context, conversation string, prev *domain.CompactionCheckpoint, model string) (string, error) {
	var prompt string
	if prev != nil && prev.Summary != "" {
		prompt = fmt.Sprintf(compactionIncrementalPrompt, prev.Summary, conversation)
	} else {
		prompt = fmt.Sprintf(compactionPrompt, conversation)
	}

	resp, err := m.llm.Chat(ctx, port.LLMChatRequest{
		Model: model,
		Messages: []port.ChatMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (m *CompactionManager) getCheckpoint(sessionID string) *domain.CompactionCheckpoint {
	m.mu.RLock()
	if cp, ok := m.checkpoints[sessionID]; ok {
		m.mu.RUnlock()
		return cp
	}
	m.mu.RUnlock()

	if m.store != nil {
		cp, err := m.store.Load(sessionID)
		if err == nil && cp != nil {
			m.mu.Lock()
			m.checkpoints[sessionID] = cp
			m.mu.Unlock()
			return cp
		}
	}
	return nil
}

func (m *CompactionManager) setCheckpoint(sessionID string, cp *domain.CompactionCheckpoint) {
	m.mu.Lock()
	m.checkpoints[sessionID] = cp
	m.mu.Unlock()

	if m.store != nil {
		_ = m.store.Save(sessionID, cp)
	}
}

func findCutPoint(messages []Message, cutTokens int) int {
	if len(messages) <= 1 {
		return 0
	}

	accumulated := 0
	for i := len(messages) - 1; i >= 0; i-- {
		accumulated += estimateMessageTokens(messages[i])
		if accumulated >= cutTokens {
			cut := findValidCutPoint(messages, i, len(messages))
			if cut > 0 && cut < len(messages) {
				for cut > 0 && cut < len(messages) && !isBlockBoundary(messages[cut-1], messages[cut]) {
					cut++
				}
			}
			if cut >= len(messages) {
				cut = len(messages) - 1
			}
			return cut
		}
	}
	return 0
}

func findValidCutPoint(messages []Message, from, to int) int {
	pairs := buildToolPairs(messages)
	for j := from; j < to; j++ {
		m := messages[j]
		if isNaturalCutPoint(m) {
			if _, inPair := pairs[j]; !inPair {
				return j
			}
		}
	}
	return from
}

func buildToolPairs(messages []Message) map[int]bool {
	pairs := make(map[int]bool)
	for i, m := range messages {
		if m.Role == RoleTool && m.ToolCallID != "" {
			for j := i - 1; j >= 0; j-- {
				prev := messages[j]
				if prev.Role == RoleAssistant {
					for _, tc := range prev.ToolCalls {
						if tc.ID == m.ToolCallID {
							pairs[i] = true
							pairs[j] = true
							break
						}
					}
					break
				}
			}
		}
	}
	return pairs
}

func isNaturalCutPoint(m Message) bool {
	switch m.Role {
	case RoleUser:
		return true
	case RoleAssistant:
		return len(m.ToolCalls) == 0
	case RoleSystem:
		return false
	case RoleTool:
		return false
	}
	return false
}

func isBlockBoundary(prev, next Message) bool {
	return prev.Role == RoleTool && next.Role == RoleUser
}

func isToolResultOrPair(messages []Message, idx int, pairs map[int]bool) bool {
	if _, ok := pairs[idx]; ok {
		return true
	}
	return messages[idx].Role == RoleTool
}

// extractLatestTodos walks messages newest-first and returns the last todowrite list.
func extractLatestTodos(messages []Message) []domain.CompactionTodoItem {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != RoleAssistant {
			continue
		}
		for j := len(m.ToolCalls) - 1; j >= 0; j-- {
			tc := m.ToolCalls[j]
			if tc.Name != "todowrite" {
				continue
			}
			if items := parseTodoArgs(tc.Arguments); len(items) > 0 {
				return items
			}
		}
	}
	return nil
}

func parseTodoArgs(args map[string]any) []domain.CompactionTodoItem {
	if args == nil {
		return nil
	}
	raw, ok := args["todos"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]domain.CompactionTodoItem, 0, len(list))
	for _, t := range list {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		content, _ := m["content"].(string)
		if content == "" {
			continue
		}
		status, _ := m["status"].(string)
		if status == "" {
			status = "pending"
		}
		priority, _ := m["priority"].(string)
		if priority == "" {
			priority = "medium"
		}
		out = append(out, domain.CompactionTodoItem{
			Content:  content,
			Status:   status,
			Priority: priority,
		})
	}
	return out
}

// formatActiveTodos renders structured todos for system-prompt injection.
func formatActiveTodos(todos []domain.CompactionTodoItem) string {
	if len(todos) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<active-todos>\n")
	for i, t := range todos {
		b.WriteString(fmt.Sprintf("%d. [%s] %s (%s)\n", i+1, t.Status, t.Content, t.Priority))
	}
	b.WriteString("</active-todos>")
	return b.String()
}

func serializeConversation(messages []Message, truncateLen int) string {
	var b strings.Builder
	for _, m := range messages {
		switch m.Role {
		case RoleUser:
			b.WriteString("[User]: ")
			b.WriteString(m.Content)
			b.WriteString("\n\n")
		case RoleAssistant:
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					argsStr, _ := json.Marshal(tc.Arguments)
					b.WriteString(fmt.Sprintf("[Assistant → tool_call: %s(%s)]\n", tc.Name, string(argsStr)))
				}
			} else {
				b.WriteString("[Assistant]: ")
				b.WriteString(m.Content)
				b.WriteString("\n\n")
			}
		case RoleTool:
			content := m.Content
			if truncateLen > 0 && len(content) > truncateLen {
				content = content[:truncateLen] + "...(truncated)"
			}
			b.WriteString(fmt.Sprintf("[Tool result: %s]: %s\n\n", m.Name, content))
		}
	}
	return b.String()
}

func estimateTokenCount(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

func estimateMessageTokens(m Message) int {
	n := 0
	n += len(m.Role) / tokenEstimateDivisor
	n += len(m.Content) / tokenEstimateDivisor
	n += len(m.Name) / tokenEstimateDivisor
	n += len(m.ToolCallID) / tokenEstimateDivisor
	for _, tc := range m.ToolCalls {
		n += len(tc.ID) / tokenEstimateDivisor
		n += len(tc.Name) / tokenEstimateDivisor
		raw, _ := json.Marshal(tc.Arguments)
		n += len(raw) / tokenEstimateDivisor
	}
	return n
}

const compactionPrompt = `You are a context compaction assistant. Analyze the conversation transcript below and produce a structured JSON summary. Preserve all critical facts, decisions, and context needed to continue the session.

<conversation>
%s
</conversation>

Output ONLY a valid JSON object (no markdown fences, no commentary) with these fields:
{
  "summary": "concise narrative summary of what happened and the current state",
  "workState": {"completed": [], "active": [], "blocked": []},
  "decisions": ["key decisions made"],
  "nextMove": "what to do next",
  "criticalContext": ["important facts that must be preserved"],
  "agentsInvolved": ["agent names"],
  "filesTouched": ["file paths"]
}`

const compactionIncrementalPrompt = `You are a context compaction assistant. Update the existing summary with new information from the conversation transcript below.

PRESERVE all still-relevant information from the previous summary. ADD new progress and facts. UPDATE completed/active/blocked statuses. REMOVE stale details. MERGE new file paths and agent names with existing ones.

<previous-summary>
%s
</previous-summary>

<conversation>
%s
</conversation>

Output ONLY a valid JSON object (no markdown fences, no commentary) with the same structure as the previous summary.`

func ToolInputKey(args map[string]any) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		if strings.HasPrefix(k, "__") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v,", k, args[k])
	}
	return b.String()
}
