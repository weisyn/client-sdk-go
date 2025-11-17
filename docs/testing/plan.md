# Go Client SDK 业务语义测试规划

---

## 📌 元信息

- **文档类型**：测试规划 / 测试策略  
- **适用范围**：`client-sdk-go`（Go Client SDK）  
- **当前版本**：v1.0  
- **文档状态**：proposal  
- **最后更新**：2025-11-17  
- **所有者**：WES SDK 团队  
- **参考规范**：遵循 WES DOCS 文档规范

---

## 1. 测试目标与范围

### 1.1 测试目标

验证 Go Client SDK 中所有业务语义服务的**真实链上交互能力**，确保：

1. **交易构建正确性**：SDK 能够正确构建符合 WES 协议的交易草稿
2. **节点交互正确性**：SDK 能够正确调用节点 API（`wes_getUTXO`, `wes_buildTransaction`, `wes_callContract`, `wes_sendRawTransaction`）
3. **结果解析正确性**：SDK 能够从链上真实数据中正确解析业务结果（ID、金额等）
4. **端到端流程完整性**：完整的"查询 → 构建 → 签名 → 提交 → 解析"链路可用

### 1.2 测试范围

**核心业务服务**（5 个）：

| 服务 | 测试功能 | 优先级 |
|------|---------|--------|
| **Token** | Transfer, BatchTransfer, Mint, Burn, GetBalance | P0 |
| **Staking** | Stake, Unstake, Delegate, Undelegate, ClaimReward | P0 |
| **Market** | CreateVesting, ClaimVesting, CreateEscrow, ReleaseEscrow, RefundEscrow, SwapAMM, AddLiquidity, RemoveLiquidity | P0 |
| **Governance** | Propose, Vote, UpdateParam | P1 |
| **Resource** | DeployContract, DeployAIModel, DeployStaticResource, GetResource | P1 |

**工具能力**（1 个）：

| 工具 | 测试功能 | 优先级 |
|------|---------|--------|
| **utils/tx_parser** | FetchAndParseTx, FindOutputsByOwner, SumAmountsByToken, FindStateOutputs | P0 |

---

## 2. 测试环境与依赖

### 2.1 前置要求

#### 2.1.1 WES 节点

**节点位置**：`/Users/qinglong/go/src/chaincodes/WES/weisyn.git`

**启动方式**：
```bash
# 方式 1：使用测试初始化脚本（推荐）
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
bash scripts/testing/common/test_init.sh

# 方式 2：直接启动测试节点
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
go run ./cmd/testing --api-only

# 方式 3：使用预编译二进制
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
./bin/weisyn-testing --daemon --env testing
```

**节点配置**：
- **环境**：`testing`（单节点共识模式）
- **API 端点**：`http://localhost:8080/jsonrpc`
- **数据目录**：`data/testing/`（测试环境会自动清理）
- **配置来源**：`configs/testing/config.json`

**节点状态检查**：
```bash
# 检查节点是否运行
curl -s http://localhost:8080/health

# 检查 JSON-RPC 是否可用
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"wes_getBlockHeight","params":[],"id":1}'
```

#### 2.1.2 SDK 项目

**SDK 位置**：`/Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git`

**编译要求**：
```bash
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
go build ./...
```

**依赖工具**：
- `go` (1.24+)
- `curl` - API 调用验证
- `jq` (可选) - JSON 解析增强

### 2.2 测试账户准备

**测试账户生成**：
- 每个测试用例应使用独立的测试账户（避免状态污染）
- 测试账户可以通过 SDK 的 `wallet.NewWallet()` 生成
- 测试账户需要先获得测试代币（通过挖矿或转账）

**测试代币获取**：
```bash
# 方式 1：通过挖矿获取（需要节点支持）
# 方式 2：通过测试脚本预分配（推荐）
# 方式 3：通过测试账户间转账（需要初始账户有余额）
```

---

## 3. 测试架构设计

### 3.1 测试分层

```
┌─────────────────────────────────────────────────────────┐
│              测试脚本层 (scripts/testing/)                │
│  - 节点启动/停止                                         │
│  - 测试环境初始化                                        │
│  - 测试用例编排                                          │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              Go 测试代码层 (test/integration/)           │
│  - 单元测试（mock 节点响应）                             │
│  - 集成测试（真实节点交互）                              │
│  - 端到端测试（完整业务流程）                            │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              SDK 业务服务层 (services/)                  │
│  - Token, Staking, Market, Governance, Resource          │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│              WES 节点 (weisyn.git)                       │
│  - JSON-RPC API                                          │
│  - 交易处理与出块                                         │
└─────────────────────────────────────────────────────────┘
```

### 3.2 测试类型

#### 3.2.1 单元测试（Unit Tests）

**位置**：`test/unit/`

**特点**：
- 使用 mock 对象模拟节点响应
- 快速执行，不依赖真实节点
- 测试 SDK 内部逻辑（交易构建、结果解析等）

**示例**：
```go
// test/unit/services/token/transfer_test.go
func TestBuildTransferTransaction(t *testing.T) {
    // Mock client
    mockClient := &MockClient{}
    // Test transaction building logic
    // ...
}
```

#### 3.2.2 集成测试（Integration Tests）

**位置**：`test/integration/`

**特点**：
- 需要真实 WES 节点运行
- 测试 SDK 与节点的真实交互
- 验证 API 调用、交易提交、结果解析

**示例**：
```go
// test/integration/services/token/transfer_test.go
func TestTokenTransfer_Integration(t *testing.T) {
    // 1. 启动/连接节点
    client := setupTestClient(t)
    defer teardownTestClient(t)
    
    // 2. 创建测试账户
    wallet := createTestWallet(t)
    
    // 3. 执行转账
    result, err := tokenService.Transfer(ctx, req, wallet)
    
    // 4. 验证结果
    assert.NoError(t, err)
    assert.NotEmpty(t, result.TxHash)
    // ...
}
```

#### 3.2.3 端到端测试（E2E Tests）

**位置**：`scripts/testing/sdk/`

**特点**：
- 使用 Shell 脚本编排完整业务流程
- 参考 `scripts/testing/contracts/hello_world_test.sh` 的模式
- 包含节点启动、测试执行、结果验证、环境清理

**示例**：
```bash
# scripts/testing/sdk/token_transfer_test.sh
#!/usr/bin/env bash
# 1. 初始化测试环境
source scripts/testing/common/test_init.sh
init_test_environment

# 2. 启动节点（如未运行）
ensure_node_running

# 3. 运行 Go 测试
go test ./test/integration/services/token/... -v

# 4. 验证测试结果
# ...
```

---

## 4. 测试用例设计

### 4.1 Token 服务测试

#### 4.1.1 Transfer（单笔转账）

**测试步骤**：
1. 创建两个测试账户（from, to）
2. 为 from 账户充值（通过挖矿或预分配）
3. 调用 `tokenService.Transfer()` 执行转账
4. 等待交易确认（轮询 `wes_getTransactionByHash`）
5. 验证结果：
   - `result.TxHash` 不为空
   - `result.Success == true`
   - 查询 to 账户余额，验证金额正确
   - 查询 from 账户余额，验证金额减少

**预期结果**：
- 交易成功提交并确认
- 余额变化正确

#### 4.1.2 BatchTransfer（批量转账）

**测试步骤**：
1. 创建多个测试账户
2. 为 from 账户充值
3. 调用 `tokenService.BatchTransfer()` 执行批量转账
4. 等待交易确认
5. 验证所有接收账户余额正确

**预期结果**：
- 批量转账成功
- 所有接收账户余额正确

#### 4.1.3 Mint（代币铸造）

**测试步骤**：
1. 准备合约 contentHash（需要先部署代币合约）
2. 调用 `tokenService.Mint()` 执行铸造
3. 等待交易确认
4. 验证接收账户余额增加

**预期结果**：
- 铸造成功
- 代币余额正确

#### 4.1.4 Burn（代币销毁）

**测试步骤**：
1. 准备有余额的测试账户
2. 调用 `tokenService.Burn()` 执行销毁
3. 等待交易确认
4. 验证账户余额减少

**预期结果**：
- 销毁成功
- 余额正确减少

#### 4.1.5 GetBalance（查询余额）

**测试步骤**：
1. 准备有余额的测试账户
2. 调用 `tokenService.GetBalance()` 查询余额
3. 验证返回余额与链上一致

**预期结果**：
- 余额查询正确

---

### 4.2 Staking 服务测试

#### 4.2.1 Stake（质押）

**测试步骤**：
1. 创建测试账户并充值
2. 调用 `stakingService.Stake()` 执行质押
3. 等待交易确认
4. 验证结果：
   - `result.StakeID` 不为空（从交易输出解析）
   - `result.TxHash` 不为空
   - 查询账户余额，验证质押金额已锁定

**预期结果**：
- 质押成功
- StakeID 正确解析
- 余额正确锁定

#### 4.2.2 Unstake（解质押）

**测试步骤**：
1. 准备有质押的账户（通过 Stake 测试创建）
2. 调用 `stakingService.Unstake()` 执行解质押
3. 等待交易确认
4. 验证结果：
   - `result.UnstakeAmount` 正确（从交易输出解析）
   - `result.RewardAmount` 正确（如果有奖励）
   - 账户余额增加

**预期结果**：
- 解质押成功
- 金额解析正确

#### 4.2.3 Delegate（委托）

**测试步骤**：
1. 创建测试账户并充值
2. 调用 `stakingService.Delegate()` 执行委托
3. 等待交易确认
4. 验证结果：
   - `result.DelegateID` 不为空（从交易输出解析）

**预期结果**：
- 委托成功
- DelegateID 正确解析

#### 4.2.4 Undelegate（取消委托）

**测试步骤**：
1. 准备有委托的账户
2. 调用 `stakingService.Undelegate()` 执行取消委托
3. 等待交易确认
4. 验证账户余额恢复

**预期结果**：
- 取消委托成功

#### 4.2.5 ClaimReward（领取奖励）

**测试步骤**：
1. 准备有奖励的账户（通过质押/委托产生）
2. 调用 `stakingService.ClaimReward()` 执行领取
3. 等待交易确认
4. 验证结果：
   - `result.RewardAmount` 正确（从交易输出解析）
   - 账户余额增加

**预期结果**：
- 领取奖励成功
- 奖励金额解析正确

---

### 4.3 Market 服务测试

#### 4.3.1 CreateVesting（创建归属计划）

**测试步骤**：
1. 创建测试账户并充值
2. 调用 `marketService.CreateVesting()` 创建归属计划
3. 等待交易确认
4. 验证结果：
   - `result.VestingID` 不为空（从交易输出解析）
   - 查询链上 UTXO，验证 TimeLock 正确设置

**预期结果**：
- 归属计划创建成功
- VestingID 正确解析
- TimeLock 正确设置

#### 4.3.2 ClaimVesting（领取归属代币）

**测试步骤**：
1. 准备有归属计划的账户（通过 CreateVesting 创建）
2. 等待归属时间到期（或手动调整节点时间）
3. 调用 `marketService.ClaimVesting()` 执行领取
4. 等待交易确认
5. 验证结果：
   - `result.ClaimAmount` 正确（从交易输出解析）
   - 账户余额增加

**预期结果**：
- 领取成功
- 金额解析正确

#### 4.3.3 CreateEscrow（创建托管）

**测试步骤**：
1. 创建买方和卖方账户
2. 买方账户充值
3. 调用 `marketService.CreateEscrow()` 创建托管
4. 等待交易确认
5. 验证结果：
   - `result.EscrowID` 不为空（从交易输出解析）
   - 查询链上 UTXO，验证 MultiKeyLock 正确设置

**预期结果**：
- 托管创建成功
- EscrowID 正确解析
- MultiKeyLock 正确设置

#### 4.3.4 ReleaseEscrow / RefundEscrow（释放/退款托管）

**测试步骤**：
1. 准备有托管的账户（通过 CreateEscrow 创建）
2. 调用 `marketService.ReleaseEscrow()` 或 `RefundEscrow()` 执行释放/退款
3. 等待交易确认（需要多方签名）
4. 验证账户余额变化

**预期结果**：
- 释放/退款成功

#### 4.3.5 SwapAMM（AMM 交换）

**测试步骤**：
1. 准备 AMM 合约地址（需要先部署 AMM 合约）
2. 准备有代币的账户
3. 调用 `marketService.SwapAMM()` 执行交换
4. 等待交易确认
5. 验证结果：
   - `result.AmountOut` 正确（从交易输出解析）
   - 账户代币余额变化正确

**预期结果**：
- 交换成功
- 输出金额解析正确

#### 4.3.6 AddLiquidity / RemoveLiquidity（添加/移除流动性）

**测试步骤**：
1. 准备 AMM 合约地址
2. 准备有代币的账户
3. 调用 `marketService.AddLiquidity()` 添加流动性
4. 等待交易确认
5. 验证结果：
   - `result.LiquidityID` 不为空（从交易输出解析）
6. 调用 `marketService.RemoveLiquidity()` 移除流动性
7. 验证结果：
   - `result.AmountA` 和 `result.AmountB` 正确（从交易输出解析）

**预期结果**：
- 流动性操作成功
- ID 和金额解析正确

---

### 4.4 Governance 服务测试

#### 4.4.1 Propose（创建提案）

**测试步骤**：
1. 创建测试账户
2. 调用 `governanceService.Propose()` 创建提案
3. 等待交易确认
4. 验证结果：
   - `result.ProposalID` 不为空（从 StateOutput 解析）
   - 查询链上 StateOutput，验证提案数据正确

**预期结果**：
- 提案创建成功
- ProposalID 正确解析

#### 4.4.2 Vote（投票）

**测试步骤**：
1. 准备有提案的链上状态（通过 Propose 创建）
2. 调用 `governanceService.Vote()` 执行投票
3. 等待交易确认
4. 验证结果：
   - `result.VoteID` 不为空（从 StateOutput 解析）

**预期结果**：
- 投票成功
- VoteID 正确解析

#### 4.4.3 UpdateParam（更新参数）

**测试步骤**：
1. 创建测试账户
2. 调用 `governanceService.UpdateParam()` 更新参数
3. 等待交易确认
4. 验证链上参数已更新

**预期结果**：
- 参数更新成功

---

### 4.5 Resource 服务测试

#### 4.5.1 DeployContract（部署合约）

**测试步骤**：
1. 准备 WASM 文件
2. 调用 `resourceService.DeployContract()` 部署合约
3. 等待交易确认
4. 验证结果：
   - `result.ContentHash` 不为空
   - `result.ContractAddress` 不为空
   - 查询链上资源，验证部署成功

**预期结果**：
- 合约部署成功
- ContentHash 和 Address 正确

#### 4.5.2 DeployAIModel（部署 AI 模型）

**测试步骤**：
1. 准备 ONNX 模型文件
2. 调用 `resourceService.DeployAIModel()` 部署模型
3. 等待交易确认
4. 验证结果：
   - `result.ContentHash` 不为空

**预期结果**：
- 模型部署成功

#### 4.5.3 DeployStaticResource（部署静态资源）

**测试步骤**：
1. 准备静态资源文件
2. 调用 `resourceService.DeployStaticResource()` 部署资源
3. 等待交易确认
4. 验证结果：
   - `result.ContentHash` 不为空

**预期结果**：
- 静态资源部署成功

#### 4.5.4 GetResource（查询资源）

**测试步骤**：
1. 准备已部署的资源 contentHash
2. 调用 `resourceService.GetResource()` 查询资源
3. 验证返回的资源信息正确

**预期结果**：
- 资源查询成功
- 资源信息正确

---

### 4.6 工具能力测试

#### 4.6.1 FetchAndParseTx（交易解析）

**测试步骤**：
1. 执行任意业务操作（如 Transfer），获取 txHash
2. 等待交易确认
3. 调用 `utils.FetchAndParseTx()` 解析交易
4. 验证解析结果：
   - 交易基本信息正确（hash, status, blockHeight）
   - inputs 和 outputs 解析正确
   - 每个输出的 outpoint 正确

**预期结果**：
- 交易解析成功
- 所有字段解析正确

#### 4.6.2 FindOutputsByOwner / SumAmountsByToken（输出查询与汇总）

**测试步骤**：
1. 解析一个交易（通过 FetchAndParseTx）
2. 调用 `utils.FindOutputsByOwner()` 查找指定地址的输出
3. 调用 `utils.SumAmountsByToken()` 汇总金额
4. 验证结果正确

**预期结果**：
- 查询和汇总结果正确

---

## 5. 测试实施计划

### 5.1 目录结构

```
client-sdk-go.git/
├── test/
│   ├── README.md                    # 测试总说明
│   │
│   ├── unit/                        # 单元测试（mock）
│   │   ├── services/
│   │   │   ├── token/
│   │   │   ├── staking/
│   │   │   ├── market/
│   │   │   ├── governance/
│   │   │   └── resource/
│   │   └── utils/
│   │
│   ├── integration/                 # 集成测试（真实节点）
│   │   ├── README.md                # 集成测试说明
│   │   ├── setup.go                 # 测试环境设置
│   │   ├── helpers.go               # 测试辅助函数
│   │   └── services/
│   │       ├── token/
│   │       │   ├── transfer_test.go
│   │       │   ├── batch_transfer_test.go
│   │       │   ├── mint_test.go
│   │       │   ├── burn_test.go
│   │       │   └── balance_test.go
│   │       ├── staking/
│   │       │   ├── stake_test.go
│   │       │   ├── unstake_test.go
│   │       │   ├── delegate_test.go
│   │       │   ├── undelegate_test.go
│   │       │   └── claim_reward_test.go
│   │       ├── market/
│   │       │   ├── vesting_test.go
│   │       │   ├── escrow_test.go
│   │       │   └── amm_test.go
│   │       ├── governance/
│   │       │   ├── propose_test.go
│   │       │   ├── vote_test.go
│   │       │   └── update_param_test.go
│   │       └── resource/
│   │           ├── deploy_test.go
│   │           └── query_test.go
│   │
│   └── e2e/                         # 端到端测试（脚本）
│       └── (可选，使用 scripts/testing/sdk/)
│
└── scripts/
    └── testing/
        └── sdk/                     # SDK 测试脚本
            ├── README.md            # SDK 测试说明
            ├── test_init.sh         # SDK 测试环境初始化
            ├── token_test.sh        # Token 服务测试脚本
            ├── staking_test.sh      # Staking 服务测试脚本
            ├── market_test.sh       # Market 服务测试脚本
            ├── governance_test.sh   # Governance 服务测试脚本
            └── resource_test.sh     # Resource 服务测试脚本
```

### 5.2 测试辅助工具

#### 5.2.1 测试环境设置（test/integration/setup.go）

**功能**：
- 节点启动/停止管理
- 测试账户创建与管理
- 测试代币预分配
- 测试环境清理

**示例**：
```go
package integration

import (
    "context"
    "testing"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/wallet"
)

// setupTestClient 设置测试客户端
func setupTestClient(t *testing.T) client.Client {
    cfg := &client.Config{
        Endpoint: "http://localhost:8080/jsonrpc",
        Protocol: client.ProtocolHTTP,
        Timeout:  30,
    }
    
    c, err := client.NewClient(cfg)
    require.NoError(t, err)
    
    // 检查节点是否运行
    ctx := context.Background()
    _, err = c.Call(ctx, "wes_getBlockHeight", []interface{}{})
    require.NoError(t, err, "节点未运行，请先启动节点")
    
    return c
}

// createTestWallet 创建测试钱包
func createTestWallet(t *testing.T) wallet.Wallet {
    w, err := wallet.NewWallet()
    require.NoError(t, err)
    return w
}

// fundTestAccount 为测试账户充值
func fundTestAccount(t *testing.T, client client.Client, address []byte, amount uint64) {
    // 通过挖矿或预分配为账户充值
    // ...
}
```

#### 5.2.2 测试辅助函数（test/integration/helpers.go）

**功能**：
- 等待交易确认
- 查询账户余额
- 验证交易结果
- 通用断言函数

**示例**：
```go
package integration

import (
    "context"
    "time"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/utils"
)

// waitForTransaction 等待交易确认
func waitForTransaction(ctx context.Context, client client.Client, txHash string, timeout time.Duration) (*utils.ParsedTx, error) {
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        parsedTx, err := utils.FetchAndParseTx(ctx, client, txHash)
        if err == nil && parsedTx != nil && parsedTx.Status == "confirmed" {
            return parsedTx, nil
        }
        time.Sleep(2 * time.Second)
    }
    
    return nil, fmt.Errorf("交易确认超时: %s", txHash)
}

// getBalance 查询账户余额
func getBalance(ctx context.Context, client client.Client, address []byte, tokenID []byte) (uint64, error) {
    // 调用 wes_getUTXO 查询余额
    // ...
}
```

### 5.3 测试脚本（scripts/testing/sdk/）

#### 5.3.1 测试初始化脚本（test_init.sh）

**功能**：
- 检查 WES 节点是否运行
- 如未运行，启动节点
- 等待节点 API 就绪
- 设置测试环境变量

**参考**：`scripts/testing/common/test_init.sh`

#### 5.3.2 服务测试脚本（*_test.sh）

**功能**：
- 调用对应的 Go 集成测试
- 收集测试结果
- 生成测试报告
- 清理测试环境

**示例**：
```bash
#!/usr/bin/env bash
# scripts/testing/sdk/token_test.sh

set -eu

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
SDK_ROOT="${PROJECT_ROOT}/sdk/client-sdk-go.git"

# 初始化测试环境
source "${PROJECT_ROOT}/scripts/testing/common/test_init.sh"
init_test_environment

# 运行 Token 服务测试
cd "${SDK_ROOT}"
go test ./test/integration/services/token/... -v -count=1

# 生成测试报告
# ...
```

---

## 6. 测试执行流程

### 6.1 快速测试（单个服务）

```bash
# 1. 启动节点（如未运行）
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
bash scripts/testing/common/test_init.sh

# 2. 运行单个服务测试
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
go test ./test/integration/services/token/... -v
```

### 6.2 完整测试（所有服务）

```bash
# 1. 启动节点
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
bash scripts/testing/common/test_init.sh

# 2. 运行所有集成测试
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
go test ./test/integration/... -v

# 3. 或使用测试脚本
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
bash scripts/testing/sdk/token_test.sh
bash scripts/testing/sdk/staking_test.sh
bash scripts/testing/sdk/market_test.sh
bash scripts/testing/sdk/governance_test.sh
bash scripts/testing/sdk/resource_test.sh
```

### 6.3 CI/CD 集成

**GitHub Actions 示例**：
```yaml
# .github/workflows/test-sdk.yml
name: Test Client SDK

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Start WES Node
        run: |
          cd weisyn.git
          go run ./cmd/testing --api-only &
          sleep 10
      
      - name: Run Tests
        run: |
          cd sdk/client-sdk-go.git
          go test ./test/integration/... -v
```

---

## 7. 测试数据管理

### 7.1 测试账户管理

**策略**：
- 每个测试用例使用独立的测试账户
- 测试账户通过 `wallet.NewWallet()` 生成
- 测试账户私钥可以硬编码（仅用于测试）

**测试账户池**：
```go
// test/integration/accounts.go
var TestAccounts = struct {
    Alice wallet.Wallet
    Bob   wallet.Wallet
    Charlie wallet.Wallet
}{
    // 使用固定的测试私钥（仅用于测试）
    Alice:   mustNewWalletFromPrivateKey("0x..."),
    Bob:     mustNewWalletFromPrivateKey("0x..."),
    Charlie: mustNewWalletFromPrivateKey("0x..."),
}
```

### 7.2 测试代币管理

**策略**：
- 测试开始前，为测试账户预分配测试代币
- 测试结束后，可以清理测试数据（可选）

**预分配方式**：
```go
// test/integration/setup.go
func setupTestAccounts(t *testing.T, client client.Client) {
    // 为测试账户充值
    fundTestAccount(t, client, TestAccounts.Alice.Address(), 1000000)
    fundTestAccount(t, client, TestAccounts.Bob.Address(), 1000000)
}
```

### 7.3 测试合约管理

**策略**：
- 测试合约（如代币合约、AMM 合约）可以预先部署
- 合约地址存储在测试配置中
- 或每次测试时动态部署

---

## 8. 测试报告与验证

### 8.1 测试报告格式

**测试报告应包含**：
- 测试用例名称
- 测试步骤
- 预期结果
- 实际结果
- 交易哈希（用于链上验证）
- 测试耗时
- 测试状态（通过/失败）

**示例**：
```
=== Token Transfer Test ===
步骤 1: 创建测试账户 ✅
步骤 2: 为账户充值 ✅
步骤 3: 执行转账 ✅
  - 交易哈希: 0x1234...
步骤 4: 等待交易确认 ✅
步骤 5: 验证余额 ✅
  - 预期余额: 1000
  - 实际余额: 1000
测试结果: ✅ 通过
耗时: 2.5s
```

### 8.2 链上验证

**验证方式**：
- 使用 `wes_getTransactionByHash` 查询交易详情
- 使用 `wes_getUTXO` 查询账户余额
- 使用 `utils.FetchAndParseTx` 解析交易结果
- 对比 SDK 返回结果与链上真实数据

---

## 9. 已知限制与注意事项

### 9.1 节点依赖

**限制**：
- 所有集成测试都需要 WES 节点运行
- 节点启动需要时间（通常 5-10 秒）
- 测试环境需要清理，避免状态污染

**建议**：
- 使用测试初始化脚本统一管理节点
- 测试前检查节点状态，避免重复启动
- 测试后可选清理测试数据

### 9.2 交易确认时间

**限制**：
- 交易提交后需要等待确认（单节点模式通常很快）
- 测试需要轮询交易状态

**建议**：
- 使用 `waitForTransaction` 辅助函数
- 设置合理的超时时间（建议 30 秒）
- 单节点模式下可以主动触发挖矿加速确认

### 9.3 测试数据隔离

**限制**：
- 多个测试用例可能共享测试账户
- 测试状态可能相互影响

**建议**：
- 每个测试用例使用独立的测试账户
- 测试前重置账户状态（如需要）
- 使用测试初始化脚本清理环境

---

## 10. 后续优化方向

### 10.1 测试覆盖率

**目标**：
- 单元测试覆盖率 > 80%
- 集成测试覆盖所有核心功能
- 端到端测试覆盖主要业务流程

### 10.2 测试性能

**优化方向**：
- 并行执行测试用例
- 复用测试账户和合约
- 优化节点启动时间

### 10.3 测试自动化

**优化方向**：
- CI/CD 集成
- 自动化测试报告生成
- 测试结果可视化

---

## 11. 参考资源

### 11.1 WES 节点测试脚本

- `scripts/testing/common/test_init.sh` - 测试环境初始化
- `scripts/testing/contracts/hello_world_test.sh` - 合约测试示例
- `test/integration/api/jsonrpc_test.go` - API 集成测试示例

### 11.2 SDK 文档

- `services/FINAL_CAPABILITY_REPORT.md` - SDK 能力报告
- `services/IMPLEMENTATION_COMPLETE.md` - 实现完成报告
- `README.md` - SDK 使用文档

---

## 12. 更新记录

### v1.0 (2025-11-17)
- ✅ 创建测试规划文档
- ✅ 定义测试架构和用例
- ✅ 规划测试实施步骤

---

