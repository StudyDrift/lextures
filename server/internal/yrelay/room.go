package yrelay

import (
	"context"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// Client is one connected WebSocket peer in a room.
type Client struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Conn   *websocket.Conn
	mu     sync.Mutex
}

// Send writes a binary WebSocket frame to the peer.
func (c *Client) Send(ctx context.Context, msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.Write(ctx, websocket.MessageBinary, msg)
}

// SendText writes a text WebSocket frame to the peer (JSON game hubs).
func (c *Client) SendText(ctx context.Context, msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.Write(ctx, websocket.MessageText, msg)
}

// Room holds all clients collaborating on the same resource.
type Room struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*Client
}

// NewRoom creates an empty room.
func NewRoom() *Room {
	return &Room{clients: make(map[uuid.UUID]*Client)}
}

// Add registers a client in the room.
func (r *Room) Add(c *Client) {
	r.mu.Lock()
	r.clients[c.ID] = c
	r.mu.Unlock()
}

// Remove unregisters a client from the room.
func (r *Room) Remove(id uuid.UUID) {
	r.mu.Lock()
	delete(r.clients, id)
	r.mu.Unlock()
}

// Len returns the number of connected clients.
func (r *Room) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Broadcast sends msg to every client except from.
func (r *Room) Broadcast(ctx context.Context, from uuid.UUID, msg []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, c := range r.clients {
		if id == from {
			continue
		}
		_ = c.Send(ctx, msg)
	}
}

// BroadcastText sends a text frame to every client except from.
func (r *Room) BroadcastText(ctx context.Context, from uuid.UUID, msg []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, c := range r.clients {
		if id == from {
			continue
		}
		_ = c.SendText(ctx, msg)
	}
}

// ForEach invokes fn for every connected client.
func (r *Room) ForEach(fn func(*Client)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		fn(c)
	}
}

// Registry is an in-process map of rooms keyed by resource ID.
type Registry struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*Room
}

// NewRegistry creates an empty room registry.
func NewRegistry() *Registry {
	return &Registry{rooms: make(map[uuid.UUID]*Room)}
}

// GetOrCreate returns the room for id, creating it if needed.
func (reg *Registry) GetOrCreate(id uuid.UUID) *Room {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	r, ok := reg.rooms[id]
	if !ok {
		r = NewRoom()
		reg.rooms[id] = r
	}
	return r
}

// MaybeDelete removes the room when it has no clients left.
func (reg *Registry) MaybeDelete(id uuid.UUID) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	r, ok := reg.rooms[id]
	if !ok {
		return
	}
	if r.Len() == 0 {
		delete(reg.rooms, id)
	}
}

// Stats returns room and client counts for observability.
func (reg *Registry) Stats() (rooms, clients int) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	rooms = len(reg.rooms)
	for _, r := range reg.rooms {
		clients += r.Len()
	}
	return rooms, clients
}
