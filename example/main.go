// Package main demonstrates how to use the Sonar SDK.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hedeqiang/sonar"
	"github.com/hedeqiang/sonar/chain/ethereum"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	mw "github.com/hedeqiang/sonar/middleware"
	"github.com/hedeqiang/sonar/retry"
)

func main() {
	// 1. Create Sonar instance with options
	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./sonar_progress.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(3*time.Second),
		sonar.WithBatchSize(500),
		sonar.WithConfirmations(2),
	)

	// 2. Register chains
	eth := ethereum.New(os.Getenv("ETHEREUM_RPC_URL"))
	if err := s.AddChain(eth); err != nil {
		log.Fatal(err)
	}

	// 3. Add middleware
	s.Use(mw.NewLogger(nil))
	s.Use(mw.NewMetrics())

	// 4. Build filter query — monitor USDT Transfer events
	usdtAddr := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(
		filter.WithAddresses(usdtAddr),
	)

	// 5. Start watching
	err := s.Watch("ethereum", q, func(log event.Log) {
		fmt.Println(log)

		// fmt.Printf("📡 Event: block=%d tx=%s addr=%s topics=%d\n",
		// 	log.BlockNumber,
		// 	log.TxHash.Hex(),
		// 	log.Address.Hex(),
		// 	len(log.Topics),
		// )
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening... Press Ctrl+C to stop.")

	// 6. Graceful shutdown on signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	fmt.Println("Done.")
}
