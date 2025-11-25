# Client SDK Go - 文档中心

**版本**: v1.0.0  


---

<div align="center">

## 📚 文档导航中心

**按角色快速找到你需要的文档**

[👨‍💻 应用开发者](#-应用开发者) • [🏗️ 架构师/贡献者](#️-架构师贡献者) • [📖 参考文档](#-参考文档)

</div>

---

## 📋 文档定位说明

### 文档层次关系

```mermaid
graph TB
    subgraph PLATFORM["🌐 平台级文档（主仓库 weisyn.git/docs）"]
        direction TB
        P1[system/platforms/contracts/<br/>智能合约平台文档]
        P2[system/architecture/<br/>系统架构文档]
        P3[components/core/ispc/<br/>ISPC 核心组件文档]
        P4[error-spec/<br/>错误规范文档]
    end
    
    subgraph SDK["🔧 SDK 文档（本仓库 client-sdk-go.git/docs）"]
        direction TB
        S1[README.md<br/>文档中心]
        S2[DEVELOPER_GUIDE.md<br/>Go SDK 开发者指南]
        S3[API_REFERENCE.md<br/>Go SDK API 参考]
        S4[BUSINESS_SCENARIOS.md<br/>Go SDK 业务场景实现]
        S5[SDK_ARCHITECTURE.md<br/>SDK 内部分层架构]
    end
    
    subgraph MAIN["📖 主 README.md"]
        M1[SDK 总览和快速开始]
    end
    
    MAIN --> S1
    S1 --> S2
    S1 --> S3
    S1 --> S4
    S1 --> S5
    
    S2 -.引用.-> P1
    S2 -.引用.-> P2
    S3 -.引用.-> P2
    S4 -.引用.-> P1
    S5 -.引用.-> P2
    
    style PLATFORM fill:#E3F2FD,color:#000
    style SDK fill:#C8E6C9,color:#000
    style MAIN fill:#FFF9C4,color:#000
```

**核心原则**：
- ✅ **平台级文档**（`weisyn.git/docs`）：系统级、平台级、规范级的权威说明
- ✅ **SDK 文档**（`client-sdk-go.git/docs`）：Go Client SDK 这一层的解读、对接与示例
- ✅ **引用关系**：SDK 文档引用平台文档，避免重复，保持一致性

**主 README.md** 的定位：
- ✅ **用户友好的入口**：快速了解 SDK，30秒上手
- ✅ **核心价值展示**：突出 SDK 的核心能力和优势
- ✅ **简洁的架构说明**：分层架构的概览
- ✅ **文档导航入口**：指向文档中心（本文件）

**docs/README.md**（本文件）的定位：
- ✅ **SDK 文档索引中心**：列出所有 SDK 相关文档及其定位
- ✅ **平台文档索引**：指向主仓库的平台级文档（只读、权威版本）
- ✅ **按角色导航**：为不同角色提供快速导航路径
- ✅ **文档分类**：按用户文档、设计文档、参考文档分类
- ✅ **使用建议**：为不同场景提供文档使用建议

---

## 👨‍💻 应用开发者

### 🚀 快速开始路径

```mermaid
graph LR
    A[主 README<br/>30秒上手] --> B[开发者指南<br/>深入学习]
    B --> C[业务场景指南<br/>实现场景]
    C --> D[API 参考<br/>查阅接口]
    
    style A fill:#E3F2FD
    style B fill:#C8E6C9
    style C fill:#FFF9C4
    style D fill:#F3E5F5
```

**推荐路径**：
1. **[主 README](../README.md)** - 了解 SDK 是什么，30秒上手
2. **[开发者指南](./DEVELOPER_GUIDE.md)** - 深入学习核心概念和开发模式
3. **[业务场景实现指南](./BUSINESS_SCENARIOS.md)** - 学习如何实现业务场景
4. **[API 参考](./API_REFERENCE.md)** - 查阅详细的 API 文档

### 📖 核心文档

#### 必读文档（P0）

- ⭐ **[主 README](../README.md)** - SDK 总览和 30 秒上手
  - SDK 简介和核心价值
  - 安装和第一个应用
  - 核心功能概览
  - 使用场景说明

- 📖 **[开发者指南](./DEVELOPER_GUIDE.md)** - 如何使用 Go SDK 开发应用
  - Go SDK 安装与环境（Go 1.24+）
  - 客户端初始化与配置
  - 业务服务使用（Token、Staking、Market 等）
  - 交易构建与签名流程
  - 事件订阅与处理
  - **引用平台文档**：平台概念（UTXO、交易模型等）请参考主仓库文档

- 📚 **[API 参考](./API_REFERENCE.md)** - Go SDK 接口详细说明
  - WESClient 类型化 API
  - 业务服务层 API（Token、Staking、Market、Governance、Resource）
  - 钱包功能 API
  - 接口参数和返回值
  - 使用示例
  - **引用平台文档**：JSON-RPC API 规范请参考主仓库文档

#### 推荐文档（P1）

- 🎯 **[业务场景实现指南](./BUSINESS_SCENARIOS.md)** - 如何使用 Go SDK 实现业务场景
  - DApp 开发场景
  - 钱包应用场景
  - 后端服务场景
  - 工具链开发场景
  - 每个场景包含：场景说明、实现步骤、代码示例

- 📡 **[WESClient 使用示例](../examples/wesclient.md)** - 类型化 API 使用
  - UTXO 操作示例
  - 资源操作示例
  - 交易操作示例
  - 事件订阅示例

#### 可选文档（P2）

- 🔧 **[业务服务指南](../guides/)** - 各业务服务的详细说明
  - Token 服务完整文档
  - Staking 服务完整文档
  - Market 服务完整文档
  - Governance 服务完整文档
  - Resource 服务完整文档

---

## 🏗️ 架构师/贡献者

### 架构设计文档

#### 核心设计文档（P1）

- 🏗️ **[SDK 内部架构](./SDK_ARCHITECTURE.md)** - SDK 内部分层架构设计
  - **聚焦 SDK 自身**：业务服务层、客户端层、钱包层设计
  - **引用平台架构**：WES 7 层架构请参考主仓库文档
  - 模块组织方式、依赖关系说明、设计决策记录

- 📊 **[应用场景分析](./APPLICATION_SCENARIOS_ANALYSIS.md)** - SDK 职责边界分析
  - **聚焦 SDK 职责**：在某场景中，Client SDK 负责哪一段？
  - **引用平台场景**：详细业务流图、跨子系统交互请参考主仓库文档
  - SDK 与 Contract SDK、Workbench、节点的职责划分

#### 规划文档（P2）

- 📈 **[架构规划](./ARCHITECTURE_PLAN.md)** - Go SDK 架构规划文档
  - **聚焦 SDK 自身**：Go SDK 未来演进方向（新增业务服务、改进 API 设计等）
  - **引用平台规划**：平台级 roadmap 请参考主仓库文档
  - 技术债务管理

- 📐 **[场景可视化指南](./SCENARIOS_VISUAL_GUIDE.md)** - SDK 相关的简明架构/场景图
  - SDK 内部分层架构图
  - SDK 与平台其他组件的交互图
  - **引用平台文档**：详细业务流图请参考主仓库文档

---

## 📖 参考文档

### SDK 参考文档（本仓库）

- 🔧 **[WES Error Spec 实施](./WES_ERROR_SPEC_IMPLEMENTATION.md)** - Go SDK 如何对接错误规范
  - 错误码映射
  - 错误处理模式
  - **引用平台文档**：WES Error Specification 请参考主仓库文档

- 📋 **[文档体系说明](./DOCUMENTATION_STRUCTURE.md)** - 文档体系与分层说明
  - SDK 文档与平台文档的关系
  - 文档组织结构
  - 与根 README 的映射关系

### 平台参考文档（主仓库）

以下文档来自 `weisyn.git/docs`，是平台级参考文档的权威版本：

- 📘 **[系统架构文档](../../../weisyn.git/docs/system/architecture/1-STRUCTURE_VIEW.md)** - WES 7 层架构详解
- 🔗 **[JSON-RPC API 规范](../../../weisyn.git/docs/reference/json-rpc/)** - JSON-RPC 2.0 API 规范
- 🔧 **[交易模型文档](../../../weisyn.git/docs/tutorials/transactions/)** - UTXO 交易模型详解
- 🌐 **[API 网关层文档](../../../weisyn.git/docs/system/architecture/)** - API 网关层设计

---

## 🎯 快速导航路径

### 新手入门路径

```mermaid
graph TB
    A[主 README<br/>30秒上手] --> B[开发者指南<br/>深入学习]
    B --> C[示例代码<br/>参考实现]
    
    A -.了解平台概念.-> P[平台文档<br/>主仓库]
    B -.了解平台概念.-> P
    
    style A fill:#E3F2FD
    style B fill:#C8E6C9
    style C fill:#FFF9C4
    style P fill:#F3E5F5
```

1. **[主 README](../README.md)** - 了解 SDK 是什么，30秒上手
2. **[开发者指南](./DEVELOPER_GUIDE.md)** - 深入学习核心概念
3. **[示例代码](../examples/)** - 参考实际应用示例
4. **平台文档**（主仓库）- 如需了解平台概念，参考平台文档

### 功能开发路径

```mermaid
graph LR
    A[核心功能] --> B[业务场景指南]
    B --> C[API 参考]
    C --> D[服务文档]
    
    style A fill:#E3F2FD
    style B fill:#C8E6C9
    style C fill:#FFF9C4
    style D fill:#F3E5F5
```

1. **[核心功能](../README.md#-核心能力)** - 了解 SDK 提供的功能
2. **[业务场景实现指南](./BUSINESS_SCENARIOS.md)** - 学习如何实现业务场景
3. **[API 参考](./API_REFERENCE.md)** - 查阅详细的 API 文档
4. **[业务服务指南](../guides/)** - 查看特定服务说明

### 深入理解路径

```mermaid
graph TB
    A[SDK 架构] --> B[SDK 内部架构]
    B --> C[应用场景分析]
    C --> D[系统架构文档<br/>主仓库]
    
    style A fill:#E3F2FD
    style B fill:#C8E6C9
    style C fill:#FFF9C4
    style D fill:#F3E5F5
```

1. **[SDK 架构](../README.md#-架构概览)** - 理解分层架构
2. **[SDK 内部架构](./SDK_ARCHITECTURE.md)** - 了解架构设计决策
3. **[应用场景分析](./APPLICATION_SCENARIOS_ANALYSIS.md)** - 理解 SDK 职责边界
4. **[系统架构文档](../../../weisyn.git/docs/system/architecture/)** - 深入平台架构（主仓库）

---

## 📋 文档分类

### 用户文档（面向应用开发者）

| 文档 | 说明 | 优先级 |
|------|------|--------|
| ⭐ [主 README](../README.md) | SDK 总览和快速开始 | P0 |
| 📖 [开发者指南](./DEVELOPER_GUIDE.md) | 如何使用 Go SDK 开发应用 | P0 |
| 📚 [API 参考](./API_REFERENCE.md) | Go SDK 接口详细说明 | P0 |
| 🎯 [业务场景实现指南](./BUSINESS_SCENARIOS.md) | 如何用 Go SDK 实现业务场景 | P1 |
| 📡 [WESClient 使用示例](../examples/wesclient.md) | 类型化 API 使用 | P1 |
| 🔧 [业务服务指南](../guides/) | 各业务服务的详细说明 | P2 |

### 设计文档（面向架构师和贡献者）

| 文档 | 说明 | 优先级 |
|------|------|--------|
| 🏗️ [SDK 内部架构](./SDK_ARCHITECTURE.md) | SDK 内部分层架构设计 | P1 |
| 📊 [应用场景分析](./APPLICATION_SCENARIOS_ANALYSIS.md) | SDK 职责边界分析 | P1 |
| 📈 [架构规划](./ARCHITECTURE_PLAN.md) | Go SDK 架构规划文档 | P2 |
| 📐 [场景可视化指南](./SCENARIOS_VISUAL_GUIDE.md) | SDK 相关的简明架构图 | P2 |

### 参考文档（面向高级开发者）

| 文档 | 说明 | 来源 |
|------|------|------|
| 🔧 [WES Error Spec 实施](./WES_ERROR_SPEC_IMPLEMENTATION.md) | Go SDK 如何对接错误规范 | SDK 文档 |
| 📋 [文档体系说明](./DOCUMENTATION_STRUCTURE.md) | 文档体系与分层说明 | SDK 文档 |
| 📘 [系统架构文档](../../../weisyn.git/docs/system/architecture/1-STRUCTURE_VIEW.md) | WES 7 层架构详解 | 主仓库 |
| 🔗 [JSON-RPC API 规范](../../../weisyn.git/docs/reference/json-rpc/) | JSON-RPC 2.0 API 规范 | 主仓库 |

---

## 💡 文档使用建议

### 如果你是新手

1. **先看主 README**：了解 SDK 是什么，完成 30 秒上手
2. **阅读开发者指南**：深入学习核心概念和最佳实践
3. **参考示例代码**：学习实际应用示例
4. **查阅平台文档**（主仓库）：如需了解平台概念（UTXO、交易模型等）

### 如果你在开发功能

1. **查看核心功能**：了解 SDK 提供的功能
2. **查阅 API 参考**：查找具体的 API 使用方法
3. **参考业务场景指南**：学习如何实现业务场景
4. **查看服务文档**：深入了解特定服务
5. **查阅平台文档**（主仓库）：如需了解平台级场景和用例

### 如果你想贡献代码

1. **阅读架构设计文档**：理解 SDK 的整体架构
2. **查看应用场景分析**：理解 SDK 的职责边界
3. **参考架构规划**：了解未来演进方向
4. **阅读系统架构文档**（主仓库）：深入理解平台架构

---

## 🔗 相关链接

### WES 平台资源

- [WES 主项目](https://github.com/weisyn/weisyn) - WES 区块链主仓库
- [WES 文档中心](../../../weisyn.git/docs/) - 完整技术文档
- [WES 系统架构](../../../weisyn.git/docs/system/architecture/) - 系统架构详解
- [WES 主 README](../../../weisyn.git/README.md) - WES 项目总览

### SDK 相关资源

- [Client SDK Go](../README.md) - Go 客户端 SDK 主 README
- [Client SDK JS](../../client-sdk-js.git/README.md) - JavaScript/TypeScript 客户端 SDK
- [Contract SDK Go](../../contract-sdk-go.git/README.md) - Go 合约 SDK
- [Contract SDK JS](../../contract-sdk-js.git/README.md) - TypeScript 合约 SDK
- [Model Workbench](../../workbench/model-workbench.git/README.md) - 模型工作台

---

  
**维护者**: WES Core Team

