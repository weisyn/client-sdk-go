## 🧱 client-sdk-go 架构边界与职责划分

> 版本：v0.1（草案）  
> 目标：明确 Go SDK 与 WES 内核 (`github.com/weisyn/go-weisyn`) 之间的边界，避免 SDK 与内部实现耦合。

---

### 1. 位置与角色

- **仓库**：`github.com/weisyn/client-sdk-go`（独立 SDK 仓库）
- **GitHub**：https://github.com/weisyn/client-sdk-go
- **架构层级**（对应 `1-STRUCTURE_VIEW.md` 中的 7 层）：
  - SDK 层（Client SDK）：位于 API 网关层之上，面向：
    - DApp / 钱包 / 后端服务
    - CLI（未来 `cmd/weisyn` 也应切换为使用本 SDK）

---

### 2. 边界约束（Hard Boundaries）

1. **禁止依赖 WES 内部包**
   - 不允许依赖：
     - `github.com/weisyn/v1/internal/...`
     - `github.com/weisyn/v1/pkg/interfaces/...`
     - `github.com/weisyn/v1/pb/...`（protobuf 类型）
   - SDK 只依赖：
     - Go 标准库
     - 通用第三方库（如 `grpc`、`btcsuite/btcutil`、`testify` 等）

2. **只通过 `internal/api` 暴露的协议访问节点**
   - JSON-RPC 2.0（主协议）
   - HTTP REST（用于健康检查、资源查询等）
   - WebSocket（后续用于事件订阅）
   - gRPC（高性能场景）

3. **不在 SDK 中重新实现链内“语义”**
   - 不复制 EUTXO、锁定条件（SingleKeyLock / HeightLock / ContractLock / DelegationLock）、`SingleKeyProof` 等内部语义。
   - 所有这些概念的权威定义和演化留在 WES 内核中，通过 API 暴露能力，而不是类型。

4. **SDK 只负责：**
   - 私钥管理（keystore、内存钱包）
   - 网络通信（HTTP/gRPC/WebSocket 客户端）
   - 高层业务语义封装（Token / Staking / Market / Governance / Resource）

---

### 3. 交易相关职责分工（采用方案 B）

#### 3.1 链内（WES）负责

- DraftJSON 的解析与验证：
  - `BuildTransactionFromDraft`
  - `ValidateDraftJSON`
- UTXO 选择、锁定条件、业务意图扩展（Intents）：
  - 对应 `DraftJSON` 中的 `inputs` / `outputs` / `intents` 字段。
- SignatureHash 计算与单密钥证明：
  - `ComputeSignatureHash`
  - `SingleKeyProof` 结构
  - 解锁证明验证插件
- 交易提交与验证：
  - `wes_sendRawTransaction`

#### 3.2 SDK（client-sdk-go）负责

- 根据业务场景构建 **DraftJSON**：
  - Token：Transfer / BatchTransfer / Mint / Burn
  - Staking：Stake / Unstake / Delegate / Undelegate
  - Market / Governance / Resource 等
- 调用链上的 **通用交易辅助 API**：
  - `wes_buildTransaction`（已有）
  - 规划中的：
    - `wes_computeSignatureHashFromDraft`
    - `wes_finalizeTransactionFromDraft`
- 使用本地私钥对链给出的 `hash` 做签名，并将 `pubkey + signature` 回传给链进行组装。

> 关键点：**签名在 SDK；签名语义与证明结构在链内。**

---

### 4. 规划中的通用交易 API（WES 侧，供 SDK 使用）

以下 API 在 `github.com/weisyn/go-weisyn/internal/api/jsonrpc/methods/tx.go` 中设计和实现，SDK 只作为调用方：

1. `wes_buildTransaction(draft)`
   - **已有**：从 DraftJSON 构建内部交易，并返回 `unsignedTx`（当前版本已经在使用）。

2. `wes_computeSignatureHashFromDraft`
   - **计划中**：从 DraftJSON/构建结果中，根据 inputIndex & sighashType 计算待签名哈希。
   - SDK 侧使用本地私钥对该 hash 做签名。

3. `wes_finalizeTransactionFromDraft`
   - **计划中**：接受 DraftJSON + inputIndex + pubkey + signature，生成带 `SingleKeyProof` 的完整交易（protobuf 序列化）。

SDK 侧的调用模式将统一为：

1. 构建 DraftJSON（业务层逻辑）。
2. `wes_computeSignatureHashFromDraft` → 得到 hash。
3. 使用 Wallet 签名 hash。
4. `wes_finalizeTransactionFromDraft` → 得到完整 tx 字节。
5. `wes_sendRawTransaction` → 提交交易。

---

### 5. 与内部 client 的关系（过渡期）

- 短期内，WES 中仍存在 `client/` 目录，用于 CLI 兼容。
- 长期目标：
  - CLI / 终端工具逐步改用 `client-sdk-go`。
  - `client/` 只保留极少量必要 glue 代码，最终退役。
- 迁移规划详见：
  - `github.com/weisyn/go-weisyn/client/CLIENT_MIGRATION_PLAN.md` (如存在)

---

### 6. 测试与演进

- 所有 **集成测试** 优先在 SDK 层实现：
  - `test/integration/services/token/*`
  - `test/integration/services/staking/*`
  - 后续扩展到 Market / Governance / Resource。
- 节点端只保留必要的 API / 内核测试：
  - JSON-RPC 方法正确性
  - 交易语义与验证正确性

> 本文件会随着 `wes_computeSignatureHashFromDraft` / `wes_finalizeTransactionFromDraft` 等 API 的落地持续更新，并作为多语言 SDK 的参考边界说明。


