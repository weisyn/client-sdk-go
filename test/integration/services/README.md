# 业务服务集成测试

本目录包含所有业务服务的集成测试，验证 SDK 与 WES 节点的真实链上交互能力。

## 📁 目录结构

```
test/integration/services/
├── README.md              # 本文档
├── token/                 # Token 服务测试
│   ├── transfer_test.go   # 转账测试
│   ├── batch_transfer_test.go  # 批量转账测试
│   ├── burn_test.go       # 销毁测试
│   └── balance_test.go    # 余额查询测试
├── staking/               # Staking 服务测试
│   ├── stake_test.go      # 质押测试
│   ├── delegate_test.go   # 委托测试
│   └── claim_reward_test.go  # 领取奖励测试
├── market/                # Market 服务测试
│   ├── vesting_test.go    # 归属计划测试
│   └── escrow_test.go     # 托管测试
├── governance/            # Governance 服务测试
│   └── propose_test.go    # 提案测试
└── resource/              # Resource 服务测试（待实现）
```

## 🎯 测试覆盖

### Token 服务 ✅

| 测试文件 | 测试功能 | 状态 |
|---------|---------|------|
| `transfer_test.go` | 单笔转账 | ✅ 完成 |
| `batch_transfer_test.go` | 批量转账 | ✅ 完成 |
| `burn_test.go` | 代币销毁 | ✅ 完成 |
| `balance_test.go` | 余额查询 | ✅ 完成 |

**待补充**：
- `mint_test.go` - 代币铸造测试

### Staking 服务 ✅

| 测试文件 | 测试功能 | 状态 |
|---------|---------|------|
| `stake_test.go` | 质押、解质押 | ✅ 完成 |
| `delegate_test.go` | 委托、取消委托 | ✅ 完成 |
| `claim_reward_test.go` | 领取奖励 | ✅ 完成 |

### Market 服务 ✅

| 测试文件 | 测试功能 | 状态 |
|---------|---------|------|
| `vesting_test.go` | 创建归属计划、领取归属代币 | ✅ 完成 |
| `escrow_test.go` | 创建托管、释放托管、退款托管 | ✅ 完成 |

**待补充**：
- `amm_test.go` - AMM 交换、流动性管理测试

### Governance 服务 ✅

| 测试文件 | 测试功能 | 状态 |
|---------|---------|------|
| `propose_test.go` | 创建提案 | ✅ 完成 |

**待补充**：
- `vote_test.go` - 投票测试
- `update_param_test.go` - 参数更新测试

### Resource 服务 ⏳

**待实现**：
- `deploy_contract_test.go` - 合约部署测试
- `deploy_ai_model_test.go` - AI 模型部署测试
- `deploy_static_resource_test.go` - 静态资源部署测试
- `get_resource_test.go` - 资源查询测试

## 🚀 运行测试

### 前置条件

1. **启动 WES 测试节点**：
```bash
# 克隆主项目
git clone https://github.com/weisyn/go-weisyn.git
cd go-weisyn

# 启动测试节点
bash scripts/testing/common/test_init.sh
```

2. **验证节点运行**：
```bash
curl http://localhost:8080/health
```

### 运行所有服务测试

```bash
# 在 client-sdk-go 目录下
go test ./test/integration/services/... -v
```

### 运行特定服务测试

```bash
# Token 服务测试
go test ./test/integration/services/token/... -v

# Staking 服务测试
go test ./test/integration/services/staking/... -v

# Market 服务测试
go test ./test/integration/services/market/... -v

# Governance 服务测试
go test ./test/integration/services/governance/... -v
```

### 运行单个测试文件

```bash
# Token 转账测试
go test ./test/integration/services/token/transfer_test.go -v

# Staking 质押测试
go test ./test/integration/services/staking/stake_test.go -v
```

### 运行特定测试函数

```bash
# 运行 Token 基本转账测试
go test ./test/integration/services/token/... -run TestTokenTransfer_Basic -v

# 运行 Staking 质押测试
go test ./test/integration/services/staking/... -run TestStaking_Stake -v
```

## 📋 测试编写规范

### 1. 测试函数命名

遵循 Go 测试命名规范：
- 测试函数以 `Test` 开头
- 使用下划线分隔服务名和功能名
- 示例：`TestTokenTransfer_Basic`、`TestStaking_Stake`

### 2. 测试结构

每个测试应包含以下步骤：

```go
func TestService_Method(t *testing.T) {
    // 1. 确保节点运行
    integration.EnsureNodeRunning(t)
    
    // 2. 设置测试客户端
    c := integration.SetupTestClient(t)
    defer integration.TeardownTestClient(t, c)
    
    // 3. 创建测试账户
    wallet := integration.CreateTestWallet(t)
    address := wallet.Address()
    
    // 4. 为账户充值（如需要）
    integration.FundTestAccount(t, c, address, amount)
    
    // 5. 创建服务实例
    service := service.NewService(c)
    
    // 6. 执行业务操作
    result, err := service.Method(ctx, &Request{...}, wallet)
    
    // 7. 验证结果
    require.NoError(t, err)
    assert.NotEmpty(t, result.TxHash)
    assert.True(t, result.Success)
    
    // 8. 等待交易确认（如需要）
    integration.WaitForTransactionConfirm(t, c, result.TxHash)
    
    // 9. 验证链上状态（如需要）
    // ...
}
```

### 3. 测试辅助函数

使用 `test/integration` 包提供的辅助函数：

| 函数 | 功能 |
|------|------|
| `EnsureNodeRunning(t)` | 确保节点运行，否则跳过测试 |
| `SetupTestClient(t)` | 创建测试客户端 |
| `TeardownTestClient(t, c)` | 清理测试客户端 |
| `CreateTestWallet(t)` | 创建测试钱包 |
| `FundTestAccount(t, c, addr, amount)` | 为账户充值 |
| `GetTestAccountBalance(t, c, addr, tokenID)` | 查询账户余额 |
| `WaitForTransactionConfirm(t, c, txHash)` | 等待交易确认 |
| `MineBlock(t, c)` | 触发挖矿 |

### 4. 错误处理

- 使用 `require` 进行必须通过的断言（失败会立即终止测试）
- 使用 `assert` 进行可继续的断言（失败会记录但继续执行）
- 对于已知的限制或依赖问题，使用 `t.Skip()` 跳过测试

### 5. 测试数据

- 使用有意义的测试数据（金额、地址等）
- 避免硬编码，使用常量或配置
- 确保测试数据不会相互干扰

## 🔍 测试验证点

每个业务服务测试应验证：

1. **交易构建正确性**
   - DraftJSON 结构正确
   - 输入输出数量符合预期
   - 锁定条件设置正确

2. **节点交互正确性**
   - API 调用成功
   - 返回数据格式正确
   - 错误处理正确

3. **结果解析正确性**
   - 交易哈希不为空
   - 业务数据（ID、金额等）正确提取
   - 状态变更正确反映

4. **端到端流程完整性**
   - 交易成功提交
   - 交易成功确认
   - 链上状态正确更新

## 📚 相关文档

- [集成测试主文档](../README.md) - 快速开始和目录结构
- [测试规划文档](../../docs/testing/plan.md) - 详细的测试策略和规划
- [架构文档](../../docs/architecture.md) - SDK 架构设计
- [业务服务文档](../../docs/modules/services.md) - 业务服务详细说明

## ⚠️ 注意事项

1. **节点依赖**：所有测试都需要 WES 节点运行，确保在运行测试前启动节点
2. **测试隔离**：每个测试应使用独立的账户，避免相互干扰
3. **资源清理**：测试完成后应清理测试数据（如需要）
4. **并发安全**：注意测试的并发执行，避免资源竞争
5. **网络超时**：设置合理的超时时间，避免测试长时间挂起

---

**最后更新**: 2025-11-17

