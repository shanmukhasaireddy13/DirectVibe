package tests

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestLoadConcurrentUsers(t *testing.T) {
	wsURL := "wss://directvibe.onrender.com/ws"
	
	// Number of concurrent clients to simulate
	// Start with 500. A standard free Render instance can typically handle 2k-5k.
	numClients := 500

	headers := make(http.Header)
	headers.Add("Origin", "https://directvibe-web.onrender.com")

	var wg sync.WaitGroup
	var matchedCount int32
	var failedCount int32

	t.Logf("Starting Load Test with %d concurrent users...", numClients)
	startTime := time.Now()

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
			if err != nil {
				atomic.AddInt32(&failedCount, 1)
				// log.Printf("Client %d failed to connect: %v", id, err)
				return
			}
			defer conn.Close()

			joinMsg := map[string]interface{}{
				"type":     "enqueue",
				"keywords": []string{"load-test"},
			}
			if err := conn.WriteJSON(joinMsg); err != nil {
				atomic.AddInt32(&failedCount, 1)
				return
			}

			// Wait for match
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			for {
				var msg map[string]interface{}
				err := conn.ReadJSON(&msg)
				if err != nil {
					atomic.AddInt32(&failedCount, 1)
					return
				}

				if msg["type"] == "match_found" {
					atomic.AddInt32(&matchedCount, 1)
					return
				}
			}
		}(i)

		// Slight stagger to avoid completely overwhelming the local network stack all at once
		time.Sleep(2 * time.Millisecond)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("==================================================")
	t.Logf("Load Test Results:")
	t.Logf("Total Simulated Users: %d", numClients)
	t.Logf("Successfully Matched:  %d", matchedCount)
	t.Logf("Failed Connections:    %d", failedCount)
	t.Logf("Total Time Taken:      %v", duration)
	t.Logf("==================================================")

	if failedCount > int32(numClients/10) {
		t.Fatalf("Too many failures during load test. Network or server might be bottlenecked.")
	}
}
