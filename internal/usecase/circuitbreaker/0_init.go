package circuitbreaker

import (
	"sync"
	"time"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.CircuitBreaker = new(CircuitBreaker)

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            string // closed, open, half-open
	failureCount     int
	failureThreshold int
	openTimeout      time.Duration
	halfOpenTimeout  time.Duration
	lastFailTime     time.Time
	successCount     int // for half-open state
}

const (
	stateClosed   = "closed"
	stateOpen     = "open"
	stateHalfOpen = "half-open"
)

func NewCircuitBreaker(
	failureThreshold int,
	openTimeout time.Duration,
	halfOpenTimeout time.Duration,
) *CircuitBreaker {
	return &CircuitBreaker{
		state:            stateClosed,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
		halfOpenTimeout:  halfOpenTimeout,
	}
}
