# 并发控制参考

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

本文档介绍 Go SDK 中的并发控制机制，包括 goroutine 管理、context 取消和资源控制。

---

## 🔗 关联文档

- **批量操作**：[批量操作参考](./batch.md)
- **Client API**：[Client API 参考](../api/client.md)

---

## 🚀 Context 使用

### 基本用法

```go
import (
    "context"
    "time"
)

// 创建带超时的 context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 使用 context 调用 SDK 方法
result, err := tokenService.Transfer(ctx, req, wallet)
```

### 取消操作

```go
ctx, cancel := context.WithCancel(context.Background())

// 在另一个 goroutine 中取消操作
go func() {
    time.Sleep(5 * time.Second)
    cancel()
}()

// SDK 方法会自动响应取消
result, err := tokenService.Transfer(ctx, req, wallet)
if err == context.Canceled {
    fmt.Println("操作已取消")
}
```

---

## 🔄 Goroutine 管理

### 使用 WaitGroup

```go
import "sync"

var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        // 处理 item
        processItem(item)
    }(item)
}

wg.Wait() // 等待所有 goroutine 完成
```

### 使用信号量控制并发

```go
concurrency := 5
sem := make(chan struct{}, concurrency)

for _, item := range items {
    sem <- struct{}{} // 获取信号量
    go func(item Item) {
        defer func() { <-sem }() // 释放信号量
        processItem(item)
    }(item)
}
```

---

## 📊 批量操作中的并发

### 使用 BatchQuery

```go
results, err := utils.BatchQuery(ctx, items, func(ctx context.Context, item Item, index int) (Result, error) {
    // 并发执行查询
    return queryItem(ctx, item)
}, &utils.BatchConfig{
    Concurrency: 5, // 限制并发数量
})
```

### 使用 ParallelExecute

```go
results, err := utils.ParallelExecute(ctx, items, func(ctx context.Context, item Item) (Result, error) {
    // 并行执行操作
    return processItem(ctx, item)
}, 5) // 并发 5 个
```

---

## ⚠️ 注意事项

- ✅ 使用 context 控制超时和取消
- ✅ 使用信号量限制并发数量
- ✅ 注意错误处理和资源清理
- ⚠️ 避免创建过多 goroutine，使用并发控制

---

## 🔗 相关文档

- **[批量操作](./batch.md)** - 批量操作工具
- **[Client API](../api/client.md)** - Client API 文档

---

