package decoder

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/hedeqiang/sonar/event"
	abiutil "github.com/hedeqiang/sonar/internal/abi"
)

// ABIDecoder decodes event logs using registered ABI event definitions.
type ABIDecoder struct {
	schema *Schema
}

// NewABIDecoder creates a new ABI-based event decoder.
func NewABIDecoder() *ABIDecoder {
	return &ABIDecoder{
		schema: NewSchema(),
	}
}

// Register parses a Solidity event signature and registers it for decoding.
// Example: "Transfer(address indexed from, address indexed to, uint256 value)"
func (d *ABIDecoder) Register(eventSignature string) error {
	parsed, err := abiutil.ParseEventSignature(eventSignature)
	if err != nil {
		return fmt.Errorf("decoder: %w", err)
	}

	d.registerParsed(parsed)
	return nil
}

// RegisterJSON registers all event definitions from a standard JSON ABI.
// Accepts a full contract ABI (array of entries); non-event entries are ignored.
//
// Example:
//
//	dec.RegisterJSON([]byte(`[{"type":"event","name":"Transfer","inputs":[...]}]`))
func (d *ABIDecoder) RegisterJSON(jsonABI []byte) error {
	events, err := abiutil.ParseJSONABI(jsonABI)
	if err != nil {
		return fmt.Errorf("decoder: %w", err)
	}

	for _, parsed := range events {
		d.registerParsed(parsed)
	}
	return nil
}

// RegisterJSONEvent registers a single event definition from a JSON ABI entry.
//
// Example:
//
//	dec.RegisterJSONEvent([]byte(`{"type":"event","name":"Transfer","inputs":[...]}`))
func (d *ABIDecoder) RegisterJSONEvent(jsonEvent []byte) error {
	parsed, err := abiutil.ParseJSONABIEvent(jsonEvent)
	if err != nil {
		return fmt.Errorf("decoder: %w", err)
	}

	d.registerParsed(parsed)
	return nil
}

func (d *ABIDecoder) registerParsed(parsed *abiutil.ParsedEvent) {
	canonical := parsed.Canonical()
	sigHash := abiutil.EventSignatureHash(canonical)

	inputs := make([]ParamDef, len(parsed.Params))
	for i, p := range parsed.Params {
		inputs[i] = ParamDef{
			Name:    p.Name,
			Type:    p.Type,
			Indexed: p.Indexed,
		}
	}

	d.schema.Add(&EventDef{
		Name:      parsed.Name,
		Signature: canonical,
		SigHash:   sigHash,
		Inputs:    inputs,
	})
}

// Decode attempts to decode a log using registered event definitions.
func (d *ABIDecoder) Decode(log event.Log) (*DecodedEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("decoder: log has no topics")
	}

	def, ok := d.schema.Lookup(log.Topics[0])
	if !ok {
		return nil, fmt.Errorf("decoder: unknown event signature %x", log.Topics[0])
	}

	decoded := &DecodedEvent{
		Name:      def.Name,
		Signature: def.Signature,
		Params:    make(map[string]interface{}),
		Indexed:   make(map[string]interface{}),
		Raw:       log,
	}

	// Decode indexed parameters from topics
	topicIdx := 1 // topic[0] is event signature
	for _, input := range def.Inputs {
		if !input.Indexed {
			continue
		}
		if topicIdx >= len(log.Topics) {
			break
		}

		name := input.Name
		if name == "" {
			name = fmt.Sprintf("arg%d", topicIdx)
		}

		val := decodeTopicValue(input.Type, log.Topics[topicIdx])
		decoded.Indexed[name] = val
		decoded.Params[name] = val
		topicIdx++
	}

	// Decode non-indexed parameters from data
	dataParams := make([]ParamDef, 0)
	for _, input := range def.Inputs {
		if !input.Indexed {
			dataParams = append(dataParams, input)
		}
	}

	if len(dataParams) > 0 && len(log.Data) > 0 {
		offset := 0
		for i, param := range dataParams {
			if offset+32 > len(log.Data) {
				break
			}

			name := param.Name
			if name == "" {
				name = fmt.Sprintf("data%d", i)
			}

			val := decodeDataValue(param.Type, log.Data[offset:offset+32])
			decoded.Params[name] = val
			offset += 32
		}
	}

	return decoded, nil
}

// decodeTopicValue decodes a single indexed parameter from a 32-byte topic.
func decodeTopicValue(typ string, topic event.Hash) interface{} {
	switch {
	case typ == "address":
		var addr event.Address
		copy(addr[:], topic[12:32])
		return addr
	case typ == "bool":
		return topic[31] != 0
	case strings.HasPrefix(typ, "uint"):
		return new(big.Int).SetBytes(topic[:])
	case strings.HasPrefix(typ, "int"):
		return new(big.Int).SetBytes(topic[:])
	case strings.HasPrefix(typ, "bytes"):
		return topic
	default:
		return topic
	}
}

// decodeDataValue decodes a single non-indexed parameter from a 32-byte ABI word.
func decodeDataValue(typ string, data []byte) interface{} {
	switch {
	case typ == "address":
		var addr event.Address
		copy(addr[:], data[12:32])
		return addr
	case typ == "bool":
		return data[31] != 0
	case strings.HasPrefix(typ, "uint"):
		return new(big.Int).SetBytes(data)
	case strings.HasPrefix(typ, "int"):
		return new(big.Int).SetBytes(data)
	case typ == "string" || typ == "bytes":
		return data
	default:
		// For fixed-size bytesN types
		if len(data) >= 8 {
			return binary.BigEndian.Uint64(data[24:32])
		}
		return data
	}
}
