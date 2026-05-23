package events

import (
	"context"
	"sync"

	"danqing-teams/internal/contract"
)

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan contract.StreamEvent]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan contract.StreamEvent]struct{})}
}

func (h *Hub) Publish(_ context.Context, teamID, taskID string, evt contract.StreamEvent) {
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

func (h *Hub) Subscribe(taskID string) (<-chan contract.StreamEvent, func()) {
	ch := make(chan contract.StreamEvent, 32)
	h.mu.Lock()
	if h.subs[taskID] == nil {
		h.subs[taskID] = make(map[chan contract.StreamEvent]struct{})
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
