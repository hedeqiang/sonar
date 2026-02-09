// Example: voting — monitor VoteAgainstRecorded events on Sepolia testnet.
//
// Demonstrates String(), JSON(), and Bind() methods on DecodedEvent.
//
// Contract: 0xbF09f23a3029F3b2AF7230767c4c830e1d7Ac2d5
// https://sepolia.etherscan.io/address/0xbF09f23a3029F3b2AF7230767c4c830e1d7Ac2d5
//
// Usage:
//
//	SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY go run ./example/voting
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// VoteEvent is a custom struct for type-safe event binding.
type VoteEvent struct {
	Caller    event.Address `abi:"caller"`
	CallerHex string        `abi:"caller"` // also works: auto-converts Address to string
}

// Voting contract JSON ABI (only event entries are used)
var votingABI = []byte(`[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "caller",
        "type": "address"
      }
    ],
    "name": "VoteAgainstRecorded",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "voteAgainst",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`)

func main() {
	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	if rpcURL == "" {
		log.Fatal("SEPOLIA_RPC_URL environment variable is required")
	}

	// Pre-seed cursor to start scanning from block 10221826
	cur := cursor.NewFile("./progress_voting.json")
	if saved, _ := cur.Load("sepolia"); saved == 0 {
		cur.Save("sepolia", 10221825) // poller will start from 10221825+1 = 10221826
	}

	s := sonar.New(
		sonar.WithCursor(cur),
		sonar.WithRetry(retry.Exponential(3)),
		sonar.WithPollInterval(5*time.Second),
		sonar.WithBatchSize(1000),
		sonar.WithConfirmations(1),
	)

	// Register Sepolia as an EVM chain
	sepolia := ethereum.NewWithID("sepolia", rpcURL)
	if err := s.AddChain(sepolia); err != nil {
		log.Fatal(err)
	}

	s.Use(mw.NewLogger(nil))

	// Register event ABI from JSON
	if err := s.RegisterEventJSON(votingABI); err != nil {
		log.Fatalf("RegisterEventJSON: %v", err)
	}

	// Monitor the Voting contract
	contractAddr := event.MustHexToAddress("0xbF09f23a3029F3b2AF7230767c4c830e1d7Ac2d5")
	q := filter.NewQuery(filter.WithAddresses(contractAddr))

	err := s.WatchDecoded("sepolia", q, func(ev *decoder.DecodedEvent) {
		// --- Method 1: String() ---
		fmt.Println("String():", ev.String())

		// --- Method 2: JSON() / MarshalJSON() ---
		jsonBytes, _ := json.MarshalIndent(ev, "", "  ")
		fmt.Println("JSON():", string(jsonBytes))

		// --- Method 3: Bind() to custom struct ---
		var vote VoteEvent
		if err := ev.Bind(&vote); err != nil {
			log.Printf("Bind error: %v", err)
			return
		}
		fmt.Printf("Bind(): caller=%s callerHex=%s\n\n", vote.Caller.Hex(), vote.CallerHex)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sonar is listening for VoteAgainstRecorded events on Sepolia...")
	fmt.Println("Contract: 0xbF09f23a3029F3b2AF7230767c4c830e1d7Ac2d5")
	fmt.Println("Starting from block: 10221826")
	fmt.Println("Press Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}
