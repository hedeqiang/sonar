// Package filter provides event log filtering capabilities.
package filter

import (
	"github.com/hedeqiang/sonar/event"
)

// Filter determines whether a log matches a given criteria.
type Filter interface {
	Match(log event.Log) bool
}

// Query describes the parameters for fetching or subscribing to event logs.
type Query struct {
	Addresses []event.Address
	Topics    [][]event.Hash
	FromBlock *uint64
	ToBlock   *uint64
}

// QueryOption configures a Query.
type QueryOption func(*Query)

// NewQuery creates a Query with the given options applied.
func NewQuery(opts ...QueryOption) Query {
	var q Query
	for _, opt := range opts {
		opt(&q)
	}
	return q
}

// WithAddresses adds contract addresses to filter on.
func WithAddresses(addrs ...event.Address) QueryOption {
	return func(q *Query) {
		q.Addresses = append(q.Addresses, addrs...)
	}
}

// WithTopics sets the topic filters.
// Each element in the outer slice corresponds to a topic position.
// Multiple hashes within an inner slice are OR-matched.
func WithTopics(topics ...[]event.Hash) QueryOption {
	return func(q *Query) {
		q.Topics = topics
	}
}

// WithFromBlock sets the starting block number for the query.
func WithFromBlock(block uint64) QueryOption {
	return func(q *Query) {
		q.FromBlock = &block
	}
}

// WithToBlock sets the ending block number for the query.
func WithToBlock(block uint64) QueryOption {
	return func(q *Query) {
		q.ToBlock = &block
	}
}

// WithBlockRange sets both the starting and ending block numbers.
func WithBlockRange(from, to uint64) QueryOption {
	return func(q *Query) {
		q.FromBlock = &from
		q.ToBlock = &to
	}
}
