// Package chain provides the multi-chain abstraction layer.
package chain

import (
	"context"

	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
)

// Chain is the core abstraction for interacting with a blockchain.
// Each supported chain (Ethereum, BSC, Polygon, etc.) must implement this interface.
type Chain interface {
	// ID returns the unique chain identifier (e.g. "ethereum", "bsc", "polygon").
	ID() string

	// LatestBlock returns the most recent block number.
	LatestBlock(ctx context.Context) (uint64, error)

	// FetchLogs retrieves historical event logs matching the given query.
	FetchLogs(ctx context.Context, query filter.Query) ([]event.Log, error)

	// Subscribe establishes a real-time subscription for new event logs.
	// Not all chains/transports support subscriptions; returns an error if unsupported.
	Subscribe(ctx context.Context, query filter.Query) (Subscription, error)
}

// Subscription represents an active real-time event subscription.
type Subscription interface {
	// Logs returns a channel that receives incoming event logs.
	Logs() <-chan event.Log

	// Err returns a channel that receives subscription errors.
	// The channel is closed when the subscription ends.
	Err() <-chan error

	// Unsubscribe terminates the subscription and closes all channels.
	Unsubscribe()
}
