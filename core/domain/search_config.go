package domain

type SearchProvider string

const (
	SearchProviderDuckDuckGo SearchProvider = "duckduckgo"
	SearchProviderBing       SearchProvider = "bing"
	SearchProviderTavily     SearchProvider = "tavily"
	SearchProviderBocha      SearchProvider = "bocha"
	SearchProviderMetaso     SearchProvider = "metaso"
	SearchProviderSearxng    SearchProvider = "searxng"
	SearchProviderBaidu      SearchProvider = "baidu"
	SearchProviderVolcengine SearchProvider = "volcengine"
	SearchProviderSofya      SearchProvider = "sofya"
)

type SearchConfig struct {
	Provider   SearchProvider `json:"provider" mapstructure:"provider"`
	BaseURL    string         `json:"baseUrl,omitempty" mapstructure:"base_url"`
	APIKey     string         `json:"apiKey,omitempty" mapstructure:"api_key"`
	TimeoutMs  int            `json:"timeoutMs" mapstructure:"timeout_ms"`
	MaxResults int            `json:"maxResults" mapstructure:"max_results"`
}

type UpsertSearchConfigRequest struct {
	Provider   SearchProvider `json:"provider" mapstructure:"provider"`
	BaseURL    string         `json:"baseUrl,omitempty" mapstructure:"base_url"`
	APIKey     string         `json:"apiKey,omitempty" mapstructure:"api_key"`
	TimeoutMs  int            `json:"timeoutMs" mapstructure:"timeout_ms"`
	MaxResults int            `json:"maxResults" mapstructure:"max_results"`
}
