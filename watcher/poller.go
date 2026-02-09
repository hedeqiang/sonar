package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hedeqiang/sonar/chain"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
)

// PollerConfig configures a Poller.
type PollerConfig struct {
	// Interval between polling cycles.
	Interval time.Duration

	// BatchSize is the maximum number of blocks to query per cycle.
	BatchSize uint64

	// Confirmations is the number of blocks to wait for finality.
	Confirmations uint64
}

// DefaultPollerConfig returns sensible defaults for polling.
func DefaultPollerConfig() PollerConfig {
	return PollerConfig{
		Interval:      2 * time.Second,
		BatchSize:     1000,
		Confirmations: 0,
	}
}

// Poller monitors a chain by periodically fetching logs in block ranges.
type Poller struct {
	chain  chain.Chain
	query  filter.Query
	cursor cursor.Cursor
	config PollerConfig

	mu      sync.Mutex
	onEvent func(event.Log)
	onError func(error)
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewPoller creates a polling watcher for the given chain.
func NewPoller(c chain.Chain, query filter.Query, cur cursor.Cursor, cfg PollerConfig) *Poller {
	return &Poller{
		chain:   c,
		query:   query,
		cursor:  cur,
		config:  cfg,
		stopped: make(chan struct{}),
	}
}

// OnEvent registers a callback for received events.
func (p *Poller) OnEvent(fn func(event.Log)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onEvent = fn
}

// OnError registers a callback for errors.
func (p *Poller) OnError(fn func(error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onError = fn
}

// Watch begins polling. Blocks until Stop is called or an unrecoverable error occurs.
func (p *Poller) Watch() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()

	defer close(p.stopped)
	defer cancel()

	// Determine start block
	lastBlock, err := p.cursor.Load(p.chain.ID())
	if err != nil {
		return fmt.Errorf("poller: load cursor: %w", err)
	}

	var fromBlock uint64
	if lastBlock > 0 {
		fromBlock = lastBlock + 1 // resume after last processed block
	} else {
		// No cursor — start from the chain's safe head so we immediately pick up events
		latest, err := p.chain.LatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("poller: get latest block: %w", err)
		}
		if latest > p.config.Confirmations {
			fromBlock = latest - p.config.Confirmations
		}
	}

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	// Run the first poll immediately instead of waiting for the first tick
	if err := p.poll(ctx, &fromBlock); err != nil {
		p.emitError(err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.poll(ctx, &fromBlock); err != nil {
				p.emitError(err)
			}
		}
	}
}

// Stop terminates the polling loop.
func (p *Poller) Stop() error {
	p.mu.Lock()
	cancel := p.cancel
	p.mu.Unlock()

	if cancel != nil {
		cancel()
		<-p.stopped
	}
	return nil
}

func (p *Poller) poll(ctx context.Context, fromBlock *uint64) error {
	latest, err := p.chain.LatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("get latest block: %w", err)
	}

	// Apply confirmations
	if latest <= p.config.Confirmations {
		return nil
	}
	safeBlock := latest - p.config.Confirmations

	if *fromBlock > safeBlock {
		return nil // already caught up
	}

	toBlock := *fromBlock + p.config.BatchSize - 1
	if toBlock > safeBlock {
		toBlock = safeBlock
	}

	// Build query with block range
	q := p.query
	q.FromBlock = fromBlock
	q.ToBlock = &toBlock

	logs, err := p.chain.FetchLogs(ctx, q)
	if err != nil {
		return fmt.Errorf("fetch logs [%d, %d]: %w", *fromBlock, toBlock, err)
	}

	for _, log := range logs {
		p.emitEvent(log)
	}

	// Save progress
	if err := p.cursor.Save(p.chain.ID(), toBlock); err != nil {
		return fmt.Errorf("save cursor: %w", err)
	}

	*fromBlock = toBlock + 1
	return nil
}

func (p *Poller) emitEvent(log event.Log) {
	p.mu.Lock()
	fn := p.onEvent
	p.mu.Unlock()
	if fn != nil {
		fn(log)
	}
}

func (p *Poller) emitError(err error) {
	p.mu.Lock()
	fn := p.onError
	p.mu.Unlock()
	if fn != nil {
		fn(err)
	}
}
