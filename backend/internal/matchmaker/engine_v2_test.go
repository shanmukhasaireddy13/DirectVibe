package matchmaker

import (
	"testing"
	"time"
)

func TestEngine_V2Matching(t *testing.T) {
	engine := NewEngine(5, 10)
	c1 := &MockClient{id: "user1", keywords: []string{"telugu"}, enqueueTime: time.Now()}
	c2 := &MockClient{id: "user2", keywords: []string{"telugu"}, enqueueTime: time.Now()}

	engine.AddClient(c1)
	engine.AddClient(c2)

	time.Sleep(50 * time.Millisecond)

	engine.mu.Lock()
	engine.processMatches()
	engine.mu.Unlock()

	c1.mu.Lock()
	matched1 := c1.matchedWith
	c1.mu.Unlock()

	c2.mu.Lock()
	matched2 := c2.matchedWith
	c2.mu.Unlock()

	if matched1 != "user2" {
		t.Errorf("expected user1 to match user2, got %s", matched1)
	}
	if matched2 != "user1" {
		t.Errorf("expected user2 to match user1, got %s", matched2)
	}
}
