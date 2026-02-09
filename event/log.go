// Package event defines the core data structures for blockchain event logs.
package event

import (
	"math/big"
	"time"
)

// Hash represents a 32-byte hash.
type Hash [32]byte

// Address represents a 20-byte Ethereum-compatible address.
type Address [20]byte

// Log represents a single event log emitted by a smart contract.
type Log struct {
	// Chain identifies which blockchain this log came from.
	Chain string

	// Address is the contract address that emitted the event.
	Address Address

	// Topics contains the indexed event parameters.
	// Topics[0] is typically the event signature hash.
	Topics []Hash

	// Data holds the non-indexed event parameters (ABI-encoded).
	Data []byte

	// BlockNumber is the block in which this log was emitted.
	BlockNumber uint64

	// BlockHash is the hash of the block containing this log.
	BlockHash Hash

	// TxHash is the transaction hash that produced this log.
	TxHash Hash

	// TxIndex is the transaction's position in the block.
	TxIndex uint

	// LogIndex is the log's position in the block.
	LogIndex uint

	// Removed indicates whether this log was reverted due to a chain reorganization.
	Removed bool

	// Timestamp is the block timestamp (if available).
	Timestamp time.Time
}

// EventSignature returns the first topic (event signature hash), or a zero hash if no topics exist.
func (l Log) EventSignature() Hash {
	if len(l.Topics) > 0 {
		return l.Topics[0]
	}
	return Hash{}
}

// Metadata holds auxiliary information about an event log's origin.
type Metadata struct {
	ChainID   string
	Network   string
	BlockTime time.Time
	GasUsed   uint64
	GasPrice  *big.Int
}
