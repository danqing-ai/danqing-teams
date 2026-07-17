package domain

// SandboxMode controls filesystem/network policy for process execution.
// Aligned with Codex CLI naming.
type SandboxMode string

const (
	SandboxModeReadOnly          SandboxMode = "read-only"
	SandboxModeWorkspaceWrite    SandboxMode = "workspace-write"
	SandboxModeDangerFullAccess  SandboxMode = "danger-full-access"
)

// SandboxNetwork controls outbound network for sandboxed processes.
type SandboxNetwork string

const (
	SandboxNetworkDeny         SandboxNetwork = "deny"
	SandboxNetworkAllow        SandboxNetwork = "allow"
	SandboxNetworkAllowlist    SandboxNetwork = "allowlist"
)

// SandboxBackend identifies the OS enforcement mechanism in use.
type SandboxBackend string

const (
	SandboxBackendSeatbelt  SandboxBackend = "seatbelt"
	SandboxBackendLandlock  SandboxBackend = "landlock"
	SandboxBackendBwrap     SandboxBackend = "bwrap"
	SandboxBackendWinToken  SandboxBackend = "win-token"
	SandboxBackendWSL2      SandboxBackend = "wsl2"
	SandboxBackendHostWeak  SandboxBackend = "host-weak"
	SandboxBackendDisabled  SandboxBackend = "disabled"
)

// ConfigSandboxSection is persisted under runtime.sandbox in config.yaml.
type ConfigSandboxSection struct {
	Enabled bool           `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	Mode    SandboxMode    `json:"mode" mapstructure:"mode" yaml:"mode"`
	Network SandboxNetwork `json:"network" mapstructure:"network" yaml:"network"`
	// Backend forces a backend when non-empty (e.g. "bwrap", "wsl2", "host-weak").
	// Empty means auto-probe.
	Backend string `json:"backend,omitempty" mapstructure:"backend" yaml:"backend,omitempty"`
}

// SandboxStatus is the probed runtime sandbox capability surface.
type SandboxStatus struct {
	Enabled        bool           `json:"enabled"`
	Mode           SandboxMode    `json:"mode"`
	Network        SandboxNetwork `json:"network"`
	Backend        SandboxBackend `json:"backend"`
	Degraded       bool           `json:"degraded"`
	DegradedReason string         `json:"degradedReason,omitempty"`
	Platform       string         `json:"platform"`
	Capabilities   []string       `json:"capabilities,omitempty"`
}
