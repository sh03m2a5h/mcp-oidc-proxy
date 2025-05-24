package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Store implements session.Store using Redis as the backend
type Store struct {
	client    redis.Cmdable
	keyPrefix string
	logger    *zap.Logger
}

// Stats holds session store statistics
type Stats struct {
	ActiveSessions int64  `json:"active_sessions"`
	TotalCreated   int64  `json:"total_created"`
	TotalDeleted   int64  `json:"total_deleted"`
	Store          string `json:"store"`
	Info           string `json:"info,omitempty"`
}

// Config holds Redis session store configuration
type Config struct {
	// Redis connection URL (redis://localhost:6379/0)
	URL string
	// Password for Redis authentication
	Password string
	// Database number (0-15)
	DB int
	// Key prefix for session keys
	KeyPrefix string
	// Connection pool size
	PoolSize int
	// Minimum idle connections
	MinIdleConns int
	// Connection timeout
	DialTimeout time.Duration
	// Read timeout
	ReadTimeout time.Duration
	// Write timeout
	WriteTimeout time.Duration
}

// DefaultConfig returns a default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		URL:          "redis://localhost:6379/0",
		KeyPrefix:    "session:",
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewStore creates a new Redis session store
func NewStore(config *Config, logger *zap.Logger) (*Store, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with config values
	if config.Password != "" {
		opt.Password = config.Password
	}
	if config.DB > 0 {
		opt.DB = config.DB
	}
	if config.PoolSize > 0 {
		opt.PoolSize = config.PoolSize
	}
	if config.MinIdleConns > 0 {
		opt.MinIdleConns = config.MinIdleConns
	}
	if config.DialTimeout > 0 {
		opt.DialTimeout = config.DialTimeout
	}
	if config.ReadTimeout > 0 {
		opt.ReadTimeout = config.ReadTimeout
	}
	if config.WriteTimeout > 0 {
		opt.WriteTimeout = config.WriteTimeout
	}

	// Create Redis client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	keyPrefix := config.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "session:"
	}

	return &Store{
		client:    client,
		keyPrefix: keyPrefix,
		logger:    logger,
	}, nil
}

// NewStoreWithClient creates a new Redis session store with an existing Redis client
func NewStoreWithClient(client redis.Cmdable, keyPrefix string, logger *zap.Logger) *Store {
	if keyPrefix == "" {
		keyPrefix = "session:"
	}
	return &Store{
		client:    client,
		keyPrefix: keyPrefix,
		logger:    logger,
	}
}

// Create creates a new session with the given key and data
func (s *Store) Create(ctx context.Context, key string, data interface{}, ttl time.Duration) (string, error) {
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Generate full key with prefix
	fullKey := s.keyPrefix + key

	// Store in Redis
	if ttl > 0 {
		err = s.client.Set(ctx, fullKey, jsonData, ttl).Err()
	} else {
		err = s.client.Set(ctx, fullKey, jsonData, 0).Err()
	}

	if err != nil {
		return "", fmt.Errorf("failed to store session in Redis: %w", err)
	}

	s.logger.Debug("Session created",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)

	return key, nil
}

// Get retrieves session data by key
func (s *Store) Get(ctx context.Context, key string, data interface{}) error {
	// Generate full key with prefix
	fullKey := s.keyPrefix + key

	// Get from Redis
	jsonData, err := s.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("session not found")
		}
		return fmt.Errorf("failed to get session from Redis: %w", err)
	}

	// Deserialize JSON data
	if err := json.Unmarshal([]byte(jsonData), data); err != nil {
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

	fullKey := s.keyPrefix + key

	// Use Lua script for atomic update operation
	script := `
		local key = KEYS[1]
		local data = ARGV[1]
		
		-- Check if key exists
		if redis.call('EXISTS', key) == 0 then
			return {err = 'session not found'}
		end
		
		-- Get current TTL
		local ttl = redis.call('TTL', key)
		
		-- Update with same TTL
		if ttl > 0 then
			redis.call('SET', key, data, 'EX', ttl)
		else
			redis.call('SET', key, data)
		end
		
		return {ok = 'updated'}
	`

	result, err := s.client.Eval(ctx, script, []string{fullKey}, string(jsonData)).Result()
	if err != nil {
		return fmt.Errorf("failed to execute update script: %w", err)
	}

	// Check result
	if resultMap, ok := result.(map[interface{}]interface{}); ok {
		if errMsg, exists := resultMap["err"]; exists {
			return fmt.Errorf("%v", errMsg)
		}
	}

	s.logger.Debug("Session updated", zap.String("key", key))
	return nil
}

// Delete removes a session by key
func (s *Store) Delete(ctx context.Context, key string) error {
	// Generate full key with prefix
	fullKey := s.keyPrefix + key

	// Delete from Redis
	deleted, err := s.client.Del(ctx, fullKey).Result()
	if err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	if deleted == 0 {
		return fmt.Errorf("session not found")
	}

	s.logger.Debug("Session deleted", zap.String("key", key))
	return nil
}

// Exists checks if a session exists
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	// Generate full key with prefix
	fullKey := s.keyPrefix + key

	// Check existence in Redis
	exists, err := s.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return exists > 0, nil
}

// Refresh extends the TTL of a session
func (s *Store) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	// Generate full key with prefix
	fullKey := s.keyPrefix + key

	// Check if session exists
	exists, err := s.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("session not found")
	}

	// Update TTL
	if ttl > 0 {
		err = s.client.Expire(ctx, fullKey, ttl).Err()
	} else {
		err = s.client.Persist(ctx, fullKey).Err()
	}

	if err != nil {
		return fmt.Errorf("failed to refresh session TTL: %w", err)
	}

	s.logger.Debug("Session TTL refreshed",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)
	return nil
}

// Close closes the Redis connection
func (s *Store) Close() error {
	if client, ok := s.client.(*redis.Client); ok {
		return client.Close()
	}
	// For redis.Cmdable interface, we can't close it directly
	return nil
}

// Cleanup removes expired sessions (Redis handles this automatically)
func (s *Store) Cleanup(ctx context.Context) error {
	// Redis automatically handles expiration, but we can implement
	// custom cleanup logic if needed
	s.logger.Debug("Session cleanup requested (Redis handles expiration automatically)")
	return nil
}

// Stats returns session store statistics
func (s *Store) Stats(ctx context.Context) (interface{}, error) {
	// Get Redis info (simplified for interface compatibility)
	info := "keyspace info not available"
	if client, ok := s.client.(*redis.Client); ok {
		if result, err := client.Do(ctx, "INFO", "keyspace").Result(); err == nil {
			info = fmt.Sprintf("%v", result)
		}
	}

	// Count sessions with our prefix using SCAN (non-blocking)
	pattern := s.keyPrefix + "*"
	var keys []string
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan sessions: %w", err)
	}

	return &Stats{
		ActiveSessions: int64(len(keys)),
		TotalCreated:   -1, // Redis doesn't track this
		TotalDeleted:   -1, // Redis doesn't track this
		Store:          "redis",
		Info:           fmt.Sprintf("%v", info),
	}, nil
}