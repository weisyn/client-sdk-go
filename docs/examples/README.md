# 示例代码

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

本文档提供 Go SDK 的示例代码，帮助开发者快速上手。

---

## 📚 示例列表

### 基础示例

- **[简单转账](./simple-transfer.md)** - 基本的代币转账示例
- **[质押流程](./staking-flow.md)** - 完整的质押、领取奖励、解质押流程
- **[批量操作](./batch-operations.md)** - 批量转账和查询示例

### 高级示例

- **[事件订阅](./event-subscription.md)** - WebSocket 事件订阅示例（待实现）

---

## 🚀 快速开始

### 运行示例

```bash
# 克隆 SDK 仓库
git clone https://github.com/weisyn/client-sdk-go.git
cd client-sdk-go

# 运行示例
go run examples/simple-transfer/main.go
```

### 前提条件

- Go 1.21 或更高版本
- WES 节点运行在 `http://localhost:8545`
- 测试账户已充值（如需要）

---

## 🔗 相关文档

- **[快速开始](../getting-started.md)** - 安装和配置
- **[业务指南](../guides/)** - 详细使用指南
- **[API 参考](../api/)** - 完整 API 文档

---

