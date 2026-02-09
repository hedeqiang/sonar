// Example: replay — backfill historical events over a specific block range.
//
// Usage:
//
//	ETH_RPC_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY go run ./example/replay
package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/hedeqiang/sonar/chain/ethereum"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	"github.com/hedeqiang/sonar/watcher"
)

func main() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	eth := ethereum.New(rpcURL)

	// Scan USDT events in a 100-block range
	// Adjust the block numbers to a recent range on mainnet
	usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(
		filter.WithAddresses(usdt),
		filter.WithBlockRange(21000000, 21000100),
	)

	// Create a replay watcher with batch size of 50 blocks
	r := watcher.NewReplay(eth, q, 50)

	var count atomic.Int64

	r.OnEvent(func(l event.Log) {
		n := count.Add(1)
		fmt.Printf("#%d [block %d] tx=%s topics=%d\n",
			n, l.BlockNumber, l.TxHash.Hex(), len(l.Topics))
	})

	r.OnError(func(err error) {
		log.Printf("Error: %v", err)
	})

	fmt.Println("Replaying USDT events from block 21000000 to 21000100...")

	if err := r.Watch(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done. Total events: %d\n", count.Load())
}
