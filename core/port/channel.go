package port

import (
	"context"
)

// ChannelType identifies an external chat platform.
type ChannelType string

const (
	ChannelWeixin ChannelType = "weixin"
	ChannelFeishu ChannelType = "feishu"
)

// InboundMessage is the normalized inbound chat message (WeKnora-style).
type InboundMessage struct {
	Type      ChannelType
	AccountID string
	PeerID    string
	ChatID    string
	ThreadID  string
	Text      string
	MessageID string
	Meta      map[string]string // e.g. context_token
}

// OutboundReply is a plain-text reply to send back to the platform.
type OutboundReply struct {
	Content string
	Meta    map[string]string
}

// ChannelDefaults are channel-level agent/model/auto-approve settings.
type ChannelDefaults struct {
	AgentID     string
	ModelID     string
	AutoApprove bool
}

// ChannelStatus is a generic runtime status snapshot.
type ChannelStatus struct {
	Type    ChannelType `json:"type"`
	Enabled bool        `json:"enabled"`
	Running bool        `json:"running"`
}

// ChannelPeerStore resolves project binding and peer→session mappings.
// Weixin keeps its own tables; Feishu (and later channels) may share a generic table.
type ChannelPeerStore interface {
	GetProjectID(ctx context.Context, channel ChannelType, accountID string) (string, error)
	GetBinding(ctx context.Context, channel ChannelType, accountID, peerID string) (sessionID string, meta map[string]string, err error)
	UpsertBinding(ctx context.Context, channel ChannelType, accountID, peerID, sessionID string, meta map[string]string) error
	UpdateBindingMeta(ctx context.Context, channel ChannelType, accountID, peerID string, meta map[string]string) error
}

// ChannelDefaultsSource loads per-channel Agent/Model/AutoApprove.
type ChannelDefaultsSource interface {
	ChannelDefaults(ctx context.Context, channel ChannelType) (ChannelDefaults, error)
}

// ChannelRuntime manages long-lived connections (long-poll / WebSocket).
type ChannelRuntime interface {
	Type() ChannelType
	SyncFromConfig(ctx context.Context) error
	Stop()
	IsRunning() bool
}

// ChannelIngress turns an InboundMessage into a Teams Session turn and returns reply text.
type ChannelIngress interface {
	HandleInbound(ctx context.Context, msg InboundMessage) (reply string, err error)
}

// StreamSender is optional; platforms may stream partial replies (future).
type StreamSender interface {
	StartStream(ctx context.Context, in *InboundMessage) (streamID string, err error)
	UpdateStreamContent(ctx context.Context, in *InboundMessage, streamID, fullContent string) error
	FinalizeStream(ctx context.Context, in *InboundMessage, streamID, finalContent string) error
	EndStream(ctx context.Context, in *InboundMessage, streamID string) error
}
