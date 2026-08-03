package tests

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestAdvancedLoadAndSkip(t *testing.T) {
	wsURL := "wss://directvibe.onrender.com/ws"
	numClients := 1000 // Test with 1000 concurrent users

	headers := make(http.Header)
	headers.Add("Origin", "https://directvibe-web.onrender.com")

	var wg sync.WaitGroup
	var initialMatches int32
	var successfulSkips int32
	var rematched int32
	var failedCount int32

	t.Logf("Starting Advanced Load Test with %d concurrent users...", numClients)
	startTime := time.Now()

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
			if err != nil {
				atomic.AddInt32(&failedCount, 1)
				return
			}
			defer conn.Close()

			// 1. Join queue
			joinMsg := map[string]interface{}{
				"type":     "enqueue",
				"keywords": []string{"advanced-load-test"},
			}
			if err := conn.WriteJSON(joinMsg); err != nil {
				atomic.AddInt32(&failedCount, 1)
				return
			}

			conn.SetReadDeadline(time.Now().Add(45 * time.Second))
			firstMatchFound := false
			skipped := false

			for {
				var msg map[string]interface{}
				err := conn.ReadJSON(&msg)
				if err != nil {
					return // Timeout or disconnect
				}

				if msg["type"] == "match_found" {
					if !firstMatchFound {
						firstMatchFound = true
						atomic.AddInt32(&initialMatches, 1)

						// 20% of users simulate a SKIP immediately
						if id%5 == 0 {
							skipped = true
							skipMsg := map[string]interface{}{
								"type": "skip",
							}
							conn.WriteJSON(skipMsg)
							atomic.AddInt32(&successfulSkips, 1)

							// Immediately enqueue again to find a new match
							conn.WriteJSON(joinMsg)
						} else {
							// Normal users wait here. They might receive a "peer_left" if their partner was one of the 20% who skipped.
						}
					} else if skipped {
						// This is the second match for the skip users!
						atomic.AddInt32(&rematched, 1)
						return
					}
				} else if msg["type"] == "peer_left" {
					// The person we matched with skipped!
					// We must automatically re-enqueue, just like the frontend does.
					if !skipped {
						conn.WriteJSON(joinMsg)
					}
				}
			}
		}(i)

		// Stagger connections slowly to avoid local ephemeral port exhaustion and Render rate limiting
		time.Sleep(3 * time.Millisecond)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("==================================================")
	t.Logf("Advanced Load Test Results:")
	t.Logf("Total Simulated Users:  %d", numClients)
	t.Logf("Initial Matches Made:   %d", initialMatches)
	t.Logf("Users Who Skipped:      %d", successfulSkips)
	t.Logf("Successfully Rematched: %d", rematched)
	t.Logf("Failed Connections:     %d", failedCount)
	t.Logf("Total Time Taken:       %v", duration)
	t.Logf("==================================================")

	if failedCount > int32(numClients/10) {
		t.Fatalf("Too many failures during load test.")
	}
}
