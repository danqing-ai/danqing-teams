package events

import (
	"context"
	"sync"

	"danqing-teams/internal/domain/model"
)

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan model.StreamEvent]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan model.StreamEvent]struct{})}
}

func (h *Hub) Publish(_ context.Context, teamID, taskID string, evt model.StreamEvent) {
	evt.TeamID = teamID
	evt.TaskID = taskID
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[taskID] {
		select {
		case ch <- evt:
		default:
		}
	}
}

func (h *Hub) Subscribe(taskID string) (<-chan model.StreamEvent, func()) {
	ch := make(chan model.StreamEvent, 32)
	h.mu.Lock()
	if h.subs[taskID] == nil {
		h.subs[taskID] = make(map[chan model.StreamEvent]struct{})
	}
	h.subs[taskID][ch] = struct{}{}
	h.mu.Unlock()
	unsub := func() {
		h.mu.Lock()
		delete(h.subs[taskID], ch)
		close(ch)
		h.mu.Unlock()
	}
	return ch, unsub
}
