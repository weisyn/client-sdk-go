# Go/JS SDK 能力对比

## 📌 版本信息

- **版本**：1.0.0
- **状态**：stable
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go/JS Client SDK

---

## 📋 概述

本文档对比 WES Client SDK Go 版本和 JavaScript/TypeScript 版本的能力差异，帮助开发者选择合适的 SDK 和了解两个 SDK 的特性。

---

## ✅ 业务服务能力对比

### 1. Token Service

| 方法 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `Transfer` | ✅ | ✅ | 单笔转账 |
| `BatchTransfer` | ✅ | ✅ | 批量转账（同一 tokenID） |
| `Mint` | ✅ | ✅ | 代币铸造 |
| `Burn` | ✅ | ✅ | 代币销毁 |
| `GetBalance` | ✅ | ✅ | 余额查询 |

**结论**：✅ **完全对齐**

---

### 2. Staking Service

| 方法 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `Stake` | ✅ | ✅ | 质押代币 |
| `Unstake` | ✅ | ✅ | 解除质押 |
| `Delegate` | ✅ | ✅ | 委托验证 |
| `Undelegate` | ✅ | ✅ | 取消委托 |
| `ClaimReward` | ✅ | ✅ | 领取奖励 |
| `Slash` | ⚠️ | ⚠️ | 罚没（架构预留） |

**结论**：✅ **完全对齐**（Slash 两边都是架构预留）

---

### 3. Market Service

| 方法 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `SwapAMM` | ✅ | ✅ | AMM 代币交换 |
| `AddLiquidity` | ✅ | ✅ | 添加流动性 |
| `RemoveLiquidity` | ✅ | ✅ | 移除流动性 |
| `CreateVesting` | ✅ | ✅ | 创建归属计划 |
| `ClaimVesting` | ✅ | ✅ | 领取归属代币 |
| `CreateEscrow` | ✅ | ✅ | 创建托管 |
| `ReleaseEscrow` | ✅ | ✅ | 释放托管 |
| `RefundEscrow` | ✅ | ✅ | 退款托管 |

**结论**：✅ **完全对齐**

---

### 4. Governance Service

| 方法 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `Propose` | ✅ | ✅ | 创建提案 |
| `Vote` | ✅ | ✅ | 投票 |
| `UpdateParam` | ✅ | ✅ | 更新参数 |

**结论**：✅ **完全对齐**

---

### 5. Resource Service

| 方法 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `DeployStaticResource` | ✅ | ✅ | 部署静态资源 |
| `DeployContract` | ✅ | ✅ | 部署智能合约 |
| `DeployAIModel` | ✅ | ✅ | 部署AI模型 |
| `GetResource` | ✅ | ✅ | 查询资源信息 |

**结论**：✅ **完全对齐**

---

## 🔧 工具层能力对比

### 1. 地址转换

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| `AddressBytesToBase58` | ✅ | ✅ | 地址转 Base58 |
| `AddressBase58ToBytes` | ✅ | ✅ | Base58 转地址 |
| `AddressHexToBase58` | ✅ | ✅ | 十六进制转 Base58 |
| `AddressBase58ToHex` | ✅ | ✅ | Base58 转十六进制 |
| **异步版本（浏览器支持）** | ❌ | ✅ | `address*Async` 系列函数 |

**结论**：
- ✅ **基础功能对齐**
- ➕ **JS SDK 额外提供异步版本**（浏览器环境必需）

---

### 2. 请求重试

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| 指数退避重试 | ✅ | ✅ | 自动重试网络错误 |
| 可配置重试参数 | ✅ | ✅ | maxRetries, delays, multiplier |
| 自定义重试判断 | ✅ | ✅ | retryable 函数 |
| 重试回调 | ✅ | ✅ | onRetry 回调 |

**结论**：✅ **完全对齐**

---

### 3. 批量操作

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| 批量查询工具 | ✅ | ✅ | `BatchQuery` / `batchQuery` |
| 批量操作工具 | ✅ | ✅ | `batchOperation` |
| 并行执行工具 | ✅ | ✅ | `ParallelExecute` / `parallelExecute` |
| 数组分批处理 | ✅ | ✅ | `BatchArray` / `batchArray` |

**结论**：✅ **完全对齐**

---

### 4. 大文件处理

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| 文件分块处理 | ✅ | ✅ | `ChunkFile` / `chunkFile`, `ProcessFileInChunks` / `processFileInChunks` |
| 流式读取 | ✅ | ✅ | `ReadFileAsStream` / `readFileAsStream` |
| 进度回调 | ✅ | ✅ | `OnProgress` / `onProgress` |
| 处理时间估算 | ✅ | ✅ | `EstimateProcessingTime` / `estimateProcessingTime` |

**结论**：✅ **完全对齐**

---

## 🔐 安全能力对比

### 1. Wallet / Keystore

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| Wallet 创建 | ✅ | ✅ | 创建新钱包 |
| 私钥导入 | ✅ | ✅ | 从私钥创建钱包 |
| 交易签名 | ✅ | ✅ | 签名交易 |
| Keystore 加密存储 | ✅ | ✅ | PBKDF2 + AES-GCM |
| Keystore 恢复 | ✅ | ✅ | 从 Keystore 恢复钱包 |
| **Node.js 完整支持** | ✅ | ✅ | PBKDF2 + AES-GCM 完整实现 |
| **浏览器完整支持** | N/A | ✅ | Web Crypto API |

**结论**：
- ✅ **基础功能对齐**
- ✅ **JS SDK 支持浏览器和 Node.js 环境**

---

## 🌐 客户端传输层对比

### 1. 传输协议

| 协议 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| HTTP | ✅ | ✅ | JSON-RPC over HTTP |
| WebSocket | ✅ | ✅ | JSON-RPC over WebSocket |
| gRPC | ✅ | ❌ | gRPC 客户端 |

**结论**：
- ⚠️ **JS SDK 缺少 gRPC 支持**
- **说明**：前端通常不需要 gRPC，这是合理的差异

---

### 2. 客户端功能

| 功能 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| 请求重试 | ✅ | ✅ | 指数退避重试 |
| 超时控制 | ✅ | ✅ | 可配置超时 |
| 调试日志 | ✅ | ✅ | 调试模式 |
| 自定义请求头 | ✅ | ✅ | 自定义 headers |
| 连接池 | ✅ | ✅ | HTTP 连接复用 |

**结论**：✅ **完全对齐**

---

## 📚 文档与测试对比

### 1. 文档

| 文档类型 | Go SDK | JS SDK | 说明 |
|----------|--------|--------|------|
| API 参考 | ✅ | ✅ | 完整的 API 文档 |
| 架构文档 | ✅ | ✅ | 架构设计说明 |
| 使用示例 | ✅ | ✅ | 代码示例 |
| 浏览器兼容性 | N/A | ✅ | 浏览器使用指南 |
| 性能优化 | ✅ | ✅ | 性能优化指南 |
| 最佳实践 | ✅ | ✅ | 最佳实践指南 |

**结论**：
- ✅ **核心文档对齐**
- ➕ **JS SDK 提供浏览器兼容性文档**（语言特性）

---

### 2. 测试

| 测试类型 | Go SDK | JS SDK | 说明 |
|----------|--------|--------|------|
| 单元测试 | ✅ | ✅ | 服务层单元测试已完成 |
| 集成测试 | ✅ | ✅ | 集成测试框架已完成 |
| 工具函数测试 | ✅ | ✅ | 基础工具测试完成 |
| 服务层测试 | ✅ | ✅ | Token/Staking/Market/Governance/Resource 测试已完成 |

**结论**：✅ **测试覆盖对齐**

---

## 🎯 语言特性差异

### JS SDK 独有能力（浏览器/前端场景）

| 能力 | 说明 | Go SDK 是否需要 |
|------|------|----------------|
| 异步地址转换 | 浏览器 Web Crypto API 支持 | ❌ 不需要 |
| 浏览器文件上传 | File API 支持 | ❌ 不需要 |
| 浏览器环境检测 | 检测运行环境 | ❌ 不需要 |
| Base64 分块编码 | 处理大数组 | ❌ 不需要 |

**结论**：这些是 JS SDK 的语言特性，Go SDK 不需要对齐。

---

### Go SDK 独有能力（服务端场景）

| 能力 | 说明 | JS SDK 是否需要 |
|------|------|----------------|
| gRPC 客户端 | 高性能 RPC | ⚠️ 可选（前端通常不需要） |
| 文件系统操作 | 直接文件读写 | ❌ 不需要（浏览器不支持） |

**结论**：这些是 Go SDK 的语言特性，JS SDK 不需要对齐。

---

## 📊 总体对比总结

### 业务能力：✅ 100% 对齐

- **Token Service**: 5/5 ✅
- **Staking Service**: 6/6 ✅
- **Market Service**: 8/8 ✅
- **Governance Service**: 3/3 ✅
- **Resource Service**: 4/4 ✅

**总计**: 26/26 业务方法完全对齐

---

### 工具层能力

| 类别 | Go SDK | JS SDK | 差异说明 |
|------|--------|--------|----------|
| 地址转换 | ✅ | ✅ | JS 额外提供异步版本 |
| 请求重试 | ✅ | ✅ | 完全对齐 |
| 批量操作 | ✅ | ✅ | 完全对齐 |
| 大文件处理 | ✅ | ✅ | 完全对齐 |

---

### 安全能力：✅ 完全对齐

- Wallet 功能：✅ 对齐
- Keystore 功能：✅ 对齐

---

### 传输层能力

| 协议 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| HTTP | ✅ | ✅ | 对齐 |
| WebSocket | ✅ | ✅ | 对齐 |
| gRPC | ✅ | ❌ | Go 独有（合理差异） |

---

### 文档与测试

| 项目 | Go SDK | JS SDK | 说明 |
|------|--------|--------|------|
| API 文档 | ✅ | ✅ | 对齐 |
| 架构文档 | ✅ | ✅ | 对齐 |
| 性能文档 | ✅ | ✅ | 对齐 |
| 浏览器兼容性 | N/A | ✅ | JS 独有 |
| 单元测试 | ✅ | ✅ | 对齐 |
| 集成测试 | ✅ | ✅ | 对齐 |

---

## 💡 使用建议

### 选择 Go SDK 的场景

- ✅ 后端服务 / 后台任务
- ✅ 需要 gRPC 高性能通信
- ✅ 需要直接文件系统操作
- ✅ 需要完整的测试覆盖

### 选择 JS SDK 的场景

- ✅ 前端应用 / Web 应用
- ✅ Node.js 工具 / 脚本
- ✅ 需要浏览器环境支持
- ✅ 需要批量操作工具
- ✅ 需要大文件处理工具

---

## 📚 相关文档

- [Go SDK README](../README.md)
- [JS SDK README](../../client-sdk-js.git/README.md)
- [Go SDK 文档](../README.md)
- [JS SDK 文档](../../client-sdk-js.git/docs/README.md)

---

**最后更新**: 2025-11-17  
**状态**: ✅ 核心功能已对齐，差异均为合理的语言特性差异

