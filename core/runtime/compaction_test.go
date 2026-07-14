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
	// Config with model entry for deepseek-v4-flash: context=1M, maxOutput=384K
	store := &testConfigStore{
		cfg: &domain.ConfigFile{
			Runtime: domain.ConfigRuntimeSection{
				Compaction: domain.ConfigCompactionSection{
					Enabled:      true,
					TurnInterval: 100,
					SubInterval:  100,
					TriggerRatio: 0.85,
					ToolTruncate: 16000,
				},
			},
			LLM: domain.ConfigLLMSection{
				Models: []domain.ModelConfig{
					{Model: "deepseek-v4-flash", ContextWindow: 1_000_000, MaxOutput: 384_000},
				},
			},
		},
	}
	modelCfg := NewModelConfigRegistry()
	mgr := NewCompactionManager(mock, stream, store, cpStore, modelCfg)

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

func TestModelConfigRegistryContextWindow(t *testing.T) {
	reg := NewModelConfigRegistry()
	reg.SetModels([]domain.ModelConfig{
		{Model: "gpt-4o", ContextWindow: 128_000},
		{Model: "gpt-4.1", ContextWindow: 1_047_576},
		{Model: "o3-mini", ContextWindow: 200_000},
		{Model: "deepseek-v4-flash", ContextWindow: 1_000_000},
		{Model: "deepseek-chat", ContextWindow: 64_000},
		{Model: "claude-sonnet-4-20250514", ContextWindow: 200_000},
		{Model: "gemini-2.5-pro", ContextWindow: 1_048_576},
		{Model: "qwen-long", ContextWindow: 10_000_000},
	})

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
		{"unknown/some-model", 128_000}, // fallback to default
	}
	for _, tt := range tests {
		got := reg.ContextWindow(tt.modelID)
		if got != tt.expected {
			t.Errorf("ContextWindow(%q) = %d, want %d", tt.modelID, got, tt.expected)
		}
	}
}

func TestModelConfigRegistryConfigOverride(t *testing.T) {
	reg := NewModelConfigRegistry()
	reg.SetModels([]domain.ModelConfig{
		{Model: "gpt-4o", ContextWindow: 128_000, MaxOutput: 16_384, Temperature: 0.5},
		{Model: "custom-model", Temperature: 0.8, TopP: 0.95},
		{Model: "deepseek-v4-flash", ContextWindow: 1_000_000, MaxOutput: 384_000},
	})

	// Config exact match returns gen params
	gp := reg.GenParams("openai/gpt-4o")
	if gp == nil || gp.Temperature != 0.5 {
		t.Errorf("config override gen params: got %v, want temperature=0.5", gp)
	}

	// Custom model from config
	gp2 := reg.GenParams("provider/custom-model")
	if gp2 == nil || gp2.Temperature != 0.8 || gp2.TopP != 0.95 {
		t.Errorf("custom model gen params: got %v, want temperature=0.8 top_p=0.95", gp2)
	}

	// Non-configured model returns nil
	gp3 := reg.GenParams("unknown/some-model")
	if gp3 != nil {
		t.Errorf("non-configured gen params: got %v, want nil", gp3)
	}

	// Context window from config
	if got := reg.ContextWindow("openai/gpt-4o"); got != 128_000 {
		t.Errorf("context window: got %d, want 128000", got)
	}
	if got := reg.ContextWindow("deepseek/deepseek-v4-flash"); got != 1_000_000 {
		t.Errorf("context window: got %d, want 1000000", got)
	}
	// Non-configured model falls back to default
	if got := reg.ContextWindow("unknown/some-model"); got != 128_000 {
		t.Errorf("non-configured context window: got %d, want 128000", got)
	}
}

func TestCompactionWithConfigModels(t *testing.T) {
	mock := llm.NewMock().Finish("done")
	stream := NewStreamEventManager(nil)

	tmpDir := t.TempDir()
	cpStore := turnlog.NewCheckpointStore(func(pid string) string { return filepath.Join(tmpDir, pid) })

	// Config with model generation param override
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
				Models: []domain.ModelConfig{
					{Model: "test-model", Temperature: 0.7},
				},
			},
		},
	}
	mgr := NewCompactionManager(mock, stream, store, cpStore, nil)

	// test-model has no context window pattern rule, so falls back to default 128K
	// default 128K - 8192 maxOutput = 119808 usable, trigger@0.85 = ~101837
	model := "provider/test-model"

	// Below threshold
	if mgr.ShouldCompact("s1", 1, 0, 80_000, model) {
		t.Error("should NOT compact when tokens (80K) < threshold (~101K)")
	}

	// Above threshold
	if !mgr.ShouldCompact("s1", 1, 0, 110_000, model) {
		t.Error("should compact when tokens (110K) > threshold (~101K)")
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
