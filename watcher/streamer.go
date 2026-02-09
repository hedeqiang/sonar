package watcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/hedeqiang/sonar/chain"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
)

// Streamer monitors a chain via WebSocket subscriptions for real-time event delivery.
type Streamer struct {
	chain chain.Chain
	query filter.Query

	mu      sync.Mutex
	onEvent func(event.Log)
	onError func(error)
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewStreamer creates a streaming watcher for the given chain.
func NewStreamer(c chain.Chain, query filter.Query) *Streamer {
	return &Streamer{
		chain:   c,
		query:   query,
		stopped: make(chan struct{}),
	}
}

// OnEvent registers a callback for received events.
func (s *Streamer) OnEvent(fn func(event.Log)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEvent = fn
}

// OnError registers a callback for errors.
func (s *Streamer) OnError(fn func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = fn
}

// Watch starts the streaming subscription. Blocks until Stop is called.
func (s *Streamer) Watch() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	defer close(s.stopped)
	defer cancel()

	sub, err := s.chain.Subscribe(ctx, s.query)
	if err != nil {
		return fmt.Errorf("streamer: subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case log, ok := <-sub.Logs():
			if !ok {
				return nil
			}
			s.emitEvent(log)
		case err, ok := <-sub.Err():
			if !ok {
				return nil
			}
			s.emitError(err)
		}
	}
}

// Stop terminates the streaming subscription.
func (s *Streamer) Stop() error {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
		<-s.stopped
	}
	return nil
}

func (s *Streamer) emitEvent(log event.Log) {
	s.mu.Lock()
	fn := s.onEvent
	s.mu.Unlock()
	if fn != nil {
		fn(log)
	}
}

func (s *Streamer) emitError(err error) {
	s.mu.Lock()
	fn := s.onError
	s.mu.Unlock()
	if fn != nil {
		fn(err)
	}
}
