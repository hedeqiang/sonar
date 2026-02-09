// Package subscriber provides event distribution patterns.
package subscriber

import (
	"github.com/hedeqiang/sonar/event"
)

// Subscriber receives event logs through a chosen delivery mechanism.
type Subscriber interface {
	// Send delivers a log to this subscriber. Non-blocking.
	Send(log event.Log)

	// Close terminates the subscriber and releases resources.
	Close()
}
