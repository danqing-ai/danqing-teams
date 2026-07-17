package runtime

import (
	"context"
	"strings"
	"testing"

	"danqing-teams/core/adapter/llm"
	"danqing-teams/core/domain"
	"danqing-teams/core/runtime/permission"
	"danqing-teams/core/runtime/tool"
)

type mockToolHandler struct {
	name  string
	risk  domain.RiskLevel
	calls int
}

func (h *mockToolHandler) Name() string              { return h.name }
func (h *mockToolHandler) RiskLevel() domain.RiskLevel { return h.risk }
func (h *mockToolHandler) Describe(args map[string]any) string { return h.name }
func (h *mockToolHandler) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: h.name, Description: "mock tool",
		Parameters: map[string]any{
			"type": "object", "properties": map[string]any{},
		},
	}
}
func (h *mockToolHandler) Execute(_ context.Context, _ map[string]any) (domain.ToolResult, error) {
	h.calls++
	return domain.ToolResult{Content: "ok"}, nil
}

func TestTrackDoomConsecutiveNotCumulative(t *testing.T) {
	tr := NewTurnRunner(nil, nil, permission.NewGate(nil), tool.NewRegistry(), nil)
	const turnID = "t1"
	const threshold = 3

	// Interleaved todowrite/write should not trip consecutive OR short alternating.
	for i := 0; i < 3; i++ {
		n := tr.trackDoom(turnID, "todowrite", "todowrite", threshold)
		if n >= threshold {
			t.Fatalf("interleaved: unexpected doom after todowrite #%d streak=%d", i+1, n)
		}
		n = tr.trackDoom(turnID, "write", "write", threshold)
		if n >= threshold {
			t.Fatalf("interleaved: unexpected doom on write streak=%d", n)
		}
	}

	// Three consecutive identical calls should trip.
	tr2 := NewTurnRunner(nil, nil, permission.NewGate(nil), tool.NewRegistry(), nil)
	var last int
	for i := 0; i < 3; i++ {
		last = tr2.trackDoom("t2", "todowrite", "todowrite", threshold)
	}
	if last < threshold {
		t.Fatalf("expected consecutive doom streak>=%d got %d", threshold, last)
	}
}

func TestDetectAlternatingLoop(t *testing.T) {
	// Need >= 8 alternating (4 A-B pairs) with threshold 3
	pat := []string{"a", "b", "a", "b", "a", "b", "a", "b"}
	if !detectAlternatingLoop(pat, 3) {
		t.Fatal("expected alternating doom")
	}
	if detectAlternatingLoop([]string{"a", "b", "a", "b", "a", "b"}, 3) {
		t.Fatal("6-long A-B should not trip (min 8)")
	}
	if detectAlternatingLoop([]string{"a", "a", "a"}, 3) {
		t.Fatal("identical streak is not alternating")
	}
}

func TestTurnRunnerDoomLoopMessagesIntegrity(t *testing.T) {
	mockLLM := llm.NewMock()
	for i := 0; i < 5; i++ {
		mockLLM.AddToolCall("todowrite", map[string]any{"todos": []any{}})
	}

	stream := NewStreamEventManager(nil)
	perm := permission.NewGate(nil)
	reg := tool.NewRegistry()
	todowriteTool := &mockToolHandler{name: "todowrite", risk: domain.RiskLow}
	reg.Register(todowriteTool)

	configStore := &testConfigStore{
		cfg: &domain.ConfigFile{
			Runtime: domain.ConfigRuntimeSection{
				Turn: domain.ConfigTurnSection{
					DoomLoopThreshold: 5,
					MaxStepsDefault:   20,
				},
			},
		},
	}

	tr := NewTurnRunner(mockLLM, stream, perm, reg, configStore)
	ctx := context.Background()

	tctx := TurnContext{
		SessionID: "test-session",
		TurnID:    "turn-doom-1",
		Agent:     domain.Agent{ID: "test-agent", Steps: 20},
		Model:     "test-model",
		MaxSteps:  20,
		WorkDir:   "/tmp",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a test assistant"},
			{Role: RoleUser, Content: "do something"},
		},
	}

	rep, msgs, err := tr.Run(ctx, tctx)
	if err != nil {
		t.Fatalf("turn 1 unexpected error: %v", err)
	}
	if rep.Status != domain.ReportFailed {
		t.Errorf("turn 1: expected ReportFailed, got %v", rep.Status)
	}
	if rep.Summary != "doom loop for todowrite" {
		t.Errorf("turn 1: expected 'doom loop for todowrite', got %q", rep.Summary)
	}
	if todowriteTool.calls < 4 {
		t.Errorf("turn 1: expected at least 4 todowrite calls before doom loop, got %d", todowriteTool.calls)
	}

	validateToolMessagePairs(t, msgs, "turn 1")

	mockLLM2 := llm.NewMock().AddText("all done")
	stream2 := NewStreamEventManager(nil)
	reg2 := tool.NewRegistry()
	reg2.Register(&mockToolHandler{name: "todowrite", risk: domain.RiskLow})
	tr2 := NewTurnRunner(mockLLM2, stream2, perm, reg2, configStore)

	tctx2 := TurnContext{
		SessionID: "test-session",
		TurnID:    "turn-doom-2",
		Agent:     domain.Agent{ID: "test-agent", Steps: 20},
		Model:     "test-model",
		MaxSteps:  20,
		WorkDir:   "/tmp",
		Messages: append(append([]Message(nil), msgs...), Message{Role: RoleUser, Content: "continue"}),
	}

	rep2, msgs2, err2 := tr2.Run(ctx, tctx2)
	if err2 != nil {
		t.Fatalf("turn 2 unexpected error: %v", err2)
	}
	if rep2.Status != domain.ReportDone {
		t.Errorf("turn 2: expected ReportDone, got %v: %s", rep2.Status, rep2.Summary)
	}

	validateToolMessagePairs(t, msgs2, "turn 2")
}

func TestTurnRunnerApprovalRejectContinues(t *testing.T) {
	mockLLM := llm.NewMock().
		AddToolCall("exec_shell", map[string]any{"command": "ls"}).
		AddText("understood, will use a safer approach")

	stream := NewStreamEventManager(nil)
	perm := permission.NewGate(nil)
	reg := tool.NewRegistry()
	reg.Register(&mockToolHandler{name: "exec_shell", risk: domain.RiskHigh})

	configStore := &testConfigStore{
		cfg: &domain.ConfigFile{
			Runtime: domain.ConfigRuntimeSection{
				Turn: domain.ConfigTurnSection{
					DoomLoopThreshold: 5,
					MaxStepsDefault:   20,
				},
			},
		},
	}

	tr := NewTurnRunner(mockLLM, stream, perm, reg, configStore)

	approved := make(chan ApprovalOutcome, 1)
	approved <- ApprovalOutcome{Approved: false, Scope: "once"}
	tr.Approval = &mockApprovalGate{result: approved}

	ctx := context.Background()
	tctx := TurnContext{
		SessionID: "test-session",
		TurnID:    "turn-approval-1",
		Agent:     domain.Agent{ID: "test-agent", Steps: 20},
		Model:     "test-model",
		MaxSteps:  20,
		WorkDir:   "/tmp",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a test assistant"},
			{Role: RoleUser, Content: "run ls"},
		},
	}

	rep, msgs, err := tr.Run(ctx, tctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Status != domain.ReportDone {
		t.Fatalf("expected ReportDone after soft reject, got %v: %s", rep.Status, rep.Summary)
	}
	foundReject := false
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "rejected") {
			foundReject = true
		}
	}
	if !foundReject {
		t.Fatal("expected tool message containing rejection")
	}
	validateToolMessagePairs(t, msgs, "approval soft reject")
}

type mockApprovalGate struct {
	result chan ApprovalOutcome
}

func (g *mockApprovalGate) WaitApproval(_ context.Context, _ string) (ApprovalOutcome, error) {
	return <-g.result, nil
}

func (g *mockApprovalGate) CreateApproval(_, _, _, _ string) string {
	return "approval-1"
}

func validateToolMessagePairs(t *testing.T, msgs []Message, label string) {
	t.Helper()

	toolByID := make(map[string]bool)
	assistantIDs := make(map[string]int)

	for _, m := range msgs {
		if m.Role == RoleTool && m.ToolCallID != "" {
			toolByID[m.ToolCallID] = true
		}
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				assistantIDs[tc.ID]++
			}
		}
	}

	for id := range assistantIDs {
		if !toolByID[id] {
			t.Errorf("%s: assistant tool_calls ID %q has no corresponding tool message", label, id)
		}
	}

	for id := range toolByID {
		if _, ok := assistantIDs[id]; !ok {
			t.Errorf("%s: orphan tool message for call ID %q (no matching assistant tool_calls)", label, id)
		}
	}
}
