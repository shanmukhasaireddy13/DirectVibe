package main

import (
	"log"

	"github.com/directvibe/backend/internal/config"
	"github.com/directvibe/backend/internal/matchmaker"
	"github.com/directvibe/backend/internal/server"
	"github.com/directvibe/backend/internal/websocket"
)

func main() {
	// Enable verbose logging for MVP
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load environment configurations (dev, stg, prod)
	cfg := config.Load()

	// Initialize the connection pool tracker
	pool := websocket.NewPool()

	// Initialize the O(1) Matchmaking Engine
	engine := matchmaker.NewEngine(cfg.StrictMatchWait, cfg.RelaxedMatchWait)

	// Initialize and start the HTTP server
	srv := server.NewServer(cfg, engine, pool)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
