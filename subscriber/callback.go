package subscriber

import (
	"github.com/hedeqiang/sonar/event"
)

// CallbackFunc is the function signature for event callbacks.
type CallbackFunc func(event.Log)

// Callback delivers event logs by invoking a callback function.
type Callback struct {
	fn   CallbackFunc
	done chan struct{}
}

// NewCallback creates a callback-based subscriber.
func NewCallback(fn CallbackFunc) *Callback {
	return &Callback{
		fn:   fn,
		done: make(chan struct{}),
	}
}

// Send invokes the callback with the log. No-op if closed.
func (c *Callback) Send(log event.Log) {
	select {
	case <-c.done:
		return
	default:
	}
	c.fn(log)
}

// Close stops the subscriber.
func (c *Callback) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}
