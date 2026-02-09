// Package abi provides internal utilities for parsing Solidity event signatures.
package abi

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/hedeqiang/sonar/event"
)

// EventSignatureHash computes the Keccak-256 hash of a canonical event signature.
func EventSignatureHash(sig string) event.Hash {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(sig))
	var out event.Hash
	copy(out[:], h.Sum(nil))
	return out
}

// ParsedEvent represents a parsed Solidity event signature.
type ParsedEvent struct {
	Name   string
	Params []ParsedParam
}

// ParsedParam represents a single parameter in an event signature.
type ParsedParam struct {
	Type    string
	Name    string
	Indexed bool
}

// Canonical returns the canonical signature string (e.g. "Transfer(address,address,uint256)").
func (p *ParsedEvent) Canonical() string {
	types := make([]string, len(p.Params))
	for i, param := range p.Params {
		types[i] = param.Type
	}
	return fmt.Sprintf("%s(%s)", p.Name, strings.Join(types, ","))
}

// ParseEventSignature parses a Solidity event signature string.
// Supported formats:
//   - "Transfer(address,address,uint256)"
//   - "Transfer(address indexed from, address indexed to, uint256 value)"
func ParseEventSignature(sig string) (*ParsedEvent, error) {
	sig = strings.TrimSpace(sig)

	parenOpen := strings.IndexByte(sig, '(')
	parenClose := strings.LastIndexByte(sig, ')')
	if parenOpen < 0 || parenClose < 0 || parenClose <= parenOpen {
		return nil, fmt.Errorf("abi: malformed event signature: %q", sig)
	}

	name := strings.TrimSpace(sig[:parenOpen])
	if name == "" {
		return nil, fmt.Errorf("abi: empty event name in signature: %q", sig)
	}

	paramsStr := strings.TrimSpace(sig[parenOpen+1 : parenClose])
	if paramsStr == "" {
		return &ParsedEvent{Name: name}, nil
	}

	parts := splitParams(paramsStr)
	params := make([]ParsedParam, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		p, err := parseParam(part)
		if err != nil {
			return nil, fmt.Errorf("abi: %w in signature %q", err, sig)
		}
		params = append(params, p)
	}

	return &ParsedEvent{Name: name, Params: params}, nil
}

func parseParam(s string) (ParsedParam, error) {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return ParsedParam{}, fmt.Errorf("empty parameter")
	}

	var p ParsedParam
	p.Type = tokens[0]

	for i := 1; i < len(tokens); i++ {
		if tokens[i] == "indexed" {
			p.Indexed = true
		} else {
			p.Name = tokens[i]
		}
	}

	return p, nil
}

// splitParams splits a parameter list string, respecting nested parentheses (e.g., tuples).
func splitParams(s string) []string {
	var parts []string
	depth := 0
	start := 0

	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}
