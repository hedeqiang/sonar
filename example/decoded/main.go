// Example: decoded — watch events with automatic ABI decoding.
//
// Usage:
//
//	ETH_RPC_URL=https://eth-mainnet.alchemyapi.io/v2/YOUR_KEY go run ./example/decoded
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
	mw "github.com/hedeqiang/sonar/middleware"
	"github.com/hedeqiang/sonar/retry"
)

func main() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL environment variable is required")
	}

	s := sonar.New(
		sonar.WithCursor(cursor.NewFile("./progress_decoded.json")),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(5),
		sonar.WithConfirmations(2),
	)

	if err := s.AddChain(ethereum.New(rpcURL)); err != nil {
		log.Fatal(err)
	}

	s.Use(mw.NewLogger(nil))

	// Register event ABIs for decoding
	s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")
	s.RegisterEvent("Approval(address indexed owner, address indexed spender, uint256 value)")

	// Monitor USDT
	usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	q := filter.NewQuery(filter.WithAddresses(usdt))

	// WatchDecoded — only successfully decoded events reach the handler
	err := s.WatchDecoded("ethereum", q, func(ev *decoder.DecodedEvent) {
		switch ev.Name {
		case "Transfer":
			from, _ := ev.Indexed["from"].(event.Address)
			to, _ := ev.Indexed["to"].(event.Address)
			value, _ := ev.Params["value"].(*big.Int)

			// USDT has 6 decimals
			amt := formatUnits(value, 6)
			fmt.Printf("[Transfer] %s -> %s : %s USDT (block %d)\n",
				from.Hex(), to.Hex(), amt, ev.Raw.BlockNumber)

		case "Approval":
			owner, _ := ev.Indexed["owner"].(event.Address)
			spender, _ := ev.Indexed["spender"].(event.Address)
			value, _ := ev.Params["value"].(*big.Int)

			fmt.Printf("[Approval] owner=%s spender=%s value=%s (block %d)\n",
				owner.Hex(), spender.Hex(), value.String(), ev.Raw.BlockNumber)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening with ABI decoding... Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}

// formatUnits formats a big.Int with the given decimal places.
func formatUnits(value *big.Int, decimals int) string {
	if value == nil {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(value, divisor)
	frac := new(big.Int).Mod(value, divisor)
	return fmt.Sprintf("%s.%0*s", whole.String(), decimals, frac.String())
}
