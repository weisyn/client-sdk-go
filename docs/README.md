# WES Client SDK (Go) 文档中心

---

## 📌 版本信息

- **版本**：0.1.0-alpha
- **状态**：draft
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK

---

## 📖 文档导航

### 🚀 快速开始

- **[快速开始](./getting-started.md)** - 安装、配置、第一个示例
- **[概述](./overview.md)** - SDK 视角的 WES 核心概念

### 📚 核心文档

- **[架构设计](./architecture.md)** - SDK 内部架构和模块划分
- **[API 参考](./api/)** - 完整的 API 文档
  - [Client API](./api/client.md) - 客户端接口
  - [Wallet API](./api/wallet.md) - 钱包功能
  - [Services API](./api/services.md) - 业务服务（Token/Staking/Market/Governance/Resource）

### 🎯 使用指南

- **[业务指南](./guides/)** - 按业务场景组织的使用指南
  - [Token 指南](./guides/token.md) - 转账、批量转账、铸造、销毁
  - [Staking 指南](./guides/staking.md) - 质押、委托、奖励领取
  - [Market 指南](./guides/market.md) - AMM、流动性、托管、归属
  - [Governance 指南](./guides/governance.md) - 提案、投票、参数更新
  - [Resource 指南](./guides/resource.md) - 合约/模型/静态资源部署

### 🔧 参考文档

- **[工具参考](./reference/)** - 底层工具和最佳实践
  - [重试机制](./reference/retry.md) - 请求重试策略
  - [批量操作](./reference/batch.md) - 批量查询和操作
  - [大文件处理](./reference/file.md) - 大文件上传和处理
  - [并发处理](./reference/concurrency.md) - Go 并发最佳实践

### 🧪 测试与故障排查

- **[测试指南](./testing.md)** - 单元测试和集成测试
- **[故障排查](./troubleshooting.md)** - 常见问题和解决方案

### 📋 其他

- **[变更日志](./changelog.md)** - SDK 版本变更记录
- **[迁移指南](./migration.md)** - 版本迁移说明

---

## 🔗 关联文档

### WES 主项目文档

> 💡 **重要**：本 SDK 文档聚焦"如何在 Go 中使用 WES"，系统级概念（共识、ISPC、EUTXO 等）请参考 WES 主项目文档。

- **[WES 项目总览](https://github.com/weisyn/weisyn/blob/main/docs/overview.md)** - WES 核心概念和定位
- **[WES 系统架构](https://github.com/weisyn/weisyn/blob/main/docs/system/architecture/README.md)** - 完整的系统架构设计
- **[JSON-RPC API 参考](https://github.com/weisyn/weisyn/blob/main/docs/reference/api.md)** - 底层 API 接口文档
- **[智能合约平台](https://github.com/weisyn/weisyn/blob/main/docs/system/platforms/contracts/README.md)** - 智能合约开发指南
- **[AI 模型平台](https://github.com/weisyn/weisyn/blob/main/docs/system/platforms/models/README.md)** - AI 模型部署和推理指南

### 其他 SDK

- **[JS SDK 文档](../client-sdk-js.git/docs/README.md)** - JavaScript/TypeScript 版本 SDK
- **[能力对比](./archive/CAPABILITY_COMPARISON.md)** - Go/JS SDK 能力对比（已归档）

---

## 📖 文档阅读建议

### 新手入门路径

1. **了解 WES** → [WES 项目总览](https://github.com/weisyn/weisyn/blob/main/docs/overview.md)
2. **快速上手** → [快速开始](./getting-started.md)
3. **理解概念** → [概述](./overview.md)
4. **实际开发** → [业务指南](./guides/)

### 进阶开发路径

1. **深入架构** → [架构设计](./architecture.md)
2. **API 参考** → [API 文档](./api/)
3. **最佳实践** → [参考文档](./reference/)
4. **问题排查** → [故障排查](./troubleshooting.md)

---

## 📝 文档说明

### 文档定位

本 SDK 文档的定位是：

- ✅ **语言绑定层**：将 WES 系统概念映射到 Go API
- ✅ **开发者视角**：聚焦"如何用代码实现业务需求"
- ✅ **工程实践**：提供最佳实践和常见问题解决方案
- ✅ **Go 特性**：充分利用 Go 的并发、接口、错误处理等特性

### 文档不包含

- ❌ **系统架构详解**：请参考 WES 主项目文档
- ❌ **协议层设计**：请参考 WES 主项目文档
- ❌ **节点部署运维**：请参考 WES 主项目文档

---

## 🔄 Go SDK 特有内容

### 并发处理

Go SDK 充分利用 Go 的并发特性：

- **Goroutine**：并发执行多个操作
- **Channel**：协程间通信
- **Context**：取消和超时控制

详见：[并发处理参考](./reference/concurrency.md)

### 服务集成

Go SDK 面向后端服务开发：

- **长期运行服务**：连接池管理、重试机制
- **微服务架构**：接口设计和适配器模式
- **gRPC 支持**：高性能 RPC 调用

详见：[服务集成指南](./guides/service-integration.md)（待创建）

---

**最后更新**: 2025-11-17

