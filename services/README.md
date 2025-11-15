# Services - 业务服务层

**版本**: 1.0.0-alpha  
**状态**: ✅ 核心功能已完成  
**最后更新**: 2025-01-23

---

## 📋 概述

业务服务层提供面向业务场景的高层 API，将底层交易复杂性抽象为直观的业务操作。所有服务都遵循统一的设计模式，使用 Wallet 接口进行签名，完全符合架构原则。

---

## 🏗️ 服务架构

### 服务分层图

```
┌─────────────────────────────────────────────────────────┐
│                  应用层调用                               │
│  tokenService.Transfer()                                 │
│  stakingService.Stake()                                  │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              业务服务层 (services/)                       │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ Token Service│  │Staking Service│ │Market Service│   │
│  │              │  │              │  │              │   │
│  │ • Transfer   │  │ • Stake      │  │ • SwapAMM    │   │
│  │ • BatchXfer  │  │ • Unstake    │  │ • AddLiq     │   │
│  │ • Mint       │  │ • Delegate   │  │ • Vesting    │   │
│  │ • Burn       │  │ • Claim      │  │ • Escrow     │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐                     │
│  │Governance    │  │Resource      │                     │
│  │Service       │  │Service       │                     │
│  │              │  │              │                     │
│  │ • Propose    │  │ • Deploy     │                     │
│  │ • Vote       │  │ • Query      │                     │
│  └──────────────┘  └──────────────┘                     │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              统一设计模式                                │
│                                                           │
│  1. 参数验证                                             │
│  2. Wallet 获取与验证                                    │
│  3. 业务逻辑（构建交易）                                 │
│  4. Wallet 签名                                          │
│  5. 提交交易 (wes_sendRawTransaction)                    │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              客户端层 (client/)                          │
│  • HTTP / gRPC / WebSocket                              │
└─────────────────────────────────────────────────────────┘
```

### 服务调用流程

```
┌─────────────────────────────────────────────────────────┐
│           业务服务调用流程 (以 Transfer 为例)            │
└─────────────────────────────────────────────────────────┘

应用层
  │
  ├─> tokenService.Transfer(ctx, req, wallet)
  │
  ↓
服务层 (services/token/transfer.go)
  │
  ├─> 1. 参数验证
  │   └─> validateTransferRequest(req)
  │
  ├─> 2. Wallet 验证
  │   └─> wallet.Address() == req.From
  │
  ├─> 3. 构建交易 (tx_builder.go)
  │   ├─> 查询 UTXO (wes_getUTXO)
  │   ├─> 选择 UTXO
  │   ├─> 计算手续费和找零
  │   ├─> 构建 DraftJSON
  │   └─> 调用 wes_buildTransaction
  │
  ├─> 4. Wallet 签名
  │   └─> wallet.SignTransaction(unsignedTx)
  │
  ├─> 5. 提交交易
  │   └─> client.SendRawTransaction(signedTxHex)
  │
  ↓
返回结果
  └─> TransferResult{TxHash, Success}
```

---

## 📦 服务列表

### 1. Token 服务 ✅

**路径**: `services/token/`  
**状态**: ✅ 核心功能已完成

**功能**:
- ✅ **Transfer** - 单笔转账（SDK 层构建交易）
- ✅ **BatchTransfer** - 批量转账（**限制：所有转账必须使用同一个 tokenID**）
- ✅ **Mint** - 代币铸造（调用 `wes_callContract`）
- ✅ **Burn** - 代币销毁（SDK 层构建交易）
- ✅ **GetBalance** - 余额查询

**架构说明**:
```
Token Service
    │
    ├─> Transfer: SDK 层构建交易
    │   └─> buildTransferTransaction() → wes_buildTransaction
    │
    ├─> BatchTransfer: SDK 层构建交易（同一 tokenID）
    │   └─> buildBatchTransferTransaction() → wes_buildTransaction
    │
    ├─> Mint: 调用合约
    │   └─> wes_callContract(return_unsigned_tx=true)
    │
    └─> Burn: SDK 层构建交易
        └─> buildBurnTransaction() → wes_buildTransaction
```

**使用示例**:
```go
tokenService := token.NewService(client)

// 单笔转账
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   fromAddr,
    To:     toAddr,
    Amount: 1000,
    TokenID: nil, // nil = 原生币
}, wallet)

// 批量转账（所有转账必须使用同一个 tokenID）
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: fromAddr,
    Transfers: []token.TransferItem{
        {To: addr1, Amount: 100, TokenID: tokenID},
        {To: addr2, Amount: 200, TokenID: tokenID}, // 必须相同
    },
}, wallet)
```

---

### 2. Staking 服务 ✅

**路径**: `services/staking/`  
**状态**: ✅ 基础结构完成

**功能**:
- ✅ **Stake** - 质押代币
- ✅ **Unstake** - 解除质押
- ✅ **Delegate** - 委托验证者
- ✅ **Undelegate** - 取消委托
- ✅ **ClaimReward** - 领取奖励
- ✅ **Slash** - 罚没（治理功能）

**使用示例**:
```go
stakingService := staking.NewService(client)

// 质押
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:     stakerAddr,
    Amount:   10000,
    Validator: validatorAddr,
}, wallet)
```

---

### 3. Market 服务 ✅

**路径**: `services/market/`  
**状态**: ✅ 基础结构完成

**功能**:
- ✅ **SwapAMM** - AMM 代币交换
- ✅ **AddLiquidity** - 添加流动性
- ✅ **RemoveLiquidity** - 移除流动性
- ✅ **CreateVesting** - 创建归属计划
- ✅ **ClaimVesting** - 领取归属代币
- ✅ **CreateEscrow** - 创建托管
- ✅ **ReleaseEscrow** - 释放托管
- ✅ **RefundEscrow** - 退款托管

---

### 4. Governance 服务 ✅

**路径**: `services/governance/`  
**状态**: ✅ 基础结构完成

**功能**:
- ✅ **Propose** - 创建提案
- ✅ **Vote** - 投票
- ✅ **UpdateParam** - 更新参数

---

### 5. Resource 服务 ✅

**路径**: `services/resource/`  
**状态**: ✅ 基础结构完成

**功能**:
- ✅ **DeployStaticResource** - 部署静态资源
- ✅ **DeployContract** - 部署智能合约
- ✅ **DeployAIModel** - 部署 AI 模型
- ✅ **GetResource** - 查询资源信息

---

## 🎯 统一设计模式

所有服务都遵循相同的设计模式：

### 1. Service 接口

```go
type Service interface {
    Method(ctx context.Context, req *Request, wallets ...wallet.Wallet) (*Result, error)
}
```

### 2. 服务结构

```go
type service struct {
    client client.Client
    wallet wallet.Wallet // 可选：默认 Wallet
}
```

### 3. 构造函数

```go
// 不带 Wallet
func NewService(client client.Client) Service

// 带默认 Wallet
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service
```

### 4. 方法实现模式

```go
func (s *service) method(ctx context.Context, req *Request, wallets ...wallet.Wallet) (*Result, error) {
    // 1. 参数验证
    if err := s.validateRequest(req); err != nil {
        return nil, err
    }

    // 2. 获取 Wallet
    w := s.getWallet(wallets...)
    if w == nil {
        return nil, fmt.Errorf("wallet is required")
    }

    // 3. 验证地址匹配
    if !bytes.Equal(w.Address(), req.From) {
        return nil, fmt.Errorf("wallet address does not match from address")
    }

    // 4. 业务逻辑（构建交易）
    unsignedTxBytes, err := buildTransaction(...)
    
    // 5. Wallet 签名
    signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
    
    // 6. 提交交易
    signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
    sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
    
    // 7. 返回结果
    return &Result{TxHash: sendResult.TxHash, Success: true}, nil
}
```

---

## 🔑 关键特性

### 1. Wallet 集成

```
┌─────────────────────────────────────────┐
│        Wallet 集成模式                   │
└─────────────────────────────────────────┘

方式1: 方法参数传递（推荐）
  tokenService.Transfer(ctx, req, wallet)

方式2: 构造函数设置
  tokenService := token.NewServiceWithWallet(client, wallet)
  tokenService.Transfer(ctx, req) // 使用默认 wallet

方式3: 混合使用
  tokenService := token.NewServiceWithWallet(client, defaultWallet)
  tokenService.Transfer(ctx, req1)        // 使用默认
  tokenService.Transfer(ctx, req2, tempWallet) // 临时切换
```

### 2. 交易构建策略

```
┌─────────────────────────────────────────┐
│        交易构建策略                      │
└─────────────────────────────────────────┘

UTXO 操作 (Transfer, Burn, BatchTransfer):
  SDK 层构建
    ├─> 查询 UTXO (wes_getUTXO)
    ├─> 选择 UTXO
    ├─> 构建 DraftJSON
    └─> 调用 wes_buildTransaction

合约调用 (Mint):
  调用节点 API
    └─> wes_callContract(return_unsigned_tx=true)
```

### 3. 批量转账限制

**重要**: 批量转账**必须**使用同一个 tokenID

```
┌─────────────────────────────────────────┐
│      批量转账 tokenID 限制               │
└─────────────────────────────────────────┘

✅ 正确:
  BatchTransferRequest{
    Transfers: []TransferItem{
      {To: addr1, Amount: 100, TokenID: tokenID},
      {To: addr2, Amount: 200, TokenID: tokenID}, // 相同
    }
  }

❌ 错误:
  BatchTransferRequest{
    Transfers: []TransferItem{
      {To: addr1, Amount: 100, TokenID: tokenID1},
      {To: addr2, Amount: 200, TokenID: tokenID2}, // 不同！
    }
  }
  // 会返回错误: "all transfers must use the same tokenID"
```

---

## 📊 服务状态统计

| 服务 | 方法数 | 状态 | 说明 |
|------|--------|------|------|
| Token | 5 | ✅ | Transfer, BatchTransfer, Mint, Burn, GetBalance |
| Staking | 6 | ✅ | Stake, Unstake, Delegate, Undelegate, ClaimReward, Slash |
| Market | 8 | ✅ | SwapAMM, AddLiquidity, RemoveLiquidity, CreateVesting, ClaimVesting, CreateEscrow, ReleaseEscrow, RefundEscrow |
| Governance | 3 | ✅ | Propose, Vote, UpdateParam |
| Resource | 4 | ✅ | DeployStaticResource, DeployContract, DeployAIModel, GetResource |

**总计**: 26 个业务方法

---

## 🔧 实现细节

### Token 服务交易构建

**Transfer**:
```
1. 查询 UTXO (wes_getUTXO)
2. 过滤匹配 tokenID 的 UTXO
3. 选择足够的 UTXO
4. 计算手续费（万分之三）
5. 计算找零
6. 构建 DraftJSON
7. 调用 wes_buildTransaction
```

**BatchTransfer**:
```
1. 验证所有转账使用同一个 tokenID
2. 查询 UTXO (wes_getUTXO)
3. 过滤匹配 tokenID 的 UTXO
4. 为每个转账选择 UTXO
5. 累计总输入和总输出
6. 计算手续费和找零（使用共同 tokenID）
7. 构建 DraftJSON
8. 调用 wes_buildTransaction
```

**Burn**:
```
1. 查询 UTXO (wes_getUTXO)
2. 过滤匹配 tokenID 的 UTXO
3. 选择足够的 UTXO
4. 计算手续费
5. 计算找零（如果有剩余）
6. 构建 DraftJSON（不创建输出或只创建找零）
7. 调用 wes_buildTransaction
```

---

## 📚 参考文档

- [主 README](../README.md) - SDK 总体文档
- [Wallet 文档](../wallet/README.md) - 钱包功能文档
- [架构设计文档](../../SDK_DESIGN.md) - SDK 设计文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team
