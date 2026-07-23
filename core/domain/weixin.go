package domain

import "time"

// ConfigChannelsSection holds external chat channel settings.
type ConfigChannelsSection struct {
	Weixin ConfigWeixinChannel `json:"weixin" mapstructure:"weixin" yaml:"weixin"`
}

// ConfigWeixinChannel configures the Weixin iLink bridge.
type ConfigWeixinChannel struct {
	Enabled          bool   `json:"enabled" mapstructure:"enabled" yaml:"enabled"`
	DefaultProjectID string `json:"defaultProjectId" mapstructure:"default_project_id" yaml:"default_project_id"`
	DefaultAgentID   string `json:"defaultAgentId" mapstructure:"default_agent_id" yaml:"default_agent_id"`
	DefaultModelID   string `json:"defaultModelId" mapstructure:"default_model_id" yaml:"default_model_id"`
	AutoApprove      bool   `json:"autoApprove" mapstructure:"auto_approve" yaml:"auto_approve"`
}

const WeixinProjectName = "微信"

// WeixinAccount is a logged-in iLink bot account.
type WeixinAccount struct {
	AccountID string    `json:"accountId"`
	Token     string    `json:"token,omitempty"`
	BaseURL   string    `json:"baseUrl,omitempty"`
	UserID    string    `json:"userId,omitempty"`
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
	Message          string `json:"message,omitempty"`
	NeedsVerifyCode  bool   `json:"needsVerifyCode,omitempty"`
}

type WeixinStatus struct {
	Enabled          bool            `json:"enabled"`
	Running          bool            `json:"running"`
	DefaultProjectID string          `json:"defaultProjectId,omitempty"`
	DefaultAgentID   string          `json:"defaultAgentId,omitempty"`
	DefaultModelID   string          `json:"defaultModelId,omitempty"`
	AutoApprove      bool            `json:"autoApprove"`
	Accounts         []WeixinAccount `json:"accounts"`
	BindingCount     int             `json:"bindingCount"`
}
