package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// WebSocket implements Transport over a WebSocket connection.
type WebSocket struct {
	url    string
	conn   *websocket.Conn
	mu     sync.Mutex
	nextID atomic.Uint64

	// connection management
	connOnce sync.Once
	connErr  error

	// subscription management
	subMu   sync.Mutex
	subs    map[uint64]chan []byte
	closed  chan struct{}
	closeOnce sync.Once
}

// NewWebSocket creates a WebSocket transport.
// The connection is established lazily on the first Call or Subscribe.
func NewWebSocket(url string) *WebSocket {
	return &WebSocket{
		url:    url,
		subs:   make(map[uint64]chan []byte),
		closed: make(chan struct{}),
	}
}

// connect establishes the WebSocket connection (called lazily, at most once).
func (ws *WebSocket) connect(ctx context.Context) error {
	ws.connOnce.Do(func() {
		dialer := websocket.Dialer{}
		conn, _, err := dialer.DialContext(ctx, ws.url, nil)
		if err != nil {
			ws.connErr = fmt.Errorf("transport/ws: dial: %w", err)
			return
		}
		ws.conn = conn
		go ws.readLoop()
	})
	return ws.connErr
}

// Call sends a JSON-RPC request over WebSocket and waits for the response.
func (ws *WebSocket) Call(ctx context.Context, method string, params ...interface{}) ([]byte, error) {
	if err := ws.connect(ctx); err != nil {
		return nil, err
	}

	if params == nil {
		params = []interface{}{}
	}

	id := ws.nextID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Create a response channel for this request
	ch := make(chan []byte, 1)
	ws.subMu.Lock()
	ws.subs[id] = ch
	ws.subMu.Unlock()

	defer func() {
		ws.subMu.Lock()
		delete(ws.subs, id)
		ws.subMu.Unlock()
	}()

	ws.mu.Lock()
	err := ws.conn.WriteJSON(req)
	ws.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("transport/ws: write: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-ch:
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal(data, &rpcResp); err != nil {
			return nil, fmt.Errorf("transport/ws: unmarshal: %w", err)
		}
		if rpcResp.Error != nil {
			return nil, rpcResp.Error
		}
		return rpcResp.Result, nil
	case <-ws.closed:
		return nil, fmt.Errorf("transport/ws: connection closed")
	}
}

// Subscribe sends a subscription request and returns a channel for incoming messages.
func (ws *WebSocket) Subscribe(ctx context.Context, method string, params ...interface{}) (<-chan []byte, func(), error) {
	if err := ws.connect(ctx); err != nil {
		return nil, nil, err
	}

	result, err := ws.Call(ctx, method, params...)
	if err != nil {
		return nil, nil, err
	}

	// Parse subscription ID from result
	var subID string
	if err := json.Unmarshal(result, &subID); err != nil {
		return nil, nil, fmt.Errorf("transport/ws: parse subscription id: %w", err)
	}

	ch := make(chan []byte, 64)

	// Use a synthetic ID based on subscription string for routing
	syntheticID := ws.nextID.Add(1)
	ws.subMu.Lock()
	ws.subs[syntheticID] = ch
	ws.subMu.Unlock()

	unsub := func() {
		ws.subMu.Lock()
		delete(ws.subs, syntheticID)
		ws.subMu.Unlock()
		close(ch)
	}

	return ch, unsub, nil
}

// Close terminates the WebSocket connection.
func (ws *WebSocket) Close() error {
	ws.closeOnce.Do(func() {
		close(ws.closed)
	})
	if ws.conn != nil {
		return ws.conn.Close()
	}
	return nil
}

// readLoop reads messages from the WebSocket and routes them to waiting callers.
func (ws *WebSocket) readLoop() {
	for {
		select {
		case <-ws.closed:
			return
		default:
		}

		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			ws.closeOnce.Do(func() {
				close(ws.closed)
			})
			return
		}

		// Try to parse the ID to route the response
		var envelope struct {
			ID     uint64          `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(message, &envelope); err != nil {
			continue
		}

		// Route response to the correct caller
		if envelope.ID != 0 {
			ws.subMu.Lock()
			if ch, ok := ws.subs[envelope.ID]; ok {
				select {
				case ch <- message:
				default:
				}
			}
			ws.subMu.Unlock()
		}

		// Handle subscription notifications (eth_subscription)
		if envelope.Method == "eth_subscription" {
			ws.subMu.Lock()
			for _, ch := range ws.subs {
				select {
				case ch <- []byte(envelope.Params):
				default:
				}
			}
			ws.subMu.Unlock()
		}
	}
}
