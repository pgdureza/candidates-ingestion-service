package circuitbreaker

import (
	"fmt"
	"time"
)

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if should transition from open to half-open
	if cb.state == stateOpen {
		if time.Since(cb.lastFailTime) > cb.openTimeout {
			cb.state = stateHalfOpen
			cb.successCount = 0
		} else {
			return fmt.Errorf("circuit breaker open")
		}
	}

	// Execute
	err := fn()

	if err != nil {
		cb.failureCount++
		cb.lastFailTime = time.Now()

		if cb.state == stateHalfOpen {
			cb.state = stateOpen
			cb.failureCount = 0
			return fmt.Errorf("circuit breaker reopened after failure: %w", err)
		}

		if cb.failureCount >= cb.failureThreshold {
			cb.state = stateOpen
			return fmt.Errorf("circuit breaker opened: %w", err)
		}

		return err
	}

	// Success
	if cb.state == stateHalfOpen {
		cb.successCount++
		if cb.successCount >= 3 { // 3 successes to close
			cb.state = stateClosed
			cb.failureCount = 0
		}
	} else if cb.state == stateClosed {
		cb.failureCount = 0
	}

	return nil
}
