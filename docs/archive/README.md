# 归档文档说明

---

## 📌 版本信息

- **版本**：0.1.0-alpha
- **状态**：archived
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK（已归档）

---

## 📖 概述

本目录包含 Go SDK 的归档文档，这些文档已按新的文档结构重新组织或整合。

---

## 🔄 文档迁移状态

| 归档文档 | 新位置 | 状态 |
|---------|--------|------|
| `architecture.md` | `../architecture.md` | ✅ 已迁移 |
| `getting-started.md` | `../getting-started.md` | ✅ 已迁移 |
| `performance.md` | `../reference/retry.md`, `../reference/batch.md`, `../reference/file.md` | ✅ 已整合 |
| `architecture_boundary.md` | 保留在归档中（详细边界说明） | 📦 归档保留 |

---

## 📝 归档文档说明

### architecture.md

**状态**：已迁移到 `../architecture.md`

**说明**：SDK 架构设计文档，包括在 WES 7 层架构中的位置和 SDK 内部分层设计。

---

### getting-started.md

**状态**：已迁移到 `../getting-started.md`

**说明**：快速开始指南，包括安装、配置和第一个示例。

---

### performance.md

**状态**：已整合到参考文档

**说明**：性能优化指南，内容已整合到：
- `../reference/retry.md` - 重试机制
- `../reference/batch.md` - 批量操作
- `../reference/file.md` - 大文件处理

---

### architecture_boundary.md

**状态**：保留在归档中

**说明**：详细的架构边界与职责划分说明，作为历史参考保留。

---

## 🔗 相关文档

- **[新文档中心](../README.md)** - 完整的文档导航
- **[JS SDK 文档](../../client-sdk-js.git/docs/README.md)** - 参考 JS SDK 文档结构

---

## 📌 注意事项

- ⚠️ 本目录中的文档为历史版本，仅供参考
- ✅ 请使用新文档结构中的文档（`../` 目录）
- 📦 归档文档保留作为历史记录，不进行更新

---
