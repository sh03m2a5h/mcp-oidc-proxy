package session

import (
	"fmt"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session/memory"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session/redis"
	"go.uber.org/zap"
)

// Factory creates session stores based on configuration
type Factory struct {
	logger *zap.Logger
}

// NewFactory creates a new session store factory
func NewFactory(logger *zap.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateStore creates a session store based on the configuration
func (f *Factory) CreateStore(config *config.SessionConfig) (Store, error) {
	switch config.Store {
	case "redis":
		return f.createRedisStore(config)
	case "memory":
		return f.createMemoryStore(config)
	default:
		return nil, fmt.Errorf("unsupported session store type: %s", config.Store)
	}
}

// createRedisStore creates a Redis session store
func (f *Factory) createRedisStore(config *config.SessionConfig) (Store, error) {
	redisConfig := &redis.Config{
		URL:          config.Redis.URL,
		Password:     config.Redis.Password,
		DB:           config.Redis.DB,
		KeyPrefix:    config.Redis.KeyPrefix,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Validate Redis configuration
	if redisConfig.URL == "" {
		return nil, fmt.Errorf("Redis URL is required for Redis session store")
	}

	if redisConfig.KeyPrefix == "" {
		redisConfig.KeyPrefix = "session:"
	}

	store, err := redis.NewStore(redisConfig, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis session store: %w", err)
	}

	f.logger.Info("Redis session store created",
		zap.String("url", redisConfig.URL),
		zap.String("key_prefix", redisConfig.KeyPrefix),
	)

	return store, nil
}

// createMemoryStore creates an in-memory session store
func (f *Factory) createMemoryStore(config *config.SessionConfig) (Store, error) {
	memoryConfig := &memory.Config{
		CleanupInterval: 5 * time.Minute,
	}

	store := memory.NewStore(memoryConfig, f.logger)

	f.logger.Info("Memory session store created",
		zap.Duration("cleanup_interval", memoryConfig.CleanupInterval),
	)

	return store, nil
}

// ValidateConfig validates session configuration
func ValidateConfig(config *config.SessionConfig) error {
	if config.Store == "" {
		return fmt.Errorf("session store type is required")
	}

	switch config.Store {
	case "redis":
		if config.Redis.URL == "" {
			return fmt.Errorf("Redis URL is required for Redis session store")
		}
		if config.Redis.DB < 0 || config.Redis.DB > 15 {
			return fmt.Errorf("Redis DB must be between 0 and 15")
		}
	case "memory":
		// Memory store has no specific requirements
	default:
		return fmt.Errorf("unsupported session store type: %s (supported: redis, memory)", config.Store)
	}

	// Validate session configuration
	if config.TTL < 0 {
		return fmt.Errorf("session TTL cannot be negative")
	}

	// Validate cookie settings
	if config.CookieName == "" {
		return fmt.Errorf("session cookie name is required")
	}

	validSameSiteValues := []string{"strict", "lax", "none", ""}
	isValidSameSite := false
	for _, valid := range validSameSiteValues {
		if config.CookieSameSite == valid {
			isValidSameSite = true
			break
		}
	}
	if !isValidSameSite {
		return fmt.Errorf("invalid cookie SameSite value: %s (valid: strict, lax, none, empty)", config.CookieSameSite)
	}

	return nil
}