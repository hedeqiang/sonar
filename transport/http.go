package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// HTTP implements Transport over HTTP JSON-RPC.
type HTTP struct {
	url    string
	client *http.Client
	nextID atomic.Uint64
}

// NewHTTP creates an HTTP transport targeting the given JSON-RPC endpoint.
func NewHTTP(url string) *HTTP {
	return &HTTP{
		url:    url,
		client: &http.Client{},
	}
}

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("rpc error: code=%d message=%s", e.Code, e.Message)
}

// Call sends an HTTP JSON-RPC request and returns the result bytes.
func (h *HTTP) Call(ctx context.Context, method string, params ...interface{}) ([]byte, error) {
	if params == nil {
		params = []interface{}{}
	}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      h.nextID.Add(1),
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("transport/http: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("transport/http: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("transport/http: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("transport/http: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body := string(respBody)
		if len(body) > 256 {
			body = body[:256]
		}
		return nil, fmt.Errorf("transport/http: HTTP %d: %s", resp.StatusCode, body)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("transport/http: unmarshal response (status %d): %w", resp.StatusCode, err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// Subscribe is not supported over HTTP and always returns an error.
func (h *HTTP) Subscribe(_ context.Context, _ string, _ ...interface{}) (<-chan []byte, func(), error) {
	return nil, nil, fmt.Errorf("transport/http: subscriptions not supported over HTTP")
}

// Close is a no-op for HTTP transport.
func (h *HTTP) Close() error {
	return nil
}
