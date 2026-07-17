package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
	"danqing-teams/core/service"
	"danqing-teams/core/runtime/permission"
	"danqing-teams/core/runtime/tool"
	"danqing-teams/core/runtime/tool/builtin"
)

var _ port.Engine = (*Engine)(nil)

type engineRunCfg struct {
	autoApprove            bool
	teamMaxDelegationDepth int
	knowledgeSearchTopK    int
}

type Engine struct {
	sessions      *service.SessionManager
	turns         *service.TurnManager
	projects      *service.ProjectManager
	approvals     *service.ApprovalManager
	turnLog       *service.TurnLogManager
	agents        *service.AgentManager
	skills        *service.SkillManager
	knowledge     *builtin.Knowledge
	llm           port.LLMProvider
	stream        port.EventStream
	sandbox       port.Sandbox
	turnRunner    *TurnRunner
	toolCatalog   *tool.Registry
	compactionMgr *CompactionManager
	modelLimits   *ModelConfigRegistry
	configStore   port.ConfigStore
	dataDir       string
	turnMessages  map[string][]Message
	mu            sync.Mutex
	approvalWait  map[string]chan ApprovalOutcome
	approvalMeta  map[string]approvalMeta
	sessionPerm   map[string]sessionPermState
	askUserWait   map[string]chan string
	cancel        map[string]context.CancelFunc
	agentChain    []string
}

type approvalMeta struct {
	SessionID string
	Reason    string
}

type sessionPermState struct {
	AllowNetwork bool
}

// ApprovalOutcome is returned when a pending approval is resolved.
type ApprovalOutcome struct {
	Approved bool
	Scope    string // once | session
	Reason   string
}

func (e *Engine) loadRunCfg(ctx context.Context) engineRunCfg {
	cfg := engineRunCfg{
		teamMaxDelegationDepth: 3,
		knowledgeSearchTopK:    3,
	}
	if e.configStore != nil {
		if c, err := e.configStore.Load(ctx); err == nil {
			rt := c.Runtime
			cfg.autoApprove = rt.AutoApprove
			cfg.teamMaxDelegationDepth = rt.Team.MaxDelegationDepth
			cfg.knowledgeSearchTopK = rt.Knowledge.SearchTopK
		}
	}
	return cfg
}

func (e *Engine) isAutoApprove() bool {
	if e.configStore != nil {
		if c, err := e.configStore.Load(context.Background()); err == nil {
			return c.Runtime.AutoApprove
		}
	}
	return false
}

func NewEngine(sessions *service.SessionManager, turns *service.TurnManager, projects *service.ProjectManager, approvals *service.ApprovalManager, turnLog *service.TurnLogManager, agents *service.AgentManager, skills *service.SkillManager, knowledge *builtin.Knowledge, llm port.LLMProvider, stream port.EventStream, checkpointStore CompactionCheckpointStore, configStore port.ConfigStore, dataDir string) *Engine {
	catalog := tool.NewRegistry()
	gate := permission.NewGate(nil)
	turnRunner := NewTurnRunner(llm, stream, gate, tool.NewRegistry(), configStore)

	modelLimits := NewModelConfigRegistry()
	modelLimits.LoadFromConfig(context.Background(), configStore)

	e := &Engine{
		sessions:      sessions,
		turns:         turns,
		projects:      projects,
		approvals:     approvals,
		turnLog:       turnLog,
		agents:        agents,
		skills:        skills,
		knowledge:     knowledge,
		llm:           llm,
		stream:        stream,
		turnRunner:    turnRunner,
		toolCatalog:   catalog,
		compactionMgr: NewCompactionManager(llm, stream, configStore, checkpointStore, modelLimits),
		modelLimits:   modelLimits,
		configStore:   configStore,
		dataDir:       dataDir,
		turnMessages:  make(map[string][]Message),
		approvalWait:  make(map[string]chan ApprovalOutcome),
		approvalMeta:  make(map[string]approvalMeta),
		sessionPerm:   make(map[string]sessionPermState),
		askUserWait:   make(map[string]chan string),
		cancel:        make(map[string]context.CancelFunc),
	}
	turnRunner.Approval = e
	turnRunner.SandboxStatus = e.sandboxStatus
	turnRunner.SessionAllowNetwork = e.sessionAllowsNetwork
	return e
}

// SetSandbox wires the process sandbox used for policy decisions and tool execution status.
func (e *Engine) SetSandbox(sb port.Sandbox) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sandbox = sb
}

func (e *Engine) sandboxStatus() domain.SandboxStatus {
	e.mu.Lock()
	sb := e.sandbox
	e.mu.Unlock()
	if sb == nil {
		return domain.SandboxStatus{Enabled: false, Backend: domain.SandboxBackendDisabled}
	}
	return sb.Status()
}

func (e *Engine) sessionAllowsNetwork(sessionID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sessionPerm[sessionID].AllowNetwork
}

func (e *Engine) RegisterTool(h tool.Handler) {
	e.toolCatalog.Register(h)
}

func (e *Engine) StartSession(ctx context.Context, s domain.Session) {
	turnID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
	go func() {
		turnCtx, cancel := context.WithCancel(context.Background())
		e.mu.Lock()
		e.cancel[turnID] = cancel
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			delete(e.cancel, turnID)
			e.mu.Unlock()
			cancel()
		}()

		agent, err := e.agents.Get(ctx, s.AgentID)
		if err != nil {
			e.stream.Publish(ctx, s.ID, "", domain.EventTurnFailed, map[string]string{"error": err.Error()})
			return
		}
		agentPtr := *agent

		reg := e.setupRegistry(s, agentPtr)
		rep, err := e.runTurn(turnCtx, s.ID, turnID, s.Content, s.ModelID, s.ProjectID, agentPtr, reg)
		e.turnLog.EndTurn(turnID, turnStatus(err, rep))
	}()
}

func (e *Engine) StartTurn(ctx context.Context, sessionID, userInput, agentID, modelID string) (string, error) {
	turnID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
	// Reset session status to active so UI shows "运行中"
	e.updateSessionStatus(sessionID, domain.SessionStatusActive)
	go func() {
		s, err := e.sessions.Get(ctx, sessionID)
		if err != nil {
			return
		}
		targetAgentID := agentID
		if targetAgentID == "" {
			targetAgentID = s.AgentID
		}
		targetModelID := modelID
		if targetModelID == "" {
			targetModelID = s.ModelID
		}
		agent, err := e.agents.Get(ctx, targetAgentID)
		if err != nil {
			return
		}
		agentPtr := *agent

		turnCtx, cancel := context.WithCancel(context.Background())
		e.mu.Lock()
		e.cancel[turnID] = cancel
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			delete(e.cancel, turnID)
			e.mu.Unlock()
			cancel()
		}()

		reg := e.setupRegistry(s, agentPtr)
		rep, err := e.runTurn(turnCtx, sessionID, turnID, userInput, targetModelID, s.ProjectID, agentPtr, reg)
		e.turnLog.EndTurn(turnID, turnStatus(err, rep))
	}()
	return turnID, nil
}

func (e *Engine) CancelTurn(ctx context.Context, turnID string) {
	e.mu.Lock()
	cancel, ok := e.cancel[turnID]
	e.mu.Unlock()
	if ok {
		cancel()
	}
}

func (e *Engine) ResumeTurn(ctx context.Context, sessionID, turnID string) {
	go func() {
		cfg := e.loadRunCfg(ctx)

		s, err := e.sessions.Get(ctx, sessionID)
		if err != nil {
			return
		}
		agent, err := e.agents.Get(ctx, s.AgentID)
		if err != nil {
			return
		}
		agentPtr := *agent

		turnCtx, cancel := context.WithCancel(context.Background())
		// Reset session status to active so UI shows "运行中" during resume
		e.updateSessionStatus(sessionID, domain.SessionStatusActive)

		e.mu.Lock()
		e.cancel[turnID] = cancel
		e.mu.Unlock()
		defer func() {
			e.mu.Lock()
			delete(e.cancel, turnID)
			e.mu.Unlock()
			cancel()
		}()

		goal := s.Content
		var replayEntries []map[string]any
		if g, entries := e.turnLog.LoadForRecovery(turnID); g != "" {
			goal = g
			replayEntries = entries
		}
		if goal == "" {
			goal = s.Content
		}

		reg := e.setupRegistry(s, agentPtr)
		e.stream.Publish(turnCtx, sessionID, turnID, domain.EventTurnStarted, domain.TurnStartedPayload{
			TurnID: turnID, AgentID: agentPtr.ID, Goal: goal,
		})
		e.stream.Publish(turnCtx, sessionID, turnID, domain.EventUserMessage, domain.UserMessagePayload{Content: goal})
		e.turnLog.Create(turnID, sessionID, s.ProjectID, agentPtr.ID, goal)
		_ = e.turns.Create(turnCtx, domain.TurnLog{ID: turnID, SessionID: sessionID, AgentID: agentPtr.ID, Goal: goal, Status: domain.TurnRunning})
		e.turnRunner.Log = func(typ string, data map[string]any) {
			e.turnLog.Append(turnID, typ, data)
		}

		sys := buildSystemPrompt(agentPtr.SystemPrompt, e.turnRunner.SkillList, e.delegatableAgents(agentPtr), "")
		messages := []Message{{Role: RoleSystem, Content: sys}}
		if hits := e.knowledge.Search(agentPtr.KnowledgeIDs, goal, cfg.knowledgeSearchTopK); len(hits) > 0 {
			content := ""
			for _, h := range hits {
				content += h + "\n"
			}
			messages = append(messages, Message{Role: RoleSystem, Content: content})
		}

		e.mu.Lock()
		prevMsgs := e.turnMessages[sessionID]
		e.mu.Unlock()
		messages = append(messages, prevMsgs...)

		messages = append(messages, Message{Role: RoleUser, Content: goal})

		for _, entry := range replayEntries {
			typ, _ := entry["type"].(string)
			switch typ {
			case "tool_call":
				data, _ := entry["data"].(map[string]any)
				name, _ := data["name"].(string)
				input, _ := data["input"].(map[string]any)
				callID, _ := data["call_id"].(string)
				if callID == "" {
					callID = turnID + "-" + name // fallback for old logs
				}
				messages = append(messages, Message{
					Role:      RoleAssistant,
					ToolCalls: []ToolCall{{ID: callID, Name: name, Arguments: input}},
				})
			case "tool_result":
				data, _ := entry["data"].(map[string]any)
				name, _ := data["name"].(string)
				output, _ := data["output"].(string)
				callID, _ := data["call_id"].(string)
				if callID == "" {
					callID = turnID + "-" + name // fallback for old logs
				}
				messages = append(messages, Message{
					Role: RoleTool, ToolCallID: callID, Name: name, Content: output,
				})
			}
		}

		e.turnRunner.Registry = reg
		rep, turnMsgs, err := e.turnRunner.Run(turnCtx, TurnContext{
			SessionID: sessionID, TurnID: turnID, Agent: agentPtr,
			Model: s.ModelID, MaxSteps: agentPtr.Steps, Messages: messages,
		})

		e.mu.Lock()
		e.turnMessages[sessionID] = append(e.turnMessages[sessionID], turnMsgs...)
		allMsgs := e.turnMessages[sessionID]
		e.mu.Unlock()

		e.turnLog.EndTurn(turnID, turnStatus(err, rep))
		e.afterTurn(sessionID, turnID, agentPtr.ID, rep, err, allMsgs, s.ModelID)
	}()
}

func (e *Engine) RecoverRunning(ctx context.Context) {
	// 1. Recover zombie turns: mark all "running" turns as failed.
	runningTurns, err := e.turns.ListByStatus(ctx, domain.TurnRunning)
	if err != nil {
		log.Printf("[RecoverRunning] list running turns: %v", err)
		return
	}
	if len(runningTurns) > 0 {
		log.Printf("[RecoverRunning] found %d zombie running turn(s), marking as failed", len(runningTurns))
	}
	for _, t := range runningTurns {
		if err := e.turns.UpdateStatus(ctx, t.ID, domain.TurnFailed); err != nil {
			log.Printf("[RecoverRunning] update turn %s status: %v", t.ID, err)
		}
		e.turnLog.EndTurn(t.ID, domain.TurnFailed)
	}

	// 2. Recover stale approvals: mark all "pending" approvals as expired.
	pendingApprovals, err := e.approvals.ListByStatus(ctx, "pending")
	if err != nil {
		log.Printf("[RecoverRunning] list pending approvals: %v", err)
	} else if len(pendingApprovals) > 0 {
		log.Printf("[RecoverRunning] found %d stale pending approval(s), marking as expired", len(pendingApprovals))
		for _, a := range pendingApprovals {
			a.Status = "expired"
			if err := e.approvals.Update(ctx, a); err != nil {
				log.Printf("[RecoverRunning] update approval %s: %v", a.ID, err)
			}
		}
	}

	// 3. Recover stuck sessions: status "active" with no running turns means the
	//    process died mid-flight. Derive the terminal status from turns — do NOT
	//    always mark failed (completed turns would incorrectly show a failure badge).
	sessions, err := e.sessions.List(ctx)
	if err != nil {
		log.Printf("[RecoverRunning] list sessions: %v", err)
		return
	}
	for _, s := range sessions {
		if s.Status != domain.SessionStatusActive {
			continue
		}
		turns, err := e.turns.ListBySession(ctx, s.ID)
		if err != nil {
			continue
		}
		hasRunning := false
		hasFailed := false
		for _, t := range turns {
			switch t.Status {
			case domain.TurnRunning:
				hasRunning = true
			case domain.TurnFailed, domain.TurnCancelled, domain.TurnTimeout:
				hasFailed = true
			}
		}
		if hasRunning {
			continue
		}
		status := domain.SessionStatusCompleted
		if hasFailed || len(turns) == 0 {
			status = domain.SessionStatusFailed
		}
		log.Printf("[RecoverRunning] session %s stuck in active with no running turns, marking as %s", s.ID, status)
		s.Status = status
		s.UpdatedAt = time.Now().UTC()
		_ = e.sessions.UpdateSession(ctx, s)
	}
}

func (e *Engine) ListTurns(sessionID string) []domain.TurnLog {
	turns, err := e.turns.ListBySession(context.Background(), sessionID)
	if err != nil {
		return nil
	}
	return turns
}

func (e *Engine) setupRegistry(s domain.Session, agent domain.Agent) *tool.Registry {
	skills := e.resolveAgentSkills(agent)
	e.turnRunner.SkillList = skills
	e.turnRunner.ToolBindings = agent.Tools

	var reg *tool.Registry
	if agentHasDelegation(agent) {
		reg = e.buildTeamRegistry(agent)
	} else {
		reg = e.buildWorkerRegistry(agent)
	}
	return reg
}

func agentHasDelegation(agent domain.Agent) bool {
	return agent.CanDelegate
}

// delegatableAgents returns the agent list to inject into the system prompt.
// Only agents with delegation enabled receive it; the coordinator itself is excluded.
func (e *Engine) delegatableAgents(agent domain.Agent) []domain.Agent {
	if !agent.CanDelegate {
		return nil
	}
	all, _ := e.agents.List(context.Background())
	var result []domain.Agent
	for _, a := range all {
		if a.ID == agent.ID {
			continue
		}
		result = append(result, a)
	}
	return result
}

// resolveAgentSkills returns only the skills bound to the given agent.
// If the agent has no SkillIDs configured, all skills are returned for backward compatibility.
func (e *Engine) resolveAgentSkills(agent domain.Agent) []domain.Skill {
	all, _ := e.skills.List(context.Background())
	if len(agent.SkillIDs) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(agent.SkillIDs))
	for _, id := range agent.SkillIDs {
		wanted[id] = struct{}{}
	}
	var result []domain.Skill
	for _, sk := range all {
		if _, ok := wanted[sk.ID]; ok {
			result = append(result, sk)
		}
	}
	return result
}

func (e *Engine) runTurn(ctx context.Context, sessionID, turnID, goal, modelID, projectID string, agent domain.Agent, reg *tool.Registry) (domain.Report, error) {
	cfg := e.loadRunCfg(ctx)

	e.turnRunner.Registry = reg
	e.stream.Publish(ctx, sessionID, turnID, domain.EventTurnStarted, domain.TurnStartedPayload{
		TurnID: turnID, AgentID: agent.ID, Goal: goal,
	})
	e.stream.Publish(ctx, sessionID, turnID, domain.EventUserMessage, domain.UserMessagePayload{Content: goal})

	e.turnLog.Create(turnID, sessionID, projectID, agent.ID, goal)
	_ = e.turns.Create(ctx, domain.TurnLog{ID: turnID, SessionID: sessionID, AgentID: agent.ID, Goal: goal, Status: domain.TurnRunning})
	e.turnRunner.Log = func(typ string, data map[string]any) {
		e.turnLog.Append(turnID, typ, data)
	}

	checkpoint := e.compactionMgr.Recover(ctx, sessionID)
	checkpointText := ""
	if checkpoint != nil && checkpoint.Summary != "" {
		checkpointText = checkpoint.Summary
	}

	sys := buildSystemPrompt(agent.SystemPrompt, e.turnRunner.SkillList, e.delegatableAgents(agent), checkpointText)
	messages := []Message{
		{Role: RoleSystem, Content: sys},
	}

	if hits := e.knowledge.Search(agent.KnowledgeIDs, goal, cfg.knowledgeSearchTopK); len(hits) > 0 {
		content := ""
		for _, h := range hits {
			content += h + "\n"
		}
		messages = append(messages, Message{Role: RoleSystem, Content: content})
	}

	e.mu.Lock()
	prevMsgs := e.turnMessages[sessionID]
	e.mu.Unlock()
	messages = append(messages, prevMsgs...)

	messages = append(messages, Message{Role: RoleUser, Content: goal})

	workDir := e.resolveWorkDir(ctx, projectID)

	rep, turnMsgs, err := e.turnRunner.Run(ctx, TurnContext{
		SessionID: sessionID,
		TurnID:    turnID,
		Agent:     agent,
		Model:     modelID,
		MaxSteps:  agent.Steps,
		WorkDir:   workDir,
		Messages:  messages,
	})

	e.mu.Lock()
	e.turnMessages[sessionID] = append(e.turnMessages[sessionID], turnMsgs...)
	allMsgs := e.turnMessages[sessionID]
	e.mu.Unlock()

	e.afterTurn(sessionID, turnID, agent.ID, rep, err, allMsgs, modelID)
	return rep, err
}

func (e *Engine) resolveWorkDir(ctx context.Context, projectID string) string {
	var dir string
	if projectID == "" {
		dir = e.dataDir
	} else {
		dir = e.projects.ResolveDir(ctx, projectID, e.dataDir)
	}
	// Ensure the working directory exists so tools can operate in it.
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func (e *Engine) afterTurn(sessionID, turnID, agentID string, rep domain.Report, err error, messages []Message, model string) {
	if err != nil {
		kind := "turn"
		msg := err.Error()
		if errors.Is(err, context.Canceled) {
			kind = "cancelled"
			msg = "cancelled"
		}
		e.stream.Publish(context.Background(), sessionID, turnID, domain.EventTurnFailed, domain.ErrorPayload{
			Message: msg, Kind: kind,
		})
		e.updateSessionStatus(sessionID, domain.SessionStatusFailed)
		_ = e.turns.UpdateStatus(context.Background(), turnID, turnStatus(err, rep))
		return
	}
	e.updateSessionStatus(sessionID, domain.SessionStatusCompleted)
	_ = e.turns.UpdateStatus(context.Background(), turnID, domain.TurnCompleted)
	e.stream.Publish(context.Background(), sessionID, turnID, domain.EventSessionCompleted, domain.SessionCompletedPayload{
		Summary: rep.Summary, Status: string(rep.Status),
	})
	turns, _ := e.turns.ListBySession(context.Background(), sessionID)
	e.maybeCompact(context.Background(), sessionID, turnID, len(turns), messages, model, rep.MaxPromptTokens)
}

func (e *Engine) updateSessionStatus(sessionID string, status domain.SessionStatus) {
	s, err := e.sessions.Get(context.Background(), sessionID)
	if err != nil {
		return
	}
	s.Status = status
	s.UpdatedAt = time.Now().UTC()
	_ = e.sessions.UpdateSession(context.Background(), s)
}

func (e *Engine) maybeCompact(ctx context.Context, sessionID, turnID string, turnCount int, messages []Message, model string, maxPromptTokens int) {
	tokenEstimate := estimateTokenCount(messages)
	if e.compactionMgr.ShouldCompact(sessionID, turnCount, tokenEstimate, maxPromptTokens, model) {
		cutIdx := e.compactionMgr.Compact(ctx, sessionID, turnID, messages, turnCount, model)
		if cutIdx > 0 {
			e.mu.Lock()
			e.turnMessages[sessionID] = messages[cutIdx:]
			e.mu.Unlock()
		}
	}
}

func (e *Engine) mountBuiltinTools(reg *tool.Registry, bindings []domain.ToolBinding) {
	for _, b := range bindings {
		if b.ToolID == "" || b.MCPServer != "" {
			continue
		}
		if h, ok := e.toolCatalog.Get(b.ToolID); ok {
			reg.Register(h)
		}
	}
}

func (e *Engine) buildTeamRegistry(agent domain.Agent) *tool.Registry {
	cfg := e.loadRunCfg(context.Background())
	delegator := &builtin.DelegateAgent{
		Stream: e.stream, Agents: e.agents,
		KnowledgeSearch: e.knowledge.Search,
		RunSubTurn: func(ctx context.Context, sessionID, modelID, parentTurnID string, workerAgent domain.Agent, goal string) (domain.Report, error) {
			childTurnID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
			workDir := e.dataDir
			projectID := ""
			var session domain.Session
			if s, err := e.sessions.Get(ctx, sessionID); err == nil {
				session = s
				workDir = e.resolveWorkDir(ctx, s.ProjectID)
				projectID = s.ProjectID
			}
			childCtx := TurnContext{
				SessionID: sessionID, TurnID: childTurnID,
				Agent: workerAgent, Model: modelID, MaxSteps: workerAgent.Steps,
				WorkDir: workDir,
			}
			e.stream.Publish(ctx, sessionID, parentTurnID, domain.EventDelegateStarted, domain.DelegateStartedPayload{
				AgentID: workerAgent.ID, Goal: goal, ChildTurnID: childTurnID,
			})

			e.mu.Lock()
			for _, id := range e.agentChain {
				if id == workerAgent.ID {
					e.mu.Unlock()
					return domain.Report{}, fmt.Errorf("circular delegation: %s", workerAgent.ID)
				}
			}
			if len(e.agentChain) >= cfg.teamMaxDelegationDepth {
				e.mu.Unlock()
				return domain.Report{}, fmt.Errorf("max delegation depth reached")
			}
			e.agentChain = append(e.agentChain, workerAgent.ID)
			e.mu.Unlock()
			defer func() {
				e.mu.Lock()
				for i := len(e.agentChain) - 1; i >= 0; i-- {
					if e.agentChain[i] == workerAgent.ID {
						e.agentChain = append(e.agentChain[:i], e.agentChain[i+1:]...)
						break
					}
				}
				e.mu.Unlock()
			}()

			oldReg := e.turnRunner.Registry
			oldSkills := e.turnRunner.SkillList
			oldBindings := e.turnRunner.ToolBindings
			oldLog := e.turnRunner.Log
			e.turnRunner.Registry = e.setupRegistry(session, workerAgent)
			defer func() {
				e.turnRunner.Registry = oldReg
				e.turnRunner.SkillList = oldSkills
				e.turnRunner.ToolBindings = oldBindings
				e.turnRunner.Log = oldLog
			}()

			sys := buildSystemPrompt(workerAgent.SystemPrompt, e.turnRunner.SkillList, nil, "")
			messages := []Message{
				{Role: RoleSystem, Content: sys},
			}
			if hits := e.knowledge.Search(workerAgent.KnowledgeIDs, goal, cfg.knowledgeSearchTopK); len(hits) > 0 {
				content := ""
				for _, h := range hits {
					content += h + "\n"
				}
				messages = append(messages, Message{Role: RoleSystem, Content: content})
			}
			messages = append(messages, Message{Role: RoleUser, Content: goal})
			childCtx.Messages = messages

			// Create turn log for child so LoadTurnLogZip can find it
			e.turnLog.Create(childTurnID, sessionID, projectID, workerAgent.ID, goal)
			_ = e.turns.Create(ctx, domain.TurnLog{ID: childTurnID, SessionID: sessionID, AgentID: workerAgent.ID, Goal: goal, Status: domain.TurnRunning})
			e.turnRunner.Log = func(typ string, data map[string]any) {
				e.turnLog.Append(childTurnID, typ, data)
			}

			e.stream.Publish(ctx, sessionID, childTurnID, domain.EventTurnStarted, domain.TurnStartedPayload{
				TurnID: childTurnID, AgentID: workerAgent.ID, Goal: goal,
			})
			e.stream.Publish(ctx, sessionID, childTurnID, domain.EventUserMessage, domain.UserMessagePayload{Content: goal})

			rep, _, err := e.turnRunner.Run(ctx, childCtx)
			e.turnLog.EndTurn(childTurnID, turnStatus(err, rep))
			status := string(rep.Status)
			if err != nil {
				status = "failed"
			}
			_ = e.turns.UpdateStatus(ctx, childTurnID, turnStatus(err, rep))
			e.stream.Publish(ctx, sessionID, parentTurnID, domain.EventDelegateCompleted, domain.DelegateCompletedPayload{
				AgentID: workerAgent.ID, Status: status, Summary: rep.Summary,
			})
			return rep, err
		},
	}
	reg := tool.NewRegistry(
		&builtin.SearchKB{Knowledge: e.knowledge, KBIDs: agent.KnowledgeIDs},
		&builtin.AskUser{
			Stream: e.stream,
			OnAsk: func(ctx context.Context, sessionID, turnID, callID, question string, options []string, defaultOpt string, formFields []domain.AskUserFormField) (string, error) {
				ch := make(chan string, 1)
				e.mu.Lock()
				e.askUserWait[callID] = ch
				e.mu.Unlock()
				e.stream.Publish(ctx, sessionID, turnID, domain.EventAskUserPending, domain.AskUserPayload{
					AskID: callID, CallID: callID, Question: question, Options: options, DefaultOpt: defaultOpt, FormFields: formFields,
				})
				select {
				case answer := <-ch:
					return answer, nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		},
		delegator,
	)
	reg.CopyMCPServersFrom(e.toolCatalog)
	e.mountBuiltinTools(reg, agent.Tools)
	reg.MountFromBindings(agent.Tools)
	return reg
}

func (e *Engine) buildWorkerRegistry(agent domain.Agent) *tool.Registry {
	reg := tool.NewRegistry(
		&builtin.SearchKB{Knowledge: e.knowledge, KBIDs: agent.KnowledgeIDs},
		&builtin.AskUser{
			Stream: e.stream,
			OnAsk: func(ctx context.Context, sessionID, turnID, callID, question string, options []string, defaultOpt string, formFields []domain.AskUserFormField) (string, error) {
				ch := make(chan string, 1)
				e.mu.Lock()
				e.askUserWait[callID] = ch
				e.mu.Unlock()
				e.stream.Publish(ctx, sessionID, turnID, domain.EventAskUserPending, domain.AskUserPayload{
					AskID: callID, CallID: callID, Question: question, Options: options, DefaultOpt: defaultOpt, FormFields: formFields,
				})
				select {
				case answer := <-ch:
					return answer, nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		},
	)
	reg.CopyMCPServersFrom(e.toolCatalog)
	e.mountBuiltinTools(reg, agent.Tools)
	reg.MountFromBindings(agent.Tools)
	return reg
}

func (e *Engine) StreamEvents(sessionID string, since int64) []domain.StreamEvent {
	return e.stream.ListSince(sessionID, since)
}
func (e *Engine) Subscribe(sessionID string) chan domain.StreamEvent {
	return e.stream.Subscribe(sessionID)
}
func (e *Engine) Unsubscribe(sessionID string, ch chan domain.StreamEvent) {
	e.stream.Unsubscribe(sessionID, ch)
}
func (e *Engine) ResolveApproval(id string, approved bool, scope string) {
	if scope == "" {
		scope = "once"
	}
	e.mu.Lock()
	ch := e.approvalWait[id]
	meta := e.approvalMeta[id]
	delete(e.approvalWait, id)
	delete(e.approvalMeta, id)
	if approved && scope == "session" && meta.Reason == permission.ReasonNetwork && meta.SessionID != "" {
		st := e.sessionPerm[meta.SessionID]
		st.AllowNetwork = true
		e.sessionPerm[meta.SessionID] = st
	}
	e.mu.Unlock()
	if ch != nil {
		select {
		case ch <- ApprovalOutcome{Approved: approved, Scope: scope, Reason: meta.Reason}:
		default:
		}
	}
}
func (e *Engine) WaitApproval(ctx context.Context, id string) (ApprovalOutcome, error) {
	if e.isAutoApprove() {
		return ApprovalOutcome{Approved: true, Scope: "once"}, nil
	}
	e.mu.Lock()
	ch := e.approvalWait[id]
	e.mu.Unlock()
	if ch == nil {
		return ApprovalOutcome{}, fmt.Errorf("approval not found")
	}
	select {
	case out := <-ch:
		return out, nil
	case <-ctx.Done():
		return ApprovalOutcome{}, ctx.Err()
	}
}
func (e *Engine) CreateApproval(sessionID, toolName, description, reason string) string {
	id := fmt.Sprintf("appr-%d", time.Now().UnixNano())
	ch := make(chan ApprovalOutcome, 1)
	e.mu.Lock()
	e.approvalWait[id] = ch
	e.approvalMeta[id] = approvalMeta{SessionID: sessionID, Reason: reason}
	e.mu.Unlock()
	_ = e.approvals.Create(context.Background(), domain.Approval{
		ID: id, SessionID: sessionID, ToolName: toolName,
		Summary: description, Description: description, Status: "pending", CreatedAt: time.Now().UTC(),
	})
	return id
}

func (e *Engine) ResolveAskUser(askID, answer string) {
	e.mu.Lock()
	ch := e.askUserWait[askID]
	delete(e.askUserWait, askID)
	e.mu.Unlock()
	if ch != nil {
		select {
		case ch <- answer:
		default:
		}
	}
}

func (e *Engine) buildTurnMessages(sessionID string, agent domain.Agent, goal string, checkpointText string) []Message {
	cfg := e.loadRunCfg(context.Background())
	sys := buildSystemPrompt(agent.SystemPrompt, e.turnRunner.SkillList, e.delegatableAgents(agent), checkpointText)
	messages := []Message{
		{Role: RoleSystem, Content: sys},
	}

	if hits := e.knowledge.Search(agent.KnowledgeIDs, goal, cfg.knowledgeSearchTopK); len(hits) > 0 {
		content := ""
		for _, h := range hits {
			content += h + "\n"
		}
		messages = append(messages, Message{Role: RoleSystem, Content: content})
	}

	e.mu.Lock()
	prevMsgs := e.turnMessages[sessionID]
	e.mu.Unlock()
	messages = append(messages, prevMsgs...)

	messages = append(messages, Message{Role: RoleUser, Content: goal})
	return messages
}

func turnStatus(err error, rep domain.Report) domain.TurnStatus {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return domain.TurnCancelled
		}
		return domain.TurnFailed
	}
	switch rep.Status {
	case domain.ReportDone:
		return domain.TurnCompleted
	case domain.ReportFailed, domain.ReportBlocked:
		return domain.TurnFailed
	default:
		return domain.TurnCompleted
	}
}
