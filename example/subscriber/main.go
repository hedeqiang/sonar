// Example: subscriber — demonstrate channel, callback, and broadcast event distribution.
//
// Usage:
//
//	ETH_RPC_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY go run ./example/subscriber
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hedeqiang/sonar"
	"github.com/hedeqiang/sonar/chain/ethereum"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	"github.com/hedeqiang/sonar/retry"
	"github.com/hedeqiang/sonar/subscriber"
)

func main() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./progress_subscriber.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(5),
		sonar.WithConfirmations(2),
	)

	if err := s.AddChain(ethereum.New(rpcURL)); err != nil {
		log.Fatal(err)
	}

	// --- Subscriber 1: Channel-based ---
	ch := subscriber.NewChannel(256)
	go func() {
		for l := range ch.Logs() {
			fmt.Printf("[Channel]  block=%d tx=%s\n", l.BlockNumber, l.TxHash.Hex())
		}
	}()

	// --- Subscriber 2: Callback-based ---
	var cbCount atomic.Int64
	cb := subscriber.NewCallback(func(l event.Log) {
		n := cbCount.Add(1)
		fmt.Printf("[Callback] #%d block=%d\n", n, l.BlockNumber)
	})

	// --- Broadcast: delivers to both subscribers ---
	bc := subscriber.NewBroadcast()
	bc.Add(ch)
	bc.Add(cb)

	// Watch and distribute via broadcast
	usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(filter.WithAddresses(usdt))

	err := s.Watch("ethereum", q, func(l event.Log) {
		bc.Send(l)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening (broadcast mode)... Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	bc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)

	fmt.Printf("Total callback invocations: %d\n", cbCount.Load())
}
