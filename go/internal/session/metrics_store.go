package session

import (
	"context"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
)

// MetricsStore wraps a Store and records metrics
type MetricsStore struct {
	store     Store
	storeType string
}

// NewMetricsStore creates a new metrics-enabled store wrapper
func NewMetricsStore(store Store, storeType string) Store {
	return &MetricsStore{
		store:     store,
		storeType: storeType,
	}
}

// Create creates a new session and records metrics
func (m *MetricsStore) Create(ctx context.Context, sessionID string, data interface{}, ttl time.Duration) (string, error) {
	start := time.Now()
	id, err := m.store.Create(ctx, sessionID, data, ttl)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("create", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("create", m.storeType).Observe(duration)
	
	return id, err
}

// Get retrieves a session and records metrics
func (m *MetricsStore) Get(ctx context.Context, sessionID string, data interface{}) error {
	start := time.Now()
	err := m.store.Get(ctx, sessionID, data)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("get", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("get", m.storeType).Observe(duration)
	
	return err
}

// Update updates a session and records metrics
func (m *MetricsStore) Update(ctx context.Context, sessionID string, data interface{}) error {
	start := time.Now()
	err := m.store.Update(ctx, sessionID, data)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("update", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("update", m.storeType).Observe(duration)
	
	return err
}

// Delete deletes a session and records metrics
func (m *MetricsStore) Delete(ctx context.Context, sessionID string) error {
	start := time.Now()
	err := m.store.Delete(ctx, sessionID)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("delete", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("delete", m.storeType).Observe(duration)
	
	return err
}

// Exists checks if a session exists and records metrics
func (m *MetricsStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	start := time.Now()
	exists, err := m.store.Exists(ctx, sessionID)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("exists", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("exists", m.storeType).Observe(duration)
	
	return exists, err
}

// Refresh refreshes a session TTL and records metrics
func (m *MetricsStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
	start := time.Now()
	err := m.store.Refresh(ctx, sessionID, ttl)
	
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	
	metrics.SessionOperationsTotal.WithLabelValues("refresh", m.storeType, status).Inc()
	metrics.SessionOperationDuration.WithLabelValues("refresh", m.storeType).Observe(duration)
	
	return err
}

// Cleanup performs cleanup operations
func (m *MetricsStore) Cleanup(ctx context.Context) error {
	return m.store.Cleanup(ctx)
}

// Stats returns session statistics and updates metrics
func (m *MetricsStore) Stats(ctx context.Context) (interface{}, error) {
	stats, err := m.store.Stats(ctx)
	if err == nil && stats != nil {
		// Type assertion to get Stats
		if s, ok := stats.(*Stats); ok {
			metrics.SessionsActive.Set(float64(s.ActiveSessions))
		}
	}
	return stats, err
}

// Close closes the store
func (m *MetricsStore) Close() error {
	return m.store.Close()
}
