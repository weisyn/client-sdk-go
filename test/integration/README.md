# 集成测试

---

## 📌 版本信息

- **版本**：1.0
- **状态**：draft
- **最后更新**：2025-01-23
- **适用范围**：Go Client SDK 集成测试

---

## 🎯 测试说明

集成测试需要真实的 WES 节点运行。测试前请确保：

1. **启动 WES 节点**：
   ```bash
   cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
   bash scripts/testing/common/test_init.sh
   # 或
   go run ./cmd/testing --api-only
   ```

2. **验证节点运行**：
   ```bash
   curl -s http://localhost:8080/health
   ```

3. **运行测试**：
   ```bash
   cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
   go test ./test/integration/... -v
   ```

---

## 📁 目录结构

```
test/integration/
├── README.md              # 本文档
├── setup.go               # 测试环境设置
├── helpers.go             # 测试辅助函数
└── services/              # 各服务测试
    ├── token/
    ├── staking/
    ├── market/
    ├── governance/
    └── resource/
```

---

## 🧪 测试用例

### Token 服务
- `transfer_test.go` - 单笔转账测试
- `batch_transfer_test.go` - 批量转账测试
- `mint_test.go` - 代币铸造测试
- `burn_test.go` - 代币销毁测试
- `balance_test.go` - 余额查询测试

### Staking 服务
- `stake_test.go` - 质押测试
- `unstake_test.go` - 解质押测试
- `delegate_test.go` - 委托测试
- `undelegate_test.go` - 取消委托测试
- `claim_reward_test.go` - 领取奖励测试

---

## ⚠️ 注意事项

1. **节点依赖**：所有测试都需要 WES 节点运行
2. **测试账户**：每个测试用例使用独立的测试账户
3. **测试代币**：测试账户需要先获得测试代币（通过挖矿或预分配）
4. **测试隔离**：测试用例之间可能共享测试账户，注意状态管理

---

