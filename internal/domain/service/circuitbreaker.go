package service

type CircuitBreaker interface {
	Execute(fn func() error) error
	State() string
}
