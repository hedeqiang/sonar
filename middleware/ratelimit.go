package middleware

import (
	"sync"
	"time"

	"github.com/hedeqiang/sonar/event"
)

// RateLimit limits the rate at which events are processed.
type RateLimit struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// NewRateLimit creates a rate-limiting middleware that allows at most one event
// per the given interval.
func NewRateLimit(interval time.Duration) *RateLimit {
	return &RateLimit{
		interval: interval,
	}
}

// Wrap decorates the handler with rate limiting.
func (r *RateLimit) Wrap(next Handler) Handler {
	return func(lg event.Log) *event.Log {
		r.mu.Lock()
		if time.Since(r.last) < r.interval {
			r.mu.Unlock()
			return nil // drop: rate limited
		}
		r.last = time.Now()
		r.mu.Unlock()

		return next(lg)
	}
}
