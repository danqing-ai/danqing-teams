package domain

// MCPServer represents a configured MCP (Model Context Protocol) server.
type MCPServer struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Transport       string            `json:"transport"`       // stdio | sse | streamable-http
	Command         string            `json:"command"`         // for stdio
	Args            string            `json:"args"`            // space-separated args for stdio
	URL             string            `json:"url"`             // for sse / streamable-http
	Env             string            `json:"env"`             // KEY=value per line
	Headers         map[string]string `json:"headers"`         // for sse / streamable-http
	EnabledTools    []string          `json:"enabledTools"`    // tool names user enabled
	DiscoveredTools []MCPToolDef      `json:"discoveredTools"` // tools discovered from server
	ToolTimeout     int               `json:"toolTimeout"`     // seconds, default 300
	Status          string            `json:"status"`          // connected | disconnected | error
	Enabled         bool              `json:"enabled"`         // user toggle
}

// MCPToolDef describes a single tool exposed by an MCP server.
type MCPToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// UpsertMCPServerRequest is the payload for creating / updating an MCP server.
type UpsertMCPServerRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Transport     string            `json:"transport"`
	Command       string            `json:"command"`
	Args          string            `json:"args"`
	URL           string            `json:"url"`
	Env           string            `json:"env"`
	Headers       map[string]string `json:"headers"`
	EnabledTools  []string          `json:"enabledTools"`
	ToolTimeout   int               `json:"toolTimeout"`
	Status        string            `json:"status"`
	Enabled       bool              `json:"enabled"`
}
