package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/directvibe/backend/internal/config"
	"github.com/directvibe/backend/internal/matchmaker"
	ws "github.com/directvibe/backend/internal/websocket"
)

// Server encapsulates the HTTP and WebSocket transport layer
type Server struct {
	cfg      *config.Config
	upgrader websocket.Upgrader
	pool     *ws.Pool
	engine   *matchmaker.Engine
}

func isAllowedOrigin(origin, frontendURLConfig string) bool {
	if frontendURLConfig == "*" || frontendURLConfig == "" {
		return true
	}

	originClean := strings.TrimRight(strings.ToLower(origin), "/")
	allowedOrigins := strings.Split(frontendURLConfig, ",")

	for _, allowed := range allowedOrigins {
		allowedClean := strings.TrimSpace(strings.TrimRight(strings.ToLower(allowed), "/"))
		if allowedClean == "*" || allowedClean == "" || originClean == allowedClean {
			return true
		}
	}
	return false
}

// NewServer initializes the HTTP server with routes and CORS config
func NewServer(cfg *config.Config, engine *matchmaker.Engine, pool *ws.Pool) *Server {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowed := isAllowedOrigin(origin, cfg.FrontendURL)
			if !allowed {
				log.Printf("WebSocket CheckOrigin rejected origin: %s (configured FRONTEND_URL: %s)", origin, cfg.FrontendURL)
			}
			return allowed
		},
	}

	return &Server{
		cfg:      cfg,
		upgrader: upgrader,
		pool:     pool,
		engine:   engine,
	}
}

// HandleWebSocket upgrades HTTP to WSS and spins up client pumps
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Use WebSocket Key + Remote Addr as a unique identifier for MVP
	clientID := r.Header.Get("Sec-WebSocket-Key")
	if clientID == "" {
		clientID = r.RemoteAddr
	}

	client := ws.NewClient(clientID, conn, s.pool, s.engine)
	s.pool.Register(client)

	// Start the decoupled IO threads
	go client.WritePump()
	go client.ReadPump()
}

// Start begins listening on the configured port
func (s *Server) Start() error {
	http.HandleFunc("/ws", s.HandleWebSocket)
	
	// Add healthcheck route
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Server starting on port %s in %s mode", s.cfg.Port, s.cfg.Env)
	return http.ListenAndServe(":"+s.cfg.Port, nil)
}
