package events

import (
	"context"

	"danqing-teams/internal/contract"
)

// Noop is a discard EventPublisher (SSE removed; clients poll timeline REST).
type Noop struct{}

func NewNoop() *Noop { return &Noop{} }

func (Noop) Publish(_ context.Context, _, _ string, _ contract.StreamEvent) {}

func (Noop) Subscribe(_ string) (<-chan contract.StreamEvent, func()) {
	ch := make(chan contract.StreamEvent)
	return ch, func() { close(ch) }
}
