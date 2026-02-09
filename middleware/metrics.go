package middleware

import (
	"sync/atomic"

	"github.com/hedeqiang/sonar/event"
)

// Metrics collects basic counters for processed events.
type Metrics struct {
	processed atomic.Uint64
	dropped   atomic.Uint64
}

// NewMetrics creates a metrics collection middleware.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Wrap decorates the handler with metrics collection.
func (m *Metrics) Wrap(next Handler) Handler {
	return func(lg event.Log) *event.Log {
		result := next(lg)
		if result != nil {
			m.processed.Add(1)
		} else {
			m.dropped.Add(1)
		}
		return result
	}
}

// Processed returns the number of successfully processed events.
func (m *Metrics) Processed() uint64 {
	return m.processed.Load()
}

// Dropped returns the number of dropped events.
func (m *Metrics) Dropped() uint64 {
	return m.dropped.Load()
}
