package decoder

import (
	"github.com/hedeqiang/sonar/event"
)

// Raw is a pass-through decoder that wraps the log without attempting ABI decoding.
// Useful when you want to receive raw logs without transformation.
type Raw struct{}

// NewRaw creates a new raw pass-through decoder.
func NewRaw() *Raw {
	return &Raw{}
}

// Decode wraps the log in a DecodedEvent with no parameter parsing.
func (r *Raw) Decode(log event.Log) (*DecodedEvent, error) {
	return &DecodedEvent{
		Name:   "raw",
		Params: nil,
		Raw:    log,
	}, nil
}

// Register is a no-op for the raw decoder.
func (r *Raw) Register(_ string) error {
	return nil
}
