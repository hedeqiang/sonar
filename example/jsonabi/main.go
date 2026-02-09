// Example: jsonabi — register events from a standard JSON ABI.
//
// Usage:
//
//	ETH_RPC_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY go run ./example/jsonabi
package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hedeqiang/sonar"
	"github.com/hedeqiang/sonar/chain/ethereum"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/decoder"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	"github.com/hedeqiang/sonar/retry"
)

// ERC-20 JSON ABI (partial — only event entries are used, others are skipped)
var erc20ABI = []byte(`[
  {
    "type": "event",
    "name": "Transfer",
    "inputs": [
      {"indexed": true,  "name": "from",  "type": "address"},
      {"indexed": true,  "name": "to",    "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ]
  },
  {
    "type": "event",
    "name": "Approval",
    "inputs": [
      {"indexed": true,  "name": "owner",   "type": "address"},
      {"indexed": true,  "name": "spender", "type": "address"},
      {"indexed": false, "name": "value",   "type": "uint256"}
    ]
  },
  {
    "type": "function",
    "name": "balanceOf",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "uint256"}]
  }
]`)

func main() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./progress_jsonabi.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(5),
		sonar.WithConfirmations(2),
	)

	if err := s.AddChain(ethereum.New(rpcURL)); err != nil {
		log.Fatal(err)
	}

	// Register events from JSON ABI — function entries are automatically skipped
	if err := s.RegisterEventJSON(erc20ABI); err != nil {
		log.Fatalf("RegisterEventJSON: %v", err)
	}

	usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(filter.WithAddresses(usdt))

	err := s.WatchDecoded("ethereum", q, func(ev *decoder.DecodedEvent) {
		value, _ := ev.Params["value"].(*big.Int)
		fmt.Printf("[%s] block=%d value=%s tx=%s\n",
			ev.Name, ev.Raw.BlockNumber, value.String(), ev.Raw.TxHash.Hex())
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening (JSON ABI mode)... Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}
