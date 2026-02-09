// Package ethereum provides the Ethereum implementation of the chain.Chain interface.
package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hedeqiang/sonar/chain"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	"github.com/hedeqiang/sonar/transport"
)

// Client is an Ethereum chain implementation.
type Client struct {
	id        string
	transport transport.Transport
}

// New creates an Ethereum client with the given RPC endpoint.
func New(rpcURL string) *Client {
	return NewWithID("ethereum", rpcURL)
}

// NewWithID creates an Ethereum-compatible client with a custom chain ID.
// This allows reuse for EVM-compatible chains (BSC, Polygon, etc.).
func NewWithID(id, rpcURL string) *Client {
	var t transport.Transport
	if strings.HasPrefix(rpcURL, "ws://") || strings.HasPrefix(rpcURL, "wss://") {
		t = transport.NewWebSocket(rpcURL)
	} else {
		t = transport.NewHTTP(rpcURL)
	}
	return &Client{
		id:        id,
		transport: t,
	}
}

// NewWithTransport creates an Ethereum client with a custom transport.
func NewWithTransport(id string, t transport.Transport) *Client {
	return &Client{
		id:        id,
		transport: t,
	}
}

// ID returns the chain identifier.
func (c *Client) ID() string {
	return c.id
}

// LatestBlock returns the latest block number.
func (c *Client) LatestBlock(ctx context.Context) (uint64, error) {
	result, err := c.transport.Call(ctx, "eth_blockNumber")
	if err != nil {
		return 0, fmt.Errorf("ethereum: eth_blockNumber: %w", err)
	}

	var hex string
	if err := json.Unmarshal(result, &hex); err != nil {
		return 0, fmt.Errorf("ethereum: parse block number: %w", err)
	}

	return parseHexUint64(hex)
}

// FetchLogs retrieves historical logs matching the query.
func (c *Client) FetchLogs(ctx context.Context, query filter.Query) ([]event.Log, error) {
	params := buildFilterParams(query)

	result, err := c.transport.Call(ctx, "eth_getLogs", params)
	if err != nil {
		return nil, fmt.Errorf("ethereum: eth_getLogs: %w", err)
	}

	var rawLogs []rpcLog
	if err := json.Unmarshal(result, &rawLogs); err != nil {
		return nil, fmt.Errorf("ethereum: parse logs: %w", err)
	}

	logs := make([]event.Log, len(rawLogs))
	for i, rl := range rawLogs {
		l, err := rl.toEventLog(c.id)
		if err != nil {
			return nil, fmt.Errorf("ethereum: convert log %d: %w", i, err)
		}
		logs[i] = l
	}

	return logs, nil
}

// Subscribe creates a real-time log subscription via WebSocket.
func (c *Client) Subscribe(ctx context.Context, query filter.Query) (chain.Subscription, error) {
	params := buildFilterParams(query)

	ch, unsub, err := c.transport.Subscribe(ctx, "eth_subscribe", "logs", params)
	if err != nil {
		return nil, fmt.Errorf("ethereum: subscribe: %w", err)
	}

	sub := newSubscription(c.id, ch, unsub)
	return sub, nil
}

// buildFilterParams converts a Query into the JSON-RPC filter object.
func buildFilterParams(query filter.Query) map[string]interface{} {
	params := make(map[string]interface{})

	if query.FromBlock != nil {
		params["fromBlock"] = fmt.Sprintf("0x%x", *query.FromBlock)
	}
	if query.ToBlock != nil {
		params["toBlock"] = fmt.Sprintf("0x%x", *query.ToBlock)
	}

	if len(query.Addresses) > 0 {
		addrs := make([]string, len(query.Addresses))
		for i, a := range query.Addresses {
			addrs[i] = a.Hex()
		}
		if len(addrs) == 1 {
			params["address"] = addrs[0]
		} else {
			params["address"] = addrs
		}
	}

	if len(query.Topics) > 0 {
		topics := make([]interface{}, len(query.Topics))
		for i, ts := range query.Topics {
			if len(ts) == 0 {
				topics[i] = nil
			} else if len(ts) == 1 {
				topics[i] = ts[0].Hex()
			} else {
				hashes := make([]string, len(ts))
				for j, h := range ts {
					hashes[j] = h.Hex()
				}
				topics[i] = hashes
			}
		}
		params["topics"] = topics
	}

	return params
}

// parseHexUint64 parses a "0x"-prefixed hex string to uint64.
func parseHexUint64(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return strconv.ParseUint(s, 16, 64)
}

// rpcLog is the JSON-RPC representation of an Ethereum log.
type rpcLog struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	BlockHash   string   `json:"blockHash"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     string   `json:"transactionIndex"`
	LogIndex    string   `json:"logIndex"`
	Removed     bool     `json:"removed"`
}

func (rl *rpcLog) toEventLog(chainID string) (event.Log, error) {
	var log event.Log
	log.Chain = chainID
	log.Removed = rl.Removed

	// Parse address
	addrBytes, err := decodeHex(rl.Address)
	if err != nil {
		return log, fmt.Errorf("parse address: %w", err)
	}
	copy(log.Address[:], padLeft(addrBytes, 20))

	// Parse topics
	log.Topics = make([]event.Hash, len(rl.Topics))
	for i, t := range rl.Topics {
		b, err := decodeHex(t)
		if err != nil {
			return log, fmt.Errorf("parse topic %d: %w", i, err)
		}
		copy(log.Topics[i][:], padLeft(b, 32))
	}

	// Parse data
	if rl.Data != "" && rl.Data != "0x" {
		log.Data, err = decodeHex(rl.Data)
		if err != nil {
			return log, fmt.Errorf("parse data: %w", err)
		}
	}

	// Parse block number
	if rl.BlockNumber != "" {
		log.BlockNumber, err = parseHexUint64(rl.BlockNumber)
		if err != nil {
			return log, fmt.Errorf("parse blockNumber: %w", err)
		}
	}

	// Parse block hash
	if rl.BlockHash != "" {
		b, err := decodeHex(rl.BlockHash)
		if err != nil {
			return log, fmt.Errorf("parse blockHash: %w", err)
		}
		copy(log.BlockHash[:], padLeft(b, 32))
	}

	// Parse tx hash
	if rl.TxHash != "" {
		b, err := decodeHex(rl.TxHash)
		if err != nil {
			return log, fmt.Errorf("parse txHash: %w", err)
		}
		copy(log.TxHash[:], padLeft(b, 32))
	}

	// Parse tx index
	if rl.TxIndex != "" {
		idx, err := parseHexUint64(rl.TxIndex)
		if err != nil {
			return log, fmt.Errorf("parse txIndex: %w", err)
		}
		log.TxIndex = uint(idx)
	}

	// Parse log index
	if rl.LogIndex != "" {
		idx, err := parseHexUint64(rl.LogIndex)
		if err != nil {
			return log, fmt.Errorf("parse logIndex: %w", err)
		}
		log.LogIndex = uint(idx)
	}

	return log, nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi := unhex(s[i])
		lo := unhex(s[i+1])
		if hi == 0xff || lo == 0xff {
			return nil, fmt.Errorf("invalid hex char at position %d", i)
		}
		b[i/2] = hi<<4 | lo
	}
	return b, nil
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0xff
	}
}

func padLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b[len(b)-size:]
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}
