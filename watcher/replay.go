package watcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/hedeqiang/sonar/chain"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
)

// Replay fetches historical event logs for a specific block range.
// Unlike Poller, it runs once and completes—useful for backfilling data.
type Replay struct {
	chain     chain.Chain
	query     filter.Query
	batchSize uint64

	mu      sync.Mutex
	onEvent func(event.Log)
	onError func(error)
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewReplay creates a replay watcher that scans a fixed block range.
// The query must have FromBlock and ToBlock set.
func NewReplay(c chain.Chain, query filter.Query, batchSize uint64) *Replay {
	if batchSize == 0 {
		batchSize = 2000
	}
	return &Replay{
		chain:     c,
		query:     query,
		batchSize: batchSize,
		stopped:   make(chan struct{}),
	}
}

// OnEvent registers a callback for received events.
func (r *Replay) OnEvent(fn func(event.Log)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onEvent = fn
}

// OnError registers a callback for errors.
func (r *Replay) OnError(fn func(error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onError = fn
}

// Watch replays historical events. Completes when the entire range is scanned.
func (r *Replay) Watch() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	defer close(r.stopped)
	defer cancel()

	if r.query.FromBlock == nil || r.query.ToBlock == nil {
		return fmt.Errorf("replay: both FromBlock and ToBlock must be set")
	}

	from := *r.query.FromBlock
	to := *r.query.ToBlock

	for from <= to {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		batchEnd := from + r.batchSize - 1
		if batchEnd > to {
			batchEnd = to
		}

		q := r.query
		q.FromBlock = &from
		q.ToBlock = &batchEnd

		logs, err := r.chain.FetchLogs(ctx, q)
		if err != nil {
			r.emitError(fmt.Errorf("fetch logs [%d, %d]: %w", from, batchEnd, err))
			from = batchEnd + 1
			continue
		}

		for _, log := range logs {
			r.emitEvent(log)
		}

		from = batchEnd + 1
	}

	return nil
}

// Stop cancels the replay.
func (r *Replay) Stop() error {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel()
		<-r.stopped
	}
	return nil
}

func (r *Replay) emitEvent(log event.Log) {
	r.mu.Lock()
	fn := r.onEvent
	r.mu.Unlock()
	if fn != nil {
		fn(log)
	}
}

func (r *Replay) emitError(err error) {
	r.mu.Lock()
	fn := r.onError
	r.mu.Unlock()
	if fn != nil {
		fn(err)
	}
}
