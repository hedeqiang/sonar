# Sonar

> 深入链上，探测每一个事件信号。

Sonar 是一个 Go 语言 SDK，用于监听多条 EVM 兼容区块链上的事件日志（Event Log）。它提供简洁的接口驱动架构，支持事件过滤、ABI 解码、实时订阅，并内置进度追踪、重试策略和中间件管道。

[English](README.md)

## 特性

- **多链支持** — 内置 Ethereum、BSC、Polygon、Arbitrum；实现一个接口即可接入任意 EVM 链
- **轮询 + 流式** — HTTP RPC 区块范围轮询，WebSocket 实时流式推送（首次使用时自动连接）
- **历史重放** — 指定区块范围批量回溯历史事件
- **ABI 解码** — 支持 Solidity 事件签名字符串或标准 JSON ABI 注册；Keccak-256 事件哈希；完整的 indexed/non-indexed 参数解码
- **灵活过滤** — 按地址、Topic、区块范围过滤，支持 AND/OR 组合
- **进度追踪** — 断点续扫，支持内存和文件两种游标实现
- **中间件管道** — 日志、指标采集、限流等中间件可插拔组合
- **重试与熔断** — 指数退避重试策略 + 熔断器，保障 RPC 调用韧性
- **事件分发** — Channel、回调函数、广播三种分发模式

## 安装

```bash
go get github.com/hedeqiang/sonar
```

要求 **Go 1.21+**。

## 快速开始

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
    // 1. 创建 Sonar 实例
    s := sonar.New(
        sonar.WithCursor(cursor.NewFile("./progress.json")),
        sonar.WithRetry(retry.Exponential(3)),
        sonar.WithPollInterval(5 * time.Second),
        sonar.WithBatchSize(5),
        sonar.WithConfirmations(2),
    )

    // 2. 注册链（HTTP 或 WebSocket）
    eth := ethereum.New(os.Getenv("ETH_RPC_URL"))
    if err := s.AddChain(eth); err != nil {
        log.Fatal(err)
    }

    // 3. 添加中间件
    s.Use(mw.NewLogger(nil))

    // 4. 注册事件 ABI 用于解码
    s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")
    s.RegisterEvent("Approval(address indexed owner, address indexed spender, uint256 value)")

    // 5. 构建过滤条件
    usdt := event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
    q := filter.NewQuery(filter.WithAddresses(usdt))

    // 6. 使用 ABI 解码监听
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

    fmt.Println("Sonar 正在监听...")

    // 7. 优雅退出
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    s.Shutdown(ctx)
}
```

## 架构总览

```
sonar/
├── sonar.go                 # SDK 入口，暴露顶层 API
├── config.go                # 全局配置
├── option.go                # Functional Options 模式
├── errors.go                # 统一错误定义
│
├── event/                   # 核心数据结构
│   ├── log.go               # Log, Address, Hash 类型定义
│   ├── batch.go             # 批量事件容器
│   └── convert.go           # Hex 字符串 ↔ Address/Hash 转换
│
├── chain/                   # 多链抽象层
│   ├── chain.go             # Chain + Subscription 接口
│   ├── registry.go          # 链注册表
│   ├── ethereum/            # 以太坊实现
│   ├── bsc/                 # BSC（复用以太坊实现）
│   ├── polygon/             # Polygon（复用以太坊实现）
│   └── arbitrum/            # Arbitrum（复用以太坊实现）
│
├── watcher/                 # 事件监听
│   ├── watcher.go           # Watcher 接口
│   ├── poller.go            # 区块轮询模式
│   ├── streamer.go          # WebSocket 流式模式
│   └── replay.go            # 历史事件重放
│
├── filter/                  # 事件过滤
│   ├── filter.go            # Filter 接口 + Query 构建器
│   ├── address.go           # 合约地址过滤
│   ├── topic.go             # Topic 过滤
│   ├── block_range.go       # 区块范围过滤
│   └── composite.go         # AND/OR 组合过滤
│
├── decoder/                 # ABI 解码
│   ├── decoder.go           # Decoder 接口
│   ├── abi.go               # ABI 解码实现（签名字符串 + JSON ABI）
│   ├── schema.go            # 事件 Schema 注册表
│   └── raw.go               # 原始日志透传
│
├── subscriber/              # 事件分发
│   ├── subscriber.go        # Subscriber 接口
│   ├── channel.go           # Go Channel 分发
│   ├── callback.go          # 回调函数分发
│   └── broadcast.go         # 一对多广播
│
├── middleware/               # 中间件
│   ├── middleware.go         # Middleware 接口 + Chain()
│   ├── logger.go            # 日志中间件
│   ├── metrics.go           # 指标采集中间件
│   └── ratelimit.go         # 限流中间件
│
├── cursor/                  # 进度追踪
│   ├── cursor.go            # Cursor 接口
│   ├── memory.go            # 内存实现（开发/测试用）
│   └── file.go              # JSON 文件持久化
│
├── retry/                   # 重试与容错
│   ├── strategy.go          # Strategy 接口 + Do()
│   ├── backoff.go           # 指数退避
│   └── circuit.go           # 熔断器
│
├── transport/               # RPC 传输层
│   ├── transport.go         # Transport 接口
│   ├── http.go              # HTTP JSON-RPC
│   └── websocket.go         # WebSocket JSON-RPC（惰性连接）
│
└── internal/                # 内部工具（不对外暴露）
    ├── hex/                 # 十六进制编解码
    ├── abi/                 # Keccak-256 哈希 + 签名/JSON 解析器
    └── syncutil/            # 并发工具
```

## 核心接口

### Chain — 链抽象

```go
type Chain interface {
    ID() string
    LatestBlock(ctx context.Context) (uint64, error)
    FetchLogs(ctx context.Context, query filter.Query) ([]event.Log, error)
    Subscribe(ctx context.Context, query filter.Query) (Subscription, error)
}
```

### Watcher — 事件监听器

```go
type Watcher interface {
    Watch() error
    Stop() error
    OnEvent(fn func(event.Log))
    OnError(fn func(error))
}
```

### Filter — 过滤器

```go
type Filter interface {
    Match(log event.Log) bool
}
```

### Decoder — 解码器

```go
type Decoder interface {
    Decode(log event.Log) (*DecodedEvent, error)
    Register(eventSignature string) error
}
```

### Cursor — 进度游标

```go
type Cursor interface {
    Load(chainID string) (uint64, error)
    Save(chainID string, block uint64) error
}
```

### Middleware — 中间件

```go
type Middleware interface {
    Wrap(next Handler) Handler
}
```

## 使用指南

### 多链监听

```go
s := sonar.New()

s.AddChain(ethereum.New("https://eth-mainnet.alchemyapi.io/v2/KEY"))
s.AddChain(bsc.New("https://bsc-dataseed.binance.org"))
s.AddChain(polygon.New("https://polygon-rpc.com"))
s.AddChain(arbitrum.New("https://arb1.arbitrum.io/rpc"))

// 一行代码监听所有链
s.WatchAll(query, func(log event.Log) {
    fmt.Printf("[%s] block=%d\n", log.Chain, log.BlockNumber)
})
```

### ABI 解码

三种注册事件 ABI 的方式：

```go
// 方式一：Solidity 事件签名字符串
s.RegisterEvent("Transfer(address indexed from, address indexed to, uint256 value)")

// 方式二：标准 JSON ABI（完整合约 ABI — 非事件类型的条目会被自动跳过）
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

// 方式三：直接使用 ABIDecoder 进行更精细的控制
dec := decoder.NewABIDecoder()
dec.RegisterJSON(contractABI)         // 完整合约 ABI
dec.RegisterJSONEvent(singleEvent)    // 单个 JSON 事件条目
dec.Register("Transfer(address,address,uint256)")  // 签名字符串
```

使用自动解码监听 — 只有成功解码的事件才会到达处理函数：

```go
s.WatchDecoded("ethereum", query, func(ev *decoder.DecodedEvent) {
    fmt.Printf("事件: %s\n", ev.Name)

    // 索引参数（来自 topics）
    from := ev.Indexed["from"].(event.Address)

    // 所有参数（索引 + 非索引）
    value := ev.Params["value"].(*big.Int)
})
```

或者在原始 Watch 处理器中手动解码：

```go
dec := decoder.NewABIDecoder()
dec.Register("Transfer(address indexed from, address indexed to, uint256 value)")

s.Watch("ethereum", query, func(log event.Log) {
    decoded, err := dec.Decode(log)
    if err != nil {
        return // 非已注册事件
    }
    fmt.Printf("[%s] from=%s\n", decoded.Name, decoded.Indexed["from"])
})
```

### 解码结果输出

`DecodedEvent` 提供三种方式消费解码后的数据：

**String() — 人类可读格式：**

```go
fmt.Println(ev.String())
// Transfer(from=0xAb5801a7..., to=0x4E83362e..., value=1000000) chain=ethereum block=21000000 tx=0xabc...
```

**JSON() / MarshalJSON() — JSON 序列化：**

```go
jsonBytes, _ := json.MarshalIndent(ev, "", "  ")
fmt.Println(string(jsonBytes))
```

```json
{
  "event": "Transfer",
  "signature": "Transfer(address,address,uint256)",
  "chain": "ethereum",
  "blockNumber": 21000000,
  "txHash": "0xabc...",
  "address": "0xdac17f...",
  "params":  { "from": "0xAb5801a7...", "to": "0x4E83362e...", "value": "1000000" },
  "indexed": { "from": "0xAb5801a7...", "to": "0x4E83362e..." },
  "data":    { "value": "1000000" }
}
```

- `params` — 全部参数（indexed + non-indexed）
- `indexed` — 仅索引参数（来自 topics）
- `data` — 仅非索引参数（来自 log data）

Address/Hash 自动转为 hex 字符串，`*big.Int` 转十进制字符串，`[]byte` 转 `0x` 前缀 hex。

**Bind() — 绑定到自定义结构体：**

```go
type TransferEvent struct {
    From  event.Address `abi:"from"`
    To    event.Address `abi:"to"`
    Value *big.Int      `abi:"value"`
}

var evt TransferEvent
if err := ev.Bind(&evt); err != nil {
    log.Fatal(err)
}
fmt.Printf("%s -> %s : %s\n", evt.From.Hex(), evt.To.Hex(), evt.Value.String())
```

字段匹配规则：优先 `abi` tag，无 tag 时按字段名（不区分大小写）。支持类型：`event.Address`、`event.Hash`、`*big.Int`、`bool`、`string`、`uint64`、`int64`、`[]byte`。

### 事件过滤

```go
// 按合约地址过滤
q := filter.NewQuery(
    filter.WithAddresses(
        event.MustHexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
    ),
)

// 按区块范围过滤
q := filter.NewQuery(
    filter.WithBlockRange(18000000, 18001000),
)

// 组合过滤器（用于拉取后的二次过滤）
f := filter.AllOf(
    filter.NewAddressFilter(addr1, addr2),
    filter.NewTopicFilter(0, transferSigHash),
)

if f.Match(log) {
    // 处理匹配的日志
}
```

### 历史事件重放

```go
q := filter.NewQuery(
    filter.WithAddresses(contractAddr),
    filter.WithBlockRange(17000000, 18000000),
)

r := watcher.NewReplay(eth, q, 2000)
r.OnEvent(func(log event.Log) {
    fmt.Printf("历史事件: block=%d\n", log.BlockNumber)
})
r.Watch() // 阻塞直到整个区间扫描完成
```

### 进度追踪

```go
// 内存模式（重启后丢失）
s := sonar.New(sonar.WithCursor(cursor.NewMemory()))

// 文件模式（持久化为 JSON）
s := sonar.New(sonar.WithCursor(cursor.NewFile("./progress.json")))

// 自定义：实现 cursor.Cursor 接口（如 Redis、数据库等）
```

### 中间件

```go
// 内置中间件
s.Use(middleware.NewLogger(nil))                        // 日志记录
s.Use(middleware.NewMetrics())                           // 指标统计
s.Use(middleware.NewRateLimit(100 * time.Millisecond))   // 限流

// 自定义中间件
type MyMiddleware struct{}

func (m *MyMiddleware) Wrap(next middleware.Handler) middleware.Handler {
    return func(log event.Log) *event.Log {
        // 前置处理
        result := next(log)
        // 后置处理
        return result
    }
}
```

### 事件分发

```go
// Channel 模式
ch := subscriber.NewChannel(256)
go func() {
    for log := range ch.Logs() {
        process(log)
    }
}()

// 回调模式
cb := subscriber.NewCallback(func(log event.Log) {
    process(log)
})

// 广播模式（一对多）
b := subscriber.NewBroadcast()
b.Add(ch)
b.Add(cb)
b.Send(log) // 同时推送给所有订阅者
```

## 扩展新链

实现 `chain.Chain` 接口并注册即可：

```go
package avalanche

import "github.com/hedeqiang/sonar/chain/ethereum"

func New(rpcURL string) *ethereum.Client {
    return ethereum.NewWithID("avalanche", rpcURL)
}
```

对于 EVM 兼容链，直接复用 `ethereum.NewWithID` — **零修改核心代码**。

对于非 EVM 链，直接实现完整的 `chain.Chain` 接口即可。

## 配置选项

| 选项 | 说明 | 默认值 |
|---|---|---|
| `WithCursor(c)` | 设置进度游标 | 内存 |
| `WithDecoder(d)` | 设置事件解码器 | 无（调用 `RegisterEvent` 时自动创建） |
| `WithRetry(s)` | 设置重试策略 | 无 |
| `WithPollInterval(d)` | 轮询间隔 | 2 秒 |
| `WithBatchSize(n)` | 每次轮询的区块数 | 1000 |
| `WithConfirmations(n)` | 确认区块数 | 0 |
| `WithMiddleware(m...)` | 添加中间件 | 无 |
| `WithLogLevel(l)` | 日志级别 | "info" |

## License

MIT
