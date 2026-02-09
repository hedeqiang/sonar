package filter

import (
	"github.com/hedeqiang/sonar/event"
)

// CompositeMode determines how child filters are combined.
type CompositeMode int

const (
	// And requires all child filters to match.
	And CompositeMode = iota
	// Or requires at least one child filter to match.
	Or
)

// CompositeFilter combines multiple filters using AND or OR logic.
type CompositeFilter struct {
	mode    CompositeMode
	filters []Filter
}

// NewCompositeFilter creates a composite filter with the given mode and children.
func NewCompositeFilter(mode CompositeMode, filters ...Filter) *CompositeFilter {
	return &CompositeFilter{mode: mode, filters: filters}
}

// AllOf is a convenience constructor for AND composition.
func AllOf(filters ...Filter) *CompositeFilter {
	return NewCompositeFilter(And, filters...)
}

// AnyOf is a convenience constructor for OR composition.
func AnyOf(filters ...Filter) *CompositeFilter {
	return NewCompositeFilter(Or, filters...)
}

// Match applies the composite logic to the log.
func (f *CompositeFilter) Match(log event.Log) bool {
	if len(f.filters) == 0 {
		return true
	}

	switch f.mode {
	case And:
		for _, child := range f.filters {
			if !child.Match(log) {
				return false
			}
		}
		return true
	case Or:
		for _, child := range f.filters {
			if child.Match(log) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
