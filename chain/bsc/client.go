// Package bsc provides a BSC (BNB Smart Chain) implementation of chain.Chain.
// BSC is EVM-compatible and reuses the Ethereum client with a different chain ID.
package bsc

import (
	"github.com/hedeqiang/sonar/chain/ethereum"
)

// New creates a BSC chain client.
func New(rpcURL string) *ethereum.Client {
	return ethereum.NewWithID("bsc", rpcURL)
}
