# 运行测试指南

---

## 🚀 快速开始

### 1. 启动 WES 节点

**终端 1**：
```bash
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
bash scripts/testing/common/test_init.sh
```

等待看到：
```
✅ 节点已启动并运行
```

### 2. 运行测试

**终端 2**：
```bash
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git

# 方式 1：使用测试脚本（推荐）
bash scripts/testing/sdk/token_test.sh

# 方式 2：直接运行 Go 测试
go test ./test/integration/services/token/... -v -timeout 120s
```

---

## 📋 测试用例列表

### Token 服务（6个测试用例）
- `TestTokenTransfer_Basic` - 基本转账功能
- `TestTokenTransfer_InvalidAddress` - 无效地址测试
- `TestTokenTransfer_InsufficientBalance` - 余额不足测试
- `TestTokenBatchTransfer_Basic` - 批量转账测试
- `TestTokenGetBalance_Basic` - 余额查询测试
- `TestTokenGetBalance_ZeroBalance` - 零余额测试

### Staking 服务（4个测试用例）
- `TestStaking_Stake` - 质押功能测试
- `TestStaking_Unstake` - 解质押功能测试
- `TestStaking_Delegate` - 委托功能测试
- `TestStaking_Undelegate` - 取消委托测试

---

## 🧪 运行单个测试用例

```bash
# Token 转账测试
go test ./test/integration/services/token/... -v -run TestTokenTransfer_Basic -timeout 60s

# Staking 质押测试
go test ./test/integration/services/staking/... -v -run TestStaking_Stake -timeout 60s
```

---

## 📊 测试输出示例

```
=== RUN   TestTokenTransfer_Basic
    transfer_test.go:44: From 地址: 0x1234...
    transfer_test.go:45: To 地址: 0x5678...
    transfer_test.go:54: From 初始余额: 1000000
    transfer_test.go:55: To 初始余额: 0
    transfer_test.go:78: 转账成功，交易哈希: 0xabcd...
    transfer_test.go:87: 交易已确认，区块高度: 10
    transfer_test.go:93: From 最终余额: 999000
    transfer_test.go:94: To 最终余额: 1000
--- PASS: TestTokenTransfer_Basic (5.23s)
```

---

## ⚠️ 常见问题

### 1. 节点未运行

**错误**：
```
节点未运行，请先启动节点
```

**解决**：
```bash
cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git
bash scripts/testing/common/test_init.sh
```

### 2. 交易确认超时

**错误**：
```
交易确认超时
```

**解决**：
- 检查节点是否正常运行
- 增加超时时间：`-timeout 180s`
- 检查网络连接

### 3. 余额不足

**错误**：
```
余额不足
```

**解决**：
- 测试会自动为账户充值（通过挖矿）
- 如果仍然失败，检查节点是否正常出块

---

## 📚 更多信息

- 测试规划：`TESTING_PLAN.md`
- 快速开始：`QUICK_START.md`
- 实施总结：`IMPLEMENTATION_SUMMARY.md`

---

