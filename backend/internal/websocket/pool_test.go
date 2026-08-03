package websocket

import (
	"testing"
)

func TestPool_RegisterUnregister(t *testing.T) {
	pool := NewPool()
	client := &Client{id: "client-1"}

	pool.Register(client)
	if pool.Get("client-1") != client {
		t.Fatalf("Failed to retrieve registered client from pool")
	}

	pool.Unregister(client)
	if pool.Get("client-1") != nil {
		t.Fatalf("Failed to unregister client from pool")
	}
}
