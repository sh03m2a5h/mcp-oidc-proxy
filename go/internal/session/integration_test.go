package session

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestData represents test session data
type TestData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func TestRedisIntegration(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	logger := zap.NewNop()
	factory := NewFactory(logger)

	config := &config.SessionConfig{
		Store:      "redis",
		TTL:        3600,
		CookieName: "session_id",
		Redis: config.RedisConfig{
			URL:       "redis://" + s.Addr(),
			KeyPrefix: "test:",
		},
	}

	store, err := factory.CreateStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{
		ID:    "user123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	t.Run("Full session lifecycle", func(t *testing.T) {
		// Create session
		sessionID, err := store.Create(ctx, "session1", testData, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, "session1", sessionID)

		// Verify session exists
		exists, err := store.Exists(ctx, "session1")
		require.NoError(t, err)
		assert.True(t, exists)

		// Get session data
		var retrieved TestData
		err = store.Get(ctx, "session1", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, testData, retrieved)

		// Update session
		updatedData := TestData{
			ID:    "user123",
			Name:  "Updated User",
			Email: "updated@example.com",
		}
		err = store.Update(ctx, "session1", updatedData)
		require.NoError(t, err)

		// Verify update
		err = store.Get(ctx, "session1", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, updatedData, retrieved)

		// Refresh session
		err = store.Refresh(ctx, "session1", 2*time.Hour)
		require.NoError(t, err)

		// Delete session
		err = store.Delete(ctx, "session1")
		require.NoError(t, err)

		// Verify deletion
		exists, err = store.Exists(ctx, "session1")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Session expiration", func(t *testing.T) {
		// Create session with short TTL
		sessionID, err := store.Create(ctx, "expiry_test", testData, 100*time.Millisecond)
		require.NoError(t, err)

		// Session should exist initially
		exists, err := store.Exists(ctx, sessionID)
		require.NoError(t, err)
		assert.True(t, exists)

		// Fast-forward miniredis time
		s.FastForward(200 * time.Millisecond)

		// Session should be expired
		exists, err = store.Exists(ctx, sessionID)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Multiple sessions", func(t *testing.T) {
		// Create multiple sessions
		for i := 0; i < 5; i++ {
			sessionID := fmt.Sprintf("multi_session_%d", i)
			data := TestData{
				ID:    fmt.Sprintf("user%d", i),
				Name:  fmt.Sprintf("User %d", i),
				Email: fmt.Sprintf("user%d@example.com", i),
			}
			_, err := store.Create(ctx, sessionID, data, time.Hour)
			require.NoError(t, err)
		}

		// Verify all sessions exist
		for i := 0; i < 5; i++ {
			sessionID := fmt.Sprintf("multi_session_%d", i)
			exists, err := store.Exists(ctx, sessionID)
			require.NoError(t, err)
			assert.True(t, exists)
		}

		// Get stats
		statsInterface, err := store.Stats(ctx)
		require.NoError(t, err)
		assert.NotNil(t, statsInterface)

		// Clean up
		for i := 0; i < 5; i++ {
			sessionID := fmt.Sprintf("multi_session_%d", i)
			err := store.Delete(ctx, sessionID)
			require.NoError(t, err)
		}
	})
}

func TestMemoryVsRedisConsistency(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	logger := zap.NewNop()
	factory := NewFactory(logger)

	// Create both memory and Redis stores
	memoryConfig := &config.SessionConfig{
		Store:      "memory",
		TTL:        3600,
		CookieName: "session_id",
	}

	redisConfig := &config.SessionConfig{
		Store:      "redis",
		TTL:        3600,
		CookieName: "session_id",
		Redis: config.RedisConfig{
			URL:       "redis://" + s.Addr(),
			KeyPrefix: "test:",
		},
	}

	memoryStore, err := factory.CreateStore(memoryConfig)
	require.NoError(t, err)
	defer memoryStore.Close()

	redisStore, err := factory.CreateStore(redisConfig)
	require.NoError(t, err)
	defer redisStore.Close()

	ctx := context.Background()
	testData := TestData{
		ID:    "user123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	stores := map[string]Store{
		"memory": memoryStore,
		"redis":  redisStore,
	}

	// Test same operations on both stores
	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			sessionKey := "consistency_test_" + name

			// Create
			sessionID, err := store.Create(ctx, sessionKey, testData, time.Hour)
			require.NoError(t, err)
			assert.Equal(t, sessionKey, sessionID)

			// Exists
			exists, err := store.Exists(ctx, sessionKey)
			require.NoError(t, err)
			assert.True(t, exists)

			// Get
			var retrieved TestData
			err = store.Get(ctx, sessionKey, &retrieved)
			require.NoError(t, err)
			assert.Equal(t, testData, retrieved)

			// Update
			updatedData := TestData{
				ID:    "user123",
				Name:  "Updated User",
				Email: "updated@example.com",
			}
			err = store.Update(ctx, sessionKey, updatedData)
			require.NoError(t, err)

			// Verify update
			err = store.Get(ctx, sessionKey, &retrieved)
			require.NoError(t, err)
			assert.Equal(t, updatedData, retrieved)

			// Refresh
			err = store.Refresh(ctx, sessionKey, 2*time.Hour)
			require.NoError(t, err)

			// Stats
			statsInterface, err := store.Stats(ctx)
			require.NoError(t, err)
			assert.NotNil(t, statsInterface)

			// Delete
			err = store.Delete(ctx, sessionKey)
			require.NoError(t, err)

			// Verify deletion
			exists, err = store.Exists(ctx, sessionKey)
			require.NoError(t, err)
			assert.False(t, exists)
		})
	}
}

func TestStoreFactory(t *testing.T) {
	logger := zap.NewNop()
	factory := NewFactory(logger)

	t.Run("Create memory store", func(t *testing.T) {
		config := &config.SessionConfig{
			Store:      "memory",
			TTL:        3600,
			CookieName: "session_id",
		}

		store, err := factory.CreateStore(config)
		require.NoError(t, err)
		assert.NotNil(t, store)
		defer store.Close()

		statsInterface, err := store.Stats(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, statsInterface)
	})

	t.Run("Create Redis store with miniredis", func(t *testing.T) {
		s, err := miniredis.Run()
		require.NoError(t, err)
		defer s.Close()

		config := &config.SessionConfig{
			Store:      "redis",
			TTL:        3600,
			CookieName: "session_id",
			Redis: config.RedisConfig{
				URL:       "redis://" + s.Addr(),
				KeyPrefix: "factory_test:",
			},
		}

		store, err := factory.CreateStore(config)
		require.NoError(t, err)
		assert.NotNil(t, store)
		defer store.Close()

		statsInterface, err := store.Stats(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, statsInterface)
	})
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.SessionConfig
		expectError bool
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
			name: "Valid Redis config",
			config: &config.SessionConfig{
				Store:      "redis",
				TTL:        3600,
				CookieName: "session_id",
				Redis: config.RedisConfig{
					URL:       "redis://localhost:6379/0",
					KeyPrefix: "test:",
				},
			},
			expectError: false,
		},
		{
			name: "Invalid store type",
			config: &config.SessionConfig{
				Store:      "invalid",
				TTL:        3600,
				CookieName: "session_id",
			},
			expectError: true,
		},
		{
			name: "Redis without URL",
			config: &config.SessionConfig{
				Store:      "redis",
				TTL:        3600,
				CookieName: "session_id",
				Redis:      config.RedisConfig{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}