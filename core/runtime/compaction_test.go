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
	mgr := NewCompactionManager(mock, stream, store, cpStore, nil)

	if !mgr.ShouldCompact("session-1", 3, 0, 0, "") {
		t.Error("ShouldCompact should return true for turnCount=3 with interval=3")
	}
	if mgr.ShouldCompact("session-1", 2, 0, 0, "") {
		t.Error("ShouldCompact should return false for turnCount=2 with interval=3")
	}
}

func TestCompactionShouldCompactWithActualUsage(t *testing.T) {
	mock := llm.NewMock().Finish("done")
	stream := NewStreamEventManager(nil)

	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })
	// maxTokens=0 means no config override; model context window drives the limit.
	store := testCompactionConfig(true, 100, 100, 0, 16000)
	mgr := NewCompactionManager(mock, stream, store, cpStore, nil)

	// deepseek/deepseek-v4-flash: context=1M, maxOutput=384K, usable=616K, trigger@0.85=523K
	model := "deepseek/deepseek-v4-flash"

	// Below threshold: should NOT compact
	if mgr.ShouldCompact("s1", 1, 0, 400_000, model) {
		t.Error("should NOT compact when actual tokens (400K) < trigger threshold (~523K)")
	}

	// Above threshold: should compact
	if !mgr.ShouldCompact("s1", 1, 0, 600_000, model) {
		t.Error("should compact when actual tokens (600K) > trigger threshold (~523K)")
	}

	// No actual usage (maxPromptTokens=0), falls back to estimation + turn count
	if mgr.ShouldCompact("s1", 1, 50000, 0, model) {
		t.Error("should NOT compact with estimation only when turn count < interval")
	}
}

func TestModelLimitsRegistryContextWindow(t *testing.T) {
	reg := NewModelLimitsRegistry()

	tests := []struct {
		modelID  string
		expected int
	}{
		{"openai/gpt-4o", 128_000},
		{"openai/gpt-4.1", 1_047_576},
		{"openai/o3-mini", 200_000},
		{"deepseek/deepseek-v4-flash", 1_000_000},
		{"deepseek/deepseek-chat", 64_000},
		{"anthropic/claude-sonnet-4-20250514", 200_000},
		{"google/gemini-2.5-pro", 1_048_576},
		{"qwen/qwen-long", 10_000_000},
		{"unknown/some-model", 128_000}, // fallback
	}
	for _, tt := range tests {
		got := reg.ContextWindow(tt.modelID)
		if got != tt.expected {
			t.Errorf("ContextWindow(%q) = %d, want %d", tt.modelID, got, tt.expected)
		}
	}
}

func TestModelLimitsRegistryConfigOverride(t *testing.T) {
	reg := NewModelLimitsRegistry()
	reg.SetLimits([]domain.ModelLimit{
		{Model: "gpt-4o", ContextWindow: 256_000, MaxOutput: 32_000},
		{Model: "custom-model", ContextWindow: 500_000, MaxOutput: 50_000},
	})

	// Config override takes priority over hardcoded pattern
	if got := reg.ContextWindow("openai/gpt-4o"); got != 256_000 {
		t.Errorf("config override: got %d, want 256000", got)
	}
	if got := reg.MaxOutputTokens("openai/gpt-4o"); got != 32_000 {
		t.Errorf("config override maxOutput: got %d, want 32000", got)
	}

	// New model from config
	if got := reg.ContextWindow("provider/custom-model"); got != 500_000 {
		t.Errorf("custom model: got %d, want 500000", got)
	}

	// Non-overridden model still uses hardcoded fallback
	if got := reg.ContextWindow("deepseek/deepseek-v4-flash"); got != 1_000_000 {
		t.Errorf("non-overridden: got %d, want 1000000", got)
	}
}

func TestCompactionWithConfigModelLimits(t *testing.T) {
	mock := llm.NewMock().Finish("done")
	stream := NewStreamEventManager(nil)

	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })

	// Config with model limit override
	store := &testConfigStore{
		cfg: &domain.ConfigFile{
			Runtime: domain.ConfigRuntimeSection{
				Compaction: domain.ConfigCompactionSection{
					Enabled:      true,
					TurnInterval: 100,
					SubInterval:  100,
					TriggerRatio: 0.85,
					ToolTruncate: 2000,
				},
			},
			LLM: domain.ConfigLLMSection{
				ModelLimits: []domain.ModelLimit{
					{Model: "test-model", ContextWindow: 32_000, MaxOutput: 4_000},
				},
			},
		},
	}
	mgr := NewCompactionManager(mock, stream, store, cpStore, nil)

	// test-model: context=32K, maxOutput=4K, usable=28K, trigger@0.85=23.8K
	model := "provider/test-model"

	// Below threshold
	if mgr.ShouldCompact("s1", 1, 0, 20_000, model) {
		t.Error("should NOT compact when tokens (20K) < threshold (~23.8K)")
	}

	// Above threshold
	if !mgr.ShouldCompact("s1", 1, 0, 25_000, model) {
		t.Error("should compact when tokens (25K) > threshold (~23.8K)")
	}
}

func TestCompactionCompactAndRecover(t *testing.T) {
	mock := llm.NewMock().Finish("done").Finish("done").
		AddText("summary: test compaction completed")
	stream := NewStreamEventManager(nil)
	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })

	mgr := NewCompactionManager(mock, stream, testCompactionConfig(true, 2, 2, 128000, 50), cpStore, nil)

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
