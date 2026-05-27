package events

import (
	"context"

	"danqing-teams/internal/domain/model"
)

// Noop is a discard EventPublisher (SSE removed; clients poll timeline REST).
type Noop struct{}

func NewNoop() *Noop { return &Noop{} }

func (Noop) Publish(_ context.Context, _, _ string, _ model.StreamEvent) {}

func (Noop) Subscribe(_ string) (<-chan model.StreamEvent, func()) {
	ch := make(chan model.StreamEvent)
	return ch, func() { close(ch) }
}
