package filter

import (
	"github.com/hedeqiang/sonar/event"
)

// TopicFilter matches logs whose topics contain any of the specified hashes
// at the configured position.
type TopicFilter struct {
	position int
	hashes   map[event.Hash]struct{}
}

// NewTopicFilter creates a filter that matches logs with any of the given
// hashes at the specified topic position (0-based).
func NewTopicFilter(position int, hashes ...event.Hash) *TopicFilter {
	m := make(map[event.Hash]struct{}, len(hashes))
	for _, h := range hashes {
		m[h] = struct{}{}
	}
	return &TopicFilter{position: position, hashes: m}
}

// Match reports whether the log has a matching topic at the configured position.
func (f *TopicFilter) Match(log event.Log) bool {
	if f.position >= len(log.Topics) {
		return false
	}
	_, ok := f.hashes[log.Topics[f.position]]
	return ok
}
