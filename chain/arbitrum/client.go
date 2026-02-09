// Package arbitrum provides an Arbitrum implementation of chain.Chain.
// Arbitrum is EVM-compatible and reuses the Ethereum client with a different chain ID.
package arbitrum

import (
	"github.com/hedeqiang/sonar/chain/ethereum"
)

// New creates an Arbitrum chain client.
func New(rpcURL string) *ethereum.Client {
	return ethereum.NewWithID("arbitrum", rpcURL)
}
