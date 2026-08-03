package websocket

import (
	"sync"
)

// Pool tracks all active websocket connections across the server
type Pool struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewPool initializes a new connection pool
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*Client),
	}
}

// Register adds a client to the pool
func (p *Pool) Register(c *Client) {
	p.mu.Lock()
	p.clients[c.id] = c
	p.mu.Unlock()
}

// Unregister removes a client from the pool
func (p *Pool) Unregister(c *Client) {
	p.mu.Lock()
	if _, ok := p.clients[c.id]; ok {
		delete(p.clients, c.id)
	}
	p.mu.Unlock()
}

// Get safely retrieves a client by ID, returning nil if not found
func (p *Pool) Get(id string) *Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[id]
}
