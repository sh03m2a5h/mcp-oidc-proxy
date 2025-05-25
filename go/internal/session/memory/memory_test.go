package memory

import (
	"context"
	"testing"
	"time"

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

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
}

func TestNewStore(t *testing.T) {
	logger := zap.NewNop()

	t.Run("with config", func(t *testing.T) {
		config := &Config{
			CleanupInterval: 10 * time.Minute,
		}
		store := NewStore(config, logger)
		assert.NotNil(t, store)
		assert.NotNil(t, store.sessions)
		assert.NotNil(t, store.cleanupTimer)
		store.Close()
	})

	t.Run("with nil config", func(t *testing.T) {
		store := NewStore(nil, logger)
		assert.NotNil(t, store)
		assert.NotNil(t, store.sessions)
		assert.NotNil(t, store.cleanupTimer)
		store.Close()
	})

	t.Run("without cleanup", func(t *testing.T) {
		config := &Config{
			CleanupInterval: 0,
		}
		store := NewStore(config, logger)
		assert.NotNil(t, store)
		assert.NotNil(t, store.sessions)
		assert.Nil(t, store.cleanupTimer)
		store.Close()
	})
}

func TestStoreOperations(t *testing.T) {
	config := &Config{
		CleanupInterval: 0, // Disable cleanup for tests
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{
		ID:    "user123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	t.Run("Create session", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "session1", testData, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, "session1", sessionID)

		// Verify session was created
		assert.Len(t, store.sessions, 1)
		assert.Contains(t, store.sessions, "session1")
	})

	t.Run("Create duplicate session", func(t *testing.T) {
		_, err := store.Create(ctx, "session1", testData, time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session already exists")
	})

	t.Run("Get session", func(t *testing.T) {
		var retrieved TestData
		err := store.Get(ctx, "session1", &retrieved)
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
		err := store.Refresh(ctx, "session1", 2*time.Hour)
		require.NoError(t, err)

		// Verify expiration was updated
		session := store.sessions["session1"]
		assert.NotNil(t, session.ExpiresAt)
		assert.True(t, session.ExpiresAt.After(time.Now().Add(time.Hour)))
	})

	t.Run("Delete session", func(t *testing.T) {
		err := store.Delete(ctx, "session1")
		require.NoError(t, err)

		var retrieved TestData
		err = store.Get(ctx, "session1", &retrieved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")

		assert.Len(t, store.sessions, 0)
	})
}

func TestStoreErrors(t *testing.T) {
	config := &Config{
		CleanupInterval: 0,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()

	t.Run("Get nonexistent session", func(t *testing.T) {
		var data TestData
		err := store.Get(ctx, "nonexistent", &data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Update nonexistent session", func(t *testing.T) {
		err := store.Update(ctx, "nonexistent", TestData{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Delete nonexistent session", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Refresh nonexistent session", func(t *testing.T) {
		err := store.Refresh(ctx, "nonexistent", time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}

func TestSessionExpiration(t *testing.T) {
	config := &Config{
		CleanupInterval: 0,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	t.Run("Create session with short TTL", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "short_session", testData, 50*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, "short_session", sessionID)

		// Session should exist initially
		exists, err := store.Exists(ctx, "short_session")
		require.NoError(t, err)
		assert.True(t, exists)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Session should be expired
		var retrieved TestData
		err = store.Get(ctx, "short_session", &retrieved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session expired")

		// Session should no longer exist
		exists, err = store.Exists(ctx, "short_session")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Create session without TTL", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "permanent_session", testData, 0)
		require.NoError(t, err)
		assert.Equal(t, "permanent_session", sessionID)

		// Session should exist
		var retrieved TestData
		err = store.Get(ctx, "permanent_session", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, testData, retrieved)

		// Session should not have expiration
		session := store.sessions["permanent_session"]
		assert.Nil(t, session.ExpiresAt)
	})

	t.Run("Refresh session to remove TTL", func(t *testing.T) {
		sessionID, err := store.Create(ctx, "ttl_session", testData, time.Hour)
		require.NoError(t, err)

		// Verify session has expiration
		session := store.sessions[sessionID]
		assert.NotNil(t, session.ExpiresAt)

		// Refresh with 0 TTL (remove expiration)
		err = store.Refresh(ctx, sessionID, 0)
		require.NoError(t, err)

		// Verify expiration was removed
		session = store.sessions[sessionID]
		assert.Nil(t, session.ExpiresAt)
	})
}

func TestCleanup(t *testing.T) {
	config := &Config{
		CleanupInterval: 0,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	// Create sessions with different TTLs
	_, err := store.Create(ctx, "permanent", testData, 0)
	require.NoError(t, err)

	_, err = store.Create(ctx, "short_lived", testData, 50*time.Millisecond)
	require.NoError(t, err)

	_, err = store.Create(ctx, "long_lived", testData, time.Hour)
	require.NoError(t, err)

	// Initially should have 3 sessions
	assert.Len(t, store.sessions, 3)

	// Wait for short session to expire
	time.Sleep(100 * time.Millisecond)

	// Manual cleanup
	err = store.Cleanup(ctx)
	require.NoError(t, err)

	// Should have 2 sessions remaining
	assert.Len(t, store.sessions, 2)
	assert.Contains(t, store.sessions, "permanent")
	assert.Contains(t, store.sessions, "long_lived")
	assert.NotContains(t, store.sessions, "short_lived")
}

func TestAutomaticCleanup(t *testing.T) {
	config := &Config{
		CleanupInterval: 100 * time.Millisecond,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	// Create a session with short TTL
	_, err := store.Create(ctx, "auto_cleanup", testData, 50*time.Millisecond)
	require.NoError(t, err)

	// Initially should have 1 session
	store.mu.RLock()
	assert.Len(t, store.sessions, 1)
	store.mu.RUnlock()

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Session should be automatically cleaned up
	store.mu.RLock()
	sessionCount := len(store.sessions)
	store.mu.RUnlock()
	assert.Equal(t, 0, sessionCount)
}

func TestStats(t *testing.T) {
	config := &Config{
		CleanupInterval: 0,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)
	defer store.Close()

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	// Create some sessions
	_, err := store.Create(ctx, "session1", testData, time.Hour)
	require.NoError(t, err)
	_, err = store.Create(ctx, "session2", testData, time.Hour)
	require.NoError(t, err)

	// Delete one session
	err = store.Delete(ctx, "session1")
	require.NoError(t, err)

	statsInterface, err := store.Stats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, statsInterface)

	// Type assert to verify structure
	if stats, ok := statsInterface.(*Stats); ok {
		assert.Equal(t, "memory", stats.Store)
		assert.Equal(t, int64(1), stats.ActiveSessions)
		assert.Equal(t, int64(2), stats.TotalCreated)
		assert.Equal(t, int64(1), stats.TotalDeleted)
		assert.Contains(t, stats.Info, "active_sessions=1")
	}
}

func TestClose(t *testing.T) {
	config := &Config{
		CleanupInterval: time.Hour,
	}
	logger := zap.NewNop()
	store := NewStore(config, logger)

	ctx := context.Background()
	testData := TestData{ID: "user123"}

	// Create some sessions
	_, err := store.Create(ctx, "session1", testData, time.Hour)
	require.NoError(t, err)
	assert.Len(t, store.sessions, 1)

	// Close store
	err = store.Close()
	require.NoError(t, err)

	// Sessions should be cleared
	assert.Len(t, store.sessions, 0)

	// Cleanup timer should be stopped
	select {
	case <-store.cleanupDone:
		// Channel should be closed
	default:
		t.Error("cleanup channel should be closed")
	}
}