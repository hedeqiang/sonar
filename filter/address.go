package filter

import (
	"github.com/hedeqiang/sonar/event"
)

// AddressFilter matches logs emitted by any of the specified contract addresses.
type AddressFilter struct {
	addresses map[event.Address]struct{}
}

// NewAddressFilter creates a filter that matches the given addresses.
func NewAddressFilter(addrs ...event.Address) *AddressFilter {
	m := make(map[event.Address]struct{}, len(addrs))
	for _, a := range addrs {
		m[a] = struct{}{}
	}
	return &AddressFilter{addresses: m}
}

// Match reports whether the log's address is in the filter set.
func (f *AddressFilter) Match(log event.Log) bool {
	_, ok := f.addresses[log.Address]
	return ok
}
