package matchmaker

import "time"

// Client defines the interface that a websocket connection must implement
// to be managed by the matchmaker. This prevents circular dependencies.
type Client interface {
	ID() string
	Keywords() []string
	EnqueueTime() time.Time
	SendMatch(otherID string, offer bool)
	HasSkipped(otherID string) bool
	AddSkip(otherID string)
	IsGhost() bool
}
