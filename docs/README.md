# WES Client SDK for Go - 文档中心

欢迎来到 WES Client SDK for Go 的文档中心。这里提供完整的 SDK 使用指南、架构说明和 API 参考。

## 📚 文档导航

### 🚀 快速开始
- **[快速开始指南](getting-started.md)** - 安装、初始化、第一个应用

### 🏗️ 架构设计
- **[架构文档](architecture.md)** - SDK 在 WES 7 层架构中的位置 + SDK 内部架构设计
- **[架构边界](architecture_boundary.md)** - SDK 架构边界与职责划分（详细版）

### 📦 模块说明
- **[业务服务层](modules/services.md)** - Token、Staking、Market、Governance、Resource 服务
- **[钱包模块](modules/wallet.md)** - 密钥管理、交易签名
- **[工具模块](modules/utils.md)** - 地址转换、交易解析等工具函数

### 📖 API 参考
- **[API 参考](reference/api.md)** - 完整的 API 接口文档（待完善）

### 📋 迁移与测试
- **[迁移指南](migration/v1_draft_signing.md)** - 从旧交易构建路径迁移到新路径
- **[测试规划](testing/plan.md)** - 业务语义测试规划

### 📊 开发报告
- **[能力报告](reports/FINAL_CAPABILITY_REPORT.md)** - 服务能力最终报告
- **[实现完成报告](reports/IMPLEMENTATION_COMPLETE.md)** - 实现完成报告
- **[AMM 实现](reports/AMM_IMPLEMENTATION.md)** - AMM 实现文档
- **[架构分析](reports/ARCHITECTURE_ANALYSIS.md)** - 架构分析文档
- **[开发文档](development/)** - 开发过程中的总结文档

## 🔗 相关资源

### WES 主链文档
- **[WES 主项目](https://github.com/weisyn/go-weisyn)** - WES 区块链核心实现
  - Go Module: `github.com/weisyn/v1`
- **[系统架构文档](https://github.com/weisyn/go-weisyn/blob/main/docs/system/architecture/1-STRUCTURE_VIEW.md)** - WES 7 层架构详解
- **[主项目 README](https://github.com/weisyn/go-weisyn/blob/main/README.md)** - WES 产品说明

### WES 生态 SDK

#### Client SDK（链外应用开发）
- **[Client SDK (Go)](https://github.com/weisyn/client-sdk-go)** ⭐ 当前仓库 - 用于链外应用开发（DApp、钱包、浏览器、后端服务）
- **[Client SDK (JS/TS)](https://github.com/weisyn/client-sdk-js)** - JavaScript/TypeScript 版本

#### Contract SDK（链上合约开发）
- **[Contract SDK (Go)](https://github.com/weisyn/contract-sdk-go)** - 用于链上智能合约开发（WASM 合约），支持 Go/Rust/AS/C

> 📖 **区别说明**：
> - **Client SDK**：链外应用通过 JSON-RPC API 与节点交互，不运行在链上
> - **Contract SDK**：智能合约代码运行在链上（WES 节点），通过 HostABI 与链交互

## 📝 文档结构说明

本 `docs/` 目录包含所有文档，按类型分类：

- **结论性文档**：架构设计、API 参考、使用指南、模块说明
- **迁移文档**：`migration/` - 版本迁移指南
- **测试文档**：`testing/` - 测试规划与说明
- **开发报告**：`reports/` - 开发过程中的能力报告、实现报告等
- **开发文档**：`development/` - 开发过程中的总结文档

---

**最后更新**: 2025-11-17
