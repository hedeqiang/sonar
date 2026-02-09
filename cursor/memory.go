package cursor

import "sync"

// Memory is an in-memory Cursor implementation.
// Suitable for development and testing; data is lost on restart.
type Memory struct {
	mu     sync.RWMutex
	blocks map[string]uint64
}

// NewMemory creates a new in-memory cursor.
func NewMemory() *Memory {
	return &Memory{
		blocks: make(map[string]uint64),
	}
}

// Load returns the last saved block number for the chain, or 0 if not found.
func (m *Memory) Load(chainID string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blocks[chainID], nil
}

// Save stores the block number for the chain.
func (m *Memory) Save(chainID string, block uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[chainID] = block
	return nil
}
