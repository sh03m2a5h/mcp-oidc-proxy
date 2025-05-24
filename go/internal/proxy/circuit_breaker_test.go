package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestCircuitBreaker_Allow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cb := NewCircuitBreaker(3, 100*time.Millisecond, logger)

	// Initially should be closed and allow requests
	assert.Equal(t, StateClosed, cb.State())
	assert.True(t, cb.Allow())

	// Record failures up to threshold
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	assert.True(t, cb.Allow())

	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	assert.True(t, cb.Allow())

	// Third failure should open the circuit
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.False(t, cb.Allow())

	// Should stay open until timeout
	assert.False(t, cb.Allow())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cb := NewCircuitBreaker(2, 100*time.Millisecond, logger)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout and allow one request
	time.Sleep(150 * time.Millisecond)
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())

	// Record success should close the circuit
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.Failures())
}

func TestCircuitBreaker_RecordFailureInHalfOpen(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cb := NewCircuitBreaker(2, 100*time.Millisecond, logger)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout and allow one request
	time.Sleep(150 * time.Millisecond)
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())

	// Record failure should re-open the circuit
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.False(t, cb.Allow())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cb := NewCircuitBreaker(2, 100*time.Millisecond, logger)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
	assert.False(t, cb.Allow())

	// Reset should close the circuit
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.Failures())
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cb := NewCircuitBreaker(3, 100*time.Millisecond, logger)

	// Record some failures (but not enough to open)
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 2, cb.Failures())

	// Success should reset failure count
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.Failures())
}