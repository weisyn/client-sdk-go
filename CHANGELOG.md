# 变更日志

**版本**：1.0  
**状态**：stable  
**最后更新**：2025-11-26  
**所有者**：WES SDK 团队

---

本文档记录 WES Client SDK for Go 的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [0.1.2-alpha] - 2025-12-31

### 改进

- 与 WES v0.1.0 版本对齐
- 更新依赖和文档

---

## [0.1.1-alpha] - 2025-11-XX

### 修复

- Workbench 集成相关问题修复

**注意**: 此版本已发布，详细信息请参考 [_dev/publishing/versions/v0.1.1-alpha/release.md](./_dev/publishing/versions/v0.1.1-alpha/release.md)

---

## [0.1.0-alpha] - 2025-11-18

### 新增

#### 核心功能
- ✅ **Client 客户端**
  - HTTP 客户端实现（JSON-RPC 2.0）
  - WebSocket 客户端实现（基础功能）
  - gRPC 客户端实现（基础功能）
  - 客户端配置和错误处理

- ✅ **Wallet 钱包**
  - 密钥生成（secp256k1）
  - 私钥导入
  - 交易签名和消息签名
  - 地址派生（Keccak-256）

- ✅ **Token 服务**
  - 单笔转账
  - 批量转账
  - 代币铸造
  - 代币销毁
  - 余额查询

- ✅ **业务服务骨架**
  - Staking 服务（类型定义和接口）
  - Market 服务（类型定义和接口）
  - Governance 服务（类型定义和接口）
  - Resource 服务（类型定义和接口，部分实现）

- ✅ **工具函数**
  - 地址转换和验证
  - 十六进制字符串转换
  - 批量操作支持

- ✅ **Keystore**
  - Keystore 加密存储（基础实现）
  - 密码验证

- ✅ **构建和测试**
  - Go modules 支持
  - 单元测试配置
  - 集成测试框架

- ✅ **文档**
  - README.md
  - API 参考文档
  - 项目结构文档
  - 实现状态文档

### 已知限制

根据 [实施计划](./_dev/IMPLEMENTATION_PLAN.md)，client-sdk-go 的核心功能已完成。当前限制主要与节点 API 和 JS SDK 的进度相关：

- ⚠️ WebSocket 事件订阅功能需要与节点 API 进一步对齐（与 client-sdk-js 保持一致）
- ⚠️ 部分业务服务（Staking、Market、Governance）的完整实现需要节点 API 支持（与 client-sdk-js 保持一致）

---

## [未发布]

### 计划中

根据 [实施计划](./_dev/IMPLEMENTATION_PLAN.md) 和与 client-sdk-js 保持一致的原则：

#### 与节点 API 对齐（与 client-sdk-js 同步）

- [ ] 完善 WebSocket 事件订阅（需要节点 API 支持）
- [ ] 完善 Staking 服务实现（需要节点 API 支持）
- [ ] 完善 Market 服务实现（需要节点 API 支持）
- [ ] 完善 Governance 服务实现（需要节点 API 支持）

#### 测试与质量

- [ ] 增加单元测试覆盖率
- [ ] 增加集成测试
- [ ] 性能优化

#### 文档与工具

- [ ] 增加更多示例代码

---

