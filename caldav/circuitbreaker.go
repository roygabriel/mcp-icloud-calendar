package caldav

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	// StateClosed allows requests through and tracks failures.
	StateClosed CircuitState = iota
	// StateOpen rejects requests immediately.
	StateOpen
	// StateHalfOpen allows a single probe request.
	StateHalfOpen
)

// ErrCircuitOpen is returned when the circuit breaker is open and rejecting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerOptions configures the circuit breaker behavior.
type CircuitBreakerOptions struct {
	Threshold    int
	ResetTimeout time.Duration
}

// DefaultCircuitBreakerOptions returns sensible defaults for the circuit breaker.
func DefaultCircuitBreakerOptions() CircuitBreakerOptions {
	return CircuitBreakerOptions{
		Threshold:    5,
		ResetTimeout: 30 * time.Second,
	}
}

// CircuitBreaker implements a three-state circuit breaker pattern.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
	now          func() time.Time // injectable for testing
}

// NewCircuitBreaker creates a new CircuitBreaker with the given options.
func NewCircuitBreaker(opts CircuitBreakerOptions) *CircuitBreaker {
	if opts.Threshold <= 0 {
		opts.Threshold = 5
	}
	if opts.ResetTimeout <= 0 {
		opts.ResetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		state:        StateClosed,
		threshold:    opts.Threshold,
		resetTimeout: opts.ResetTimeout,
		now:          time.Now,
	}
}

// Allow checks whether a request is allowed through the circuit breaker.
// Returns nil if allowed, ErrCircuitOpen if the breaker is open.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if cb.now().Sub(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		// Only one probe at a time; reject additional requests while probing.
		return ErrCircuitOpen
	default:
		return nil
	}
}

// RecordSuccess records a successful request, resetting the breaker to closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure records a failed request and opens the breaker if the threshold is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = cb.now()
	if cb.failures >= cb.threshold {
		cb.state = StateOpen
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// isServerError returns true if the error should count as a circuit breaker failure.
// Connection errors and HTTP 5xx responses count; 4xx responses do not.
func isServerError(err error) bool {
	if err == nil {
		return false
	}

	// Connection errors always count.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// HTTP status-based errors: only 5xx counts.
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}

	// Default: count unknown errors as server errors.
	return true
}

// HTTPStatusError wraps an HTTP status code as an error for circuit breaker classification.
type HTTPStatusError struct {
	StatusCode int
}

// Error returns a string representation of the HTTP status error.
func (e *HTTPStatusError) Error() string {
	return http.StatusText(e.StatusCode)
}
