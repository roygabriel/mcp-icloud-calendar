package caldav

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StartsInClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerOptions())
	if cb.State() != StateClosed {
		t.Errorf("initial state = %d, want StateClosed", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Errorf("unexpected error in closed state: %v", err)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 3, ResetTimeout: 10 * time.Second})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Errorf("state = %d, want StateOpen after %d failures", cb.State(), 3)
	}
	if err := cb.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 2, ResetTimeout: 5 * time.Second})
	cb.now = func() time.Time { return now }

	// Trip the breaker.
	cb.RecordFailure()
	cb.RecordFailure()

	// Advance time past reset timeout.
	cb.now = func() time.Time { return now.Add(6 * time.Second) }

	err := cb.Allow()
	if err != nil {
		t.Errorf("expected nil error for half-open probe, got %v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("state = %d, want StateHalfOpen", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenRejectsSecondRequest(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 1, ResetTimeout: 5 * time.Second})
	cb.now = func() time.Time { return now }

	cb.RecordFailure()

	// Advance past reset timeout → half-open.
	cb.now = func() time.Time { return now.Add(6 * time.Second) }
	_ = cb.Allow() // first probe allowed

	// Second request in half-open should be rejected.
	err := cb.Allow()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen for second request in half-open, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenSuccessCloses(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 1, ResetTimeout: 5 * time.Second})
	cb.now = func() time.Time { return now }

	cb.RecordFailure()

	// Advance past reset timeout → half-open.
	cb.now = func() time.Time { return now.Add(6 * time.Second) }
	_ = cb.Allow()

	// Success in half-open closes the breaker.
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("state = %d, want StateClosed after successful probe", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 1, ResetTimeout: 5 * time.Second})
	cb.now = func() time.Time { return now }

	cb.RecordFailure()

	// Advance past reset timeout → half-open.
	cb.now = func() time.Time { return now.Add(6 * time.Second) }
	_ = cb.Allow()

	// Failure in half-open re-opens.
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("state = %d, want StateOpen after failed probe", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 3, ResetTimeout: 10 * time.Second})

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // reset
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateClosed {
		t.Errorf("state = %d, want StateClosed (failures should have been reset)", cb.State())
	}
}

func TestCircuitBreaker_DefaultOptions(t *testing.T) {
	opts := DefaultCircuitBreakerOptions()
	if opts.Threshold != 5 {
		t.Errorf("Threshold = %d, want 5", opts.Threshold)
	}
	if opts.ResetTimeout != 30*time.Second {
		t.Errorf("ResetTimeout = %v, want 30s", opts.ResetTimeout)
	}
}

func TestCircuitBreaker_DefaultsForZeroValues(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOptions{})
	if cb.threshold != 5 {
		t.Errorf("threshold = %d, want 5 (default)", cb.threshold)
	}
	if cb.resetTimeout != 30*time.Second {
		t.Errorf("resetTimeout = %v, want 30s (default)", cb.resetTimeout)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOptions{Threshold: 100, ResetTimeout: 1 * time.Second})
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Allow()
			cb.RecordFailure()
			cb.RecordSuccess()
			_ = cb.State()
		}()
	}

	wg.Wait()
}

func TestIsServerError_Nil(t *testing.T) {
	if isServerError(nil) {
		t.Error("nil error should not be a server error")
	}
}

func TestIsServerError_NetError(t *testing.T) {
	err := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	if !isServerError(err) {
		t.Error("net.Error should be a server error")
	}
}

func TestIsServerError_HTTP5xx(t *testing.T) {
	err := &HTTPStatusError{StatusCode: 500}
	if !isServerError(err) {
		t.Error("HTTP 500 should be a server error")
	}
	err = &HTTPStatusError{StatusCode: 503}
	if !isServerError(err) {
		t.Error("HTTP 503 should be a server error")
	}
}

func TestIsServerError_HTTP4xx(t *testing.T) {
	err := &HTTPStatusError{StatusCode: 404}
	if isServerError(err) {
		t.Error("HTTP 404 should NOT be a server error")
	}
	err = &HTTPStatusError{StatusCode: 400}
	if isServerError(err) {
		t.Error("HTTP 400 should NOT be a server error")
	}
}

func TestIsServerError_UnknownError(t *testing.T) {
	err := errors.New("some unknown error")
	if !isServerError(err) {
		t.Error("unknown errors should be treated as server errors")
	}
}

func TestHTTPStatusError_Error(t *testing.T) {
	err := &HTTPStatusError{StatusCode: 503}
	if err.Error() != "Service Unavailable" {
		t.Errorf("Error() = %q, want Service Unavailable", err.Error())
	}
}
