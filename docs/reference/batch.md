# 批量操作参考

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

SDK 提供了批量操作工具，可以高效处理大量数据，支持并发控制和进度监控。

---

## 🔗 关联文档

- **API 参考**：[Services API](../api/services.md)
- **并发控制**：[并发参考](./concurrency.md)

---

## 📦 导入

```go
import "github.com/weisyn/client-sdk-go/utils"
```

---

## 🔍 批量查询

### BatchQuery()

批量查询多个项目，支持并发控制和错误处理。

```go
func BatchQuery[T any, R any](
    ctx context.Context,
    items []T,
    queryFn func(ctx context.Context, item T, index int) (R, error),
    config *BatchConfig,
) (*BatchQueryResult[R], error)
```

### 示例：批量查询余额

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/utils"
    "github.com/weisyn/client-sdk-go/services/token"
)

addresses := [][]byte{
    addr1,
    addr2,
    addr3,
    // ... 更多地址
}

tokenService := token.NewService(client)

results, err := utils.BatchQuery(ctx, addresses, func(ctx context.Context, addr []byte, index int) (uint64, error) {
    return tokenService.GetBalance(ctx, addr, nil)
}, &utils.BatchConfig{
    BatchSize:   50,
    Concurrency: 5,
    OnProgress: func(progress utils.BatchProgress) {
        fmt.Printf("进度: %d/%d (%.1f%%)\n", 
            progress.Completed, progress.Total, 
            float64(progress.Percentage))
    },
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("成功: %d, 失败: %d\n", results.Success, results.Failed)
for i, balance := range results.Results {
    fmt.Printf("地址 %d 余额: %d\n", i, balance)
}
```

---

## ⚡ 并行执行

### ParallelExecute()

并行执行多个操作，限制并发数量。

```go
func ParallelExecute[T any, R any](
    ctx context.Context,
    items []T,
    executeFn func(ctx context.Context, item T) (R, error),
    concurrency int,
) ([]R, error)
```

### 示例：并行转账

```go
transfers := []token.TransferItem{
    {To: addr1, Amount: 100},
    {To: addr2, Amount: 200},
    {To: addr3, Amount: 300},
}

results, err := utils.ParallelExecute(ctx, transfers, func(ctx context.Context, transfer token.TransferItem) (string, error) {
    result, err := tokenService.Transfer(ctx, &token.TransferRequest{
        From:   wallet.Address(),
        To:     transfer.To,
        Amount: transfer.Amount,
    }, wallet)
    if err != nil {
        return "", err
    }
    return result.TxHash, nil
}, 5) // 并发 5 个
```

---

## 📊 数组分批处理

### BatchArray()

将数组分成多个批次。

```go
func BatchArray[T any](array []T, batchSize int) [][]T
```

### 示例

```go
items := []string{"item1", "item2", "item3", "item4", "item5"}
batches := utils.BatchArray(items, 2)
// batches = [["item1", "item2"], ["item3", "item4"], ["item5"]]
```

---

## ⚙️ 配置选项

### BatchConfig

```go
type BatchConfig struct {
    BatchSize   int                          // 批量大小，默认 50
    Concurrency int                          // 并发数量，默认 5
    OnProgress  func(progress BatchProgress) // 进度回调函数
}
```

### BatchProgress

```go
type BatchProgress struct {
    Completed  int // 已完成数量
    Total      int // 总数量
    Percentage int // 进度百分比（0-100）
    Success    int // 成功数量
    Failed     int // 失败数量
}
```

---

## 🎯 使用建议

- ✅ 批量大小建议：10-50 个
- ✅ 并发数量建议：3-10 个（根据网络和节点性能调整）
- ✅ 使用进度回调监控处理进度
- ⚠️ 注意错误处理，批量操作可能部分成功

---

## 🔗 相关文档

- **[Services API](../api/services.md)** - 业务服务 API
- **[并发参考](./concurrency.md)** - Go 并发特性

---

