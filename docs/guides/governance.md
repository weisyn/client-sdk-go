# Governance 服务指南

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

Governance Service 提供治理相关功能，包括提案创建、投票和参数更新。

---

## 🔗 关联文档

- **API 参考**：[Services API - Governance](../api/services.md#-governance-service)
- **WES 协议**：[WES 治理机制](https://github.com/weisyn/weisyn/blob/main/docs/system/platforms/governance/README.md)（待确认）

---

## 🚀 快速开始

### 创建服务

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/governance"
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

governanceService := governance.NewService(cli)
```

---

## 📝 创建提案

### 基本提案

```go
ctx := context.Background()

result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
    Title:   "增加最小质押金额",
    Content: "建议将最小质押金额从 1000 增加到 5000",
    Type:    governance.ProposalTypeParameterChange,
    Metadata: map[string]string{
        "param_key":   "min_stake_amount",
        "param_value": "5000",
    },
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("提案创建成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("提案 ID: %s\n", result.ProposalID)
```

### 提案类型

```go
const (
    ProposalTypeParameterChange   = "ParameterChange"   // 参数变更
    ProposalTypeContractUpgrade   = "ContractUpgrade"   // 合约升级
    ProposalTypeResourceDeployment = "ResourceDeployment" // 资源部署
    ProposalTypeOther            = "Other"             // 其他
)
```

---

## 🗳️ 投票

### 基本投票

```go
result, err := governanceService.Vote(ctx, &governance.VoteRequest{
    ProposalID: proposalID,
    Support:    true, // true = 支持, false = 反对
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("投票成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("投票 ID: %s\n", result.VoteID)
```

### 投票选择

```go
// 支持
supportResult, err := governanceService.Vote(ctx, &governance.VoteRequest{
    ProposalID: proposalID,
    Support:    true,
}, w)

// 反对
againstResult, err := governanceService.Vote(ctx, &governance.VoteRequest{
    ProposalID: proposalID,
    Support:    false,
}, w)
```

---

## ⚙️ 参数更新

### 更新治理参数

```go
result, err := governanceService.UpdateParam(ctx, &governance.UpdateParamRequest{
    Key:   "min_stake_amount",
    Value: "5000",
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("参数更新成功！交易哈希: %s\n", result.TxHash)
```

### 注意事项

- ⚠️ 参数更新通常需要治理提案通过后才能执行
- ✅ SDK 只负责提交参数更新交易，不负责验证治理权限

---

## 🎯 典型场景

### 场景 1：完整的治理流程

```go
func completeGovernanceFlow(
    ctx context.Context,
    proposerWallet, voterWallet wallet.Wallet,
    governanceService governance.Service,
) error {
    // 1. 创建提案
    proposalResult, err := governanceService.Propose(ctx, &governance.ProposeRequest{
        Title:   "更新最小质押金额",
        Content: "建议将最小质押金额从 1000 增加到 5000",
        Type:    governance.ProposalTypeParameterChange,
        Metadata: map[string]string{
            "param_key":   "min_stake_amount",
            "param_value": "5000",
        },
    }, proposerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("提案 ID: %s\n", proposalResult.ProposalID)
    
    // 2. 投票
    voteResult, err := governanceService.Vote(ctx, &governance.VoteRequest{
        ProposalID: proposalResult.ProposalID,
        Support:    true,
    }, voterWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("投票 ID: %s\n", voteResult.VoteID)
    
    // 3. 等待投票期结束后，执行参数更新
    // ... 等待投票期结束 ...
    
    updateResult, err := governanceService.UpdateParam(ctx, &governance.UpdateParamRequest{
        Key:   "min_stake_amount",
        Value: "5000",
    }, proposerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("参数已更新\n")
    return nil
}
```

---

## ⚠️ 常见错误

### 提案已存在

```go
result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
    Title:   "重复提案",
    Content: "...",
    Type:    governance.ProposalTypeParameterChange,
}, w)
if err != nil {
    if strings.Contains(err.Error(), "proposal already exists") {
        log.Fatal("提案已存在")
    }
    log.Fatal(err)
}
```

### 投票已存在

```go
result, err := governanceService.Vote(ctx, &governance.VoteRequest{
    ProposalID: proposalID,
    Support:    true,
}, w)
if err != nil {
    if strings.Contains(err.Error(), "vote already exists") {
        log.Fatal("已投票，不能重复投票")
    }
    log.Fatal(err)
}
```

---

## 🔗 相关文档

- **[API 参考](../api/services.md#-governance-service)** - 完整 API 文档
- **[Staking 指南](./staking.md)** - 质押服务指南
- **[故障排查](../troubleshooting.md)** - 常见问题

---

