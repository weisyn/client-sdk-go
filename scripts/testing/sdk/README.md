# SDK 测试脚本

---

## 📌 版本信息

- **版本**：1.0
- **状态**：draft
- **最后更新**：2025-01-23
- **适用范围**：Go Client SDK 测试脚本

---

## 🎯 脚本说明

这些脚本用于自动化执行 SDK 的集成测试，包括：

1. **节点启动管理**：自动启动/停止 WES 节点
2. **测试环境初始化**：清理测试数据，准备测试环境
3. **测试执行**：运行 Go 集成测试
4. **结果报告**：生成测试报告

---

## 📁 脚本列表

- `test_init.sh` - SDK 测试环境初始化
- `token_test.sh` - Token 服务测试脚本
- `staking_test.sh` - Staking 服务测试脚本（待实现）
- `market_test.sh` - Market 服务测试脚本（待实现）
- `governance_test.sh` - Governance 服务测试脚本（待实现）
- `resource_test.sh` - Resource 服务测试脚本（待实现）

---

## 🚀 使用方法

### 运行单个服务测试

```bash
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
bash scripts/testing/sdk/token_test.sh
```

### 运行所有测试

```bash
cd /Users/qinglong/go/src/chaincodes/WES/sdk/client-sdk-go.git
bash scripts/testing/sdk/token_test.sh
bash scripts/testing/sdk/staking_test.sh
# ... 其他服务测试
```

---

## ⚠️ 前置要求

1. **WES 节点**：需要 WES 节点代码在 `/Users/qinglong/go/src/chaincodes/WES/weisyn.git`
2. **Go 环境**：需要 Go 1.24+ 环境
3. **测试依赖**：已安装 `github.com/stretchr/testify`

---

