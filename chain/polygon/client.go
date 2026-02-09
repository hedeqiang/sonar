// Package polygon provides a Polygon (formerly Matic) implementation of chain.Chain.
// Polygon is EVM-compatible and reuses the Ethereum client with a different chain ID.
package polygon

import (
	"github.com/hedeqiang/sonar/chain/ethereum"
)

// New creates a Polygon chain client.
func New(rpcURL string) *ethereum.Client {
	return ethereum.NewWithID("polygon", rpcURL)
}
