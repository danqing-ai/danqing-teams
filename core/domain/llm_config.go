package domain

type LLMProviderType string

const (
	LLMProviderOpenAI    LLMProviderType = "openai"
	LLMProviderAnthropic LLMProviderType = "anthropic"
	LLMProviderLocal     LLMProviderType = "local"
	LLMProviderMock      LLMProviderType = "mock"
)

type LLMModelRef struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type LLMProviderConfig struct {
	ID        string          `json:"id"`
	Provider  LLMProviderType `json:"provider"`
	Name      string          `json:"name"`
	APIKey    string          `json:"apiKey,omitempty"`
	BaseURL   string          `json:"baseUrl,omitempty"`
	Models    []LLMModelRef   `json:"models,omitempty"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

type UpsertLLMProviderConfigRequest struct {
	Provider LLMProviderType `json:"provider"`
	Name     string          `json:"name"`
	APIKey   string          `json:"apiKey,omitempty"`
	BaseURL  string          `json:"baseUrl,omitempty"`
	Models   []LLMModelRef   `json:"models,omitempty"`
}

type LLMModel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Enabled    bool   `json:"enabled"`
}

// LLMProviderPreset is a template for quickly creating a provider config.
// It ships via config.yaml or built-in defaults and is exposed to the frontend
// so users can pick a preset instead of filling every field manually.
type LLMProviderPreset struct {
	ID          string          `json:"id"`          // e.g. "deepseek"
	Name        string          `json:"name"`        // display name, e.g. "DeepSeek"
	Provider    LLMProviderType `json:"provider"`    // "openai" | "anthropic" | ...
	BaseURL     string          `json:"baseUrl"`     // default API endpoint
	Icon        string          `json:"icon"`        // emoji or icon key
	Description string          `json:"description"` // short human-readable description
}
