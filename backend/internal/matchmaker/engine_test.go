package matchmaker

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockClient implements the Client interface for testing
type MockClient struct {
	id          string
	keywords    []string
	enqueueTime time.Time
	mu          sync.Mutex
	matchedWith string
	isOffer     bool
}

func (m *MockClient) ID() string { return m.id }
func (m *MockClient) Keywords() []string { return m.keywords }
func (m *MockClient) EnqueueTime() time.Time { return m.enqueueTime }
func (m *MockClient) SendMatch(otherID string, offer bool) {
	m.mu.Lock()
	m.matchedWith = otherID
	m.isOffer = offer
	m.mu.Unlock()
}

func TestEngine_ConcurrencyMemoryLeak(t *testing.T) {
	engine := NewEngine(1, 2)
	
	var wg sync.WaitGroup
	numClients := 5000 // Heavy stress test with 5000 concurrent joins

	// 1. Enqueue 5000 clients concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := &MockClient{
				id:          fmt.Sprintf("client-%d", i),
				keywords:    []string{fmt.Sprintf("unique-%d", i)},
				enqueueTime: time.Now(),
			}
			engine.AddClient(client)
		}(i)
	}

	wg.Wait()
	
	// Wait for worker to process all adds (5000 channel messages)
	time.Sleep(500 * time.Millisecond)

	engine.mu.Lock()
	if len(engine.clients) != numClients {
		t.Fatalf("Expected %d clients in map, got %d", numClients, len(engine.clients))
	}
	engine.mu.Unlock()

	// 2. Remove all 5000 clients concurrently (simulating abrupt disconnects)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := &MockClient{
				id: fmt.Sprintf("client-%d", i),
			}
			engine.RemoveClient(client)
		}(i)
	}

	wg.Wait()
	
	// Wait for worker to process all removes
	time.Sleep(500 * time.Millisecond)

	// 3. Assert absolutely zero memory footprint (maps and queues must be totally empty)
	engine.mu.Lock()
	defer engine.mu.Unlock()

	if len(engine.clients) != 0 {
		t.Errorf("Memory Leak: engine.clients map is not empty, size: %d", len(engine.clients))
	}
	if engine.queue.Size != 0 {
		t.Errorf("Memory Leak: engine.queue size is not 0, size: %d", engine.queue.Size)
	}
	if len(engine.index.KeywordMap) != 0 {
		t.Errorf("Memory Leak: engine.index map is not empty, size: %d", len(engine.index.KeywordMap))
	}
}

func TestEngine_SpamSkipHandling(t *testing.T) {
	engine := NewEngine(1, 2)
	client := &MockClient{
		id:          "spammer",
		keywords:    []string{"spam"},
		enqueueTime: time.Now(),
	}

	// User clicks "Skip" rapidly 1000 times
	for i := 0; i < 1000; i++ {
		engine.RemoveClient(client)
		engine.AddClient(client)
	}
	
	time.Sleep(200 * time.Millisecond)
	
	engine.mu.Lock()
	defer engine.mu.Unlock()
	
	// The client should only exist exactly once in memory despite 1000 skip clicks
	if len(engine.clients) != 1 {
		t.Errorf("Expected exactly 1 client after spamming, got %d", len(engine.clients))
	}
}
