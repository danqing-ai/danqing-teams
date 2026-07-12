package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"danqing-teams/core/adapter/llm"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"
	"danqing-teams/core/store/turnlog"
)

type testConfigStore struct {
	cfg *domain.ConfigFile
}

func (s *testConfigStore) Load(_ context.Context) (*domain.ConfigFile, error) {
	return s.cfg, nil
}

func (s *testConfigStore) Save(_ context.Context, _ *domain.ConfigFile) error {
	return nil
}

var _ port.ConfigStore = (*testConfigStore)(nil)

func testCompactionConfig(enabled bool, turnInterval, subInterval, maxTokens, cutTokens int) *testConfigStore {
	return &testConfigStore{
		cfg: &domain.ConfigFile{
			Runtime: domain.ConfigRuntimeSection{
				Compaction: domain.ConfigCompactionSection{
					Enabled:       enabled,
					TurnInterval:  turnInterval,
					SubInterval:   subInterval,
					MaxTokens:     maxTokens,
					CutTokens:     cutTokens,
					TriggerRatio:  0.85,
					ToolTruncate:  2000,
				},
			},
		},
	}
}

func TestCompactionShouldCompact(t *testing.T) {
	mock := llm.NewMock().Finish("done")
	stream := NewStreamEventManager(nil)

	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })
	store := testCompactionConfig(true, 3, 2, 128000, 16000)
	mgr := NewCompactionManager(mock, stream, store, cpStore)

	if !mgr.ShouldCompact("session-1", 3, 0) {
		t.Error("ShouldCompact should return true for turnCount=3 with interval=3")
	}
	if mgr.ShouldCompact("session-1", 2, 0) {
		t.Error("ShouldCompact should return false for turnCount=2 with interval=3")
	}
}

func TestCompactionCompactAndRecover(t *testing.T) {
	mock := llm.NewMock().Finish("done").Finish("done").
		AddText("summary: test compaction completed")
	stream := NewStreamEventManager(nil)
	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })

	mgr := NewCompactionManager(mock, stream, testCompactionConfig(true, 2, 2, 128000, 50), cpStore)

	messages := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant"},
		{Role: RoleUser, Content: "Hello, this is a very long message that should exceed the token cut limit so compaction triggers"},
		{Role: RoleAssistant, Content: "I understand. " + string(make([]byte, 200))},
		{Role: RoleUser, Content: "Another long message to ensure enough tokens for the cut point"},
		{Role: RoleAssistant, Content: "Responding again with " + string(make([]byte, 200))},
	}

	cutIdx := mgr.Compact(context.Background(), "session-1", "turn-test", messages, 2, "mock/test")
	if cutIdx <= 0 {
		t.Error("Compact should return non-zero cut index")
	}

	cp := mgr.Recover(context.Background(), "session-1")
	if cp == nil {
		t.Fatal("Recover should return checkpoint after compaction")
	}
	if cp.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want session-1", cp.SessionID)
	}
	if cp.TurnCount != 2 {
		t.Errorf("TurnCount = %d, want 2", cp.TurnCount)
	}
	if cp.TurnID != "turn-test" {
		t.Errorf("TurnID = %q, want turn-test", cp.TurnID)
	}
	if cp.Summary == "" {
		t.Error("Summary should not be empty")
	}

	loaded, err := cpStore.Load("session-1")
	if err != nil {
		t.Fatalf("Load from store: %v", err)
	}
	if loaded == nil {
		t.Fatal("Checkpoint should be persisted to store")
	}
	if loaded.Summary != cp.Summary {
		t.Errorf("Loaded summary = %q, want %q", loaded.Summary, cp.Summary)
	}
}
