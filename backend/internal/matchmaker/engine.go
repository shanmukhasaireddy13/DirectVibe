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
	ghostEvictions := 0

	curr := e.queue.Head
	for curr != nil {
		node1 := curr
		next := curr.Next
		
		if node1.Client.IsGhost() {
			if ghostEvictions < 5 {
				e.cleanupNode(node1)
				ghostEvictions++
			}
			curr = next
			continue
		}
		
		waitDuration := time.Since(node1.Client.EnqueueTime())

		var match *Node

		if len(node1.Client.Keywords()) == 0 {
			match = e.findRandomMatch(node1.Client, next, &ghostEvictions)
		} else {
			if waitDuration < e.relaxedWait {
				match = e.findBestMatch(node1.Client, &ghostEvictions)
			} else {
				match = e.findRandomMatch(node1.Client, next, &ghostEvictions)
			}
		}

		if match != nil {
			// We have a pair!
			e.cleanupNode(node1)
			e.cleanupNode(match)

			node1.Client.SendMatch(match.Client.ID(), true)
			match.Client.SendMatch(node1.Client.ID(), false)
			
			curr = e.queue.Head // Restart sweep from new head
		} else {
			curr = next
		}
	}
}

func (e *Engine) findRandomMatch(c Client, start *Node, evictions *int) *Node {
	curr := start
	for curr != nil {
		next := curr.Next
		if curr.Client.ID() != c.ID() && !c.HasSkipped(curr.Client.ID()) {
			if curr.Client.IsGhost() {
				if *evictions < 5 {
					e.cleanupNode(curr)
					*evictions++
				}
				curr = next
				continue
			}
			return curr
		}
		curr = next
	}
	return nil
}

func (e *Engine) findBestMatch(c Client, evictions *int) *Node {
	bucket := e.index.GetSmallestBucket(c.Keywords())
	if bucket == nil {
		return nil
	}

	var bestMatch *Node
	bestScore := -1
	var bestWait time.Duration

	// Build a map of caller's keywords for fast O(1) intersection check
	cKeywords := make(map[string]struct{})
	for _, kw := range c.Keywords() {
		cKeywords[kw] = struct{}{}
	}

	for _, potentialMatch := range bucket {
		if potentialMatch.Client.ID() == c.ID() {
			continue
		}
		
		if c.HasSkipped(potentialMatch.Client.ID()) {
			continue
		}

		if potentialMatch.Client.IsGhost() {
			if *evictions < 5 {
				e.cleanupNode(potentialMatch)
				*evictions++
			}
			continue
		}

		score := 0
		for _, kw := range potentialMatch.Client.Keywords() {
			if _, exists := cKeywords[kw]; exists {
				score++
			}
		}

		wait := time.Since(potentialMatch.Client.EnqueueTime())

		if score > bestScore {
			bestScore = score
			bestMatch = potentialMatch
			bestWait = wait
		} else if score == bestScore {
			if wait > bestWait { // Break tie with oldest waiting
				bestMatch = potentialMatch
				bestWait = wait
			}
		}
	}

	return bestMatch
}
