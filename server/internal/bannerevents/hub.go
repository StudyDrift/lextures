// Package bannerevents provides an in-memory pub/sub hub that fans out
// maintenance-banner change signals to WebSocket subscribers so all screens
// clear or refresh the banner in real time (plan 18.6).
package bannerevents

import "sync"

// Event is the payload broadcast to banner WebSocket subscribers.
type Event struct {
	Type   string `json:"type"`
	Action string `json:"action"` // "upserted" | "cleared"
	ID     string `json:"id,omitempty"`
	Scope  string `json:"scope,omitempty"` // "global" | "org"
	OrgID  string `json:"orgId,omitempty"`
}

// Hub fans out banner events to all subscribers (platform-wide).
type Hub struct {
	mu      sync.RWMutex
	clients map[uint64]chan Event
	seq     uint64
}

// New returns a new Hub.
func New() *Hub {
	return &Hub{clients: make(map[uint64]chan Event)}
}

// Subscribe registers a listener. It returns a channel that receives banner
// events and a function to unsubscribe.
func (h *Hub) Subscribe() (<-chan Event, func()) {
	if h == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.seq++
	id := h.seq
	h.clients[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.clients, id)
		h.mu.Unlock()
	}
}

// Publish sends ev to all subscribers (non-blocking; drops on a full channel so
// a slow client never stalls the writer).
func (h *Hub) Publish(ev Event) {
	if h == nil {
		return
	}
	if ev.Type == "" {
		ev.Type = "banner_changed"
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Cleared broadcasts that a banner was deleted or deactivated.
func (h *Hub) Cleared(id, scope, orgID string) {
	h.Publish(Event{
		Type:   "banner_changed",
		Action: "cleared",
		ID:     id,
		Scope:  scope,
		OrgID:  orgID,
	})
}

// Upserted broadcasts that a banner was created or updated.
func (h *Hub) Upserted(id, scope, orgID string) {
	h.Publish(Event{
		Type:   "banner_changed",
		Action: "upserted",
		ID:     id,
		Scope:  scope,
		OrgID:  orgID,
	})
}
