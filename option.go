package sonar

import (
	"time"

	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/decoder"
	"github.com/hedeqiang/sonar/middleware"
	"github.com/hedeqiang/sonar/retry"
	"github.com/hedeqiang/sonar/watcher"
)

// Option configures a Sonar instance.
type Option func(*Sonar)

// WithCursor sets the progress cursor for resumable scanning.
func WithCursor(c cursor.Cursor) Option {
	return func(s *Sonar) {
		s.cursor = c
	}
}

// WithRetry sets the retry strategy for failed RPC calls.
func WithRetry(strategy retry.Strategy) Option {
	return func(s *Sonar) {
		s.retry = strategy
	}
}

// WithMiddleware adds middleware to the event processing pipeline.
func WithMiddleware(mw ...middleware.Middleware) Option {
	return func(s *Sonar) {
		s.middlewares = append(s.middlewares, mw...)
	}
}

// WithPollerConfig overrides the default polling configuration.
func WithPollerConfig(cfg watcher.PollerConfig) Option {
	return func(s *Sonar) {
		s.config.Poller = cfg
	}
}

// WithPollInterval sets the polling interval.
func WithPollInterval(d time.Duration) Option {
	return func(s *Sonar) {
		s.config.Poller.Interval = d
	}
}

// WithBatchSize sets the maximum number of blocks per polling cycle.
func WithBatchSize(size uint64) Option {
	return func(s *Sonar) {
		s.config.Poller.BatchSize = size
	}
}

// WithConfirmations sets the number of confirmation blocks to wait.
func WithConfirmations(n uint64) Option {
	return func(s *Sonar) {
		s.config.Poller.Confirmations = n
	}
}

// WithLogLevel sets the log verbosity level.
func WithLogLevel(level string) Option {
	return func(s *Sonar) {
		s.config.LogLevel = level
	}
}

// WithDecoder sets the event decoder for ABI decoding.
func WithDecoder(d decoder.Decoder) Option {
	return func(s *Sonar) {
		s.decoder = d
	}
}
