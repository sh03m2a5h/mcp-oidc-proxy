package session

import (
	"testing"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFactory(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)
	assert.NotNil(t, factory)
	assert.Equal(t, logger, factory.logger)
}

func TestCreateMemoryStore(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)

	config := &config.SessionConfig{
		Store: "memory",
		TTL:   3600,
	}

	store, err := factory.CreateStore(config)
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Verify it's actually a memory store by testing interface compliance
	statsInterface, err := store.Stats(nil)
	require.NoError(t, err)
	assert.NotNil(t, statsInterface)

	store.Close()
}

func TestCreateRedisStore(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)

	config := &config.SessionConfig{
		Store: "redis",
		Redis: config.RedisConfig{
			URL:       "redis://localhost:6379/0",
			KeyPrefix: "test:",
		},
	}

	// This test will fail if Redis is not available, which is expected
	store, err := factory.CreateStore(config)
	if err != nil {
		// If Redis is not available, we expect a connection error
		assert.Contains(t, err.Error(), "failed to create Redis session store")
		return
	}

	// If Redis is available, verify the store was created
	assert.NotNil(t, store)
	statsInterface, err := store.Stats(nil)
	require.NoError(t, err)
	assert.NotNil(t, statsInterface)

	store.Close()
}

func TestCreateRedisStoreWithoutURL(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)

	config := &config.SessionConfig{
		Store: "redis",
		Redis: config.RedisConfig{
			// Missing URL
			KeyPrefix: "test:",
		},
	}

	store, err := factory.CreateStore(config)
	assert.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "Redis URL is required")
}

func TestCreateUnsupportedStore(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)

	config := &config.SessionConfig{
		Store: "unsupported",
	}

	store, err := factory.CreateStore(config)
	assert.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "unsupported session store type")
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.SessionConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid memory config",
			config: &config.SessionConfig{
				Store:      "memory",
				TTL:        3600,
				CookieName: "session_id",
			},
			expectError: false,
		},
		{
			name: "Valid redis config",
			config: &config.SessionConfig{
				Store:      "redis",
				TTL:        3600,
				CookieName: "session_id",
				Redis: config.RedisConfig{
					URL: "redis://localhost:6379/0",
					DB:  0,
				},
			},
			expectError: false,
		},
		{
			name: "Missing store type",
			config: &config.SessionConfig{
				TTL:        3600,
				CookieName: "session_id",
			},
			expectError: true,
			errorMsg:    "session store type is required",
		},
		{
			name: "Unsupported store type",
			config: &config.SessionConfig{
				Store:      "invalid",
				TTL:        3600,
				CookieName: "session_id",
			},
			expectError: true,
			errorMsg:    "unsupported session store type",
		},
		{
			name: "Redis without URL",
			config: &config.SessionConfig{
				Store:      "redis",
				TTL:        3600,
				CookieName: "session_id",
				Redis: config.RedisConfig{
					DB: 0,
				},
			},
			expectError: true,
			errorMsg:    "Redis URL is required",
		},
		{
			name: "Redis with invalid DB",
			config: &config.SessionConfig{
				Store:      "redis",
				TTL:        3600,
				CookieName: "session_id",
				Redis: config.RedisConfig{
					URL: "redis://localhost:6379/0",
					DB:  16, // Invalid DB number
				},
			},
			expectError: true,
			errorMsg:    "Redis DB must be between 0 and 15",
		},
		{
			name: "Negative TTL",
			config: &config.SessionConfig{
				Store:      "memory",
				TTL:        -1,
				CookieName: "session_id",
			},
			expectError: true,
			errorMsg:    "session TTL cannot be negative",
		},
		{
			name: "Missing cookie name",
			config: &config.SessionConfig{
				Store: "memory",
				TTL:   3600,
			},
			expectError: true,
			errorMsg:    "session cookie name is required",
		},
		{
			name: "Invalid SameSite value",
			config: &config.SessionConfig{
				Store:          "memory",
				TTL:            3600,
				CookieName:     "session_id",
				CookieSameSite: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid cookie SameSite value",
		},
		{
			name: "Valid SameSite strict",
			config: &config.SessionConfig{
				Store:          "memory",
				TTL:            3600,
				CookieName:     "session_id",
				CookieSameSite: "strict",
			},
			expectError: false,
		},
		{
			name: "Valid SameSite lax",
			config: &config.SessionConfig{
				Store:          "memory",
				TTL:            3600,
				CookieName:     "session_id",
				CookieSameSite: "lax",
			},
			expectError: false,
		},
		{
			name: "Valid SameSite none",
			config: &config.SessionConfig{
				Store:          "memory",
				TTL:            3600,
				CookieName:     "session_id",
				CookieSameSite: "none",
			},
			expectError: false,
		},
		{
			name: "Valid empty SameSite",
			config: &config.SessionConfig{
				Store:          "memory",
				TTL:            3600,
				CookieName:     "session_id",
				CookieSameSite: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}