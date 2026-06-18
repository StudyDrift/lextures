// Package canvassubmissionsyncevents fans out Canvas submission sync WebSocket messages per job.
package canvassubmissionsyncevents

import (
	"sync"

	"github.com/google/uuid"
)

// Message is a JSON-serializable WebSocket payload (complete or error).
type Message struct {
	Type    string         `json:"type"`
	Message string         `json:"message,omitempty"`
	Grade   map[string]any `json:"grade,omitempty"`
}

// Hub broadcasts sync events to subscribers keyed by job ID.
type Hub struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[uint64]chan Message
	seq  uint64
}

// New returns a new Hub.
func New() *Hub {
	return &Hub{subs: make(map[uuid.UUID]map[uint64]chan Message)}
}

// Broadcast sends a message to all subscribers for jobID (non-blocking; drops on full channel).
func (h *Hub) Broadcast(jobID uuid.UUID, msg Message) {
	if h == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[jobID] {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Subscribe registers a listener for jobID. Call unsubscribe when done.
func (h *Hub) Subscribe(jobID uuid.UUID) (<-chan Message, func()) {
	if h == nil {
		ch := make(chan Message)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan Message, 8)
	h.mu.Lock()
	h.seq++
	id := h.seq
	if h.subs[jobID] == nil {
		h.subs[jobID] = make(map[uint64]chan Message)
	}
	h.subs[jobID][id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[jobID], id)
		if len(h.subs[jobID]) == 0 {
			delete(h.subs, jobID)
		}
		h.mu.Unlock()
	}
}