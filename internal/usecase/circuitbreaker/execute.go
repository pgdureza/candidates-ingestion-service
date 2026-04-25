package circuitbreaker

import (
	"fmt"
	"time"
)

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 1. Check if should transition from open to half-open
	if cb.state == stateOpen {
		if time.Since(cb.lastFailTime) > cb.openTimeout {
			if cb.logger != nil {
				cb.logger.Info("Timeout reached; transitioning from OPEN to HALF-OPEN")
			}
			cb.state = stateHalfOpen
			cb.successCount = 0
		} else {
			remaining := cb.openTimeout - time.Since(cb.lastFailTime)
			if cb.logger != nil {
				cb.logger.Warn("Execution rejected: Circuit is OPEN", "retry_after", remaining.String())
			}
			return fmt.Errorf("[CircuitBreaker] state is open, retry in %v", remaining)
		}
	}

	// 2. Execute the function
	err := fn()

	if err != nil {
		cb.failureCount++
		cb.lastFailTime = time.Now()

		if cb.logger != nil {
			cb.logger.Debug("Function execution failed", "error", err, "failure_count", cb.failureCount)
		}

		// Failure in Half-Open means immediate back to Open
		if cb.state == stateHalfOpen {
			cb.state = stateOpen
			if cb.logger != nil {
				cb.logger.Error("Half-Open probe failed; reverting to OPEN state")
			}
			return fmt.Errorf("[CircuitBreaker] failed in half-open: %w", err)
		}

		// Check if threshold reached to trip the breaker
		if cb.failureCount >= cb.failureThreshold {
			cb.state = stateOpen
			if cb.logger != nil {
				cb.logger.Error("Failure threshold reached; tripping circuit to OPEN",
					"threshold", cb.failureThreshold,
					"total_failures", cb.failureCount)
			}
			return fmt.Errorf("[CircuitBreaker] state changed to open: %w", err)
		}

		return err
	}

	// 3. Success Handling
	if cb.state == stateHalfOpen {
		cb.successCount++
		if cb.logger != nil {
			cb.logger.Info("Half-Open success recorded", "current_successes", cb.successCount, "required", 3)
		}

		if cb.successCount >= 3 {
			cb.state = stateClosed
			cb.failureCount = 0
			if cb.logger != nil {
				cb.logger.Info("Circuit fully recovered; transitioning to CLOSED")
			}
		}
	} else if cb.state == stateClosed {
		// Reset failure count on a successful call in closed state
		if cb.failureCount > 0 {
			cb.failureCount = 0
		}
	}

	return nil
}
