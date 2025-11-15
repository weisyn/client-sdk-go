# Market Service - 市场服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 基础结构完成  
**最后更新**: 2025-01-23

---

## 📋 概述

Market Service 提供去中心化市场相关的业务操作，包括 AMM 代币交换、流动性管理、归属计划和托管等功能。所有操作都使用 Wallet 接口进行签名，符合 SDK 架构原则。

---

## 🔧 核心功能

### 1. SwapAMM - AMM 代币交换 ✅

**功能**: 在 AMM 池中进行代币交换

**使用示例**:
```go
marketService := market.NewService(client)

result, err := marketService.SwapAMM(ctx, &market.SwapRequest{
    From:      fromAddr,
    TokenIn:   tokenInID,
    TokenOut:  tokenOutID,
    AmountIn:  1000,
    MinAmountOut: 950, // 滑点保护
}, wallet)
```

### 2. AddLiquidity - 添加流动性 ✅

**功能**: 向 AMM 池添加流动性

**使用示例**:
```go
result, err := marketService.AddLiquidity(ctx, &market.AddLiquidityRequest{
    From:     providerAddr,
    TokenA:   tokenAID,
    TokenB:   tokenBID,
    AmountA:  1000,
    AmountB:  2000,
}, wallet)
```

### 3. RemoveLiquidity - 移除流动性 ✅

**功能**: 从 AMM 池移除流动性

**使用示例**:
```go
result, err := marketService.RemoveLiquidity(ctx, &market.RemoveLiquidityRequest{
    From:     providerAddr,
    TokenA:   tokenAID,
    TokenB:   tokenBID,
    Liquidity: liquidityAmount,
}, wallet)
```

### 4. CreateVesting - 创建归属计划 ✅

**功能**: 创建代币归属计划

**使用示例**:
```go
result, err := marketService.CreateVesting(ctx, &market.CreateVestingRequest{
    From:     creatorAddr,
    To:       beneficiaryAddr,
    Amount:   10000,
    TokenID:  tokenID,
    StartTime: startTimestamp,
    Duration:  86400 * 365, // 1年
}, wallet)
```

### 5. ClaimVesting - 领取归属代币 ✅

**功能**: 领取归属计划中的代币

**使用示例**:
```go
result, err := marketService.ClaimVesting(ctx, &market.ClaimVestingRequest{
    From:     beneficiaryAddr,
    VestingID: vestingID,
}, wallet)
```

### 6. CreateEscrow - 创建托管 ✅

**功能**: 创建代币托管

**使用示例**:
```go
result, err := marketService.CreateEscrow(ctx, &market.CreateEscrowRequest{
    From:     senderAddr,
    To:       recipientAddr,
    Amount:   1000,
    TokenID:  tokenID,
    Condition: conditionData,
}, wallet)
```

### 7. ReleaseEscrow - 释放托管 ✅

**功能**: 释放托管代币给接收方

**使用示例**:
```go
result, err := marketService.ReleaseEscrow(ctx, &market.ReleaseEscrowRequest{
    From:     senderAddr,
    EscrowID: escrowID,
}, wallet)
```

### 8. RefundEscrow - 退款托管 ✅

**功能**: 退款托管代币给发送方

**使用示例**:
```go
result, err := marketService.RefundEscrow(ctx, &market.RefundEscrowRequest{
    From:     senderAddr,
    EscrowID: escrowID,
}, wallet)
```

---

## 🏗️ 服务架构

```
┌─────────────────────────────────────────┐
│        Market Service 架构              │
└─────────────────────────────────────────┘

Market Service
    │
    ├─> SwapAMM: AMM 代币交换
    ├─> AddLiquidity: 添加流动性
    ├─> RemoveLiquidity: 移除流动性
    ├─> CreateVesting: 创建归属计划
    ├─> ClaimVesting: 领取归属代币
    ├─> CreateEscrow: 创建托管
    ├─> ReleaseEscrow: 释放托管
    └─> RefundEscrow: 退款托管
```

---

## 📚 API 参考

### Service 接口

```go
type Service interface {
    SwapAMM(ctx context.Context, req *SwapRequest, wallets ...wallet.Wallet) (*SwapResult, error)
    AddLiquidity(ctx context.Context, req *AddLiquidityRequest, wallets ...wallet.Wallet) (*AddLiquidityResult, error)
    RemoveLiquidity(ctx context.Context, req *RemoveLiquidityRequest, wallets ...wallet.Wallet) (*RemoveLiquidityResult, error)
    CreateVesting(ctx context.Context, req *CreateVestingRequest, wallets ...wallet.Wallet) (*CreateVestingResult, error)
    ClaimVesting(ctx context.Context, req *ClaimVestingRequest, wallets ...wallet.Wallet) (*ClaimVestingResult, error)
    CreateEscrow(ctx context.Context, req *CreateEscrowRequest, wallets ...wallet.Wallet) (*CreateEscrowResult, error)
    ReleaseEscrow(ctx context.Context, req *ReleaseEscrowRequest, wallets ...wallet.Wallet) (*ReleaseEscrowResult, error)
    RefundEscrow(ctx context.Context, req *RefundEscrowRequest, wallets ...wallet.Wallet) (*RefundEscrowResult, error)
}
```

---

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

