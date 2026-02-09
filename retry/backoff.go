package retry

import (
	"math"
	"time"
)

// Backoff implements exponential backoff with a configurable maximum number of attempts.
type Backoff struct {
	// MaxAttempts is the maximum number of retry attempts. 0 means no retries.
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// MaxDelay caps the backoff delay.
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay grows. Defaults to 2.
	Multiplier float64
}

// Exponential creates a Backoff strategy with sensible defaults.
func Exponential(maxAttempts int) *Backoff {
	return &Backoff{
		MaxAttempts:  maxAttempts,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2,
	}
}

// Next returns the delay for the given attempt number.
func (b *Backoff) Next(attempt int) (time.Duration, bool) {
	if attempt > b.MaxAttempts {
		return 0, false
	}

	multiplier := b.Multiplier
	if multiplier == 0 {
		multiplier = 2
	}

	delay := float64(b.InitialDelay) * math.Pow(multiplier, float64(attempt-1))
	d := time.Duration(delay)
	if d > b.MaxDelay {
		d = b.MaxDelay
	}

	return d, true
}
