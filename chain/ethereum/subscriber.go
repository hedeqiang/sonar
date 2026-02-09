package ethereum

import (
	"encoding/json"
	"sync"

	"github.com/hedeqiang/sonar/event"
)

// Subscription wraps a WebSocket subscription for Ethereum logs.
type Subscription struct {
	chainID string
	logs    chan event.Log
	errs    chan error
	unsub   func()
	done    chan struct{}
	once    sync.Once
}

func newSubscription(chainID string, raw <-chan []byte, unsub func()) *Subscription {
	s := &Subscription{
		chainID: chainID,
		logs:    make(chan event.Log, 64),
		errs:    make(chan error, 1),
		unsub:   unsub,
		done:    make(chan struct{}),
	}
	go s.consume(raw)
	return s
}

// Logs returns the channel of incoming event logs.
func (s *Subscription) Logs() <-chan event.Log {
	return s.logs
}

// Err returns the error channel.
func (s *Subscription) Err() <-chan error {
	return s.errs
}

// Unsubscribe terminates the subscription.
func (s *Subscription) Unsubscribe() {
	s.once.Do(func() {
		close(s.done)
		if s.unsub != nil {
			s.unsub()
		}
	})
}

func (s *Subscription) consume(raw <-chan []byte) {
	defer close(s.logs)
	defer close(s.errs)

	for {
		select {
		case <-s.done:
			return
		case msg, ok := <-raw:
			if !ok {
				return
			}

			var notification struct {
				Result rpcLog `json:"result"`
			}
			if err := json.Unmarshal(msg, &notification); err != nil {
				// Try direct log format
				var rl rpcLog
				if err2 := json.Unmarshal(msg, &rl); err2 != nil {
					select {
					case s.errs <- err:
					default:
					}
					continue
				}
				notification.Result = rl
			}

			log, err := notification.Result.toEventLog(s.chainID)
			if err != nil {
				select {
				case s.errs <- err:
				default:
				}
				continue
			}

			select {
			case s.logs <- log:
			case <-s.done:
				return
			}
		}
	}
}
