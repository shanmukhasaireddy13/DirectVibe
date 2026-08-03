package matchmaker

import (
	"sync"
	"time"
)

// EventType defines the type of event in the consumer channel
type EventType int

const (
	EventAdd EventType = iota
	EventRemove
)

// Event represents an action to be performed by the Matchmaker
type Event struct {
	Type   EventType
	Client Client
}

// Engine coordinates matching, implementing the Time-Weighted Strategy Pattern.
// It uses a mutex to protect all O(1) data structures.
type Engine struct {
	mu          sync.Mutex
	queue       *DLL
	index       *Index
	clients     map[string]*Node // Global map: SocketID -> DLL Node
	events      chan Event       // Buffered channel for Producer-Consumer

	strictWait  time.Duration
	relaxedWait time.Duration
}

// NewEngine initializes the matchmaking engine and starts background workers.
func NewEngine(strictWaitSec, relaxedWaitSec int) *Engine {
	e := &Engine{
		queue:       NewDLL(),
		index:       NewIndex(),
		clients:     make(map[string]*Node),
		events:      make(chan Event, 2048), // Large buffer to decouple network I/O
		strictWait:  time.Duration(strictWaitSec) * time.Second,
		relaxedWait: time.Duration(relaxedWaitSec) * time.Second,
	}

	go e.worker()
	go e.matcherLoop()
	return e
}

// AddClient pushes a client to the match queue (Producer)
func (e *Engine) AddClient(c Client) {
	e.events <- Event{Type: EventAdd, Client: c}
}

// RemoveClient removes a client from the match queue (Producer)
func (e *Engine) RemoveClient(c Client) {
	e.events <- Event{Type: EventRemove, Client: c}
}

// worker sequentially processes queue modifications
func (e *Engine) worker() {
	for evt := range e.events {
		e.mu.Lock()
		switch evt.Type {
		case EventAdd:
			e.handleAdd(evt.Client)
		case EventRemove:
			e.handleRemove(evt.Client)
		}
		e.mu.Unlock()
	}
}

func (e *Engine) handleAdd(c Client) {
	// Ensure not already queued
	if _, exists := e.clients[c.ID()]; exists {
		return
	}

	node := e.queue.PushBack(c)
	e.clients[c.ID()] = node

	if len(c.Keywords()) > 0 {
		e.index.Add(c.Keywords(), node)
	}
}

func (e *Engine) handleRemove(c Client) {
	node, exists := e.clients[c.ID()]
	if !exists {
		return
	}

	e.cleanupNode(node)
}

func (e *Engine) cleanupNode(node *Node) {
	e.queue.Remove(node)
	delete(e.clients, node.Client.ID())

	if len(node.Client.Keywords()) > 0 {
		e.index.Remove(node.Client.Keywords(), node.Client.ID())
	}
}

// matcherLoop constantly sweeps the queue for matches using Strategy degradation
func (e *Engine) matcherLoop() {
	ticker := time.NewTicker(250 * time.Millisecond)
	for range ticker.C {
		e.mu.Lock()
		e.processMatches()
		e.mu.Unlock()
	}
}

func (e *Engine) processMatches() {
	// Iterate through the oldest users first
	for e.queue.Head != nil {
		node1 := e.queue.Head
		waitDuration := time.Since(node1.Client.EnqueueTime())

		var match *Node

		if len(node1.Client.Keywords()) == 0 {
			// No keywords? Fallback immediately to oldest next user
			match = node1.Next
		} else {
			if waitDuration < e.strictWait {
				// Strict Strategy: Find match in same keyword bucket
				match = e.findKeywordMatch(node1.Client)
			} else if waitDuration < e.relaxedWait {
				// Relaxed Strategy: (Simplified to finding any keyword match)
				match = e.findKeywordMatch(node1.Client)
			} else {
				// Fallback Strategy: Drop keywords, pop oldest waiting user
				match = node1.Next
			}
		}

		if match != nil {
			// We have a pair!
			e.cleanupNode(node1)
			e.cleanupNode(match)

			// Tell the clients they matched
			// Client 1 is the "Caller" (Offer), Client 2 is "Callee" (Answer)
			node1.Client.SendMatch(match.Client.ID(), true)
			match.Client.SendMatch(node1.Client.ID(), false)
		} else {
			// Oldest user can't find a match yet, break sweep until next tick
			break
		}
	}
}

func (e *Engine) findKeywordMatch(c Client) *Node {
	// Look through the user's buckets
	for _, kw := range c.Keywords() {
		if bucket, exists := e.index.KeywordMap[kw]; exists {
			for _, potentialMatch := range bucket {
				if potentialMatch.Client.ID() != c.ID() {
					return potentialMatch
				}
			}
		}
	}
	return nil
}
