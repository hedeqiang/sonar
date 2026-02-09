package decoder

import (
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
		Data:      make(map[string]interface{}),
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
		decodeDataParams(log.Data, dataParams, decoded.Params, decoded.Data)
	}

	return decoded, nil
}

// isDynamic returns true if the ABI type is dynamically-sized.
func isDynamic(typ string) bool {
	if typ == "string" || typ == "bytes" {
		return true
	}
	// T[] dynamic arrays
	if strings.HasSuffix(typ, "[]") {
		return true
	}
	// tuple types starting with "(" may be dynamic (simplified: treat all tuples as dynamic)
	if strings.HasPrefix(typ, "(") {
		return true
	}
	return false
}

// decodeDataParams decodes all non-indexed parameters from the ABI-encoded data blob.
// It handles both static (inline) and dynamic (offset-referenced) types.
// Values are written to both out (all params) and dataOut (data-only params).
func decodeDataParams(data []byte, params []ParamDef, out, dataOut map[string]interface{}) {
	// Phase 1: read the head section — 32 bytes per parameter.
	// For static types, the value is inline.
	// For dynamic types, the 32 bytes hold an offset (in bytes) into the data blob.
	for i, param := range params {
		headOffset := i * 32
		if headOffset+32 > len(data) {
			break
		}

		name := param.Name
		if name == "" {
			name = fmt.Sprintf("data%d", i)
		}

		word := data[headOffset : headOffset+32]

		var val interface{}
		if isDynamic(param.Type) {
			// word is a uint256 byte-offset pointing into the data blob
			dynOffset := new(big.Int).SetBytes(word).Uint64()
			val = decodeDynamicValue(param.Type, data, dynOffset)
		} else {
			val = decodeStaticValue(param.Type, word)
		}
		out[name] = val
		dataOut[name] = val
	}
}

// decodeStaticValue decodes a single static (fixed-size) parameter from a 32-byte ABI word.
func decodeStaticValue(typ string, word []byte) interface{} {
	switch {
	case typ == "address":
		var addr event.Address
		copy(addr[:], word[12:32])
		return addr

	case typ == "bool":
		return word[31] != 0

	case strings.HasPrefix(typ, "uint"):
		return new(big.Int).SetBytes(word)

	case strings.HasPrefix(typ, "int"):
		return decodeSigned(word)

	case strings.HasPrefix(typ, "bytes"):
		// bytesN (fixed-size): return the left-aligned N bytes
		n := parseBytesN(typ)
		if n > 0 && n <= 32 {
			result := make([]byte, n)
			copy(result, word[:n])
			return result
		}
		return word

	default:
		return word
	}
}

// decodeDynamicValue decodes a dynamic type (string, bytes, T[]) from the data blob
// starting at the given byte offset.
func decodeDynamicValue(typ string, data []byte, offset uint64) interface{} {
	if offset+32 > uint64(len(data)) {
		return nil
	}

	switch {
	case typ == "string":
		return decodeDynamicBytes(data, offset, true)

	case typ == "bytes":
		return decodeDynamicBytes(data, offset, false)

	case strings.HasSuffix(typ, "[]"):
		// Dynamic array: T[]
		elemType := strings.TrimSuffix(typ, "[]")
		return decodeDynamicArray(elemType, data, offset)

	default:
		// Fallback: read as raw bytes
		return decodeDynamicBytes(data, offset, false)
	}
}

// decodeDynamicBytes decodes a bytes/string value at the given offset.
// Layout: [32-byte length][padded content]
func decodeDynamicBytes(data []byte, offset uint64, asString bool) interface{} {
	if offset+32 > uint64(len(data)) {
		return nil
	}
	length := new(big.Int).SetBytes(data[offset : offset+32]).Uint64()

	contentStart := offset + 32
	if contentStart+length > uint64(len(data)) {
		// Truncated — return what we have
		length = uint64(len(data)) - contentStart
	}

	content := make([]byte, length)
	copy(content, data[contentStart:contentStart+length])

	if asString {
		return string(content)
	}
	return content
}

// decodeDynamicArray decodes a dynamic array T[] at the given offset.
// Layout: [32-byte count][element0][element1]...
func decodeDynamicArray(elemType string, data []byte, offset uint64) interface{} {
	if offset+32 > uint64(len(data)) {
		return nil
	}

	count := new(big.Int).SetBytes(data[offset : offset+32]).Uint64()
	if count == 0 {
		return []interface{}{}
	}

	result := make([]interface{}, 0, count)
	elemOffset := offset + 32

	if isDynamic(elemType) {
		// Each element in the head is an offset relative to the array's data section
		for i := uint64(0); i < count; i++ {
			headPos := elemOffset + i*32
			if headPos+32 > uint64(len(data)) {
				break
			}
			relOffset := new(big.Int).SetBytes(data[headPos : headPos+32]).Uint64()
			val := decodeDynamicValue(elemType, data, elemOffset+relOffset)
			result = append(result, val)
		}
	} else {
		// Static elements are packed sequentially
		for i := uint64(0); i < count; i++ {
			pos := elemOffset + i*32
			if pos+32 > uint64(len(data)) {
				break
			}
			val := decodeStaticValue(elemType, data[pos:pos+32])
			result = append(result, val)
		}
	}

	return result
}

// decodeSigned interprets a 32-byte big-endian word as a signed two's complement integer.
func decodeSigned(word []byte) *big.Int {
	val := new(big.Int).SetBytes(word)
	// If the high bit is set, it's negative
	if word[0]&0x80 != 0 {
		// two's complement: val - 2^256
		max := new(big.Int).Lsh(big.NewInt(1), 256)
		val.Sub(val, max)
	}
	return val
}

// parseBytesN extracts N from "bytesN" (e.g., "bytes32" → 32, "bytes4" → 4).
// Returns 0 if the type is not a valid fixed bytesN.
func parseBytesN(typ string) int {
	if !strings.HasPrefix(typ, "bytes") {
		return 0
	}
	suffix := typ[5:]
	if suffix == "" {
		return 0 // "bytes" without a number is dynamic
	}
	n := 0
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	if n < 1 || n > 32 {
		return 0
	}
	return n
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
		return decodeSigned(topic[:])
	case strings.HasPrefix(typ, "bytes"):
		n := parseBytesN(typ)
		if n > 0 && n <= 32 {
			result := make([]byte, n)
			copy(result, topic[:n])
			return result
		}
		// dynamic bytes/string as indexed → topic is keccak256 hash
		return topic
	default:
		return topic
	}
}
