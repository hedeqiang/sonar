// Package decoder provides event log decoding capabilities.
package decoder

import (
	"github.com/hedeqiang/sonar/event"
)

// Decoder decodes raw event logs into structured data.
type Decoder interface {
	// Decode parses a raw log into a DecodedEvent.
	// Returns ErrDecode if the log cannot be matched or parsed.
	Decode(log event.Log) (*DecodedEvent, error)

	// Register adds an event ABI signature to the decoder.
	// The signature should be in Solidity format, e.g. "Transfer(address,address,uint256)".
	Register(eventSignature string) error
}

// DecodedEvent contains the decoded representation of an event log.
type DecodedEvent struct {
	// Name is the event name (e.g. "Transfer").
	Name string

	// Signature is the full event signature (e.g. "Transfer(address,address,uint256)").
	Signature string

	// Params holds the decoded parameter values keyed by parameter name.
	// For anonymous parameters, numeric indices are used as keys.
	Params map[string]interface{}

	// Indexed holds the decoded indexed (topic) parameters.
	Indexed map[string]interface{}

	// Raw is the original unmodified event log.
	Raw event.Log
}
