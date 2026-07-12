package domain

type MCPServer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Type     string `json:"type"`
}
