// Package cursor provides progress tracking for event log scanning.
package cursor

// Cursor tracks the last processed block for each chain,
// allowing resumable event scanning.
type Cursor interface {
	// Load returns the last saved block number for the given chain ID.
	// Returns 0 if no progress has been saved.
	Load(chainID string) (uint64, error)

	// Save persists the current block number for the given chain ID.
	Save(chainID string, block uint64) error
}
