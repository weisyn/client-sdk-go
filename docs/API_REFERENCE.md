# Client SDK Go - API 参考

**版本**: v1.0.0  
**最后更新**: 2025-01-23

---

## 📋 文档定位

> 📌 **重要说明**：本文档提供 **Go SDK API 的详细参考**。  
> 如需了解底层 JSON-RPC API 规范，请参考主仓库文档。

**本文档目标**：
- 提供完整的 API 接口说明
- 包含参数、返回值、使用示例
- 按模块组织（WESClient、业务服务、钱包等）

---

## 📚 API 概览

### WESClient 类型化 API

`WESClient` 提供类型化的 RPC 封装，是所有服务的基础：

```go
type WESClient interface {
    // UTXO 操作
    ListUTXOs(ctx context.Context, address []byte) ([]*UTXO, error)
    
    // 资源操作
    GetResource(ctx context.Context, resourceID [32]byte) (*ResourceInfo, error)
    GetResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error)
    
    // 交易操作
    GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error)
    GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error)
    SubmitTransaction(ctx context.Context, tx *Transaction) (*SubmitTxResult, error)
    
    // 事件操作
    GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error)
    SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error)
    
    // 节点信息
    GetNodeInfo(ctx context.Context) (*NodeInfo, error)
    
    // 连接管理
    Close() error
}
```

### 业务服务 API

- [Token 服务](#token-服务)
- [Staking 服务](#staking-服务)
- [Market 服务](#market-服务)
- [Governance 服务](#governance-服务)
- [Resource 服务](#resource-服务)
- [Transaction 服务](#transaction-服务)
- [Event 服务](#event-服务)

---

## 🔧 详细 API 文档

### WESClient 类型化 API

#### ListUTXOs

查询指定地址下的所有 UTXO。

```go
func (c *wesClient) ListUTXOs(ctx context.Context, address []byte) ([]*UTXO, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `address []byte` - 地址（20 字节）

**返回值**：
- `[]*UTXO` - UTXO 列表
- `error` - 错误

**示例**：

```go
utxos, err := wesClient.ListUTXOs(ctx, address)
if err != nil {
    log.Fatal(err)
}

for _, utxo := range utxos {
    fmt.Printf("UTXO: %s:%d, 金额: %d\n", utxo.TxID, utxo.OutputIndex, utxo.Amount)
}
```

#### GetResource

查询单个资源信息。

```go
func (c *wesClient) GetResource(ctx context.Context, resourceID [32]byte) (*ResourceInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `resourceID [32]byte` - 资源 ID（32 字节）

**返回值**：
- `*ResourceInfo` - 资源信息
- `error` - 错误

**示例**：

```go
resource, err := wesClient.GetResource(ctx, resourceID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("资源类型: %s, 所有者: %x\n", resource.ResourceType, resource.Owner)
```

#### GetResources

查询资源列表（支持过滤）。

```go
func (c *wesClient) GetResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `filters *ResourceFilters` - 过滤条件

**ResourceFilters 结构**：

```go
type ResourceFilters struct {
    ResourceType *ResourceType // 资源类型（可选）
    Owner        *[20]byte     // 所有者地址（可选）
    Limit        int           // 限制数量
    Offset       int           // 偏移量
}
```

**返回值**：
- `[]*ResourceInfo` - 资源列表
- `error` - 错误

**示例**：

```go
resources, err := wesClient.GetResources(ctx, &ResourceFilters{
    ResourceType: &ResourceTypeContract,
    Limit:        20,
    Offset:       0,
})
```

#### GetTransaction

查询单个交易信息。

```go
func (c *wesClient) GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `txID string` - 交易 ID

**返回值**：
- `*TransactionInfo` - 交易信息
- `error` - 错误

#### GetTransactionHistory

查询交易历史（支持过滤）。

```go
func (c *wesClient) GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `filters *TransactionFilters` - 过滤条件

**TransactionFilters 结构**：

```go
type TransactionFilters struct {
    ResourceID *[32]byte // 资源 ID（可选）
    TxID       *string   // 交易 ID（可选）
    Limit      int       // 限制数量
    Offset     int       // 偏移量
}
```

**返回值**：
- `[]*TransactionInfo` - 交易列表
- `error` - 错误

#### SubmitTransaction

提交已签名的交易。

```go
func (c *wesClient) SubmitTransaction(ctx context.Context, tx *Transaction) (*SubmitTxResult, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `tx *Transaction` - 已签名的交易

**返回值**：
- `*SubmitTxResult` - 提交结果（包含交易哈希）
- `error` - 错误

#### GetEvents

查询事件列表（支持过滤）。

```go
func (c *wesClient) GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `filters *EventFilters` - 过滤条件

**EventFilters 结构**：

```go
type EventFilters struct {
    ResourceID *[32]byte // 资源 ID（可选）
    EventName  *string   // 事件名称（可选）
    Limit      int       // 限制数量
    Offset     int       // 偏移量
}
```

**返回值**：
- `[]*EventInfo` - 事件列表
- `error` - 错误

#### SubscribeEvents

订阅事件（WebSocket）。

```go
func (c *wesClient) SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `filters *EventFilters` - 过滤条件

**返回值**：
- `<-chan *EventInfo` - 事件通道
- `error` - 错误

**示例**：

```go
events, err := wesClient.SubscribeEvents(ctx, &EventFilters{
    ResourceID: &resourceID,
    EventName:  &eventName,
})

if err != nil {
    log.Fatal(err)
}

for event := range events {
    fmt.Printf("收到事件: %s, 数据: %x\n", event.Topic, event.Data)
}
```

#### GetNodeInfo

获取节点信息。

```go
func (c *wesClient) GetNodeInfo(ctx context.Context) (*NodeInfo, error)
```

**返回值**：
- `*NodeInfo` - 节点信息
- `error` - 错误

**NodeInfo 结构**：

```go
type NodeInfo struct {
    RPCVersion  string
    ChainID     string
    BlockHeight uint64
}
```

---

### Token 服务

#### Transfer

单笔转账。

```go
func (s *tokenService) Transfer(ctx context.Context, req *TransferRequest, wallet wallet.Wallet) (*TransferResult, error)
```

**TransferRequest 结构**：

```go
type TransferRequest struct {
    From    []byte  // 发送方地址
    To      []byte  // 接收方地址
    Amount  uint64  // 金额
    TokenID []byte  // 代币 ID（nil 表示原生币）
}
```

**返回值**：
- `*TransferResult` - 转账结果（包含交易哈希）
- `error` - 错误

#### BatchTransfer

批量转账（所有转账必须使用同一个 tokenID）。

```go
func (s *tokenService) BatchTransfer(ctx context.Context, req *BatchTransferRequest, wallet wallet.Wallet) (*BatchTransferResult, error)
```

**BatchTransferRequest 结构**：

```go
type BatchTransferRequest struct {
    From     []byte         // 发送方地址
    Transfers []TransferItem // 转账列表
}

type TransferItem struct {
    To      []byte // 接收方地址
    Amount  uint64 // 金额
    TokenID []byte // 代币 ID（必须相同）
}
```

#### Mint

代币铸造。

```go
func (s *tokenService) Mint(ctx context.Context, req *MintRequest, wallet wallet.Wallet) (*MintResult, error)
```

**MintRequest 结构**：

```go
type MintRequest struct {
    To          []byte // 接收方地址
    Amount      uint64 // 金额
    TokenID     []byte // 代币 ID
    ContractAddr []byte // 合约地址
}
```

#### Burn

代币销毁。

```go
func (s *tokenService) Burn(ctx context.Context, req *BurnRequest, wallet wallet.Wallet) (*BurnResult, error)
```

**BurnRequest 结构**：

```go
type BurnRequest struct {
    From    []byte // 发送方地址
    Amount  uint64 // 金额
    TokenID []byte // 代币 ID
}
```

#### GetBalance

查询余额。

```go
func (s *tokenService) GetBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error)
```

**参数**：
- `ctx context.Context` - 上下文
- `address []byte` - 地址
- `tokenID []byte` - 代币 ID（nil 表示原生币）

**返回值**：
- `uint64` - 余额
- `error` - 错误

---

### Staking 服务

#### Stake

质押代币。

```go
func (s *stakingService) Stake(ctx context.Context, req *StakeRequest, wallet wallet.Wallet) (*StakeResult, error)
```

**StakeRequest 结构**：

```go
type StakeRequest struct {
    From      []byte // 质押方地址
    Amount    uint64 // 金额
    Validator []byte // 验证者地址
}
```

#### Unstake

解除质押。

```go
func (s *stakingService) Unstake(ctx context.Context, req *UnstakeRequest, wallet wallet.Wallet) (*UnstakeResult, error)
```

#### Delegate

委托验证者。

```go
func (s *stakingService) Delegate(ctx context.Context, req *DelegateRequest, wallet wallet.Wallet) (*DelegateResult, error)
```

#### Undelegate

取消委托。

```go
func (s *stakingService) Undelegate(ctx context.Context, req *UndelegateRequest, wallet wallet.Wallet) (*UndelegateResult, error)
```

#### ClaimReward

领取奖励。

```go
func (s *stakingService) ClaimReward(ctx context.Context, req *ClaimRewardRequest, wallet wallet.Wallet) (*ClaimRewardResult, error)
```

---

### Market 服务

#### SwapAMM

AMM 代币交换。

```go
func (s *marketService) SwapAMM(ctx context.Context, req *SwapAMMRequest, wallet wallet.Wallet) (*SwapAMMResult, error)
```

**SwapAMMRequest 结构**：

```go
type SwapAMMRequest struct {
    ContractAddr string // AMM 合约地址
    TokenIn       []byte // 输入代币 ID
    AmountIn      uint64 // 输入金额
    TokenOut      []byte // 输出代币 ID
    MinAmountOut  uint64 // 最小输出金额（滑点保护）
}
```

#### AddLiquidity

添加流动性。

```go
func (s *marketService) AddLiquidity(ctx context.Context, req *AddLiquidityRequest, wallet wallet.Wallet) (*AddLiquidityResult, error)
```

#### RemoveLiquidity

移除流动性。

```go
func (s *marketService) RemoveLiquidity(ctx context.Context, req *RemoveLiquidityRequest, wallet wallet.Wallet) (*RemoveLiquidityResult, error)
```

#### CreateVesting

创建归属计划。

```go
func (s *marketService) CreateVesting(ctx context.Context, req *CreateVestingRequest, wallet wallet.Wallet) (*CreateVestingResult, error)
```

#### CreateEscrow

创建托管。

```go
func (s *marketService) CreateEscrow(ctx context.Context, req *CreateEscrowRequest, wallet wallet.Wallet) (*CreateEscrowResult, error)
```

---

### Governance 服务

#### Propose

创建提案。

```go
func (s *governanceService) Propose(ctx context.Context, req *ProposeRequest, wallet wallet.Wallet) (*ProposeResult, error)
```

**ProposeRequest 结构**：

```go
type ProposeRequest struct {
    Title   string // 提案标题
    Content string // 提案内容
    Type    ProposalType // 提案类型
}
```

#### Vote

投票。

```go
func (s *governanceService) Vote(ctx context.Context, req *VoteRequest, wallet wallet.Wallet) (*VoteResult, error)
```

**VoteRequest 结构**：

```go
type VoteRequest struct {
    ProposalID string // 提案 ID
    Support    bool   // true = 支持, false = 反对
}
```

#### UpdateParam

更新参数。

```go
func (s *governanceService) UpdateParam(ctx context.Context, req *UpdateParamRequest, wallet wallet.Wallet) (*UpdateParamResult, error)
```

---

### Resource 服务

#### GetResource

查询单个资源信息。

```go
func (s *resourceService) GetResource(ctx context.Context, resourceID [32]byte) (*ResourceInfo, error)
```

#### GetResources

查询资源列表。

```go
func (s *resourceService) GetResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error)
```

#### DeployContract

部署智能合约（支持锁定条件）。

```go
func (s *resourceService) DeployContract(ctx context.Context, req *DeployContractRequest, wallet wallet.Wallet) (*DeployContractResult, error)
```

**DeployContractRequest 结构**：

```go
type DeployContractRequest struct {
    From                []byte            // 部署方地址
    WasmPath            string            // WASM 文件路径（可选）
    WasmContent         []byte            // WASM 文件内容
    ContractName        string            // 合约名称
    InitArgs            []byte            // 初始化参数
    
    // ✅ 锁定条件列表（支持 7 种类型）
    LockingConditions   []LockingCondition
    
    // ✅ 锁定条件验证选项
    ValidateLockingConditions bool  // 是否在SDK层验证（默认true）
    AllowContractLockCycles    bool  // 是否允许ContractLock循环（默认false）
}
```

**LockingCondition 结构**：

```go
type LockingCondition struct {
    Type LockType  // 锁定类型（SingleKey/MultiKey/Contract/Delegation/Threshold/Time/Height）
    Keys [][]byte // 密钥列表（SingleKey/MultiKey）
    // ... 其他字段根据类型不同
}
```

#### DeployAIModel

部署 AI 模型。

```go
func (s *resourceService) DeployAIModel(ctx context.Context, req *DeployAIModelRequest, wallet wallet.Wallet) (*DeployAIModelResult, error)
```

#### DeployStaticResource

部署静态资源。

```go
func (s *resourceService) DeployStaticResource(ctx context.Context, req *DeployStaticResourceRequest, wallet wallet.Wallet) (*DeployStaticResourceResult, error)
```

---

### Transaction 服务

#### GetTransaction

查询单个交易信息。

```go
func (s *transactionService) GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error)
```

#### GetTransactionHistory

查询交易历史。

```go
func (s *transactionService) GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error)
```

#### SubmitTransaction

提交交易。

```go
func (s *transactionService) SubmitTransaction(ctx context.Context, tx *Transaction, wallet wallet.Wallet) (*SubmitTxResult, error)
```

---

### Event 服务

#### GetEvents

查询事件列表。

```go
func (s *eventService) GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error)
```

#### SubscribeEvents

订阅事件。

```go
func (s *eventService) SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error)
```

---

### Wallet 功能

#### NewWallet

创建新钱包。

```go
func NewWallet() (Wallet, error)
```

#### NewWalletFromPrivateKey

从私钥创建钱包。

```go
func NewWalletFromPrivateKey(privateKeyHex string) (Wallet, error)
```

#### LoadFromKeystore

从 Keystore 加载钱包。

```go
func LoadFromKeystore(keystorePath string, password string) (Wallet, error)
```

#### Address

获取地址。

```go
func (w *wallet) Address() []byte
```

#### SignTransaction

签名交易。

```go
func (w *wallet) SignTransaction(unsignedTxBytes []byte) ([]byte, error)
```

#### SignMessage

签名消息。

```go
func (w *wallet) SignMessage(messageBytes []byte) ([]byte, error)
```

---

## 🔗 相关文档

- [开发者指南](./DEVELOPER_GUIDE.md) - 如何使用 API
- [业务场景实现指南](./BUSINESS_SCENARIOS.md) - API 使用示例
- [JSON-RPC API 规范](../../../weisyn.git/docs/reference/json-rpc/) - 底层 API 规范（主仓库）
- [Client API 设计](../_dev/CLIENT_API_DESIGN.md) - WESClient API 详细设计
- [Services 设计](../_dev/SERVICES_DESIGN.md) - 服务层详细设计

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team
