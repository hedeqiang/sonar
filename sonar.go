// Package sonar provides a multi-chain event log monitoring SDK.
//
// Sonar — a deep probe for every on-chain event signal.
//
// Usage:
//
//	s := sonar.New(
//	    sonar.WithCursor(cursor.NewMemory()),
//	    sonar.WithRetry(retry.Exponential(3)),
//	)
//
//	s.AddChain(ethereum.New("https://mainnet.infura.io/v3/KEY"))
//
//	q := filter.NewQuery(
//	    filter.WithAddresses(addr),
//	)
//
//	s.Watch("ethereum", q, func(log event.Log) {
//	    fmt.Println("event:", log.BlockNumber)
//	})
package sonar

import (
	"context"
	"fmt"
	"sync"

	"github.com/hedeqiang/sonar/chain"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/decoder"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	"github.com/hedeqiang/sonar/middleware"
	"github.com/hedeqiang/sonar/retry"
	"github.com/hedeqiang/sonar/watcher"
)

// Sonar is the main SDK entry point for multi-chain event log monitoring.
type Sonar struct {
	registry    *chain.Registry
	cursor      cursor.Cursor
	retry       retry.Strategy
	decoder     decoder.Decoder
	middlewares []middleware.Middleware
	config      Config

	mu       sync.Mutex
	watchers map[string]watcher.Watcher
	shutdown bool
}

// New creates a new Sonar instance with the given options.
func New(opts ...Option) *Sonar {
	s := &Sonar{
		registry: chain.NewRegistry(),
		cursor:   cursor.NewMemory(),
		config:   DefaultConfig(),
		watchers: make(map[string]watcher.Watcher),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AddChain registers a chain implementation. Returns an error if the chain ID
// is already registered.
func (s *Sonar) AddChain(c chain.Chain) error {
	return s.registry.Register(c)
}

// Watch begins monitoring the specified chain for events matching the query.
// The handler is called for each event log that passes through the middleware pipeline.
// This method launches a background goroutine and returns immediately.
func (s *Sonar) Watch(chainID string, query filter.Query, handler func(event.Log)) error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return ErrShutdown
	}
	s.mu.Unlock()

	c, ok := s.registry.Get(chainID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrChainNotFound, chainID)
	}

	// Build the middleware pipeline
	finalHandler := buildHandler(handler, s.middlewares)

	// Create poller watcher
	p := watcher.NewPoller(c, query, s.cursor, s.config.Poller)
	p.OnEvent(func(log event.Log) {
		result := finalHandler(log)
		if result == nil {
			return // dropped by middleware
		}
	})
	p.OnError(func(err error) {
		// TODO: configurable error handler
		fmt.Printf("[sonar] chain=%s error: %v\n", chainID, err)
	})

	s.mu.Lock()
	if _, exists := s.watchers[chainID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrAlreadyRunning, chainID)
	}
	s.watchers[chainID] = p
	s.mu.Unlock()

	go p.Watch()

	return nil
}

// WatchAll begins monitoring all registered chains with the same query and handler.
func (s *Sonar) WatchAll(query filter.Query, handler func(event.Log)) error {
	for _, c := range s.registry.All() {
		if err := s.Watch(c.ID(), query, handler); err != nil {
			return err
		}
	}
	return nil
}

// Use appends middleware to the processing pipeline.
// Must be called before Watch.
func (s *Sonar) Use(mw ...middleware.Middleware) {
	s.middlewares = append(s.middlewares, mw...)
}

// Shutdown gracefully stops all watchers.
func (s *Sonar) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.shutdown = true
	watchers := make(map[string]watcher.Watcher, len(s.watchers))
	for k, v := range s.watchers {
		watchers[k] = v
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, w := range watchers {
			w.Stop()
		}
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Chains returns the IDs of all registered chains.
func (s *Sonar) Chains() []string {
	return s.registry.IDs()
}

// Decoder returns the registered decoder, or nil if none is set.
func (s *Sonar) Decoder() decoder.Decoder {
	return s.decoder
}

// RegisterEvent registers an event ABI signature for decoding.
// If no decoder has been configured, an ABIDecoder is created automatically.
// Example: s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")
func (s *Sonar) RegisterEvent(eventSignature string) error {
	s.ensureDecoder()
	return s.decoder.Register(eventSignature)
}

// RegisterEventJSON registers all event definitions from a standard JSON ABI.
// Non-event entries (functions, constructors, etc.) are ignored.
// If no decoder has been configured, an ABIDecoder is created automatically.
//
// Example:
//
//	s.RegisterEventJSON([]byte(`[{"type":"event","name":"Transfer","inputs":[...]}]`))
func (s *Sonar) RegisterEventJSON(jsonABI []byte) error {
	s.ensureDecoder()
	dec, ok := s.decoder.(*decoder.ABIDecoder)
	if !ok {
		return fmt.Errorf("sonar: RegisterEventJSON requires ABIDecoder")
	}
	return dec.RegisterJSON(jsonABI)
}

func (s *Sonar) ensureDecoder() {
	if s.decoder == nil {
		s.decoder = decoder.NewABIDecoder()
	}
}

// WatchDecoded begins monitoring the specified chain and delivers decoded events.
// Events that cannot be decoded are silently skipped.
// The decoder must have event signatures registered via RegisterEvent.
func (s *Sonar) WatchDecoded(chainID string, query filter.Query, handler func(*decoder.DecodedEvent)) error {
	if s.decoder == nil {
		return fmt.Errorf("sonar: no decoder configured; call RegisterEvent first")
	}

	dec := s.decoder
	return s.Watch(chainID, query, func(log event.Log) {
		decoded, err := dec.Decode(log)
		if err != nil {
			return // skip unrecognized events
		}
		handler(decoded)
	})
}

// buildHandler constructs the middleware pipeline with the user handler at the end.
func buildHandler(handler func(event.Log), mws []middleware.Middleware) middleware.Handler {
	terminal := func(log event.Log) *event.Log {
		handler(log)
		return &log
	}
	return middleware.Chain(terminal, mws...)
}
