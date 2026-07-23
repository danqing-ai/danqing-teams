package domain

import "time"

// ConfigChannelsSection holds external chat channel settings.
type ConfigChannelsSection struct {
	Weixin ConfigWeixinChannel `json:"weixin" mapstructure:"weixin" yaml:"weixin"`
}

// ConfigWeixinChannel configures the Weixin iLink bridge.
// Project binding lives on each WeixinAccount (one account → one project).
type ConfigWeixinChannel struct {
	Enabled        bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	DefaultAgentID string `json:"defaultAgentId" mapstructure:"default_agent_id" yaml:"default_agent_id"`
	DefaultModelID string `json:"defaultModelId" mapstructure:"default_model_id" yaml:"default_model_id"`
	AutoApprove    bool   `json:"autoApprove" mapstructure:"auto_approve" yaml:"auto_approve"`

	// DefaultProjectID is deprecated (migrated onto WeixinAccount.ProjectID).
	// Kept only so one-shot migration can read old YAML.
	DefaultProjectID string `json:"defaultProjectId,omitempty" mapstructure:"default_project_id" yaml:"default_project_id,omitempty"`
}

// WeixinAccount is a logged-in iLink bot account bound to one Teams project.
type WeixinAccount struct {
	AccountID string    `json:"accountId"`
	Token     string    `json:"token,omitempty"`
	BaseURL   string    `json:"baseUrl,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	ProjectID string    `json:"projectId,omitempty"`
	SyncBuf   string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WeixinBinding maps one Weixin peer to one Teams session (1:1).
type WeixinBinding struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"accountId"`
	PeerUserID   string    `json:"peerUserId"`
	SessionID    string    `json:"sessionId"`
	ContextToken string    `json:"contextToken,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type WeixinLoginStartResult struct {
	SessionKey string `json:"sessionKey"`
	QRCodeURL  string `json:"qrcodeUrl"`
	AccountID  string `json:"accountId,omitempty"`
}

type WeixinLoginWaitResult struct {
	Connected        bool   `json:"connected"`
	AlreadyConnected bool   `json:"alreadyConnected,omitempty"`
	AccountID        string `json:"accountId,omitempty"`
	UserID           string `json:"userId,omitempty"`
	ProjectID        string `json:"projectId,omitempty"`
	Message          string `json:"message,omitempty"`
	NeedsVerifyCode  bool   `json:"needsVerifyCode,omitempty"`
}

type WeixinStatus struct {
	Enabled        bool            `json:"enabled"`
	Running        bool            `json:"running"`
	DefaultAgentID string          `json:"defaultAgentId,omitempty"`
	DefaultModelID string          `json:"defaultModelId,omitempty"`
	AutoApprove    bool            `json:"autoApprove"`
	Accounts       []WeixinAccount `json:"accounts"`
	BindingCount   int             `json:"bindingCount"`
}
