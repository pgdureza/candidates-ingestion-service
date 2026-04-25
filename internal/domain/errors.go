package domain

type CircuitBreakerError struct {
	Err error
}

func (e *CircuitBreakerError) Error() string {
	return e.Err.Error()
}

func NewCircuitBreakerError(err error) *CircuitBreakerError {
	return &CircuitBreakerError{Err: err}
}
