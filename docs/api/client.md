# Client API 参考

---

## 📌 版本信息

- **版本**：0.1.0-alpha
- **状态**：draft
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK

---

## 📖 概述

`Client` 是 SDK 的核心接口，负责与 WES 节点通信。它封装了 JSON-RPC/gRPC/WebSocket 调用、请求重试、错误处理等功能。

---

## 🔗 关联文档

- **底层 API**：[WES JSON-RPC API 参考](https://github.com/weisyn/weisyn/blob/main/docs/reference/api.md)
- **架构说明**：[SDK 架构设计](../architecture.md)

---

## 📦 导入

```go
import "github.com/weisyn/client-sdk-go/client"
```

---

## 🏗️ Client 接口

### Client Interface

```go
type Client interface {
    // Call 调用 JSON-RPC 方法
    Call(ctx context.Context, method string, params interface{}) (interface{}, error)
    
    // SendRawTransaction 发送已签名的原始交易
    SendRawTransaction(ctx context.Context, signedTxHex string) (*SendTxResult, error)
    
    // Subscribe 订阅事件（WebSocket 支持）
    Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error)
    
    // Close 关闭连接
    Close() error
}
```

---

## ⚙️ 配置

### Config

```go
type Config struct {
    Endpoint string        // 节点端点（如 "http://localhost:8545"）
    Protocol Protocol      // 协议：ProtocolHTTP / ProtocolGRPC / ProtocolWebSocket
    Timeout  time.Duration // 超时时间，默认 30 秒
    Debug    bool          // 调试模式，默认 false
    Retry    *RetryConfig  // 重试配置（可选）
}
```

### RetryConfig

```go
type RetryConfig struct {
    MaxRetries      int           // 最大重试次数，默认 3
    InitialDelay    time.Duration // 首次重试延迟，默认 500ms
    MaxDelay        time.Duration // 最大重试延迟，默认 10s
    BackoffMultiplier float64     // 退避乘数，默认 2
    Retryable       func(error) bool // 判断错误是否可重试的函数
    OnRetry         func(int, error) // 重试回调函数
}
```

---

## 🚀 使用示例

### 基本使用

```go
import (
    "context"
    "time"
    
    "github.com/weisyn/client-sdk-go/client"
)

// 创建客户端
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Timeout:  30 * time.Second,
}
c, err := client.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer c.Close()

// 调用 JSON-RPC 方法
ctx := context.Background()
blockNumber, err := c.Call(ctx, "wes_blockNumber", nil)

// 查询 UTXO
utxos, err := c.Call(ctx, "wes_getUTXO", []interface{}{addressBase58})
```

### 配置重试机制

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: &client.RetryConfig{
        MaxRetries:      5,
        InitialDelay:    500 * time.Millisecond,
        MaxDelay:        10 * time.Second,
        BackoffMultiplier: 2,
        Retryable: func(err error) bool {
            // 只重试网络错误或 5xx 错误
            if netErr, ok := err.(*client.NetworkError); ok {
                return true
            }
            return false
        },
        OnRetry: func(attempt int, err error) {
            log.Printf("重试第 %d 次: %v", attempt, err)
        },
    },
}
```

### gRPC 客户端

```go
cfg := &client.Config{
    Endpoint: "localhost:9090",
    Protocol: client.ProtocolGRPC,
}
c, err := client.NewClient(cfg)
```

### WebSocket 事件订阅

```go
cfg := &client.Config{
    Endpoint: "ws://localhost:8081",
    Protocol: client.ProtocolWebSocket,
}
wsClient, err := client.NewClient(cfg)

filter := &client.EventFilter{
    Topics: []string{"Transfer", "Mint"},
    From:   fromAddress,
    To:     toAddress,
}

events, err := wsClient.Subscribe(ctx, filter)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    log.Printf("收到事件: %s, 数据: %x", event.Topic, event.Data)
}
```

---

## 📚 常用 JSON-RPC 方法

### 查询方法

| 方法 | 说明 | 参数 | 返回 |
|------|------|------|------|
| `wes_blockNumber` | 获取当前区块高度 | `nil` | `number` |
| `wes_getUTXO` | 查询 UTXO | `[address]` | `{ utxos: [...] }` |
| `wes_getTransactionByHash` | 查询交易 | `[txHash]` | `{ hash, status, ... }` |
| `wes_getResource` | 查询资源 | `[resourceId]` | `{ type, size, ... }` |

### 交易方法

| 方法 | 说明 | 参数 | 返回 |
|------|------|------|------|
| `wes_buildTransaction` | 构建交易 | `[draft]` | `{ unsigned_tx, ... }` |
| `wes_computeSignatureHashFromDraft` | 计算签名哈希 | `[draft, inputIndex]` | `string` |
| `wes_finalizeTransactionFromDraft` | 完成交易 | `[draft, signatures, ...]` | `{ signed_tx }` |
| `wes_sendRawTransaction` | 发送交易 | `[signedTxHex]` | `{ tx_hash }` |

### 合约方法

| 方法 | 说明 | 参数 | 返回 |
|------|------|------|------|
| `wes_callContract` | 调用合约 | `[contractAddr, method, params, ...]` | `{ result, unsigned_tx? }` |

> 💡 **完整 API 列表**：详见 [WES JSON-RPC API 参考](https://github.com/weisyn/weisyn/blob/main/docs/reference/api.md)

---

## 🔍 错误处理

### 错误类型

```go
// NetworkError - 网络错误
if netErr, ok := err.(*client.NetworkError); ok {
    log.Printf("网络错误: %v", netErr)
}

// JSONRPCError - JSON-RPC 错误
if rpcErr, ok := err.(*client.JSONRPCError); ok {
    log.Printf("RPC 错误: %d, %s", rpcErr.Code, rpcErr.Message)
}
```

### 错误分类

| 错误类型 | 说明 | 是否可重试 |
|---------|------|-----------|
| `NetworkError` | 网络连接错误 | ✅ 是 |
| `JSONRPCError` | JSON-RPC 协议错误 | ⚠️ 部分（5xx 可重试） |
| `TransactionError` | 交易错误（余额不足等） | ❌ 否 |
| `ValidationError` | 参数验证错误 | ❌ 否 |

---

## 🔗 相关文档

- **[Wallet API](./wallet.md)** - 钱包功能
- **[Services API](./services.md)** - 业务服务
- **[重试机制](../reference/retry.md)** - 重试配置详解
- **[故障排查](../troubleshooting.md)** - 常见问题

---

**最后更新**: 2025-11-17

