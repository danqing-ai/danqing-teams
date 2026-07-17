package domain

// ConfigBrowserSection is persisted under runtime.browser in config.yaml.
type ConfigBrowserSection struct {
	Enabled        bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	ExecutablePath string `json:"executablePath,omitempty" mapstructure:"executable_path" yaml:"executable_path,omitempty"`
	CDPURL         string `json:"cdpUrl,omitempty" mapstructure:"cdp_url" yaml:"cdp_url,omitempty"`
}

// BrowserStatus is the probed headless browser capability surface.
type BrowserStatus struct {
	Available      bool   `json:"available"`
	Enabled        bool   `json:"enabled"`
	Engine         string `json:"engine"` // chrome | edge | chromium | cdp | none
	Path           string `json:"path,omitempty"`
	Mode           string `json:"mode"` // launch | attach | none
	DegradedReason string `json:"degradedReason,omitempty"`
}
