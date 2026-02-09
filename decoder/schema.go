package decoder

import (
	"sync"

	"github.com/hedeqiang/sonar/event"
)

// Schema maps event signature hashes to their parsed definitions.
type Schema struct {
	mu     sync.RWMutex
	events map[event.Hash]*EventDef
}

// EventDef describes a parsed event definition.
type EventDef struct {
	Name      string
	Signature string
	SigHash   event.Hash
	Inputs    []ParamDef
}

// ParamDef describes a single event parameter.
type ParamDef struct {
	Name    string
	Type    string
	Indexed bool
}

// NewSchema creates an empty event schema registry.
func NewSchema() *Schema {
	return &Schema{
		events: make(map[event.Hash]*EventDef),
	}
}

// Add registers an event definition.
func (s *Schema) Add(def *EventDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[def.SigHash] = def
}

// Lookup finds the event definition for the given topic0 hash.
func (s *Schema) Lookup(sigHash event.Hash) (*EventDef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	def, ok := s.events[sigHash]
	return def, ok
}

// Has reports whether the schema contains a definition for the given hash.
func (s *Schema) Has(sigHash event.Hash) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.events[sigHash]
	return ok
}
