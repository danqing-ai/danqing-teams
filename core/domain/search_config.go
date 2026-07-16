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
	SearchProviderBrave      SearchProvider = "brave"
)

type SearchConfig struct {
	Provider     SearchProvider `json:"provider" mapstructure:"provider" yaml:"provider"`
	BaseURL      string         `json:"baseUrl,omitempty" mapstructure:"base_url" yaml:"base_url,omitempty"`
	APIKey       string         `json:"apiKey,omitempty" mapstructure:"api_key" yaml:"api_key,omitempty"`
	TimeoutMs    int            `json:"timeoutMs" mapstructure:"timeout_ms" yaml:"timeout_ms"`
	MaxResults   int            `json:"maxResults" mapstructure:"max_results" yaml:"max_results"`
	Proxy        string         `json:"proxy,omitempty" mapstructure:"proxy" yaml:"proxy,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty" mapstructure:"user_agent" yaml:"user_agent,omitempty"`
	HTMLFallback *bool          `json:"htmlFallback,omitempty" mapstructure:"html_fallback" yaml:"html_fallback,omitempty"`
}

// HTMLFallbackEnabled returns whether empty/failed API searches should fall back to HTML scrapers.
// Default is true when unset.
func (c SearchConfig) HTMLFallbackEnabled() bool {
	if c.HTMLFallback == nil {
		return true
	}
	return *c.HTMLFallback
}

type UpsertSearchConfigRequest struct {
	Provider     SearchProvider `json:"provider" mapstructure:"provider"`
	BaseURL      string         `json:"baseUrl,omitempty" mapstructure:"base_url"`
	APIKey       string         `json:"apiKey,omitempty" mapstructure:"api_key"`
	TimeoutMs    int            `json:"timeoutMs" mapstructure:"timeout_ms"`
	MaxResults   int            `json:"maxResults" mapstructure:"max_results"`
	Proxy        string         `json:"proxy,omitempty" mapstructure:"proxy"`
	UserAgent    string         `json:"userAgent,omitempty" mapstructure:"user_agent"`
	HTMLFallback *bool          `json:"htmlFallback,omitempty" mapstructure:"html_fallback"`
}
