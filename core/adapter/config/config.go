package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var _ port.SearchConfigStore = (*Loader)(nil)
var _ port.ConfigStore = (*Loader)(nil)

// Loader reads and writes the user-editable .dq-teams/config.yaml configuration.
// It is the source of truth for settings that should be readable and editable
// by all entry points (server, cli, tui). Viper is used for loading, defaults,
// and environment-variable binding; yaml.v3 is used for writing so that only
// the touched sections are persisted and other fields are preserved.
type Loader struct {
	path string
	v    *viper.Viper
	mu   sync.RWMutex
}

// NewLoader returns a config loader for the given path.
// If path is empty, it defaults to .dq-teams/config.yaml (config dir).
// Data paths (data.dir, data.database) are always resolved relative to cwd,
// keeping config (.dq-teams/) and data (data/) cleanly separated.
func NewLoader(path string) *Loader {
	if path == "" {
		path = ".dq-teams/config.yaml"
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	setDefaults(v)
	bindEnv(v)
	return &Loader{path: path, v: v}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("data.dir", "./data")
	v.SetDefault("data.database", "./data/teams.db")
	v.SetDefault("data.store", "sqlite")
	v.SetDefault("server.listen_addr", "0.0.0.0:7801")
	v.SetDefault("instance.id", "")
	v.SetDefault("runtime.auto_approve", false)
	v.SetDefault("runtime.turn.doom_loop_threshold", 5)
	v.SetDefault("runtime.turn.max_steps_default", 20)
	v.SetDefault("runtime.team.max_delegation_depth", 3)
	v.SetDefault("runtime.memory.recall_top_k", 3)
	v.SetDefault("runtime.knowledge.search_top_k", 3)
	v.SetDefault("runtime.compaction.enabled", false)
	v.SetDefault("runtime.compaction.model", "")
	v.SetDefault("runtime.compaction.max_tokens", 128000)
	v.SetDefault("runtime.compaction.trigger_ratio", 0.85)
	v.SetDefault("runtime.compaction.cut_tokens", 16000)
	v.SetDefault("runtime.compaction.turn_interval", 6)
	v.SetDefault("runtime.compaction.sub_interval", 4)
	v.SetDefault("runtime.compaction.tool_truncate", 2000)
	v.SetDefault("search.provider", string(domain.SearchProviderDuckDuckGo))
	v.SetDefault("search.base_url", "")
	v.SetDefault("search.api_key", "")
	v.SetDefault("search.timeout_ms", 15000)
	v.SetDefault("search.max_results", 5)
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("data.dir", "TEAMS_DATA_DIR")
	_ = v.BindEnv("data.database", "TEAMS_DB_PATH")
	_ = v.BindEnv("data.store", "TEAMS_STORE")
	_ = v.BindEnv("server.listen_addr", "TEAMS_ADDR")
	_ = v.BindEnv("runtime.auto_approve", "TEAMS_AUTO_APPROVE")
	_ = v.BindEnv("instance.id", "TEAMS_INSTANCE_ID")
}

// Path returns the resolved config file path.
func (l *Loader) Path() string { return l.path }

// Load reads the configuration file (if it exists), applies defaults and
// environment-variable overrides, and returns the resolved config.
func (l *Loader) Load(_ context.Context) (*domain.ConfigFile, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if err := l.v.ReadInConfig(); err != nil {
		// Ignore "file not found" so that defaults + env vars still work.
		if !isConfigNotFound(err) {
			return nil, err
		}
	}

	var cfg domain.ConfigFile
	if err := l.v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cwd, _ := os.Getwd()

	if cfg.Data.Dir == "" {
		cfg.Data.Dir = "./data"
	}
	if !filepath.IsAbs(cfg.Data.Dir) {
		cfg.Data.Dir = filepath.Join(cwd, cfg.Data.Dir)
	}
	if cfg.Data.Database == "" {
		cfg.Data.Database = cfg.Data.Dir + "/teams.db"
	}
	if !filepath.IsAbs(cfg.Data.Database) {
		cfg.Data.Database = filepath.Join(cwd, cfg.Data.Database)
	}

	if cfg.Search.Provider == "" {
		cfg.Search.Provider = domain.SearchProviderDuckDuckGo
	}

	// Fill in default LLM presets if none are configured.
	if len(cfg.LLM.Providers) == 0 {
		cfg.LLM.Providers = defaultLLMPresets()
	}
	return &cfg, nil
}

// defaultLLMPresets returns the built-in provider presets for mainstream
// model vendors. Users can override these in .dq-teams/config.yaml.
func defaultLLMPresets() []domain.LLMProviderPreset {
	return []domain.LLMProviderPreset{
		{ID: "openai", Name: "OpenAI", Provider: "openai", BaseURL: "https://api.openai.com/v1", Icon: "🟢", Description: "GPT 系列、o 系列推理模型"},
		{ID: "anthropic", Name: "Anthropic", Provider: "anthropic", BaseURL: "https://api.anthropic.com", Icon: "🟠", Description: "Claude Sonnet、Opus、Haiku"},
		{ID: "deepseek", Name: "DeepSeek", Provider: "openai", BaseURL: "https://api.deepseek.com", Icon: "🔵", Description: "DeepSeek 系列"},
		{ID: "google", Name: "Google Gemini", Provider: "openai", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Icon: "🔷", Description: "Gemini Pro、Flash"},
		{ID: "zhipu", Name: "智谱 (Zhipu)", Provider: "openai", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Icon: "🟣", Description: "GLM 系列"},
		{ID: "qwen", Name: "通义千问 (Qwen)", Provider: "openai", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Icon: "🟡", Description: "Qwen Max、Plus、Turbo"},
		{ID: "moonshot", Name: "Moonshot (Kimi)", Provider: "openai", BaseURL: "https://api.kimi.com/coding/v1", Icon: "🌙", Description: "Kimi 系列"},
		{ID: "ollama", Name: "Ollama (Local)", Provider: "openai", BaseURL: "http://localhost:11434/v1", Icon: "🦙", Description: "本地模型，通过 Ollama 运行"},
	}
}

// Save writes the full configuration back to the YAML file.
func (l *Loader) Save(_ context.Context, cfg *domain.ConfigFile) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if cfg == nil {
		return nil
	}

	root := map[string]any{}
	if data, err := os.ReadFile(l.path); err == nil {
		_ = yaml.Unmarshal(data, &root)
	} else if !os.IsNotExist(err) {
		return err
	}

	root["data"] = cfg.Data
	root["server"] = cfg.Server
	root["instance"] = cfg.Instance
	root["runtime"] = cfg.Runtime
	root["search"] = cfg.Search
	root["llm"] = cfg.LLM

	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return err
	}
	out, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, out, 0600)
}

// Get returns the search configuration for the app manager.
func (l *Loader) Get(ctx context.Context) (domain.SearchConfig, error) {
	cfg, err := l.Load(ctx)
	if err != nil {
		return domain.SearchConfig{}, err
	}
	return cfg.Search, nil
}

// Upsert persists the search configuration to the YAML file, preserving all
// other top-level keys.
func (l *Loader) Upsert(ctx context.Context, cfg domain.SearchConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	root := map[string]any{}
	if data, err := os.ReadFile(l.path); err == nil {
		_ = yaml.Unmarshal(data, &root)
	} else if !os.IsNotExist(err) {
		return err
	}

	search, _ := root["search"].(map[string]any)
	if search == nil {
		search = map[string]any{}
		root["search"] = search
	}

	search["provider"] = string(cfg.Provider)
	if cfg.BaseURL != "" {
		search["base_url"] = cfg.BaseURL
	} else {
		delete(search, "base_url")
	}
	if cfg.APIKey != "" {
		search["api_key"] = cfg.APIKey
	} else {
		delete(search, "api_key")
	}
	search["timeout_ms"] = cfg.TimeoutMs
	search["max_results"] = cfg.MaxResults

	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return err
	}
	out, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, out, 0600)
}

func isConfigNotFound(err error) bool {
	if err == nil {
		return true
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return true
	}
	return os.IsNotExist(err)
}
