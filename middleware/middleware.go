// Package middleware provides interceptors for event log processing pipelines.
package middleware

import (
	"github.com/hedeqiang/sonar/event"
)

// Handler processes an event log and returns a (possibly modified) log.
// Returning a nil pointer signals that the log should be dropped.
type Handler func(log event.Log) *event.Log

// Middleware wraps a Handler, adding cross-cutting behavior (logging, metrics, etc.).
type Middleware interface {
	// Wrap returns a new Handler that decorates the given inner handler.
	Wrap(next Handler) Handler
}

// Chain composes multiple middlewares into a single Handler, applying them
// in the order provided (first middleware is outermost).
func Chain(handler Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i].Wrap(handler)
	}
	return handler
}
