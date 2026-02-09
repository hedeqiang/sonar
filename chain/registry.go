package chain

import (
	"fmt"
	"sync"
)

// Registry holds registered Chain implementations for lookup by ID.
type Registry struct {
	mu     sync.RWMutex
	chains map[string]Chain
}

// NewRegistry creates an empty chain registry.
func NewRegistry() *Registry {
	return &Registry{
		chains: make(map[string]Chain),
	}
}

// Register adds a chain to the registry. Returns an error if a chain with
// the same ID is already registered.
func (r *Registry) Register(c Chain) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := c.ID()
	if _, exists := r.chains[id]; exists {
		return fmt.Errorf("chain: %q already registered", id)
	}
	r.chains[id] = c
	return nil
}

// Get returns the chain with the given ID, or nil if not found.
func (r *Registry) Get(id string) (Chain, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.chains[id]
	return c, ok
}

// All returns all registered chains.
func (r *Registry) All() []Chain {
	r.mu.RLock()
	defer r.mu.RUnlock()
	chains := make([]Chain, 0, len(r.chains))
	for _, c := range r.chains {
		chains = append(chains, c)
	}
	return chains
}

// IDs returns the IDs of all registered chains.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.chains))
	for id := range r.chains {
		ids = append(ids, id)
	}
	return ids
}
