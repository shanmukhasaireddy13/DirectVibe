package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/directvibe/backend/internal/matchmaker"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

// Client represents a single user's websocket connection
type Client struct {
	id          string
	conn        *websocket.Conn
	pool        *Pool
	engine      *matchmaker.Engine
	send        chan []byte
	
	keywords    []string
	enqueueTime time.Time
	
	// Anti-spam throttling
	lastSkip    time.Time

	ctx         context.Context
	cancel      context.CancelFunc

	currentPeerID string
	peerMu        sync.RWMutex

	skipHistory map[string]time.Time
	skipMu      sync.RWMutex
	lastSeen    time.Time
}

func (c *Client) SetPeer(peerID string) {
	c.peerMu.Lock()
	c.currentPeerID = peerID
	c.peerMu.Unlock()
}

func (c *Client) GetPeer() string {
	c.peerMu.RLock()
	defer c.peerMu.RUnlock()
	return c.currentPeerID
}

func (c *Client) notifyPeerLeft() {
	peerID := c.GetPeer()
	if peerID == "" {
		return
	}
	c.SetPeer("")
	c.AddSkip(peerID)
	
	targetClient := c.pool.Get(peerID)
	if targetClient != nil {
		targetClient.SetPeer("")
		targetClient.AddSkip(c.id)
		msg := map[string]interface{}{
			"type": "peer_left",
		}
		b, _ := json.Marshal(msg)
		select {
		case targetClient.send <- b:
		default:
		}
	}
}

func (c *Client) AddSkip(otherID string) {
	c.skipMu.Lock()
	c.skipHistory[otherID] = time.Now()
	c.skipMu.Unlock()
}

func (c *Client) HasSkipped(otherID string) bool {
	c.skipMu.RLock()
	defer c.skipMu.RUnlock()
	t, exists := c.skipHistory[otherID]
	if !exists {
		return false
	}
	if time.Since(t) > 5*time.Second {
		return false
	}
	return true
}

func (c *Client) updateLastSeen() {
	c.peerMu.Lock()
	c.lastSeen = time.Now()
	c.peerMu.Unlock()
}

func (c *Client) IsGhost() bool {
	c.peerMu.RLock()
	t := c.lastSeen
	c.peerMu.RUnlock()
	// Must be greater than pingPeriod (54s) + margin
	return time.Since(t) > 65*time.Second
}

func (c *Client) janitor() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.skipMu.Lock()
			now := time.Now()
			for id, t := range c.skipHistory {
				if now.Sub(t) > 5*time.Second {
					delete(c.skipHistory, id)
				}
			}
			c.skipMu.Unlock()
		}
	}
}

// NewClient creates a new client
func NewClient(id string, conn *websocket.Conn, pool *Pool, engine *matchmaker.Engine) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		id:          id,
		conn:        conn,
		pool:        pool,
		engine:      engine,
		send:        make(chan []byte, 256),
		ctx:         ctx,
		cancel:      cancel,
		skipHistory: make(map[string]time.Time),
		lastSeen:    time.Now(),
	}
	go c.janitor()
	return c
}

// --- matchmaker.Client interface implementation ---

func (c *Client) ID() string {
	return c.id
}
func (c *Client) Keywords() []string {
	return c.keywords
}
func (c *Client) EnqueueTime() time.Time {
	return c.enqueueTime
}
func (c *Client) SendMatch(otherID string, offer bool) {
	c.SetPeer(otherID)
	msg := map[string]interface{}{
		"type":     "match_found",
		"peer_id":  otherID,
		"is_offer": offer,
	}
	b, _ := json.Marshal(msg)
	
	select {
	case c.send <- b:
	default:
		// Queue full
	}
}

// --- WebSocket lifecycle loops ---

func (c *Client) ReadPump() {
	defer func() {
		// Clean up on disconnect
		c.cancel() // Immediately kill the WritePump goroutine
		c.notifyPeerLeft()
		c.engine.RemoveClient(c)
		c.pool.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { 
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.updateLastSeen()
		return nil 
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		c.updateLastSeen()
		c.handleMessage(message)
	}
}

func (c *Client) handleMessage(message []byte) {
	var payload map[string]interface{}
	if err := json.Unmarshal(message, &payload); err != nil {
		return
	}

	msgType, ok := payload["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "enqueue":
		c.engine.RemoveClient(c) // Remove if already queued
		
		c.keywords = []string{}
		if kws, ok := payload["keywords"].([]interface{}); ok {
			for _, kw := range kws {
				if s, ok := kw.(string); ok {
					c.keywords = append(c.keywords, s)
				}
			}
		}
		c.enqueueTime = time.Now()
		c.engine.AddClient(c)

	case "skip":
		c.notifyPeerLeft()
		c.engine.RemoveClient(c)
		
		// Immediately re-enqueue to find a new match
		c.enqueueTime = time.Now()
		c.engine.AddClient(c)

	case "webrtc_signal":
		targetID, ok := payload["target_id"].(string)
		if !ok {
			return
		}
		
		targetClient := c.pool.Get(targetID)
		if targetClient != nil {
			relayMsg := map[string]interface{}{
				"type":      "webrtc_signal",
				"sender_id": c.id,
				"signal":    payload["signal"],
			}
			b, _ := json.Marshal(relayMsg)
			select {
			case targetClient.send <- b:
			default:
			}
		}

	case "chat_message":
		targetID, ok := payload["target_id"].(string)
		if !ok {
			return
		}
		
		targetClient := c.pool.Get(targetID)
		if targetClient != nil {
			relayMsg := map[string]interface{}{
				"type":      "chat_message",
				"sender_id": c.id,
				"text":      payload["text"],
			}
			b, _ := json.Marshal(relayMsg)
			select {
			case targetClient.send <- b:
			default:
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// Send heartbeat Keep-Alive Ping
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
