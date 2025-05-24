package session

import (
	"context"
	"time"
)

// Store defines the interface for session storage
type Store interface {
	// Create creates a new session with the given key and data
	// Returns the session ID
	// If ttl is 0, the session does not expire
	Create(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error)

	// Get retrieves session data by key
	// The data parameter should be a pointer to the target struct
	Get(ctx context.Context, key string, data interface{}) error

	// Update updates existing session data
	Update(ctx context.Context, key string, data interface{}) error

	// Delete removes a session by key
	Delete(ctx context.Context, key string) error

	// Exists checks if a session exists
	Exists(ctx context.Context, key string) (bool, error)

	// Refresh extends the TTL of a session
	Refresh(ctx context.Context, key string, ttl time.Duration) error

	// Close closes the store connection
	Close() error

	// Cleanup removes expired sessions (optional)
	Cleanup(ctx context.Context) error

	// Stats returns session store statistics (optional)
	Stats(ctx context.Context) (interface{}, error)
}

// Stats holds session store statistics
type Stats struct {
	ActiveSessions int64  `json:"active_sessions"`
	TotalCreated   int64  `json:"total_created"`
	TotalDeleted   int64  `json:"total_deleted"`
	Store          string `json:"store"`
	Info           string `json:"info,omitempty"`
}