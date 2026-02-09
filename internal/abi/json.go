package abi

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONABIEntry represents a single entry in an Ethereum JSON ABI array.
type JSONABIEntry struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Inputs    []JSONABIInput  `json:"inputs"`
	Anonymous bool            `json:"anonymous"`
}

// JSONABIInput represents a single input parameter in a JSON ABI entry.
type JSONABIInput struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Indexed    bool           `json:"indexed"`
	Components []JSONABIInput `json:"components,omitempty"` // for tuple types
}

// ParseJSONABI parses a full JSON ABI (array of entries) and returns only the
// event definitions as ParsedEvent values.
func ParseJSONABI(jsonData []byte) ([]*ParsedEvent, error) {
	var entries []JSONABIEntry
	if err := json.Unmarshal(jsonData, &entries); err != nil {
		return nil, fmt.Errorf("abi: parse JSON ABI: %w", err)
	}

	var events []*ParsedEvent
	for _, entry := range entries {
		if entry.Type != "event" {
			continue
		}

		parsed, err := jsonEntryToEvent(entry)
		if err != nil {
			return nil, err
		}
		events = append(events, parsed)
	}

	return events, nil
}

// ParseJSONABIEvent parses a single JSON ABI event entry.
func ParseJSONABIEvent(jsonData []byte) (*ParsedEvent, error) {
	var entry JSONABIEntry
	if err := json.Unmarshal(jsonData, &entry); err != nil {
		return nil, fmt.Errorf("abi: parse JSON ABI event: %w", err)
	}

	if entry.Type != "" && entry.Type != "event" {
		return nil, fmt.Errorf("abi: expected type \"event\", got %q", entry.Type)
	}

	return jsonEntryToEvent(entry)
}

func jsonEntryToEvent(entry JSONABIEntry) (*ParsedEvent, error) {
	if entry.Name == "" {
		return nil, fmt.Errorf("abi: event entry has no name")
	}

	params := make([]ParsedParam, len(entry.Inputs))
	for i, input := range entry.Inputs {
		typ := resolveType(input)
		params[i] = ParsedParam{
			Type:    typ,
			Name:    input.Name,
			Indexed: input.Indexed,
		}
	}

	return &ParsedEvent{
		Name:   entry.Name,
		Params: params,
	}, nil
}

// resolveType converts a JSON ABI input to its canonical Solidity type string.
// Handles tuple types by recursively building "(type1,type2,...)" notation.
func resolveType(input JSONABIInput) string {
	if len(input.Components) == 0 {
		return input.Type
	}

	// Tuple type — build canonical representation
	suffix := ""
	base := input.Type
	if idx := strings.Index(base, "["); idx >= 0 {
		suffix = base[idx:] // e.g., "[]" or "[3]"
		base = base[:idx]
	}

	componentTypes := make([]string, len(input.Components))
	for i, comp := range input.Components {
		componentTypes[i] = resolveType(comp)
	}

	return "(" + strings.Join(componentTypes, ",") + ")" + suffix
}
