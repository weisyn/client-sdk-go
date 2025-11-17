# Market 服务指南

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

Market Service 提供市场相关功能，包括 AMM 代币交换、流动性管理、托管和归属计划。

---

## 🔗 关联文档

- **API 参考**：[Services API - Market](../api/services.md#-market-service)
- **WES 协议**：[WES 市场机制](https://github.com/weisyn/weisyn/blob/main/docs/system/platforms/market/README.md)（待确认）

---

## 🚀 快速开始

### 创建服务

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/market"
    "github.com/weisyn/client-sdk-go/wallet"
)

cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
}
cli, err := client.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}

w, err := wallet.NewWallet()
if err != nil {
    log.Fatal(err)
}

marketService := market.NewService(cli)
```

---

## 💱 AMM 代币交换

### 基本交换

```go
ctx := context.Background()

result, err := marketService.SwapAMM(ctx, &market.SwapAMMRequest{
    ContractAddr: ammContractAddr,
    TokenIn:      tokenA,
    AmountIn:     1000000,
    TokenOut:     tokenB,
    MinAmountOut: 900000, // 滑点保护：最小输出量
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("交换成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("实际输出: %d\n", result.AmountOut)
```

### 滑点保护

```go
// 设置最小输出量，防止滑点过大
result, err := marketService.SwapAMM(ctx, &market.SwapAMMRequest{
    ContractAddr: ammContractAddr,
    TokenIn:      tokenA,
    AmountIn:     1000000,
    TokenOut:     tokenB,
    MinAmountOut: 950000, // 至少获得 95% 的预期输出
}, w)
```

---

## 💧 流动性管理

### 添加流动性

```go
result, err := marketService.AddLiquidity(ctx, &market.AddLiquidityRequest{
    ContractAddr: ammContractAddr,
    TokenA:       tokenA,
    AmountA:      1000000,
    TokenB:       tokenB,
    AmountB:      1000000,
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("添加流动性成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("流动性 ID: %s\n", result.LiquidityID)
```

### 移除流动性

```go
result, err := marketService.RemoveLiquidity(ctx, &market.RemoveLiquidityRequest{
    ContractAddr: ammContractAddr,
    LiquidityID:  liquidityID,
    Amount:       500000, // 移除部分流动性
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("移除流动性成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("获得 Token A: %d\n", result.AmountA)
fmt.Printf("获得 Token B: %d\n", result.AmountB)
```

---

## 🔒 托管（Escrow）

### 创建托管

```go
sellerWallet, _ := wallet.NewWallet()

result, err := marketService.CreateEscrow(ctx, &market.CreateEscrowRequest{
    Buyer:   w.Address(),
    Seller:  sellerWallet.Address(),
    Amount:  1000000,
    TokenID: nil, // nil 表示原生币
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("创建托管成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("托管 ID: %s\n", result.EscrowID)
```

### 释放托管（给卖方）

```go
// 卖方操作
sellerMarketService := market.NewService(cli)

result, err := sellerMarketService.ReleaseEscrow(ctx, &market.ReleaseEscrowRequest{
    EscrowID: escrowID,
}, sellerWallet)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("释放托管成功！交易哈希: %s\n", result.TxHash)
```

### 退款托管（给买方）

```go
// 买方操作（例如：交易取消或过期）
result, err := marketService.RefundEscrow(ctx, &market.RefundEscrowRequest{
    EscrowID: escrowID,
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("退款成功！交易哈希: %s\n", result.TxHash)
```

---

## 📅 归属计划（Vesting）

### 创建归属计划

```go
recipientWallet, _ := wallet.NewWallet()
unlockTime := time.Now().Add(30 * 24 * time.Hour).Unix() // 30 天后解锁

result, err := marketService.CreateVesting(ctx, &market.CreateVestingRequest{
    Recipient: recipientWallet.Address(),
    Amount:    10000000,
    TokenID:   tokenID,
    UnlockTime: unlockTime,
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("创建归属计划成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("归属 ID: %s\n", result.VestingID)
```

### 领取归属代币

```go
// 接收者操作（解锁时间到达后）
recipientMarketService := market.NewService(cli)

result, err := recipientMarketService.ClaimVesting(ctx, &market.ClaimVestingRequest{
    VestingID: vestingID,
}, recipientWallet)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("领取归属代币成功！交易哈希: %s\n", result.TxHash)
```

---

## 🎯 典型场景

### 场景 1：完整的 AMM 流动性流程

```go
func completeAMMFlow(
    ctx context.Context,
    providerWallet wallet.Wallet,
    ammContractAddr []byte,
    tokenA, tokenB []byte,
    marketService market.Service,
) error {
    // 1. 添加流动性
    addResult, err := marketService.AddLiquidity(ctx, &market.AddLiquidityRequest{
        ContractAddr: ammContractAddr,
        TokenA:       tokenA,
        AmountA:      1000000,
        TokenB:       tokenB,
        AmountB:      1000000,
    }, providerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("流动性 ID: %s\n", addResult.LiquidityID)
    
    // 2. 等待一段时间后，移除部分流动性
    // ... 等待 ...
    
    removeResult, err := marketService.RemoveLiquidity(ctx, &market.RemoveLiquidityRequest{
        ContractAddr: ammContractAddr,
        LiquidityID:  addResult.LiquidityID,
        Amount:       500000, // 移除一半
    }, providerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("获得 Token A: %d\n", removeResult.AmountA)
    fmt.Printf("获得 Token B: %d\n", removeResult.AmountB)
    return nil
}
```

---

## ⚠️ 常见错误

### 滑点过大

```go
result, err := marketService.SwapAMM(ctx, &market.SwapAMMRequest{
    ContractAddr: ammContractAddr,
    TokenIn:      tokenA,
    AmountIn:     1000000,
    TokenOut:     tokenB,
    MinAmountOut: 999999, // 设置过高的最小输出量
}, w)
if err != nil {
    if strings.Contains(err.Error(), "slippage") {
        log.Fatal("滑点过大，交易失败")
    }
    log.Fatal(err)
}
```

---

## 🔗 相关文档

- **[API 参考](../api/services.md#-market-service)** - 完整 API 文档
- **[Token 指南](./token.md)** - 代币操作指南
- **[故障排查](../troubleshooting.md)** - 常见问题

---

