// Package feedevents provides an in-memory pub/sub hub that fans out course-feed
// change signals to WebSocket subscribers per course, so the SPA refreshes in
// real time when channels or messages change (including from the CLI).
package feedevents

import "sync"

// Event is the payload broadcast to feed WebSocket subscribers. It mirrors the
// shape the SPA expects: {"type":"feed","scope":"channels"|"messages",...}.
type Event struct {
	Type      string `json:"type"`
	Scope     string `json:"scope"`
	ChannelID string `json:"channelId,omitempty"`
}

// Hub fans out feed events to subscribers keyed by course code.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[uint64]chan Event
	seq     uint64
}

// New returns a new Hub.
func New() *Hub {
	return &Hub{clients: make(map[string]map[uint64]chan Event)}
}

// Subscribe registers a listener for the given course code. It returns a channel
// that receives feed events and a function to unsubscribe.
func (h *Hub) Subscribe(courseCode string) (<-chan Event, func()) {
	if h == nil {
		ch := make(chan Event, 1)
		return ch, func() {}
	}
	ch := make(chan Event, 16)
	h.mu.Lock()
	h.seq++
	id := h.seq
	if h.clients[courseCode] == nil {
		h.clients[courseCode] = make(map[uint64]chan Event)
	}
	h.clients[courseCode][id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.clients[courseCode], id)
		if len(h.clients[courseCode]) == 0 {
			delete(h.clients, courseCode)
		}
		h.mu.Unlock()
	}
}

// Publish sends ev to all subscribers of courseCode (non-blocking; drops on a
// full channel so a slow client never stalls the writer).
func (h *Hub) Publish(courseCode string, ev Event) {
	if h == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[courseCode] {
		select {
		case ch <- ev:
		default:
		}
	}
}

// ChannelsChanged broadcasts that the course's channel list changed.
func (h *Hub) ChannelsChanged(courseCode string) {
	h.Publish(courseCode, Event{Type: "feed", Scope: "channels"})
}

// MessagesChanged broadcasts that messages in the given channel changed.
func (h *Hub) MessagesChanged(courseCode, channelID string) {
	h.Publish(courseCode, Event{Type: "feed", Scope: "messages", ChannelID: channelID})
}
