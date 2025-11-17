# 集成测试

集成测试需要真实的 WES 节点运行，验证 SDK 与节点的真实链上交互能力。

## 🚀 快速开始

### 1. 启动 WES 节点

**方式 1：使用测试初始化脚本（推荐）**

```bash
# 克隆 WES 主项目
git clone https://github.com/weisyn/go-weisyn.git
cd go-weisyn

# 运行测试初始化脚本（会自动编译、启动节点）
bash scripts/testing/common/test_init.sh
```

**方式 2：手动启动测试节点**

```bash
# 克隆 WES 主项目
git clone https://github.com/weisyn/go-weisyn.git
cd go-weisyn

# 编译测试节点
go build -o bin/weisyn-testing ./cmd/testing

# 启动测试节点（仅 API 模式，不参与共识）
./bin/weisyn-testing --api-only --env testing
```

**方式 3：使用 Go 直接运行**

```bash
cd go-weisyn
go run ./cmd/testing --api-only --env testing
```

### 2. 验证节点运行

节点启动后，默认监听以下端点：

- **JSON-RPC API**：`http://localhost:8080/jsonrpc`
- **健康检查**：`http://localhost:8080/health`
- **HTTP REST API**：`http://localhost:8080/api/v1/*`

**验证命令**：

```bash
# 检查节点健康状态
curl http://localhost:8080/health

# 检查 JSON-RPC 是否可用
curl -X POST http://localhost:8080/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"wes_blockNumber","params":[],"id":1}'

# 预期返回：{"jsonrpc":"2.0","id":1,"result":"0x..."}
```

### 3. 运行测试

```bash
# 克隆 SDK 项目（如果还没有）
git clone https://github.com/weisyn/client-sdk-go.git
cd client-sdk-go

# 运行所有集成测试
go test ./test/integration/... -v

# 运行特定服务测试
go test ./test/integration/services/token/... -v
```

## 📁 目录结构

```
test/integration/
├── README.md              # 本文档
├── setup.go               # 测试环境设置和客户端管理
├── helpers.go             # 测试辅助函数（挖矿、交易确认等）
└── services/              # 各服务测试
    ├── README.md          # 业务服务测试说明
    ├── token/             # Token 服务测试
    ├── staking/           # Staking 服务测试
    ├── market/            # Market 服务测试
    ├── governance/        # Governance 服务测试
    └── resource/          # Resource 服务测试
```

## ⚙️ 测试配置

### 节点端点配置

测试默认连接到 `http://localhost:8080/jsonrpc`，可以通过环境变量修改：

```bash
# 设置自定义节点端点
export WES_NODE_ENDPOINT=http://localhost:8080/jsonrpc

# 运行测试
go test ./test/integration/... -v
```

### 测试超时配置

默认超时时间（定义在 `setup.go`）：
- **客户端超时**：30 秒
- **交易确认超时**：60 秒
- **交易确认轮询间隔**：2 秒

## 🔧 测试辅助函数

### 节点管理

| 函数 | 功能 | 说明 |
|------|------|------|
| `EnsureNodeRunning(t)` | 确保节点运行 | 如果节点未运行，测试会失败并提示启动命令 |

**使用示例**：
```go
func TestExample(t *testing.T) {
    integration.EnsureNodeRunning(t)  // 检查节点是否运行
    // ... 测试代码
}
```

### 客户端管理

| 函数 | 功能 | 说明 |
|------|------|------|
| `SetupTestClient(t)` | 创建测试客户端 | 连接到 WES 节点，验证连接成功 |
| `TeardownTestClient(t, c)` | 清理测试客户端 | 关闭连接，释放资源 |

**使用示例**：
```go
func TestExample(t *testing.T) {
    c := integration.SetupTestClient(t)
    defer integration.TeardownTestClient(t, c)
    // ... 使用客户端进行测试
}
```

### 账户管理

| 函数 | 功能 | 说明 |
|------|------|------|
| `CreateTestWallet(t)` | 创建测试钱包 | 生成新的随机钱包 |
| `FundTestAccount(t, c, addr, amount)` | 为账户充值 | 通过挖矿为账户充值原生币 |
| `GetTestAccountBalance(t, c, addr, tokenID)` | 查询账户余额 | 查询指定地址和代币的余额 |

**使用示例**：
```go
func TestExample(t *testing.T) {
    c := integration.SetupTestClient(t)
    defer integration.TeardownTestClient(t, c)
    
    // 创建测试账户
    wallet := integration.CreateTestWallet(t)
    address := wallet.Address()
    
    // 为账户充值
    integration.FundTestAccount(t, c, address, 1000000)
    
    // 查询余额
    balance := integration.GetTestAccountBalance(t, c, address, nil)
    t.Logf("账户余额: %d", balance)
}
```

### 交易管理

| 函数 | 功能 | 说明 |
|------|------|------|
| `WaitForTransactionWithTest(t, c, txHash)` | 等待交易确认 | 轮询查询交易状态，直到确认 |
| `TriggerMining(t, c, minerAddr)` | 触发挖矿 | 启动挖矿，等待区块生成，然后停止 |

**使用示例**：
```go
func TestExample(t *testing.T) {
    // ... 提交交易
    result, err := service.Method(ctx, req, wallet)
    require.NoError(t, err)
    
    // 触发挖矿以确认交易
    integration.TriggerMining(t, c, wallet.Address())
    
    // 等待交易确认
    parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
    require.Equal(t, "confirmed", parsedTx.Status)
}
```

## 🐛 故障排查

### 问题 1：节点未运行

**错误信息**：
```
节点未运行，请先启动节点: http://localhost:8080/jsonrpc
```

**解决方案**：
1. 检查节点是否已启动：
   ```bash
   curl http://localhost:8080/health
   ```
2. 如果未启动，按照"快速开始"部分的步骤启动节点
3. 检查端口是否被占用：
   ```bash
   lsof -i :8080
   ```

### 问题 2：连接超时

**错误信息**：
```
context deadline exceeded
```

**解决方案**：
1. 检查节点是否正常运行
2. 检查防火墙设置
3. 确认节点端点配置正确（默认 `http://localhost:8080/jsonrpc`）

### 问题 3：账户余额不足

**错误信息**：
```
insufficient balance
```

**解决方案**：
1. 确保在测试前调用 `FundTestAccount` 为账户充值
2. 检查挖矿是否成功（查看日志）
3. 等待足够时间让 UTXO 可用（`FundTestAccount` 会自动等待）

### 问题 4：交易未确认

**错误信息**：
```
交易确认超时
```

**解决方案**：
1. 确保调用 `TriggerMining` 触发挖矿
2. 增加超时时间（修改 `TransactionConfirmTimeout`）
3. 检查节点是否正常出块

## 📚 完整文档

- **[业务服务测试说明](services/README.md)** - 各服务测试的详细说明和运行指南
- **[测试规划文档](../../docs/testing/plan.md)** - 详细的测试策略和规划
- **[架构文档](../../docs/architecture.md)** - SDK 架构设计

## 🔗 相关资源

- **[WES 主项目](https://github.com/weisyn/go-weisyn)** - WES 区块链核心实现
- **[WES 测试文档](https://github.com/weisyn/go-weisyn/tree/main/scripts/testing)** - WES 节点测试相关脚本

---

**最后更新**: 2025-11-17
