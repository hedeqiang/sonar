package subscriber

import (
	"sync"

	"github.com/hedeqiang/sonar/event"
)

// Broadcast distributes event logs to multiple subscribers.
type Broadcast struct {
	mu   sync.RWMutex
	subs []Subscriber
}

// NewBroadcast creates a new broadcast dispatcher.
func NewBroadcast() *Broadcast {
	return &Broadcast{}
}

// Add registers a subscriber to receive broadcast events.
func (b *Broadcast) Add(sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, sub)
}

// Send delivers a log to all registered subscribers.
func (b *Broadcast) Send(log event.Log) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subs {
		sub.Send(log)
	}
}

// Close shuts down all registered subscribers.
func (b *Broadcast) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, sub := range b.subs {
		sub.Close()
	}
	b.subs = nil
}

// Len returns the number of registered subscribers.
func (b *Broadcast) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
