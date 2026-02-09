package event

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// HexToAddress converts a "0x"-prefixed hex string to an Address.
func HexToAddress(s string) (Address, error) {
	b, err := decodeHexBytes(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid address %q: %w", s, err)
	}
	var addr Address
	if len(b) > 20 {
		copy(addr[:], b[len(b)-20:])
	} else {
		copy(addr[20-len(b):], b)
	}
	return addr, nil
}

// MustHexToAddress is like HexToAddress but panics on error.
func MustHexToAddress(s string) Address {
	addr, err := HexToAddress(s)
	if err != nil {
		panic(err)
	}
	return addr
}

// HexToHash converts a "0x"-prefixed hex string to a Hash.
func HexToHash(s string) (Hash, error) {
	b, err := decodeHexBytes(s)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid hash %q: %w", s, err)
	}
	var h Hash
	if len(b) > 32 {
		copy(h[:], b[len(b)-32:])
	} else {
		copy(h[32-len(b):], b)
	}
	return h, nil
}

// MustHexToHash is like HexToHash but panics on error.
func MustHexToHash(s string) Hash {
	h, err := HexToHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

// Hex returns the "0x"-prefixed hex encoding of the address.
func (a Address) Hex() string {
	return "0x" + hex.EncodeToString(a[:])
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.Hex()
}

// Hex returns the "0x"-prefixed hex encoding of the hash.
func (h Hash) Hex() string {
	return "0x" + hex.EncodeToString(h[:])
}

// String implements fmt.Stringer.
func (h Hash) String() string {
	return h.Hex()
}

func decodeHexBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}
