# Staking Service - 质押服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 已迁移到新架构（Draft+Hash+Finalize）  
**最后更新**: 2025-01-23

⚠️ **重要更新**：所有 Staking 操作已迁移到新的签名架构路径（`build*Draft` + `wes_computeSignatureHashFromDraft` + `wes_finalizeTransactionFromDraft`）。旧的 `build*Transaction` 函数已标记为废弃，请使用新路径。

---

## 📋 概述

Staking Service 提供质押相关的业务操作，包括质押、解质押、委托、取消委托、领取奖励和罚没等功能。所有操作都使用 Wallet 接口进行签名，符合 SDK 架构原则。

---

## 🔧 核心功能

### 架构说明

所有 Staking 操作现在使用新的签名架构：

1. **构建交易草稿** (`build*Draft`)：在 SDK 层构建 `DraftJSON`
2. **计算签名哈希** (`wes_computeSignatureHashFromDraft`)：调用节点 API 获取签名哈希
3. **签名哈希** (`Wallet.SignHash`)：使用钱包对哈希进行签名
4. **完成交易** (`wes_finalizeTransactionFromDraft`)：调用节点 API 生成带 `SingleKeyProof` 的交易
5. **提交交易** (`wes_sendRawTransaction`)：提交已签名的交易

**手续费规则**：手续费从接收者扣除，发送者不需要支付手续费，找零 = 输入金额 - 输出金额。

---

### 1. Stake - 质押 ✅

**功能**: 质押代币到验证者

**新路径流程**:
```
1. buildStakeDraft() → DraftJSON
2. wes_computeSignatureHashFromDraft() → 签名哈希
3. Wallet.SignHash() → 签名
4. wes_finalizeTransactionFromDraft() → 完整交易
5. wes_sendRawTransaction() → 提交
```

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
- [迁移指南](../../MIGRATION_GUIDE.md) - 从旧路径迁移到新路径

---

## 📝 迁移说明

### 旧路径（已废弃）

旧的 `build*Transaction` 函数（如 `buildStakeTransaction`, `buildDelegateTransaction` 等）已标记为 `Deprecated`，不再推荐使用。这些函数返回 `unsignedTx`，然后使用 `Wallet.SignTransaction` 签名，最后提交。

### 新路径（推荐）

所有操作现在使用 `build*Draft` + `wes_computeSignatureHashFromDraft` + `wes_finalizeTransactionFromDraft` 路径，确保：
- SDK 只负责私钥管理和哈希签名
- 节点负责复杂的 EUTXO/lock/proof 逻辑
- 架构边界清晰，易于维护和扩展

详细迁移指南请参考 [MIGRATION_GUIDE.md](../../MIGRATION_GUIDE.md)。

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

