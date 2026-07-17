package turnlog

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"danqing-teams/core/domain"
)

func testProjector(root string) func(string) string {
	return func(projectID string) string {
		if projectID == "" {
			return root
		}
		return filepath.Join(root, projectID)
	}
}

func writeToolPair(s *TurnLogStore, turnID, callID, name string) {
	s.Append(turnID, "tool_call", map[string]any{
		"call_id": callID, "name": name, "input": map[string]any{"x": 1},
	})
	s.Append(turnID, "tool_result", map[string]any{
		"call_id": callID, "name": name, "output": "ok",
	})
}

func countTypes(entries []map[string]any) (calls, results int) {
	for _, e := range entries {
		switch e["type"] {
		case "tool_call":
			calls++
		case "tool_result":
			results++
		}
	}
	return calls, results
}

func TestLoadForRecoveryDropsUnpairedTrailingToolCall(t *testing.T) {
	root := t.TempDir()
	s := NewTurnLogStore(testProjector(root))

	if err := s.Create("turn-1", "sess-1", "proj-a", "agent-1", "do stuff"); err != nil {
		t.Fatal(err)
	}
	writeToolPair(s, "turn-1", "c1", "read_file")
	writeToolPair(s, "turn-1", "c2", "exec_shell")
	// Interrupted after tool_call logged, before tool_result.
	s.Append("turn-1", "tool_call", map[string]any{
		"call_id": "c3", "name": "write_file", "input": map[string]any{"path": "x"},
	})
	s.EndTurn("turn-1", domain.TurnCancelled)

	goal, entries := s.LoadForRecovery("turn-1")
	if goal != "do stuff" {
		t.Fatalf("goal: want %q, got %q", "do stuff", goal)
	}
	calls, results := countTypes(entries)
	if calls != 2 || results != 2 {
		t.Fatalf("want 2 paired tool calls/results, got calls=%d results=%d entries=%d", calls, results, len(entries))
	}
	// Trailing unpaired call must not appear.
	for _, e := range entries {
		if e["type"] == "tool_call" {
			data, _ := e["data"].(map[string]any)
			if id, _ := data["call_id"].(string); id == "c3" {
				t.Fatal("unpaired trailing tool_call c3 should have been dropped")
			}
		}
	}
}

func TestLoadForRecoveryKeepsCompletePairs(t *testing.T) {
	root := t.TempDir()
	s := NewTurnLogStore(testProjector(root))

	if err := s.Create("turn-2", "sess-1", "proj-a", "agent-1", "goal"); err != nil {
		t.Fatal(err)
	}
	writeToolPair(s, "turn-2", "c1", "read_file")
	s.EndTurn("turn-2", domain.TurnCompleted)

	goal, entries := s.LoadForRecovery("turn-2")
	if goal != "goal" {
		t.Fatalf("goal: %q", goal)
	}
	calls, results := countTypes(entries)
	if calls != 1 || results != 1 {
		t.Fatalf("want 1 pair, got calls=%d results=%d", calls, results)
	}
}

func TestCreateReopensWithoutDuplicateStart(t *testing.T) {
	root := t.TempDir()
	s := NewTurnLogStore(testProjector(root))

	if err := s.Create("turn-3", "sess-1", "proj-a", "agent-1", "goal"); err != nil {
		t.Fatal(err)
	}
	writeToolPair(s, "turn-3", "c1", "read_file")
	s.EndTurn("turn-3", domain.TurnCancelled)

	// Resume: Create should reopen append, not write another start.
	if err := s.Create("turn-3", "sess-1", "proj-a", "agent-1", "goal"); err != nil {
		t.Fatal(err)
	}
	s.Append("turn-3", "tool_call", map[string]any{
		"call_id": "c2", "name": "exec_shell", "input": map[string]any{},
	})
	s.Append("turn-3", "tool_result", map[string]any{
		"call_id": "c2", "name": "exec_shell", "output": "done",
	})
	s.EndTurn("turn-3", domain.TurnCompleted)

	raw, err := s.LoadRawLog("turn-3")
	if err != nil {
		t.Fatal(err)
	}
	starts := 0
	dec := json.NewDecoder(bytes.NewReader(raw))
	for dec.More() {
		var e EntryJSON
		if err := dec.Decode(&e); err != nil {
			t.Fatal(err)
		}
		if e.Type == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("want exactly 1 start entry after resume, got %d", starts)
	}
}

func TestLoadForRecoveryRehydratesFromDisk(t *testing.T) {
	root := t.TempDir()
	s1 := NewTurnLogStore(testProjector(root))
	if err := s1.Create("turn-4", "sess-1", "proj-a", "agent-1", "disk-goal"); err != nil {
		t.Fatal(err)
	}
	writeToolPair(s1, "turn-4", "c1", "read_file")
	s1.EndTurn("turn-4", domain.TurnCancelled)

	// Simulate process restart: new store, empty memory.
	s2 := NewTurnLogStore(testProjector(root))
	goal, entries := s2.LoadForRecovery("turn-4")
	if goal != "disk-goal" {
		t.Fatalf("goal from disk: want %q, got %q", "disk-goal", goal)
	}
	calls, results := countTypes(entries)
	if calls != 1 || results != 1 {
		t.Fatalf("disk recovery pairs: calls=%d results=%d", calls, results)
	}

	// Resume Create on fresh store should reopen existing file.
	if err := s2.Create("turn-4", "sess-1", "proj-a", "agent-1", "disk-goal"); err != nil {
		t.Fatal(err)
	}
	s2.Append("turn-4", "tool_call", map[string]any{
		"call_id": "c2", "name": "exec_shell", "input": map[string]any{},
	})
	s2.EndTurn("turn-4", domain.TurnCompleted)

	listed := s2.ListEntries("turn-4")
	starts := 0
	for _, e := range listed {
		if e.Type == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("after disk reopen want 1 start, got %d", starts)
	}
}
