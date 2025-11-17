# 简单转账示例

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

本示例演示如何使用 Go SDK 进行简单的代币转账。

---

## 💻 完整代码

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
    // 1. 创建客户端
    cfg := &client.Config{
        Endpoint: "http://localhost:8545",
        Protocol: client.ProtocolHTTP,
    }
    c, err := client.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // 2. 创建钱包
    w, err := wallet.NewWallet()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("钱包地址: %x\n", w.Address())
    
    // 3. 创建 Token 服务
    tokenService := token.NewService(c)
    
    // 4. 查询余额
    ctx := context.Background()
    balance, err := tokenService.GetBalance(ctx, w.Address(), nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("当前余额: %d\n", balance)
    
    // 5. 转账（需要先充值账户）
    recipientAddr := make([]byte, 20)
    recipientAddr[0] = 0x02
    
    result, err := tokenService.Transfer(ctx, &token.TransferRequest{
        From:   w.Address(),
        To:     recipientAddr,
        Amount: 1000,
        TokenID: nil, // nil 表示原生币
    }, w)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("转账成功！交易哈希: %s\n", result.TxHash)
}
```

---

## 🔗 相关文档

- **[Token 指南](../guides/token.md)** - 详细使用指南
- **[快速开始](../getting-started.md)** - 安装和配置

---

