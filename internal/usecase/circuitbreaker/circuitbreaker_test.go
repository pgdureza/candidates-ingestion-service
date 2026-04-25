package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond, 50*time.Millisecond)

	// Success should keep it closed
	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error { return nil })
		assert.NoError(t, err)
		assert.Equal(t, stateClosed, cb.State())
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond, 50*time.Millisecond)

	// First failure
	err := cb.Execute(func() error { return errors.New("fail 1") })
	assert.Error(t, err)
	assert.Equal(t, stateClosed, cb.State())

	// Second failure - should open
	err = cb.Execute(func() error { return errors.New("fail 2") })
	assert.Error(t, err)
	assert.Equal(t, stateOpen, cb.State())
}

func TestCircuitBreakerFastFailWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond, 50*time.Millisecond)

	// Fail once to open
	_ = cb.Execute(func() error { return errors.New("fail") })
	assert.Equal(t, stateOpen, cb.State())

	// Next call should fast-fail
	err := cb.Execute(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state is open")
}

func TestCircuitBreakerHalfOpenTransition(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond, 50*time.Millisecond)

	// Open the circuit
	_ = cb.Execute(func() error { return errors.New("fail") })
	assert.Equal(t, stateOpen, cb.State())

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Next execution should transition to half-open
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, stateHalfOpen, cb.State())
}

func TestCircuitBreakerClosesAfterSuccesses(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond, 50*time.Millisecond)

	// Open circuit
	_ = cb.Execute(func() error { return errors.New("fail") })
	assert.Equal(t, stateOpen, cb.State())

	// Wait and transition to half-open
	time.Sleep(100 * time.Millisecond)
	_ = cb.Execute(func() error { return nil })
	assert.Equal(t, stateHalfOpen, cb.State())

	// 3 successes should close
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error { return nil })
	}
	assert.Equal(t, stateClosed, cb.State())
}
