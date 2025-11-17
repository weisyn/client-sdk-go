# 快速开始

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

本指南将帮助你快速开始使用 WES Client SDK (Go)，包括安装、配置和第一个示例。

---

## 🔗 关联文档

- **WES 安装**：[WES 节点安装指南](https://github.com/weisyn/weisyn/blob/main/docs/tutorials/installation.md)（待确认）
- **架构说明**：[SDK 架构设计](./architecture.md)

---

## 📦 安装

### 使用 Go Modules

```bash
go get github.com/weisyn/client-sdk-go
```

### 导入包

```go
import (
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/wallet"
)
```

---

## 🚀 第一个示例

### 简单转账

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/wallet"
)

func main() {
    // 1. 初始化客户端
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

    // 2. 创建或导入钱包
    // 方式 1：创建新钱包
    w, err := wallet.NewWallet()
    if err != nil {
        log.Fatal(err)
    }
    
    // 方式 2：从私钥导入
    // privateKeyHex := "0x..."
    // w, err := wallet.NewWalletFromPrivateKey(privateKeyHex)
    // if err != nil {
    //     log.Fatal(err)
    // }

    // 3. 创建 Token 服务
    tokenService := token.NewTokenService(c, w)

    // 4. 查询余额
    ctx := context.Background()
    balance, err := tokenService.GetBalance(ctx, w.Address(), nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("余额: %s\n", balance.String())

    // 5. 执行转账
    recipient := wallet.MustAddressFromHex("0x...") // 接收方地址
    amount := big.NewInt(1000000)                    // 1 WES（假设 6 位小数）

    result, err := tokenService.Transfer(ctx, &token.TransferRequest{
        From:   w.Address(),
        To:     recipient,
        Amount: amount,
        TokenID: nil, // nil 表示原生币
    }, w)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("转账成功！交易哈希: %s\n", result.TxHash)
}
```

---

## 🔧 配置

### Client 配置

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",  // 节点端点
    Protocol: client.ProtocolHTTP,     // 协议：HTTP/gRPC/WebSocket
    Timeout:  30 * time.Second,        // 超时时间
    Debug:    false,                   // 调试模式
    Retry: &client.RetryConfig{       // 重试配置（可选）
        MaxRetries:      3,
        InitialDelay:    500 * time.Millisecond,
        MaxDelay:        10 * time.Second,
        BackoffMultiplier: 2,
    },
}
```

### 环境变量配置

```go
import "os"

endpoint := os.Getenv("WES_NODE_ENDPOINT")
if endpoint == "" {
    endpoint = "http://localhost:8545" // 默认值
}

cfg := &client.Config{
    Endpoint: endpoint,
    Protocol: client.ProtocolHTTP,
}
```

---

## 📚 核心概念

### 1. Client

`Client` 是与 WES 节点通信的核心接口：

```go
c, err := client.NewClient(&client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
})
```

### 2. Wallet

`Wallet` 提供密钥管理和签名功能：

```go
// 创建新钱包
w, err := wallet.NewWallet()

// 从私钥导入
w, err := wallet.FromPrivateKey("0x...")

// 获取地址
address := w.Address() // [20]byte

// 签名交易
signature := w.SignTransaction(unsignedTx)
```

### 3. Services

业务服务提供高级 API：

```go
// Token 服务
tokenService := token.NewTokenService(c, w)
result, err := tokenService.Transfer(ctx, &token.TransferRequest{...}, w)

// Staking 服务
stakingService := staking.NewStakingService(c, w)
result, err := stakingService.Stake(ctx, &staking.StakeRequest{...}, w)
```

---

## 🎯 下一步

- **[概述](./overview.md)** - 了解 SDK 视角的 WES 核心概念
- **[Token 指南](./guides/token.md)** - 学习 Token 服务的使用
- **[API 参考](./api/)** - 查看完整的 API 文档

---

## 🔗 相关文档

- **[WES 项目总览](https://github.com/weisyn/weisyn/blob/main/docs/overview.md)** - WES 核心概念和定位
- **[WES 系统架构](https://github.com/weisyn/weisyn/blob/main/docs/system/architecture/README.md)** - 完整的系统架构设计
- **[JSON-RPC API 参考](https://github.com/weisyn/weisyn/blob/main/docs/reference/api.md)** - 底层 API 接口文档

---

**最后更新**: 2025-11-17
