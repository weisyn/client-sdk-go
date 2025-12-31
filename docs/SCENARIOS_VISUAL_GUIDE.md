# Client SDK Go - 场景可视化指南

**版本**: v1.0.0  


---

## 📋 文档定位

> 📌 **重要说明**：本文档提供 **SDK 相关的简明架构/场景图**。  
> 如需了解详细业务流图，请参考主仓库文档。

**本文档目标**：
- 提供 SDK 内部分层架构图
- 提供 SDK 与平台其他组件的交互图
- 可视化场景流程

---

## 🏗️ 架构图

### SDK 内部分层架构

```mermaid
graph TB
    subgraph "L3: 业务服务层（业务开发者使用）"
        direction LR
        BUSINESS[services/<br/>业务语义封装<br/>Token/Staking/Market等]
        BUSINESS --> TOKEN[token/<br/>转账/铸造/销毁]
        BUSINESS --> STAKING[staking/<br/>质押/解质押]
        BUSINESS --> MARKET[market/<br/>托管/释放]
        BUSINESS --> GOV[governance/<br/>提案/投票]
        BUSINESS --> RESOURCE_DEPLOY[resource/<br/>资源部署]
    end
    
    subgraph "L2: 中层服务层（Explorer 场景）"
        direction LR
        MIDDLE[services/<br/>Resource/Transaction/Event]
        MIDDLE --> RESOURCE_SVC[resource/<br/>资源查询]
        MIDDLE --> TX_SVC[transaction/<br/>交易查询]
        MIDDLE --> EVENT_SVC[event/<br/>事件订阅]
    end
    
    subgraph "L1: 底层客户端（RPC 封装）"
        direction LR
        CLIENT[client/<br/>WESClient]
        CLIENT --> HTTP[HTTP Client]
        CLIENT --> GRPC[gRPC Client]
        CLIENT --> WS[WebSocket Client]
    end
    
    subgraph "钱包层（独立）"
        direction LR
        WALLET[wallet/<br/>Wallet]
        KEYSTORE[Keystore]
    end
    
    BUSINESS --> MIDDLE
    MIDDLE --> CLIENT
    CLIENT --> NODE[WES 节点]
    BUSINESS -.签名.-> WALLET
    MIDDLE -.签名.-> WALLET
    
    style BUSINESS fill:#FF9800,color:#fff
    style MIDDLE fill:#4CAF50,color:#fff
    style CLIENT fill:#2196F3,color:#fff
    style WALLET fill:#FFC107,color:#000
    style NODE fill:#9E9E9E,color:#fff
```

### SDK 在 WES 7 层架构中的位置

```mermaid
graph TB
    subgraph DEV_ECOSYSTEM["🎨 应用层 & 开发者生态"]
        direction TB
        subgraph SDK_LAYER["SDK 工具链"]
            direction LR
            CLIENT_SDK["Client SDK<br/>Go/JS/Python/Java<br/>📱 DApp·钱包·浏览器<br/>⭐ client-sdk-go<br/>链外应用开发"]
            CONTRACT_SDK["Contract SDK (WASM)<br/>Go/Rust/AS/C<br/>📜 智能合约开发<br/>链上合约开发<br/>github.com/weisyn/contract-sdk-go"]
            AI_SDK["AI SDK (ONNX)"]
        end
        subgraph END_USER_APPS["终端应用"]
            direction LR
            WALLET_APP["Wallet<br/>钱包应用"]
            EXPLORER["Explorer<br/>区块浏览器"]
            DAPP["DApp<br/>去中心化应用"]
        end
    end
    
    subgraph API_GATEWAY["🌐 API 网关层"]
        direction LR
        JSONRPC["JSON-RPC 2.0<br/>:28680"]
        HTTP["HTTP REST<br/>/api/v1/*"]
        GRPC["gRPC<br/>:28682"]
        WS["WebSocket<br/>:28681"]
    end
    
    subgraph BIZ_LAYER["💼 业务服务层"]
        APP_SVC["App Service<br/>应用编排·生命周期"]
    end
    
    WALLET_APP --> CLIENT_SDK
    EXPLORER --> CLIENT_SDK
    DAPP --> CLIENT_SDK
    
    CLIENT_SDK --> JSONRPC
    CLIENT_SDK --> HTTP
    CLIENT_SDK --> GRPC
    CLIENT_SDK --> WS
    
    JSONRPC --> APP_SVC
    HTTP --> APP_SVC
    GRPC --> APP_SVC
    WS --> APP_SVC
    
    style CLIENT_SDK fill:#81C784,color:#fff,stroke:#4CAF50,stroke-width:3px
    style API_GATEWAY fill:#64B5F6,color:#fff
    style BIZ_LAYER fill:#FFB74D,color:#333
```

---

## 📊 数据流图

### 查询流程

```mermaid
sequenceDiagram
    participant App as 应用层
    participant Service as Service 层 (L2/L3)
    participant Client as WESClient (L1)
    participant Node as WES 节点
    
    App->>Service: GetResource(resourceID)
    Service->>Client: GetResource(resourceID)
    Client->>Node: wes_getResource RPC
    Node-->>Client: ResourceInfo
    Client-->>Service: ResourceInfo
    Service-->>App: ResourceInfo
```

### 交易流程

```mermaid
sequenceDiagram
    participant App as 应用层
    participant Service as Service 层 (L3)
    participant Builder as TransactionBuilder
    participant Client as WESClient (L1)
    participant Wallet as Wallet
    participant Node as WES 节点
    
    App->>Service: Transfer(...)
    Service->>Builder: BuildTransaction(...)
    Builder->>Client: ListUTXOs(...)
    Client->>Node: wes_getUTXO RPC
    Node-->>Client: UTXO[]
    Client-->>Builder: UTXO[]
    Builder->>Builder: 构造交易草稿 (DraftJSON)
    Builder->>Client: wes_buildTransaction RPC
    Client->>Node: wes_buildTransaction RPC
    Node-->>Client: UnsignedTx
    Client-->>Builder: UnsignedTx
    Builder-->>Service: UnsignedTx
    Service->>Wallet: SignTransaction(unsignedTx)
    Wallet-->>Service: SignedTx
    Service->>Client: SubmitTransaction(signedTx)
    Client->>Node: wes_sendRawTransaction RPC
    Node-->>Client: TxHash
    Client-->>Service: TxHash
    Service-->>App: TxHash
```

### 事件订阅流程

```mermaid
sequenceDiagram
    participant App as 应用层
    participant Service as EventService (L2)
    participant Client as WESClient (L1)
    participant Node as WES 节点
    
    App->>Service: SubscribeEvents(filters)
    Service->>Client: SubscribeEvents(filters)
    Client->>Node: wes_subscribeEvents RPC (WebSocket)
    Node-->>Client: EventStream
    Client-->>Service: EventStream
    Service-->>App: EventStream
    
    loop 事件流
        Node->>Client: Event
        Client->>Service: Event
        Service->>App: Event
    end
```

---

## 🎯 场景图

### DApp 开发场景

```mermaid
graph TB
    subgraph DAPP["DApp 应用"]
        FRONTEND[前端 UI]
        BACKEND[后端服务]
    end
    
    subgraph SDK["Client SDK"]
        TOKEN_SVC[Token Service]
        STAKING_SVC[Staking Service]
        WES_CLIENT[WESClient]
    end
    
    subgraph NODE["WES 节点"]
        API[API Gateway]
    end
    
    FRONTEND --> BACKEND
    BACKEND --> TOKEN_SVC
    BACKEND --> STAKING_SVC
    TOKEN_SVC --> WES_CLIENT
    STAKING_SVC --> WES_CLIENT
    WES_CLIENT --> API
    
    style DAPP fill:#E3F2FD
    style SDK fill:#C8E6C9
    style NODE fill:#FFF9C4
```

### 钱包应用场景

```mermaid
graph TB
    subgraph WALLET_APP["钱包应用"]
        UI[用户界面]
        WALLET_MGR[钱包管理]
        TX_MGR[交易管理]
    end
    
    subgraph SDK["Client SDK"]
        WALLET[Wallet]
        SERVICES[业务服务]
        WES_CLIENT[WESClient]
    end
    
    subgraph NODE["WES 节点"]
        API[API Gateway]
    end
    
    UI --> WALLET_MGR
    UI --> TX_MGR
    WALLET_MGR --> WALLET
    TX_MGR --> SERVICES
    SERVICES --> WES_CLIENT
    WES_CLIENT --> API
    
    style WALLET_APP fill:#E3F2FD
    style SDK fill:#C8E6C9
    style NODE fill:#FFF9C4
```

### Workbench Explorer 场景

```mermaid
graph TB
    subgraph WORKBENCH["Workbench"]
        RESOURCE_EXPLORER[Resource Explorer]
        HISTORY_TAB[History Tab]
        EVENTS_TAB[Events Tab]
    end
    
    subgraph SDK["Client SDK"]
        RESOURCE_SVC[Resource Service]
        TX_SVC[Transaction Service]
        EVENT_SVC[Event Service]
        WES_CLIENT[WESClient]
    end
    
    subgraph NODE["WES 节点"]
        API[API Gateway]
    end
    
    RESOURCE_EXPLORER --> RESOURCE_SVC
    HISTORY_TAB --> TX_SVC
    EVENTS_TAB --> EVENT_SVC
    RESOURCE_SVC --> WES_CLIENT
    TX_SVC --> WES_CLIENT
    EVENT_SVC --> WES_CLIENT
    WES_CLIENT --> API
    
    style WORKBENCH fill:#E3F2FD
    style SDK fill:#C8E6C9
    style NODE fill:#FFF9C4
```

---

## 🔗 相关文档

- [SDK 架构](./SDK_ARCHITECTURE.md) - 架构设计
- [应用场景分析](./APPLICATION_SCENARIOS_ANALYSIS.md) - 场景分析
- [WES 系统架构文档](../../../weisyn.git/docs/system/architecture/1-STRUCTURE_VIEW.md) - 平台架构（主仓库）

---

  
**维护者**: WES Core Team
