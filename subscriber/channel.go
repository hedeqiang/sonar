package subscriber

import (
	"github.com/hedeqiang/sonar/event"
)

// Channel delivers event logs through a Go channel.
type Channel struct {
	ch     chan event.Log
	done   chan struct{}
}

// NewChannel creates a channel-based subscriber with the given buffer size.
func NewChannel(bufSize int) *Channel {
	if bufSize <= 0 {
		bufSize = 128
	}
	return &Channel{
		ch:   make(chan event.Log, bufSize),
		done: make(chan struct{}),
	}
}

// Logs returns the channel to read events from.
func (c *Channel) Logs() <-chan event.Log {
	return c.ch
}

// Send delivers a log to the channel. Drops the log if the channel is full.
func (c *Channel) Send(log event.Log) {
	select {
	case c.ch <- log:
	case <-c.done:
	default:
		// drop: subscriber is not keeping up
	}
}

// Close shuts down the subscriber.
func (c *Channel) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}
