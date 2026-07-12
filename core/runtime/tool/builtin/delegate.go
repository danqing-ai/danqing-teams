package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
	"danqing-teams/core/service"
)

type ListAgents struct {
	Agents *service.AgentManager
}

func (h *ListAgents) Name() string                { return "list_agents" }
func (h *ListAgents) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *ListAgents) Describe(args map[string]any) string {
	return "list_agents"
}
func (h *ListAgents) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "list_agents",
		Description: "List all available agents for delegation.\n\n" +
			"- Use before delegating to discover available agents, their IDs, and descriptions.\n" +
			"- Returns a JSON array of agents with id, name, and description fields.",
		Parameters: map[string]any{"type": "object"},
	}
}
func (h *ListAgents) Execute(ctx context.Context, _ map[string]any) (domain.ToolResult, error) {
	agents, err := h.Agents.List(ctx)
	if err != nil {
		return domain.ToolResult{}, err
	}
	var result []map[string]string
	for _, a := range agents {
		result = append(result, map[string]string{"id": a.ID, "name": a.Name, "description": a.Description})
	}
	raw, _ := json.Marshal(result)
	return domain.ToolResult{Content: string(raw)}, nil
}

type DelegateAgent struct {
	Stream          port.EventStream
	Agents          *service.AgentManager
	KnowledgeSearch func(kbIDs []string, query string, k int) []string
	RunSubTurn      func(ctx context.Context, sessionID, modelID, parentTurnID string, agent domain.Agent, goal string) (domain.Report, error)
}

func (h *DelegateAgent) Name() string                { return "delegate_agent" }
func (h *DelegateAgent) RiskLevel() domain.RiskLevel { return domain.RiskLow }
func (h *DelegateAgent) Describe(args map[string]any) string {
	agentID, _ := args["agent_id"].(string)
	goal, _ := args["goal"].(string)
	if len(goal) > 80 {
		goal = goal[:80] + "..."
	}
	return agentID + ": " + goal
}
func (h *DelegateAgent) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "delegate_agent",
		Description: "Delegate a task to a subagent and get back a structured report.\n\n" +
			"- agent_id: the ID of the agent to delegate to (use list_agents first).\n" +
			"- goal: a clear, specific description of what the subagent should accomplish.\n" +
			"- context: optional additional background information.\n" +
			"- Assign complete subtasks, not single actions — let subagents decide how to use their tools.\n" +
			"- Launch multiple subagents in parallel when their work is independent.\n" +
			"- Read subagent reports before deciding the next step.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent_id": map[string]any{"type": "string"},
				"goal":     map[string]any{"type": "string"},
				"context":  map[string]any{"type": "string"},
			},
			"required": []string{"agent_id", "goal"},
		},
	}
}

func (h *DelegateAgent) Execute(ctx context.Context, input map[string]any) (domain.ToolResult, error) {
	agentID, _ := input["agent_id"].(string)
	goal, _ := input["goal"].(string)
	if extra, ok := input["context"].(string); ok && extra != "" {
		goal = goal + "\nContext: " + extra
	}
	sessionID, _ := input["__session_id"].(string)
	modelID, _ := input["__model_id"].(string)
	parentTurnID, _ := input["__turn_id"].(string)

	agent, err := h.Agents.Get(ctx, agentID)
	if err != nil {
		return domain.ToolResult{}, fmt.Errorf("unknown agent %q", agentID)
	}

	report, err := h.RunSubTurn(ctx, sessionID, modelID, parentTurnID, *agent, goal)
	if err != nil {
		return domain.ToolResult{}, err
	}
	report.WorkerID = agent.ID
	report.WorkerName = agent.Name
	report.TraceID = parentTurnID

	raw, _ := json.Marshal(report)
	return domain.ToolResult{Content: formatSessionResult(report), Meta: map[string]any{"report": string(raw)}}, nil
}

func formatSessionResult(report domain.Report) string {
	b, _ := json.Marshal(report)
	return fmt.Sprintf("<session_result>%s</session_result>", string(b))
}
