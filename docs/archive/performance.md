# 性能优化指南

---

## 📌 版本信息

- **版本**：1.0.0-alpha
- **状态**：draft
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK（已归档）

---

## 📋 概述

本文档介绍 WES Client SDK (Go) 的性能优化功能和使用方法。

---

## 🚀 性能优化功能

### 1. 请求重试机制

SDK 提供了指数退避重试机制，自动处理网络请求失败的情况。

#### 配置重试

```go
import (
    "github.com/weisyn/client-sdk-go/client"
)

config := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Timeout:  30,
    Retry: &client.RetryConfig{
        MaxRetries:        3,              // 最大重试次数
        InitialDelay:      1000,          // 初始延迟（毫秒）
        MaxDelay:          10000,          // 最大延迟（毫秒）
        BackoffMultiplier: 2.0,           // 退避倍数
        OnRetry: func(attempt int, err error) {
            log.Printf("Retry attempt %d: %v", attempt, err)
        },
    },
}

cli, err := client.NewHTTPClient(config)
```

#### 默认重试配置

- **最大重试次数**: 3
- **初始延迟**: 1000ms
- **最大延迟**: 10000ms
- **退避倍数**: 2.0

#### 可重试的错误

以下错误会自动重试：
- 网络错误（连接失败、超时等）
- DNS 错误
- HTTP 5xx 错误（服务器错误）
- HTTP 429 错误（请求过多）

---

## 📊 性能建议

### 1. 批量操作

#### 批量转账

**推荐配置**：
- 使用 `BatchTransfer` 方法，一次交易处理多个转账
- 所有转账必须使用同一个 tokenID
- 批量大小：建议 10-50 个转账

**示例**：
```go
tokenService := token.NewService(client)

result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: fromAddr,
    Transfers: []token.TransferItem{
        {To: addr1, Amount: 100, TokenID: tokenID},
        {To: addr2, Amount: 200, TokenID: tokenID},
        // ... 更多转账
    },
}, wallet)
```

#### 批量查询

**推荐配置**：
- 使用 `utils.BatchQuery` 批量查询工具
- 控制并发数量（建议 5-10 个）
- 使用 context 控制超时

**示例**：
```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/utils"
)

// 使用批量查询工具
addresses := [][]byte{addr1, addr2, addr3, ...}
result, err := utils.BatchQuery(
    ctx,
    addresses,
    func(ctx context.Context, addr []byte, index int) (uint64, error) {
        return tokenService.GetBalance(ctx, addr, nil)
    },
    &utils.BatchConfig{
        BatchSize: 50,
        Concurrency: 5,
        OnProgress: func(progress utils.BatchProgress) {
            fmt.Printf("Progress: %d%%\n", progress.Percentage)
        },
    },
)

if err != nil {
    return err
}

// 处理结果
for i, balance := range result.Results {
    fmt.Printf("Address %d balance: %d\n", i, balance)
}

// 处理错误
for _, err := range result.Errors {
    fmt.Printf("Error at index %d: %v\n", err.Index, err.Error)
}
```

---

### 2. 网络请求优化

#### 连接池配置

**推荐配置**：
- 使用 HTTP/2（如果支持）
- 设置合理的超时时间
- 启用连接复用

**示例**：
```go
config := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Timeout:  30, // 30秒超时
    Retry:    client.DefaultRetryConfig(),
}
```

#### 重试配置建议

**生产环境**：
```go
retryConfig := &client.RetryConfig{
    MaxRetries:        5,              // 生产环境可以增加重试次数
    InitialDelay:      1000,
    MaxDelay:          30000,          // 增加最大延迟
    BackoffMultiplier: 2.0,
}
```

**开发环境**：
```go
retryConfig := &client.RetryConfig{
    MaxRetries:        2,              // 开发环境减少重试次数
    InitialDelay:      500,
    MaxDelay:          5000,
    BackoffMultiplier: 2.0,
}
```

---

### 3. 交易构建优化

#### UTXO 选择策略

**推荐**：
- 使用贪心算法选择 UTXO（最小化输入数量）
- 预先查询并缓存常用地址的 UTXO
- 避免频繁查询 UTXO

**示例**：
```go
// 预先查询 UTXO
utxos, err := client.Call(ctx, "wes_getUTXO", []interface{}{addressHex})
if err != nil {
    return err
}

// 缓存 UTXO 信息（根据业务需求）
// ...
```

---

## 🔧 性能监控

### 请求耗时统计

```go
start := time.Now()
result, err := client.Call(ctx, "wes_getBalance", params)
duration := time.Since(start)

log.Printf("Request took %v", duration)
```

### 重试统计

```go
retryCount := 0
retryConfig := &client.RetryConfig{
    MaxRetries: 3,
    OnRetry: func(attempt int, err error) {
        retryCount++
        log.Printf("Retry %d: %v", attempt, err)
    },
}
```

---

### 3. 大文件处理

SDK 提供了大文件处理工具，支持分块处理和流式读取，避免一次性加载大文件到内存。

#### 分块处理文件

**推荐配置**：
- 使用 `ProcessFileInChunks` 分块处理大文件
- 分块大小：建议 1-5MB
- 并发数量：建议 3-5 个

**示例**：
```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/utils"
)

// 读取文件并分块处理
data, err := os.ReadFile("large_file.bin")
if err != nil {
    return err
}

results, err := utils.ProcessFileInChunks(
    context.Background(),
    data,
    func(chunk []byte, index int) (string, error) {
        // 处理每个分块
        return processChunk(chunk), nil
    },
    &utils.ChunkConfig{
        ChunkSize: 5 * 1024 * 1024, // 5MB
        Concurrency: 3,
        OnProgress: func(progress utils.FileProgress) {
            fmt.Printf("Progress: %d%%\n", progress.Percentage)
        },
    },
)
```

#### 流式读取文件

**推荐配置**：
- 使用 `ReadFileAsStream` 流式读取大文件
- 使用 `ReadFileInChunks` 分块读取并处理

**示例**：
```go
// 流式读取文件（带进度回调）
data, err := utils.ReadFileAsStream("large_file.bin", func(progress utils.FileProgress) {
    fmt.Printf("Reading: %d%%\n", progress.Percentage)
})

// 分块读取并处理（不一次性加载到内存）
err := utils.ReadFileInChunks("large_file.bin", func(chunk []byte, index int) error {
    // 处理每个分块
    return processChunk(chunk)
}, &utils.ChunkConfig{
    ChunkSize: 5 * 1024 * 1024,
    OnProgress: func(progress utils.FileProgress) {
        fmt.Printf("Processing: %d%%\n", progress.Percentage)
    },
})
```

#### 处理时间估算

```go
fileSize := int64(100 * 1024 * 1024) // 100MB
chunkSize := int64(5 * 1024 * 1024)  // 5MB
processingSpeed := int64(10 * 1024 * 1024) // 10MB/s

estimatedTime := utils.EstimateProcessingTime(fileSize, chunkSize, processingSpeed)
fmt.Printf("Estimated processing time: %v\n", estimatedTime)
```

---

## 📚 相关文档

- [主 README](../README.md) - SDK 总体文档
- [服务文档](./modules/services.md) - 业务服务文档
- [架构文档](./architecture.md) - SDK 架构设计

---

**最后更新**: 2025-11-17

