// Example: basic — poll raw event logs from Ethereum.
//
// Usage:
//
//	ETH_RPC_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY go run ./example/basic
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
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	// 1. Create Sonar instance
	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./progress_basic.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(5),
		sonar.WithConfirmations(2),
	)

	// 2. Register Ethereum chain
	if err := s.AddChain(ethereum.New(rpcURL)); err != nil {
		log.Fatal(err)
	}

	// 3. Add logging middleware
	s.Use(mw.NewLogger(nil))

	// 4. Monitor USDT contract
	usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(filter.WithAddresses(usdt))

	// 5. Watch raw events
	err := s.Watch("ethereum", q, func(l event.Log) {
		fmt.Printf("[block %d] tx=%s addr=%s topics=%d data=%d bytes\n",
			l.BlockNumber,
			l.TxHash.Hex(),
			l.Address.Hex(),
			len(l.Topics),
			len(l.Data),
		)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening for raw events... Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}
