# Market Service - 市场服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 已迁移到新架构（Draft+Hash+Finalize）  
**最后更新**: 2025-01-23

---

## ⚠️ 重要更新

**Market 模块已完全迁移到新架构（Draft+Hash+Finalize）**，所有操作现在使用：
- `build*Draft` 函数构建交易草稿
- `wes_computeSignatureHashFromDraft` 计算签名哈希
- `Wallet.SignHash` 对哈希进行签名
- `wes_finalizeTransactionFromDraft` 生成完整交易
- `wes_sendRawTransaction` 提交交易

旧的 `build*Transaction` 函数已标记为废弃，将在未来版本中移除。

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

### 架构说明

Market Service 采用新的 **Draft+Hash+Finalize** 架构：

1. **构建草稿（Draft）**：SDK 层构建交易草稿（DraftJSON）
2. **计算哈希（Hash）**：调用节点 API 计算签名哈希
3. **签名哈希（Sign）**：使用 Wallet 对哈希进行签名
4. **完成交易（Finalize）**：调用节点 API 生成完整交易
5. **提交交易（Submit）**：提交已签名的交易

### 架构图

```
┌─────────────────────────────────────────┐
│        Market Service 架构              │
└─────────────────────────────────────────┘

Market Service
    │
    ├─> CreateVesting: 创建归属计划
    │   └─> buildVestingDraft → computeHash → signHash → finalize → submit
    ├─> ClaimVesting: 领取归属代币
    │   └─> buildClaimVestingDraft → computeHash → signHash → finalize → submit
    ├─> CreateEscrow: 创建托管
    │   └─> buildEscrowDraft → computeHash → signHash → finalize → submit
    ├─> ReleaseEscrow: 释放托管
    │   └─> buildReleaseEscrowDraft → computeHash → signHash → finalize → submit
    └─> RefundEscrow: 退款托管
        └─> buildRefundEscrowDraft → computeHash → signHash → finalize → submit
```

### 手续费规则

**重要**：手续费从接收者扣除，发送者不需要支付手续费。发送者只需要满足输出金额即可，找零 = 输入金额 - 输出金额。

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

## 📖 新路径流程

### CreateVesting（创建归属计划）

1. 调用 `buildVestingDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### ClaimVesting（领取归属代币）

1. 调用 `buildClaimVestingDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### CreateEscrow（创建托管）

1. 调用 `buildEscrowDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### ReleaseEscrow（释放托管）

1. 调用 `buildReleaseEscrowDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### RefundEscrow（退款托管）

1. 调用 `buildRefundEscrowDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

## 🔄 迁移说明

### 旧路径（已废弃）

旧路径使用 `build*Transaction` 函数直接构建未签名交易，然后使用 `Wallet.SignTransaction` 签名：

```go
// ⚠️ 已废弃：不再使用
unsignedTxBytes, err := buildVestingTransaction(...)
signedTxBytes, err := wallet.SignTransaction(unsignedTxBytes)
```

### 新路径（推荐）

新路径使用 `build*Draft` + `wes_computeSignatureHashFromDraft` + `wes_finalizeTransactionFromDraft`：

```go
// ✅ 推荐：使用新路径
draftJSON, inputIndex, err := buildVestingDraft(...)
hashResult, err := client.Call(ctx, "wes_computeSignatureHashFromDraft", ...)
sigBytes, err := wallet.SignHash(hashBytes)
finalResult, err := client.Call(ctx, "wes_finalizeTransactionFromDraft", ...)
```

详细迁移指南请参考：[MIGRATION_GUIDE.md](../../MIGRATION_GUIDE.md)

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档
- [迁移指南](../../MIGRATION_GUIDE.md) - 从旧路径迁移到新路径

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

