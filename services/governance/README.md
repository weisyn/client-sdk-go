# Governance Service - 治理服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 基础结构完成  
**最后更新**: 2025-01-23

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

```
┌─────────────────────────────────────────┐
│      Governance Service 架构            │
└─────────────────────────────────────────┘

Governance Service
    │
    ├─> Propose: 创建提案
    ├─> Vote: 投票
    └─> UpdateParam: 更新参数
```

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

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

