# 批量操作示例

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

本示例演示如何使用批量操作工具进行批量转账和查询。

---

## 💻 完整代码

### 批量转账

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/wallet"
)

func main() {
    cfg := &client.Config{
        Endpoint: "http://localhost:8545",
        Protocol: client.ProtocolHTTP,
    }
    c, err := client.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    w, err := wallet.NewWallet()
    if err != nil {
        log.Fatal(err)
    }
    
    tokenService := token.NewService(c)
    ctx := context.Background()
    
    // 批量转账
    result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
        From: w.Address(),
        Transfers: []token.TransferItem{
            {To: addr1, Amount: 100},
            {To: addr2, Amount: 200},
            {To: addr3, Amount: 300},
        },
        TokenID: nil,
    }, w)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("批量转账成功！交易哈希: %s\n", result.TxHash)
}
```

### 批量查询余额

```go
import "github.com/weisyn/client-sdk-go/utils"

addresses := [][]byte{addr1, addr2, addr3}

results, err := utils.BatchQuery(ctx, addresses, func(ctx context.Context, addr []byte, index int) (uint64, error) {
    return tokenService.GetBalance(ctx, addr, nil)
}, &utils.BatchConfig{
    BatchSize:   50,
    Concurrency: 5,
    OnProgress: func(progress utils.BatchProgress) {
        fmt.Printf("进度: %d/%d\n", progress.Completed, progress.Total)
    },
})

if err != nil {
    log.Fatal(err)
}

for i, balance := range results.Results {
    fmt.Printf("地址 %d 余额: %d\n", i, balance)
}
```

---

## 🔗 相关文档

- **[批量操作参考](../reference/batch.md)** - 详细使用指南
- **[Token 指南](../guides/token.md)** - Token 服务指南

---

