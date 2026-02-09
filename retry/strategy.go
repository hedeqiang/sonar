// Package retry provides retry strategies and a circuit breaker for resilient operations.
package retry

import (
	"context"
	"time"
)

// Strategy defines a retry policy.
type Strategy interface {
	// Next returns the delay before the next retry attempt.
	// Returns false if no more retries should be attempted.
	Next(attempt int) (delay time.Duration, ok bool)
}

// Do executes fn, retrying according to the given strategy on non-nil errors.
// It respects context cancellation.
func Do(ctx context.Context, s Strategy, fn func(ctx context.Context) error) error {
	var attempt int
	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		attempt++
		delay, ok := s.Next(attempt)
		if !ok {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}
