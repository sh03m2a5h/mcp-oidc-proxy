package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

func TestDefaultConfigSimple(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "redis://localhost:6379/0", config.URL)
	assert.Equal(t, "session:", config.KeyPrefix)
	assert.Equal(t, 10, config.PoolSize)
	assert.Equal(t, 5, config.MinIdleConns)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 3*time.Second, config.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.WriteTimeout)
}

func TestNewStoreWithInvalidURLSimple(t *testing.T) {
	config := &Config{
		URL: "invalid-url",
	}
	logger := zap.NewNop()

	store, err := NewStore(config, logger)
	assert.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "failed to parse Redis URL")
}

func TestRedisStoreWithMiniredis(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	config := &Config{
		URL:       "redis://" + s.Addr(),
		KeyPrefix: "test:",
	}
	logger := zap.NewNop()

	store, err := NewStore(config, logger)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{
		ID:    "user123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	t.Run("Create and Get session", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "session1", testData, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, "session1", sessionID)

		var retrieved TestData
		err = store.Get(ctx, "session1", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, testData, retrieved)
	})

	t.Run("Update session", func(t *testing.T) {
		updatedData := TestData{
			ID:    "user123",
			Name:  "Updated User",
			Email: "updated@example.com",
		}

		err := store.Update(ctx, "session1", updatedData)
		require.NoError(t, err)

		var retrieved TestData
		err = store.Get(ctx, "session1", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, updatedData, retrieved)
	})

	t.Run("Session exists", func(t *testing.T) {
		exists, err := store.Exists(ctx, "session1")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = store.Exists(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Refresh session", func(t *testing.T) {
		// Create a session first
		testData := TestData{ID: "refresh_test", Name: "Test"}
		sessionID, err := store.Create(ctx, "refresh_session", testData, time.Hour)
		require.NoError(t, err)

		// Refresh the session TTL
		err = store.Refresh(ctx, sessionID, 2*time.Hour)
		require.NoError(t, err)

		// Session should still exist
		exists, err := store.Exists(ctx, sessionID)
		require.NoError(t, err)
		assert.True(t, exists)

		// Clean up
		err = store.Delete(ctx, sessionID)
		require.NoError(t, err)
	})

	t.Run("Delete session", func(t *testing.T) {
		err := store.Delete(ctx, "session1")
		require.NoError(t, err)

		var retrieved TestData
		err = store.Get(ctx, "session1", &retrieved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}

func TestSessionExpiration(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	config := &Config{
		URL:       "redis://" + s.Addr(),
		KeyPrefix: "expire_test:",
	}
	logger := zap.NewNop()

	store, err := NewStore(config, logger)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	t.Run("Session with TTL", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "ttl_session", testData, 100*time.Millisecond)
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
}

func TestStatsSimple(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	config := &Config{
		URL:       "redis://" + s.Addr(),
		KeyPrefix: "stats_test:",
	}
	logger := zap.NewNop()

	store, err := NewStore(config, logger)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	// Create some sessions
	_, err = store.Create(ctx, "session1", testData, time.Hour)
	require.NoError(t, err)
	_, err = store.Create(ctx, "session2", testData, time.Hour)
	require.NoError(t, err)

	statsInterface, err := store.Stats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, statsInterface)

	// Type assert to verify structure
	if stats, ok := statsInterface.(*Stats); ok {
		assert.Equal(t, "redis", stats.Store)
		assert.Equal(t, int64(2), stats.ActiveSessions)
	}
}

func TestCleanupSimple(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	config := &Config{
		URL:       "redis://" + s.Addr(),
		KeyPrefix: "cleanup_test:",
	}
	logger := zap.NewNop()

	store, err := NewStore(config, logger)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	err = store.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestNewStoreWithClient(t *testing.T) {
	// Start miniredis server
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	logger := zap.NewNop()

	store := NewStoreWithClient(client, "test:", logger)
	assert.NotNil(t, store)
	assert.Equal(t, "test:", store.keyPrefix)
	assert.Equal(t, client, store.client)
	assert.Equal(t, logger, store.logger)

	// Test basic operations
	ctx := context.Background()
	testData := TestData{ID: "user123", Name: "Test"}

	sessionID, err := store.Create(ctx, "test_session", testData, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "test_session", sessionID)

	var retrieved TestData
	err = store.Get(ctx, "test_session", &retrieved)
	require.NoError(t, err)
	assert.Equal(t, testData, retrieved)
}