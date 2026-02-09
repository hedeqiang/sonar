# Sonar

> A deep probe for every on-chain event signal.

Sonar is a Go SDK for monitoring event logs across multiple EVM-compatible blockchains. It provides a clean, interface-driven architecture for filtering, decoding, and subscribing to smart contract events with built-in progress tracking, retry strategies, and middleware support.

[简体中文](README.zh-CN.md)

## Features

- **Multi-chain** — Ethereum, BSC, Polygon, Arbitrum out of the box; add any EVM chain by implementing one interface
- **Polling & Streaming** — Block-range polling via HTTP RPC, real-time streaming via WebSocket (auto-connect on first use)
- **Historical Replay** — Backfill past events over a specific block range
- **ABI Decoding** — Register via Solidity signature string or standard JSON ABI; Keccak-256 event hashing; full indexed/non-indexed parameter decoding
- **Flexible Filtering** — Filter by address, topic, block range, or compose filters with AND/OR logic
- **Progress Tracking** — Resume scanning from the last processed block (in-memory or file-based)
- **Middleware Pipeline** — Plug in logging, metrics, rate limiting, or custom middleware
- **Retry & Circuit Breaker** — Exponential backoff and circuit breaker for RPC resilience
- **Event Distribution** — Deliver events via channels, callbacks, or broadcast to multiple subscribers

## Installation

```bash
go get github.com/hedeqiang/sonar
```

Requires **Go 1.21+**.

## Quick Start

```go
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
    // 1. Create a Sonar instance
    s := sonar.New(
        sonar.WithCursor(cursor.NewFile("./progress.json")),
        sonar.WithRetry(retry.Exponential(3)),
        sonar.WithPollInterval(5 * time.Second),
        sonar.WithBatchSize(5),
        sonar.WithConfirmations(2),
    )

    // 2. Register a chain (HTTP or WebSocket)
    eth := ethereum.New(os.Getenv("ETH_RPC_URL"))
    if err := s.AddChain(eth); err != nil {
        log.Fatal(err)
    }

    // 3. Add middleware
    s.Use(mw.NewLogger(nil))

    // 4. Register event ABI for decoding
    s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")
    s.RegisterEvent("Approval(address indexed owner, address indexed spender, uint256 value)")

    // 5. Build a filter query
    usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
    q := filter.NewQuery(filter.WithAddresses(usdt))

    // 6. Watch with ABI decoding
    err := s.WatchDecoded("ethereum", q, func(ev *decoder.DecodedEvent) {
        from, _ := ev.Indexed["from"].(event.Address)
        to, _ := ev.Indexed["to"].(event.Address)
        value, _ := ev.Params["value"].(*big.Int)

        fmt.Printf("[%s] %s -> %s : %s (block %d)\n",
            ev.Name, from.Hex(), to.Hex(), value.String(), ev.Raw.BlockNumber)
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Sonar is listening...")

    // 7. Graceful shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    s.Shutdown(ctx)
}
```

## Architecture

```
sonar/
├── sonar.go                 # SDK entry point
├── config.go                # Global configuration
├── option.go                # Functional options
├── errors.go                # Sentinel errors
│
├── event/                   # Core data structures
│   ├── log.go               # Log, Address, Hash types
│   ├── batch.go             # Batch container
│   └── convert.go           # Hex ↔ Address/Hash helpers
│
├── chain/                   # Multi-chain abstraction
│   ├── chain.go             # Chain + Subscription interfaces
│   ├── registry.go          # Chain registry
│   ├── ethereum/            # Ethereum implementation
│   ├── bsc/                 # BSC (reuses Ethereum)
│   ├── polygon/             # Polygon (reuses Ethereum)
│   └── arbitrum/            # Arbitrum (reuses Ethereum)
│
├── watcher/                 # Event monitoring
│   ├── watcher.go           # Watcher interface
│   ├── poller.go            # Block-range polling
│   ├── streamer.go          # WebSocket streaming
│   └── replay.go            # Historical replay
│
├── filter/                  # Event filtering
│   ├── filter.go            # Filter interface + Query builder
│   ├── address.go           # Address filter
│   ├── topic.go             # Topic filter
│   ├── block_range.go       # Block range filter
│   └── composite.go         # AND/OR composition
│
├── decoder/                 # ABI decoding
│   ├── decoder.go           # Decoder interface
│   ├── abi.go               # ABI decoder (string + JSON ABI)
│   ├── schema.go            # Event schema registry
│   └── raw.go               # Raw pass-through decoder
│
├── subscriber/              # Event distribution
│   ├── subscriber.go        # Subscriber interface
│   ├── channel.go           # Go channel delivery
│   ├── callback.go          # Callback delivery
│   └── broadcast.go         # One-to-many broadcast
│
├── middleware/               # Processing pipeline
│   ├── middleware.go         # Middleware interface + Chain()
│   ├── logger.go            # Logging middleware
│   ├── metrics.go           # Metrics middleware
│   └── ratelimit.go         # Rate limiting middleware
│
├── cursor/                  # Progress tracking
│   ├── cursor.go            # Cursor interface
│   ├── memory.go            # In-memory (dev/test)
│   └── file.go              # JSON file persistence
│
├── retry/                   # Resilience
│   ├── strategy.go          # Strategy interface + Do()
│   ├── backoff.go           # Exponential backoff
│   └── circuit.go           # Circuit breaker
│
├── transport/               # RPC transport
│   ├── transport.go         # Transport interface
│   ├── http.go              # HTTP JSON-RPC
│   └── websocket.go         # WebSocket JSON-RPC (lazy connect)
│
└── internal/                # Internal utilities
    ├── hex/                 # Hex encoding
    ├── abi/                 # Keccak-256 hashing + signature/JSON parser
    └── syncutil/            # Concurrency helpers
```

## Core Interfaces

### Chain

```go
type Chain interface {
    ID() string
    LatestBlock(ctx context.Context) (uint64, error)
    FetchLogs(ctx context.Context, query filter.Query) ([]event.Log, error)
    Subscribe(ctx context.Context, query filter.Query) (Subscription, error)
}
```

### Watcher

```go
type Watcher interface {
    Watch() error
    Stop() error
    OnEvent(fn func(event.Log))
    OnError(fn func(error))
}
```

### Filter

```go
type Filter interface {
    Match(log event.Log) bool
}
```

### Decoder

```go
type Decoder interface {
    Decode(log event.Log) (*DecodedEvent, error)
    Register(eventSignature string) error
}
```

### Cursor

```go
type Cursor interface {
    Load(chainID string) (uint64, error)
    Save(chainID string, block uint64) error
}
```

### Middleware

```go
type Middleware interface {
    Wrap(next Handler) Handler
}
```

## Usage Guide

### Multi-Chain Monitoring

```go
s := sonar.New()

s.AddChain(ethereum.New("https://eth-mainnet.alchemyapi.io/v2/KEY"))
s.AddChain(bsc.New("https://bsc-dataseed.binance.org"))
s.AddChain(polygon.New("https://polygon-rpc.com"))
s.AddChain(arbitrum.New("https://arb1.arbitrum.io/rpc"))

// Watch all chains with one call
s.WatchAll(query, func(log event.Log) {
    fmt.Printf("[%s] block=%d\n", log.Chain, log.BlockNumber)
})
```

### ABI Decoding

Three ways to register event ABIs:

```go
// Method 1: Solidity signature string
s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")

// Method 2: Standard JSON ABI (full contract ABI — non-event entries are skipped)
s.RegisterEventJSON([]byte(`[
  {
    "type": "event",
    "name": "Transfer",
    "inputs": [
      {"indexed": true,  "name": "from",  "type": "address"},
      {"indexed": true,  "name": "to",    "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ]
  },
  {"type": "function", "name": "balanceOf", "inputs": []}
]`))

// Method 3: Use ABIDecoder directly for advanced control
dec := decoder.NewABIDecoder()
dec.RegisterJSON(contractABI)         // full contract ABI
dec.RegisterJSONEvent(singleEvent)    // single JSON event entry
dec.Register("Transfer(address,address,uint256)")  // signature string
```

Watch with automatic decoding — only successfully decoded events reach the handler:

```go
s.WatchDecoded("ethereum", query, func(ev *decoder.DecodedEvent) {
    fmt.Printf("Event: %s\n", ev.Name)

    // Indexed parameters (from topics)
    from := ev.Indexed["from"].(event.Address)

    // All parameters (indexed + non-indexed)
    value := ev.Params["value"].(*big.Int)
})
```

Or decode manually in a raw Watch handler:

```go
dec := decoder.NewABIDecoder()
dec.Register("Transfer(address indexed from, address indexed to, uint256 value)")

s.Watch("ethereum", query, func(log event.Log) {
    decoded, err := dec.Decode(log)
    if err != nil {
        return // not a registered event
    }
    fmt.Printf("[%s] from=%s\n", decoded.Name, decoded.Indexed["from"])
})
```

### Filtering

```go
// By contract address
q := filter.NewQuery(
    filter.WithAddresses(
        event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
    ),
)

// By block range
q := filter.NewQuery(
    filter.WithBlockRange(18000000, 18001000),
)

// Composable filters for post-fetch filtering
f := filter.AllOf(
    filter.NewAddressFilter(addr1, addr2),
    filter.NewTopicFilter(0, transferSigHash),
)

if f.Match(log) {
    // process
}
```

### Historical Replay

```go
q := filter.NewQuery(
    filter.WithAddresses(contractAddr),
    filter.WithBlockRange(17000000, 18000000),
)

r := watcher.NewReplay(eth, q, 2000)
r.OnEvent(func(log event.Log) {
    fmt.Printf("Historical event at block %d\n", log.BlockNumber)
})
r.Watch() // blocks until the range is fully scanned
```

### Progress Tracking

```go
// In-memory (lost on restart)
s := sonar.New(sonar.WithCursor(cursor.NewMemory()))

// File-based (persists to disk as JSON)
s := sonar.New(sonar.WithCursor(cursor.NewFile("./progress.json")))

// Custom: implement the cursor.Cursor interface (e.g., Redis, database)
```

### Middleware

```go
// Built-in middleware
s.Use(middleware.NewLogger(nil))       // log every event
s.Use(middleware.NewMetrics())         // count processed/dropped events
s.Use(middleware.NewRateLimit(100*time.Millisecond)) // throttle

// Custom middleware
type MyMiddleware struct{}

func (m *MyMiddleware) Wrap(next middleware.Handler) middleware.Handler {
    return func(log event.Log) *event.Log {
        // pre-processing
        result := next(log)
        // post-processing
        return result
    }
}
```

### Event Distribution

```go
// Channel-based
ch := subscriber.NewChannel(256)
go func() {
    for log := range ch.Logs() {
        process(log)
    }
}()

// Callback-based
cb := subscriber.NewCallback(func(log event.Log) {
    process(log)
})

// Broadcast to multiple subscribers
b := subscriber.NewBroadcast()
b.Add(ch)
b.Add(cb)
b.Send(log) // delivered to both
```

## Adding a New Chain

Implement the `chain.Chain` interface and register it:

```go
package avalanche

import "github.com/hedeqiang/sonar/chain/ethereum"

func New(rpcURL string) *ethereum.Client {
    return ethereum.NewWithID("avalanche", rpcURL)
}
```

For EVM-compatible chains, reuse `ethereum.NewWithID` — zero core code changes required.

For non-EVM chains, implement the full `chain.Chain` interface directly.

## Configuration Options

| Option | Description | Default |
|---|---|---|
| `WithCursor(c)` | Set progress cursor | In-memory |
| `WithDecoder(d)` | Set event decoder | None (auto-created on `RegisterEvent`) |
| `WithRetry(s)` | Set retry strategy | None |
| `WithPollInterval(d)` | Polling interval | 2s |
| `WithBatchSize(n)` | Blocks per poll cycle | 1000 |
| `WithConfirmations(n)` | Confirmation blocks | 0 |
| `WithMiddleware(m...)` | Add middleware | None |
| `WithLogLevel(l)` | Log verbosity | "info" |

## License

MIT
