package tests

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestLiveProductionSignaling(t *testing.T) {
	wsURL := "wss://directvibe.onrender.com/ws"
	
	headers := make(http.Header)
	headers.Add("Origin", "https://directvibe-web.onrender.com")

	// Create Client A
	connA, respA, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		status := "no response"
		if respA != nil {
			status = respA.Status
		}
		log.Fatalf("Client A failed to connect to live server %s: %v (Status: %s)", wsURL, err, status)
	}
	defer connA.Close()

	// Create Client B
	connB, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		log.Fatalf("Client B failed to connect to live server %s: %v", wsURL, err)
	}
	defer connB.Close()

	// Join the queue
	joinMsg := map[string]interface{}{
		"type":     "enqueue",
		"keywords": []string{"e2e-live-test"},
	}

	connA.WriteJSON(joinMsg)
	connB.WriteJSON(joinMsg)

	// Wait for Match
	var peerA, peerB string
	matchChanA := make(chan map[string]interface{})
	matchChanB := make(chan map[string]interface{})

	go func() {
		for {
			var msg map[string]interface{}
			if err := connA.ReadJSON(&msg); err != nil { return }
			if msg["type"] == "match_found" {
				matchChanA <- msg
				return
			}
		}
	}()

	go func() {
		for {
			var msg map[string]interface{}
			if err := connB.ReadJSON(&msg); err != nil { return }
			if msg["type"] == "match_found" {
				matchChanB <- msg
				return
			}
		}
	}()

	select {
	case matchA := <-matchChanA:
		peerA = matchA["peer_id"].(string)
		log.Printf("Client A matched with: %s (Is Offer: %v)", peerA, matchA["offer"]) // Check offer or is_offer
	case <-time.After(10 * time.Second):
		log.Fatal("Client A failed to receive match_found within 10s")
	}

	select {
	case matchB := <-matchChanB:
		peerB = matchB["peer_id"].(string)
		log.Printf("Client B matched with: %s", peerB)
	case <-time.After(10 * time.Second):
		log.Fatal("Client B failed to receive match_found within 10s")
	}

	// Verify Signal Relay (Client A sending to Client B)
	// We'll test the EXACT batching scenario the frontend used to fail on
	
	go func() {
		// Client A rapidly sends Offer + multiple candidates to test if the backend buffers/relays them correctly
		offerPayload := map[string]interface{}{
			"type": "webrtc_signal",
			"target_id": peerA,
			"signal": map[string]interface{}{
				"type": "offer",
				"sdp": "v=0\r\no=jdoe 2890844526 2890842807 IN IP4 10.47.16.5\r\n",
			},
		}
		connA.WriteJSON(offerPayload)
		
		for i := 0; i < 5; i++ {
			candidatePayload := map[string]interface{}{
				"type": "webrtc_signal",
				"target_id": peerA,
				"signal": map[string]interface{}{
					"type": "candidate",
					"candidate": "candidate:12345 1 udp 2122260223 192.168.1.100 55000 typ host generation 0",
				},
			}
			connA.WriteJSON(candidatePayload)
		}
		log.Println("Client A sent 1 Offer + 5 ICE Candidates in rapid succession")
	}()

	// Verify Client B receives exactly 6 WebRTC signals
	receivedOffer := false
	candidateCount := 0

	for i := 0; i < 6; i++ {
		c := make(chan map[string]interface{})
		go func() {
			var msg map[string]interface{}
			connB.ReadJSON(&msg)
			c <- msg
		}()

		select {
		case msg := <-c:
			if msg["type"] == "webrtc_signal" {
				sig, ok := msg["signal"].(map[string]interface{})
				if ok {
					if sig["type"] == "offer" {
						receivedOffer = true
						log.Println("Client B successfully received relay: OFFER")
					} else if sig["type"] == "candidate" {
						candidateCount++
						log.Printf("Client B successfully received relay: CANDIDATE (%d/5)", candidateCount)
					}
				}
			}
		case <-time.After(3 * time.Second):
			log.Fatalf("Timeout waiting for relayed signal %d/6", i+1)
		}
	}

	if !receivedOffer {
		log.Fatal("Client B did NOT receive the offer signal!")
	}
	if candidateCount != 5 {
		log.Fatalf("Client B expected 5 candidates, got %d", candidateCount)
	}

	log.Println("SUCCESS: E2E Live Signaling Server is routing and relaying rapidly batched WebRTC signals perfectly.")
}
