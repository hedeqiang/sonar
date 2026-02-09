package filter

import (
	"github.com/hedeqiang/sonar/event"
)

// BlockRangeFilter matches logs within a specified block number range (inclusive).
type BlockRangeFilter struct {
	from *uint64
	to   *uint64
}

// NewBlockRangeFilter creates a filter matching logs within [from, to].
// A nil value means unbounded on that side.
func NewBlockRangeFilter(from, to *uint64) *BlockRangeFilter {
	return &BlockRangeFilter{from: from, to: to}
}

// Match reports whether the log's block number falls within the range.
func (f *BlockRangeFilter) Match(log event.Log) bool {
	if f.from != nil && log.BlockNumber < *f.from {
		return false
	}
	if f.to != nil && log.BlockNumber > *f.to {
		return false
	}
	return true
}
