// Example: multichain — monitor events across multiple EVM chains simultaneously.
//
// Usage:
//
//	ETH_RPC_URL=https://... BSC_RPC_URL=https://... POLYGON_RPC_URL=https://... go run ./example/multichain
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
	"github.com/hedeqiang/sonar/chain/bsc"
	"github.com/hedeqiang/sonar/chain/ethereum"
	"github.com/hedeqiang/sonar/chain/polygon"
	"github.com/hedeqiang/sonar/cursor"
	"github.com/hedeqiang/sonar/event"
	"github.com/hedeqiang/sonar/filter"
	mw "github.com/hedeqiang/sonar/middleware"
	"github.com/hedeqiang/sonar/retry"
)

func main() {
	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./progress_multichain.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(10),
		sonar.WithConfirmations(2),
	)

	s.Use(mw.NewLogger(nil))

	// Register multiple chains — only those with a configured RPC URL
	chains := map[string]struct {
		envKey  string
		addFunc func(string) error
	}{
		"ethereum": {"ETH_RPC_URL", func(url string) error { return s.AddChain(ethereum.New(url)) }},
		"bsc":      {"BSC_RPC_URL", func(url string) error { return s.AddChain(bsc.New(url)) }},
		"polygon":  {"POLYGON_RPC_URL", func(url string) error { return s.AddChain(polygon.New(url)) }},
	}

	registered := 0
	for name, cfg := range chains {
		url := os.Getenv(cfg.envKey)
		if url == "" {
			fmt.Printf("Skipping %s (set %s to enable)\n", name, cfg.envKey)
			continue
		}
		if err := cfg.addFunc(url); err != nil {
			log.Fatalf("Failed to add %s: %v", name, err)
		}
		fmt.Printf("Registered chain: %s\n", name)
		registered++
	}

	if registered == 0 {
		log.Fatal("No chains configured. Set at least one of: ETH_RPC_URL, BSC_RPC_URL, POLYGON_RPC_URL")
	}

	// USDT addresses per chain (mainnet)
	usdtAddresses := map[string]string{
		"ethereum": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		"bsc":      "0x55d398326f99059fF775485246999027B3197955",
		"polygon":  "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
	}

	// Watch each registered chain
	for chainID, addr := range usdtAddresses {
		if os.Getenv(chains[chainID].envKey) == "" {
			continue
		}

		contractAddr := event.MustHexToAddress(addr)
		q := filter.NewQuery(filter.WithAddresses(contractAddr))

		err := s.Watch(chainID, q, func(l event.Log) {
			fmt.Printf("[%s] block=%d tx=%s\n",
				l.Chain, l.BlockNumber, l.TxHash.Hex())
		})
		if err != nil {
			log.Fatalf("Watch %s: %v", chainID, err)
		}
	}

	fmt.Printf("\nSonar is listening on %d chain(s)... Press Ctrl+C to stop.\n", registered)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}
