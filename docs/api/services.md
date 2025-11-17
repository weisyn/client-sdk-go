# Services API 参考

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

Services 提供业务语义接口，封装了完整的交易构建和提交流程。开发者只需关注业务参数，无需了解底层实现。

---

## 🔗 关联文档

- **业务指南**：[业务使用指南](../guides/)
- **底层 API**：[WES JSON-RPC API 参考](https://github.com/weisyn/weisyn/blob/main/docs/reference/api.md)

---

## 📦 导入

```go
import (
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/services/staking"
    "github.com/weisyn/client-sdk-go/services/market"
    "github.com/weisyn/client-sdk-go/services/governance"
    "github.com/weisyn/client-sdk-go/services/resource"
)
```

---

## 🏗️ 服务概览

| 服务 | 职责 | 主要方法 |
|------|------|---------|
| **TokenService** | 代币操作 | `Transfer`, `BatchTransfer`, `Mint`, `Burn`, `GetBalance` |
| **StakingService** | 质押操作 | `Stake`, `Unstake`, `Delegate`, `Undelegate`, `ClaimReward` |
| **MarketService** | 市场操作 | `SwapAMM`, `AddLiquidity`, `RemoveLiquidity`, `CreateEscrow`, `CreateVesting` |
| **GovernanceService** | 治理操作 | `Propose`, `Vote`, `UpdateParam` |
| **ResourceService** | 资源操作 | `DeployContract`, `DeployAIModel`, `DeployStaticResource`, `GetResource` |

---

## 💰 Token Service

### 创建服务

```go
tokenService := token.NewTokenService(client, wallet)
```

### Transfer() - 转账

```go
func (s *TokenService) Transfer(
    ctx context.Context,
    req *TransferRequest,
    wallet wallet.Wallet,
) (*TransferResult, error)
```

**参数**：
- `req.From`: 发送方地址（`[20]byte`）
- `req.To`: 接收方地址（`[20]byte`）
- `req.Amount`: 金额（`*big.Int`）
- `req.TokenID`: 代币 ID（`[32]byte`，`nil` 表示原生币）

**返回**：
- `TxHash`: 交易哈希
- `Success`: 是否成功

**示例**：
```go
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   wallet.Address(),
    To:     recipient,
    Amount: big.NewInt(1000000),
    TokenID: nil, // 原生币
}, wallet)
```

**关联 JSON-RPC**：
- `wes_getUTXO` - 查询输入 UTXO
- `wes_buildTransaction` - 构建交易
- `wes_sendRawTransaction` - 发送交易

---

### BatchTransfer() - 批量转账

```go
func (s *TokenService) BatchTransfer(
    ctx context.Context,
    req *BatchTransferRequest,
    wallet wallet.Wallet,
) (*BatchTransferResult, error)
```

**参数**：
- `req.From`: 发送方地址
- `req.Transfers`: 转账列表（所有转账必须使用同一个 `TokenID`）
  - `To`: 接收方地址
  - `Amount`: 金额
- `req.TokenID`: 代币 ID（所有转账共享）

**示例**：
```go
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: wallet.Address(),
    Transfers: []token.TransferItem{
        {To: addr1, Amount: big.NewInt(100000)},
        {To: addr2, Amount: big.NewInt(200000)},
    },
    TokenID: tokenID, // 所有转账使用同一个 tokenID
}, wallet)
```

---

### Mint() - 代币铸造

```go
func (s *TokenService) Mint(
    ctx context.Context,
    req *MintRequest,
    wallet wallet.Wallet,
) (*MintResult, error)
```

**参数**：
- `req.To`: 接收方地址
- `req.Amount`: 铸造数量
- `req.TokenID`: 代币 ID
- `req.ContractAddr`: 合约地址（代币合约）

**关联 JSON-RPC**：
- `wes_callContract` - 调用代币合约的 `mint` 方法

---

### Burn() - 代币销毁

```go
func (s *TokenService) Burn(
    ctx context.Context,
    req *BurnRequest,
    wallet wallet.Wallet,
) (*BurnResult, error)
```

**参数**：
- `req.From`: 销毁方地址
- `req.Amount`: 销毁数量
- `req.TokenID`: 代币 ID
- `req.ContractAddr`: 合约地址（代币合约）

---

### GetBalance() - 查询余额

```go
func (s *TokenService) GetBalance(
    ctx context.Context,
    address [20]byte,
    tokenID *[32]byte,
) (*big.Int, error)
```

**参数**：
- `address`: 地址（`[20]byte`）
- `tokenID`: 代币 ID（`nil` 表示原生币）

**返回**：`*big.Int` - 余额

**示例**：
```go
// 查询原生币余额
balance, err := tokenService.GetBalance(ctx, wallet.Address(), nil)

// 查询代币余额
tokenBalance, err := tokenService.GetBalance(ctx, wallet.Address(), tokenID)
```

**关联 JSON-RPC**：
- `wes_getUTXO` - 查询 UTXO 并汇总余额

---

## 🏛️ Staking Service

### Stake() - 质押

```go
func (s *StakingService) Stake(
    ctx context.Context,
    req *StakeRequest,
    wallet wallet.Wallet,
) (*StakeResult, error)
```

**参数**：
- `req.From`: 质押者地址
- `req.ValidatorAddr`: 验证者地址
- `req.Amount`: 质押金额
- `req.LockBlocks`: 锁定期（区块数，可选）

**返回**：
- `TxHash`: 交易哈希
- `StakeID`: 质押 ID（用于后续操作）

**关联 JSON-RPC**：
- `wes_buildTransaction` - 构建质押交易（使用 ContractLock + HeightLock）

---

### Unstake() - 解质押

```go
func (s *StakingService) Unstake(
    ctx context.Context,
    req *UnstakeRequest,
    wallet wallet.Wallet,
) (*UnstakeResult, error)
```

**参数**：
- `req.From`: 质押者地址
- `req.StakeID`: 质押 ID

**返回**：
- `TxHash`: 交易哈希
- `Amount`: 解质押金额
- `Reward`: 奖励金额

---

### Delegate() - 委托

```go
func (s *StakingService) Delegate(
    ctx context.Context,
    req *DelegateRequest,
    wallet wallet.Wallet,
) (*DelegateResult, error)
```

**参数**：
- `req.From`: 委托者地址
- `req.ValidatorAddr`: 验证者地址
- `req.Amount`: 委托金额

**返回**：
- `TxHash`: 交易哈希
- `DelegateID`: 委托 ID

---

### ClaimReward() - 领取奖励

```go
func (s *StakingService) ClaimReward(
    ctx context.Context,
    req *ClaimRewardRequest,
    wallet wallet.Wallet,
) (*ClaimRewardResult, error)
```

**参数**：
- `req.From`: 质押者/委托者地址
- `req.StakeID`: 质押 ID（可选）
- `req.DelegateID`: 委托 ID（可选）

**返回**：
- `TxHash`: 交易哈希
- `Reward`: 奖励金额

---

## 🏪 Market Service

### SwapAMM() - AMM 代币交换

```go
func (s *MarketService) SwapAMM(
    ctx context.Context,
    req *SwapAMMRequest,
    wallet wallet.Wallet,
) (*SwapAMMResult, error)
```

**参数**：
- `req.From`: 交换者地址
- `req.ContractAddr`: AMM 合约地址
- `req.TokenIn`: 输入代币 ID
- `req.AmountIn`: 输入金额
- `req.TokenOut`: 输出代币 ID
- `req.AmountOutMin`: 最小输出金额（滑点保护）

**关联 JSON-RPC**：
- `wes_callContract` - 调用 AMM 合约的 `swap` 方法

---

### CreateEscrow() - 创建托管

```go
func (s *MarketService) CreateEscrow(
    ctx context.Context,
    req *CreateEscrowRequest,
    wallet wallet.Wallet,
) (*CreateEscrowResult, error)
```

**参数**：
- `req.From`: 买方地址
- `req.Seller`: 卖方地址
- `req.Amount`: 托管金额
- `req.TokenID`: 代币 ID（`nil` 表示原生币）

**返回**：
- `TxHash`: 交易哈希
- `EscrowID`: 托管 ID

**关联 JSON-RPC**：
- `wes_buildTransaction` - 构建托管交易（使用 MultiKeyLock）

---

### CreateVesting() - 创建归属计划

```go
func (s *MarketService) CreateVesting(
    ctx context.Context,
    req *CreateVestingRequest,
    wallet wallet.Wallet,
) (*CreateVestingResult, error)
```

**参数**：
- `req.From`: 创建者地址
- `req.Recipient`: 接收者地址
- `req.Amount`: 总金额
- `req.TokenID`: 代币 ID
- `req.UnlockTime`: 解锁时间（Unix 时间戳）

**关联 JSON-RPC**：
- `wes_buildTransaction` - 构建归属交易（使用 TimeLock + SingleKeyLock）

---

## 🗳️ Governance Service

### Propose() - 创建提案

```go
func (s *GovernanceService) Propose(
    ctx context.Context,
    req *ProposeRequest,
    wallet wallet.Wallet,
) (*ProposeResult, error)
```

**参数**：
- `req.Proposer`: 提案者地址
- `req.ProposalData`: 提案数据
  - `Title`: 提案标题
  - `Description`: 提案描述
  - `Action`: 提案类型
  - `Params`: 提案参数

**返回**：
- `TxHash`: 交易哈希
- `ProposalID`: 提案 ID（stateID）

**关联 JSON-RPC**：
- `wes_buildTransaction` - 构建提案交易（使用 StateOutput）

---

### Vote() - 投票

```go
func (s *GovernanceService) Vote(
    ctx context.Context,
    req *VoteRequest,
    wallet wallet.Wallet,
) (*VoteResult, error)
```

**参数**：
- `req.Voter`: 投票者地址
- `req.ProposalID`: 提案 ID
- `req.Choice`: 投票选择（1=支持, 0=反对, -1=弃权）
- `req.Weight`: 投票权重

**返回**：
- `TxHash`: 交易哈希
- `VoteID`: 投票 ID

---

## 📦 Resource Service

### DeployContract() - 部署智能合约

```go
func (s *ResourceService) DeployContract(
    ctx context.Context,
    req *DeployContractRequest,
    wallet wallet.Wallet,
) (*DeployContractResult, error)
```

**参数**：
- `req.From`: 部署者地址
- `req.WasmBytes`: WASM 字节码（`[]byte`）
- `req.Name`: 合约名称（可选）
- `req.Description`: 合约描述（可选）

**返回**：
- `TxHash`: 交易哈希
- `ContractID`: 合约 ID（资源哈希）

**关联 JSON-RPC**：
- `wes_deployResource` - 部署资源

---

### DeployAIModel() - 部署 AI 模型

```go
func (s *ResourceService) DeployAIModel(
    ctx context.Context,
    req *DeployAIModelRequest,
    wallet wallet.Wallet,
) (*DeployAIModelResult, error)
```

**参数**：
- `req.From`: 部署者地址
- `req.ModelBytes`: ONNX 模型字节码（`[]byte`）
- `req.Name`: 模型名称（可选）
- `req.Framework`: 框架（如 "ONNX"）

**关联 JSON-RPC**：
- `wes_deployResource` - 部署资源

---

### DeployStaticResource() - 部署静态资源

```go
func (s *ResourceService) DeployStaticResource(
    ctx context.Context,
    req *DeployStaticResourceRequest,
    wallet wallet.Wallet,
) (*DeployStaticResourceResult, error)
```

**参数**：
- `req.From`: 部署者地址
- `req.FileContent`: 文件内容（`[]byte`）
- `req.MimeType`: MIME 类型（如 "image/png"）

---

### GetResource() - 查询资源

```go
func (s *ResourceService) GetResource(
    ctx context.Context,
    resourceID [32]byte,
) (*ResourceInfo, error)
```

**参数**：
- `resourceID`: 资源 ID（32 字节哈希）

**返回**：
- `ResourceID`: 资源 ID
- `Type`: 资源类型（"contract" | "model" | "static"）
- `Size`: 资源大小（字节）
- `MimeType`: MIME 类型（静态资源）

**注意**：此方法不需要 Wallet

---

## 🔗 相关文档

- **[Token 指南](../guides/token.md)** - Token 服务详细指南
- **[Staking 指南](../guides/staking.md)** - Staking 服务详细指南
- **[Market 指南](../guides/market.md)** - Market 服务详细指南
- **[Governance 指南](../guides/governance.md)** - Governance 服务详细指南
- **[Resource 指南](../guides/resource.md)** - Resource 服务详细指南

---

**最后更新**: 2025-11-17

