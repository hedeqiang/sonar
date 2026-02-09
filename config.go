package sonar

import (
	"time"

	"github.com/hedeqiang/sonar/watcher"
)

// Config holds the global configuration for a Sonar instance.
type Config struct {
	// Poller configures the default polling behavior.
	Poller watcher.PollerConfig

	// LogLevel controls log verbosity ("debug", "info", "warn", "error").
	LogLevel string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Poller: watcher.PollerConfig{
			Interval:      2 * time.Second,
			BatchSize:     1000,
			Confirmations: 0,
		},
		LogLevel: "info",
	}
}
