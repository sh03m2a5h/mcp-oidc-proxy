package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Store implements session.Store using in-memory storage
type Store struct {
	mu           sync.RWMutex
	sessions     map[string]*sessionData
	logger       *zap.Logger
	cleanupDone  chan struct{}
	cleanupTimer *time.Timer
	stats        sessionStats
}

// sessionData holds session information
type sessionData struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// sessionStats tracks session statistics
type sessionStats struct {
	totalCreated int64
	totalDeleted int64
}

// Stats holds session store statistics
type Stats struct {
	ActiveSessions int64  `json:"active_sessions"`
	TotalCreated   int64  `json:"total_created"`
	TotalDeleted   int64  `json:"total_deleted"`
	Store          string `json:"store"`
	Info           string `json:"info,omitempty"`
}

// Config holds memory session store configuration
type Config struct {
	// CleanupInterval for removing expired sessions
	CleanupInterval time.Duration
}

// DefaultConfig returns a default memory store configuration
func DefaultConfig() *Config {
	return &Config{
		CleanupInterval: 5 * time.Minute,
	}
}

// NewStore creates a new memory session store
func NewStore(config *Config, logger *zap.Logger) *Store {
	if config == nil {
		config = DefaultConfig()
	}

	store := &Store{
		sessions:    make(map[string]*sessionData),
		logger:      logger,
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup routine
	if config.CleanupInterval > 0 {
		store.startCleanup(config.CleanupInterval)
	}

	return store
}

// startCleanup starts the background cleanup routine
func (s *Store) startCleanup(interval time.Duration) {
	s.cleanupTimer = time.AfterFunc(interval, func() {
		s.cleanup()
		s.startCleanup(interval) // Reschedule
	})
}

// cleanup removes expired sessions
func (s *Store) cleanup() {
	now := time.Now()
	var expiredKeys []string

	// First pass: identify expired sessions with read lock
	s.mu.RLock()
	for key, session := range s.sessions {
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.mu.RUnlock()

	// Second pass: delete expired sessions with write lock (if any found)
	if len(expiredKeys) > 0 {
		s.mu.Lock()
		for _, key := range expiredKeys {
			// Double-check expiration in case session was updated
			if session, exists := s.sessions[key]; exists {
				if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
					delete(s.sessions, key)
					s.stats.totalDeleted++
				}
			}
		}
		s.mu.Unlock()

		s.logger.Debug("Cleaned up expired sessions", zap.Int("count", len(expiredKeys)))
	}
}

// Create creates a new session with the given key and data
func (s *Store) Create(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if _, exists := s.sessions[key]; exists {
		return "", fmt.Errorf("session already exists")
	}

	session := &sessionData{
		Data:      jsonData,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set expiration if TTL is provided
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		session.ExpiresAt = &expiresAt
	}

	s.sessions[key] = session
	s.stats.totalCreated++

	s.logger.Debug("Session created",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)

	return key, nil
}

// Get retrieves session data by key
func (s *Store) Get(ctx context.Context, key string, data interface{}) error {
	s.mu.RLock()
	session, exists := s.sessions[key]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if session is expired
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		// Remove expired session
		s.mu.Lock()
		delete(s.sessions, key)
		s.stats.totalDeleted++
		s.mu.Unlock()
		return fmt.Errorf("session expired")
	}

	// Deserialize JSON data
	if err := json.Unmarshal(session.Data, data); err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	s.logger.Debug("Session retrieved", zap.String("key", key))
	return nil
}

// Update updates existing session data
func (s *Store) Update(ctx context.Context, key string, data interface{}) error {
	// Serialize new data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[key]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if session is expired
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		delete(s.sessions, key)
		s.stats.totalDeleted++
		return fmt.Errorf("session expired")
	}

	// Update data and timestamp
	session.Data = jsonData
	session.UpdatedAt = time.Now()

	s.logger.Debug("Session updated", zap.String("key", key))
	return nil
}

// Delete removes a session by key
func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[key]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(s.sessions, key)
	s.stats.totalDeleted++

	s.logger.Debug("Session deleted", zap.String("key", key))
	return nil
}

// Exists checks if a session exists
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	session, exists := s.sessions[key]
	s.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Check if session is expired
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		// Remove expired session
		s.mu.Lock()
		delete(s.sessions, key)
		s.stats.totalDeleted++
		s.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Refresh extends the TTL of a session
func (s *Store) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[key]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if session is expired
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		delete(s.sessions, key)
		s.stats.totalDeleted++
		return fmt.Errorf("session expired")
	}

	// Update expiration
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		session.ExpiresAt = &expiresAt
	} else {
		session.ExpiresAt = nil // No expiration
	}
	session.UpdatedAt = time.Now()

	s.logger.Debug("Session TTL refreshed",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)
	return nil
}

// Close closes the store and stops cleanup routine
func (s *Store) Close() error {
	if s.cleanupTimer != nil {
		s.cleanupTimer.Stop()
	}
	close(s.cleanupDone)

	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear all sessions
	s.sessions = make(map[string]*sessionData)
	
	s.logger.Debug("Memory session store closed")
	return nil
}

// Cleanup manually triggers cleanup of expired sessions
func (s *Store) Cleanup(ctx context.Context) error {
	s.cleanup()
	return nil
}

// Stats returns session store statistics
func (s *Store) Stats(ctx context.Context) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &Stats{
		ActiveSessions: int64(len(s.sessions)),
		TotalCreated:   s.stats.totalCreated,
		TotalDeleted:   s.stats.totalDeleted,
		Store:          "memory",
		Info:           fmt.Sprintf("active_sessions=%d", len(s.sessions)),
	}, nil
}