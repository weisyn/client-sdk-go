# Governance Service - 治理服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 已迁移到新架构（Draft+Hash+Finalize）  
**最后更新**: 2025-01-23

---

## ⚠️ 重要更新

**Governance 模块已完全迁移到新架构（Draft+Hash+Finalize）**，所有操作现在使用：
- `build*Draft` 函数构建交易草稿
- `wes_computeSignatureHashFromDraft` 计算签名哈希
- `Wallet.SignHash` 对哈希进行签名
- `wes_finalizeTransactionFromDraft` 生成完整交易
- `wes_sendRawTransaction` 提交交易

旧的 `build*Transaction` 函数已标记为废弃，将在未来版本中移除。

---

## 📋 概述

Governance Service 提供链上治理相关的业务操作，包括创建提案、投票和更新参数等功能。所有操作都使用 Wallet 接口进行签名，符合 SDK 架构原则。

---

## 🔧 核心功能

### 1. Propose - 创建提案 ✅

**功能**: 创建链上治理提案

**使用示例**:
```go
governanceService := governance.NewService(client)

result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
    From:        proposerAddr,
    Title:       "提案标题",
    Description: "提案描述",
    ProposalType: "parameter_change",
    Parameters:  parameterData,
}, wallet)
```

### 2. Vote - 投票 ✅

**功能**: 对提案进行投票

**使用示例**:
```go
result, err := governanceService.Vote(ctx, &governance.VoteRequest{
    From:       voterAddr,
    ProposalID: proposalID,
    Option:     "yes", // "yes", "no", "abstain"
    Weight:     1000,  // 投票权重
}, wallet)
```

### 3. UpdateParam - 更新参数 ✅

**功能**: 更新链上参数（需要治理权限）

**使用示例**:
```go
result, err := governanceService.UpdateParam(ctx, &governance.UpdateParamRequest{
    From:     adminAddr,
    ParamKey: "fee_rate",
    ParamValue: "0.0003", // 新的费率
}, wallet)
```

---

## 🏗️ 服务架构

### 架构说明

Governance Service 采用新的 **Draft+Hash+Finalize** 架构：

1. **构建草稿（Draft）**：SDK 层构建交易草稿（DraftJSON）
2. **计算哈希（Hash）**：调用节点 API 计算签名哈希
3. **签名哈希（Sign）**：使用 Wallet 对哈希进行签名
4. **完成交易（Finalize）**：调用节点 API 生成完整交易
5. **提交交易（Submit）**：提交已签名的交易

### 架构图

```
┌─────────────────────────────────────────┐
│      Governance Service 架构            │
└─────────────────────────────────────────┘

Governance Service
    │
    ├─> Propose: 创建提案
    │   └─> buildProposeDraft → computeHash → signHash → finalize → submit
    ├─> Vote: 投票
    │   └─> buildVoteDraft → computeHash → signHash → finalize → submit
    └─> UpdateParam: 更新参数
        └─> buildUpdateParamDraft → computeHash → signHash → finalize → submit
```

### 手续费规则

**重要**：手续费从接收者扣除，发送者不需要支付手续费。发送者只需要满足输出金额即可，找零 = 输入金额 - 输出金额。

---

## 📚 API 参考

### Service 接口

```go
type Service interface {
    Propose(ctx context.Context, req *ProposeRequest, wallets ...wallet.Wallet) (*ProposeResult, error)
    Vote(ctx context.Context, req *VoteRequest, wallets ...wallet.Wallet) (*VoteResult, error)
    UpdateParam(ctx context.Context, req *UpdateParamRequest, wallets ...wallet.Wallet) (*UpdateParamResult, error)
}
```

---

## 📖 新路径流程

### Propose（创建提案）

1. 调用 `buildProposeDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### Vote（投票）

1. 调用 `buildVoteDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

### UpdateParam（更新参数）

1. 调用 `buildUpdateParamDraft` 构建交易草稿
2. 调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
3. 使用 `Wallet.SignHash` 对哈希进行签名
4. 调用 `wes_finalizeTransactionFromDraft` 生成完整交易
5. 调用 `wes_sendRawTransaction` 提交交易

## 🔄 迁移说明

### 旧路径（已废弃）

旧路径使用 `build*Transaction` 函数直接构建未签名交易，然后使用 `Wallet.SignTransaction` 签名：

```go
// ⚠️ 已废弃：不再使用
unsignedTxBytes, err := buildProposeTransaction(...)
signedTxBytes, err := wallet.SignTransaction(unsignedTxBytes)
```

### 新路径（推荐）

新路径使用 `build*Draft` + `wes_computeSignatureHashFromDraft` + `wes_finalizeTransactionFromDraft`：

```go
// ✅ 推荐：使用新路径
draftJSON, inputIndex, err := buildProposeDraft(...)
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

