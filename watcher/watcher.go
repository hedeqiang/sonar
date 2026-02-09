// Package watcher provides event log monitoring implementations.
package watcher

import (
	"github.com/hedeqiang/sonar/event"
)

// Watcher monitors a blockchain for event logs.
type Watcher interface {
	// Watch begins monitoring for events. Blocks until the context is cancelled
	// or Stop is called. Returns nil on graceful stop.
	Watch() error

	// Stop gracefully shuts down the watcher.
	Stop() error

	// OnEvent registers a callback invoked for each received event log.
	OnEvent(fn func(event.Log))

	// OnError registers a callback invoked when an error occurs.
	OnError(fn func(error))
}
