# Staking Service - 质押服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 基础结构完成  
**最后更新**: 2025-01-23

---

## 📋 概述

Staking Service 提供质押相关的业务操作，包括质押、解质押、委托、取消委托、领取奖励和罚没等功能。所有操作都使用 Wallet 接口进行签名，符合 SDK 架构原则。

---

## 🔧 核心功能

### 1. Stake - 质押 ✅

**功能**: 质押代币到验证者

**使用示例**:
```go
stakingService := staking.NewService(client)

result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:     stakerAddr,
    Amount:   10000,
    Validator: validatorAddr,
}, wallet)
```

### 2. Unstake - 解质押 ✅

**功能**: 从验证者解质押代币

**使用示例**:
```go
result, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
    From:     stakerAddr,
    Amount:   5000,
    Validator: validatorAddr,
}, wallet)
```

### 3. Delegate - 委托 ✅

**功能**: 委托代币给验证者

**使用示例**:
```go
result, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
    From:     delegatorAddr,
    To:       validatorAddr,
    Amount:   1000,
}, wallet)
```

### 4. Undelegate - 取消委托 ✅

**功能**: 取消对验证者的委托

**使用示例**:
```go
result, err := stakingService.Undelegate(ctx, &staking.UndelegateRequest{
    From:     delegatorAddr,
    To:       validatorAddr,
    Amount:   500,
}, wallet)
```

### 5. ClaimReward - 领取奖励 ✅

**功能**: 领取质押奖励

**使用示例**:
```go
result, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
    From:     stakerAddr,
    Validator: validatorAddr,
}, wallet)
```

### 6. Slash - 罚没 ✅

**功能**: 罚没验证者（治理功能）

**使用示例**:
```go
result, err := stakingService.Slash(ctx, &staking.SlashRequest{
    Validator: validatorAddr,
    Amount:    1000,
    Reason:    "double_sign",
}, wallet)
```

---

## 🏗️ 服务架构

```
┌─────────────────────────────────────────┐
│        Staking Service 架构             │
└─────────────────────────────────────────┘

Staking Service
    │
    ├─> Stake: 质押代币
    ├─> Unstake: 解质押代币
    ├─> Delegate: 委托代币
    ├─> Undelegate: 取消委托
    ├─> ClaimReward: 领取奖励
    └─> Slash: 罚没（治理）
```

---

## 📚 API 参考

### Service 接口

```go
type Service interface {
    Stake(ctx context.Context, req *StakeRequest, wallets ...wallet.Wallet) (*StakeResult, error)
    Unstake(ctx context.Context, req *UnstakeRequest, wallets ...wallet.Wallet) (*UnstakeResult, error)
    Delegate(ctx context.Context, req *DelegateRequest, wallets ...wallet.Wallet) (*DelegateResult, error)
    Undelegate(ctx context.Context, req *UndelegateRequest, wallets ...wallet.Wallet) (*UndelegateResult, error)
    ClaimReward(ctx context.Context, req *ClaimRewardRequest, wallets ...wallet.Wallet) (*ClaimRewardResult, error)
    Slash(ctx context.Context, req *SlashRequest, wallets ...wallet.Wallet) (*SlashResult, error)
}
```

---

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

