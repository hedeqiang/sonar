package middleware

import (
	"log"

	"github.com/hedeqiang/sonar/event"
)

// Logger logs each event log that passes through the pipeline.
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a logging middleware using the provided logger.
// If logger is nil, the default standard logger is used.
func NewLogger(l *log.Logger) *Logger {
	if l == nil {
		l = log.Default()
	}
	return &Logger{logger: l}
}

// Wrap decorates the handler with event logging.
func (l *Logger) Wrap(next Handler) Handler {
	return func(lg event.Log) *event.Log {
		sig := lg.EventSignature()
		l.logger.Printf("[sonar] chain=%s block=%d tx=%x logIndex=%d addr=%x topic0=%x",
			lg.Chain,
			lg.BlockNumber,
			lg.TxHash[:8],
			lg.LogIndex,
			lg.Address[:8],
			sig[:8],
		)
		return next(lg)
	}
}
