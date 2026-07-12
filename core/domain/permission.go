package domain

type PermAction string

const (
	PermAsk  PermAction = "ask"
	PermDeny PermAction = "deny"
)

type PermissionRule struct {
	Pattern string     `json:"pattern"`
	Action  PermAction `json:"action"`
}
