// Package decoder provides event log decoding capabilities.
package decoder

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"

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

	// Params holds all decoded parameter values keyed by parameter name (indexed + non-indexed).
	Params map[string]interface{}

	// Indexed holds only the decoded indexed (topic) parameters.
	Indexed map[string]interface{}

	// Data holds only the decoded non-indexed (data) parameters.
	Data map[string]interface{}

	// Raw is the original unmodified event log.
	Raw event.Log
}

// String returns a human-readable representation of the decoded event.
func (e *DecodedEvent) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s(", e.Name)

	first := true
	for k, v := range e.Params {
		if !first {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s=%s", k, formatValue(v))
		first = false
	}
	b.WriteString(")")

	fmt.Fprintf(&b, " chain=%s block=%d tx=%s",
		e.Raw.Chain, e.Raw.BlockNumber, e.Raw.TxHash.Hex())

	return b.String()
}

// JSON returns the decoded event as a JSON-serializable map.
// Address and Hash types are hex-encoded, *big.Int becomes a decimal string,
// and []byte becomes a "0x"-prefixed hex string.
func (e *DecodedEvent) JSON() map[string]interface{} {
	m := map[string]interface{}{
		"event":       e.Name,
		"signature":   e.Signature,
		"chain":       e.Raw.Chain,
		"blockNumber": e.Raw.BlockNumber,
		"txHash":      e.Raw.TxHash.Hex(),
		"logIndex":    e.Raw.LogIndex,
		"address":     e.Raw.Address.Hex(),
		"removed":     e.Raw.Removed,
	}

	params := make(map[string]interface{}, len(e.Params))
	for k, v := range e.Params {
		params[k] = jsonValue(v)
	}
	m["params"] = params

	indexed := make(map[string]interface{}, len(e.Indexed))
	for k, v := range e.Indexed {
		indexed[k] = jsonValue(v)
	}
	m["indexed"] = indexed

	data := make(map[string]interface{}, len(e.Data))
	for k, v := range e.Data {
		data[k] = jsonValue(v)
	}
	m["data"] = data

	return m
}

// MarshalJSON implements json.Marshaler.
func (e *DecodedEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.JSON())
}

// Bind decodes the event parameters into a user-defined struct.
// Fields are matched by the "abi" struct tag, or by case-insensitive field name.
// Supported field types: event.Address, event.Hash, *big.Int, bool, string,
// uint8–uint64, int8–int64, []byte.
//
// Example:
//
//	type TransferEvent struct {
//	    From  event.Address `abi:"from"`
//	    To    event.Address `abi:"to"`
//	    Value *big.Int      `abi:"value"`
//	}
//
//	var evt TransferEvent
//	decoded.Bind(&evt)
func (e *DecodedEvent) Bind(out interface{}) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("decoder: Bind requires a non-nil pointer to struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("decoder: Bind requires a pointer to struct, got %s", rv.Kind())
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		// Determine the ABI parameter name
		paramName := field.Tag.Get("abi")
		if paramName == "-" {
			continue
		}
		if paramName == "" {
			paramName = field.Name
		}

		// Look up value in Params (case-insensitive fallback)
		val, ok := e.Params[paramName]
		if !ok {
			val, ok = findParamInsensitive(e.Params, paramName)
		}
		if !ok {
			continue // no matching param, skip
		}

		fv := rv.Field(i)
		if err := assignValue(fv, val); err != nil {
			return fmt.Errorf("decoder: field %s: %w", field.Name, err)
		}
	}

	return nil
}

// formatValue returns a human-readable string for a decoded parameter value.
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case event.Address:
		return val.Hex()
	case event.Hash:
		return val.Hex()
	case *big.Int:
		if val == nil {
			return "0"
		}
		return val.String()
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []byte:
		return "0x" + hex.EncodeToString(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// jsonValue converts a decoded parameter value to a JSON-friendly representation.
func jsonValue(v interface{}) interface{} {
	switch val := v.(type) {
	case event.Address:
		return val.Hex()
	case event.Hash:
		return val.Hex()
	case *big.Int:
		if val == nil {
			return "0"
		}
		return val.String()
	case []byte:
		return "0x" + hex.EncodeToString(val)
	default:
		return v
	}
}

func findParamInsensitive(params map[string]interface{}, name string) (interface{}, bool) {
	lower := strings.ToLower(name)
	for k, v := range params {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return nil, false
}

func assignValue(fv reflect.Value, val interface{}) error {
	if val == nil {
		return nil
	}

	// Direct assignability check
	rv := reflect.ValueOf(val)
	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)
		return nil
	}

	// Handle pointer types: *big.Int, etc.
	if fv.Kind() == reflect.Ptr {
		if rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
			return nil
		}
		// If field is *T and value is T, wrap it
		if fv.Type().Elem() == rv.Type() {
			ptr := reflect.New(rv.Type())
			ptr.Elem().Set(rv)
			fv.Set(ptr)
			return nil
		}
	}

	// Numeric conversions from *big.Int
	if bi, ok := val.(*big.Int); ok && bi != nil {
		switch fv.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fv.SetUint(bi.Uint64())
			return nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fv.SetInt(bi.Int64())
			return nil
		}
	}

	// string from Address/Hash
	if fv.Kind() == reflect.String {
		fv.SetString(formatValue(val))
		return nil
	}

	// []byte
	if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.Uint8 {
		switch v := val.(type) {
		case []byte:
			fv.SetBytes(v)
			return nil
		case event.Hash:
			fv.SetBytes(v[:])
			return nil
		}
	}

	return fmt.Errorf("cannot assign %T to %s", val, fv.Type())
}
