package retry

import (
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// Closed means the circuit is healthy; requests flow normally.
	Closed State = iota
	// Open means too many failures have occurred; requests are rejected.
	Open
	// HalfOpen means the circuit is testing whether the downstream has recovered.
	HalfOpen
)

// CircuitBreaker prevents cascading failures by tracking error rates and
// temporarily halting requests when failures exceed a threshold.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        State
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

// NewCircuitBreaker creates a circuit breaker that opens after threshold
// consecutive failures and resets after resetTimeout.
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

// Allow reports whether a request is permitted.
// In Closed state, it always allows. In Open state, it checks if the reset
// timeout has elapsed (transitioning to HalfOpen). In HalfOpen state, it allows
// one probe request.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = HalfOpen
			return true
		}
		return false
	case HalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation, resetting the breaker to Closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = Closed
}

// RecordFailure records a failed operation. If failures reach the threshold,
// the breaker transitions to Open.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = Open
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) CurrentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
