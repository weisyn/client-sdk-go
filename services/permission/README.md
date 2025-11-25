# Permission Service - 权限管理服务

Permission Service 提供资源权限管理功能，包括所有权转移、协作者管理、委托授权和时间/高度锁。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/permission"

permissionService := permission.NewService(client)

// 转移所有权
result, err := permissionService.TransferOwnership(ctx, permission.TransferOwnershipIntent{
    ResourceID:      "0x...:0",
    NewOwnerAddress: "WES1...",
    Memo:            "转移给新所有者",
}, wallet)

// 更新协作者
result, err := permissionService.UpdateCollaborators(ctx, permission.UpdateCollaboratorsIntent{
    ResourceID:         "0x...:0",
    RequiredSignatures: 2,
    Collaborators:      []string{"WES1...", "WES1..."},
}, wallet)

// 授予委托授权
result, err := permissionService.GrantDelegation(ctx, permission.GrantDelegationIntent{
    ResourceID:      "0x...:0",
    DelegateAddress: "WES1...",
    Operations:      []string{"reference", "execute", "query"},
    ExpiryBlocks:    14400,
}, wallet)

// 设置时间锁
result, err := permissionService.SetTimeOrHeightLock(ctx, permission.SetTimeOrHeightLockIntent{
    ResourceID:      "0x...:0",
    UnlockTimestamp: &unlockTimestamp,
}, wallet)
```

## 📚 API 参考

### TransferOwnership - 转移所有权

转移资源的所有权到新地址。

**参数**：
- `ResourceID`: 资源 ID（格式：`txId:outputIndex`）
- `NewOwnerAddress`: 新所有者地址（Base58 或 hex）
- `Memo`: 可选备注

**返回**：
- `TxHash`: 交易哈希
- `Success`: 是否成功

### UpdateCollaborators - 更新协作者

更新资源的协作者列表和签名要求（MultiKey 管理）。

**参数**：
- `ResourceID`: 资源 ID
- `RequiredSignatures`: 需要的签名数（M）
- `Collaborators`: 协作者地址列表

**返回**：
- `TxHash`: 交易哈希
- `Success`: 是否成功

### GrantDelegation - 授予委托授权

授予其他地址临时使用资源的权限。

**参数**：
- `ResourceID`: 资源 ID
- `DelegateAddress`: 被委托者地址
- `Operations`: 授权操作类型（`reference`, `execute`, `query`, `consume`, `transfer`, `stake`, `vote`）
- `ExpiryBlocks`: 过期区块数（0 = 永不过期）
- `MaxValuePerOperation`: 单次操作最大价值（可选）

**返回**：
- `TxHash`: 交易哈希
- `Success`: 是否成功

### SetTimeOrHeightLock - 设置时间/高度锁

设置资源在指定时间或区块高度之前无法使用。

**参数**：
- `ResourceID`: 资源 ID
- `UnlockTimestamp`: 解锁时间戳（Unix 秒，可选）
- `UnlockHeight`: 解锁区块高度（可选）

**注意**：`UnlockTimestamp` 和 `UnlockHeight` 必须提供其中一个，不能同时提供。

**返回**：
- `TxHash`: 交易哈希
- `Success`: 是否成功

## 🔧 交易构建器

如果需要更细粒度的控制，可以直接使用交易构建器：

```go
// 构建未签名交易
unsignedTx, err := permission.BuildTransferOwnershipTx(ctx, client, intent)

// 然后手动签名和提交
// ... 签名流程 ...
```

## 📖 详细文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#6-permission-服务-)**

---

**最后更新**: 2025-11-XX

