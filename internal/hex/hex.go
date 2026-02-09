// Package hex provides utilities for encoding and decoding hexadecimal strings
// with the "0x" prefix commonly used in Ethereum.
package hex

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Encode returns the hexadecimal encoding of src with "0x" prefix.
func Encode(src []byte) string {
	return "0x" + hex.EncodeToString(src)
}

// Decode decodes a hex string (with or without "0x" prefix) into bytes.
func Decode(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

// MustDecode is like Decode but panics on error.
func MustDecode(s string) []byte {
	b, err := Decode(s)
	if err != nil {
		panic(fmt.Sprintf("hex: invalid hex string %q: %v", s, err))
	}
	return b
}

// EncodeUint64 encodes a uint64 as a "0x"-prefixed hex string.
func EncodeUint64(n uint64) string {
	return fmt.Sprintf("0x%x", n)
}
