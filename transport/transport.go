// Package transport provides RPC transport layer abstractions.
package transport

import (
	"context"
)

// Transport sends JSON-RPC requests and returns raw responses.
type Transport interface {
	// Call sends a JSON-RPC request and returns the result bytes.
	Call(ctx context.Context, method string, params ...interface{}) ([]byte, error)

	// Subscribe establishes a streaming subscription (WebSocket only).
	// Returns a channel of raw messages and an unsubscribe function.
	Subscribe(ctx context.Context, method string, params ...interface{}) (<-chan []byte, func(), error)

	// Close terminates the transport connection.
	Close() error
}
