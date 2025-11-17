# 概述

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

本文档从 SDK 视角解释 WES 的核心概念，帮助开发者理解如何在 Go 中使用 WES。

---

## 🔗 关联文档

- **WES 系统架构**：[WES 系统架构文档](https://github.com/weisyn/weisyn/blob/main/docs/system/architecture/README.md)
- **快速开始**：[快速开始指南](./getting-started.md)

---

## 🏗️ WES 核心概念（SDK 视角）

### UTXO 模型

WES 使用 UTXO（未花费交易输出）模型：

```go
// UTXO 表示一个可花费的输出
type UTXO struct {
    Outpoint Outpoint  // 交易输出索引
    Output   Output    // 输出内容（金额、锁定条件等）
}

// 查询 UTXO
utxos, err := client.GetUTXO(ctx, address)
```

**SDK 封装**：
- `client.GetUTXO()` - 查询地址的 UTXO
- `services` 自动选择 UTXO 构建交易

---

### 锁定条件

WES 支持 7 种锁定条件：

| 锁定条件 | Go 类型 | 用途 |
|---------|---------|------|
| `SingleKeyLock` | `SingleKeyLock` | 单签名锁定 |
| `MultiKeyLock` | `MultiKeyLock` | 多签名锁定 |
| `ContractLock` | `ContractLock` | 合约锁定 |
| `DelegationLock` | `DelegationLock` | 委托锁定 |
| `ThresholdLock` | `ThresholdLock` | 阈值锁定 |
| `TimeLock` | `TimeLock` | 时间锁定 |
| `HeightLock` | `HeightLock` | 高度锁定 |

**SDK 封装**：
- `services` 自动选择合适的锁定条件
- 开发者无需直接操作锁定条件

---

### 交易构建流程

```go
// 1. 查询 UTXO
utxos, err := client.GetUTXO(ctx, fromAddress)

// 2. 构建交易草稿
draft := &tx.Draft{
    Inputs: []tx.Input{...},
    Outputs: []tx.Output{...},
}

// 3. 构建未签名交易
unsignedTx, err := client.BuildTransaction(ctx, draft)

// 4. 签名交易
signature := wallet.SignTransaction(unsignedTx)

// 5. 完成交易
signedTx, err := client.FinalizeTransaction(ctx, draft, []Signature{signature})

// 6. 提交交易
txHash, err := client.SendRawTransaction(ctx, signedTx)
```

**SDK 封装**：
- `services` 自动完成上述流程
- 开发者只需调用业务方法

---

### 业务服务

SDK 提供 5 个业务服务：

| 服务 | Go 包 | 职责 |
|------|-------|------|
| **Token** | `services/token` | 代币操作（转账、铸造、销毁） |
| **Staking** | `services/staking` | 质押操作（质押、委托、奖励） |
| **Market** | `services/market` | 市场操作（AMM、流动性、托管） |
| **Governance** | `services/governance` | 治理操作（提案、投票） |
| **Resource** | `services/resource` | 资源操作（合约/模型部署） |

**使用示例**：
```go
// Token 服务
tokenService := token.NewTokenService(client, wallet)
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   fromAddress,
    To:     toAddress,
    Amount: amount,
    TokenID: nil,
})

// Staking 服务
stakingService := staking.NewStakingService(client, wallet)
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:        fromAddress,
    ValidatorAddr: validatorAddress,
    Amount:      amount,
})
```

---

## 🔄 JSON-RPC 方法映射

SDK 封装了底层 JSON-RPC 方法：

| JSON-RPC 方法 | SDK 方法 | 说明 |
|--------------|---------|------|
| `wes_getUTXO` | `client.GetUTXO()` | 查询 UTXO |
| `wes_buildTransaction` | `client.BuildTransaction()` | 构建交易 |
| `wes_computeSignatureHashFromDraft` | `client.ComputeSignatureHash()` | 计算签名哈希 |
| `wes_finalizeTransactionFromDraft` | `client.FinalizeTransaction()` | 完成交易 |
| `wes_sendRawTransaction` | `client.SendRawTransaction()` | 发送交易 |
| `wes_callContract` | `client.CallContract()` | 调用合约 |

**SDK 封装**：
- 业务服务自动调用底层 JSON-RPC 方法
- 开发者无需直接调用 JSON-RPC

---

## 🎯 典型使用流程

### 1. 初始化

```go
// 创建客户端
client, err := client.NewClient(&client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
})

// 创建或导入钱包
wallet, err := wallet.NewWallet()
```

### 2. 使用业务服务

```go
// 创建服务实例
tokenService := token.NewTokenService(client, wallet)

// 调用业务方法
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   wallet.Address(),
    To:     recipientAddress,
    Amount: amount,
    TokenID: nil,
})
```

### 3. 错误处理

```go
result, err := tokenService.Transfer(ctx, req)
if err != nil {
    switch e := err.(type) {
    case *client.NetworkError:
        // 网络错误
    case *client.TransactionError:
        // 交易错误
    case *client.ValidationError:
        // 参数验证错误
    default:
        // 其他错误
    }
}
```

---

## 🔗 相关文档

- **[快速开始](./getting-started.md)** - 安装和配置
- **[架构设计](./architecture.md)** - SDK 内部架构
- **[API 参考](./api/)** - 完整 API 文档

---

**最后更新**: 2025-11-17

