# SDK 迁移指南

**版本**: 1.0.0  
**最后更新**: 2025-11-17

---

## 📋 概述

本文档说明如何从旧的交易构建和签名路径迁移到新的统一路径。新路径提供了更好的架构边界分离，确保 SDK 不依赖 WES 内部类型。

---

## ⚠️ 废弃的旧路径

以下函数和模式已被废弃，将在未来版本中移除：

### Token Service

- `buildTransferTransaction()` - 已废弃
- `buildBatchTransferTransaction()` - 已废弃  
- `buildBurnTransaction()` - 已废弃

**旧路径流程**:
```
buildXXXTransaction() → 返回 unsignedTxBytes
  ↓
Wallet.SignTransaction(unsignedTxBytes) → 返回 signedTxBytes
  ↓
wes_sendRawTransaction(signedTxHex)
```

**问题**:
- SDK 需要知道 WES 内部 protobuf 格式才能正确签名
- 签名逻辑复杂，容易出错
- 无法支持多输入签名

---

## ✅ 新的推荐路径

### 统一签名流程

所有 Token 操作（Transfer、BatchTransfer、Burn）现在使用统一的签名流程：

```
buildXXXDraft() → 返回 DraftJSON + inputIndex
  ↓
wes_computeSignatureHashFromDraft(draft, inputIndex) → 返回 hash + unsignedTx
  ↓
Wallet.SignHash(hash) → 返回 signature
  ↓
wes_finalizeTransactionFromDraft(draft, unsignedTx, inputIndex, pubkey, signature) → 返回 tx
  ↓
wes_sendRawTransaction(tx)
```

### 多输入签名（批量转账）

对于批量转账等需要多个输入签名的场景：

```
buildBatchTransferDraft() → 返回 DraftJSON + inputIndices[]
  ↓
for each inputIndex in inputIndices:
  wes_computeSignatureHashFromDraft(draft, inputIndex) → hash
  Wallet.SignHash(hash) → signature
  signatures.append({inputIndex, pubkey, signature})
  ↓
wes_finalizeTransactionFromDraft(draft, unsignedTx, signatures[]) → 返回 tx
  ↓
wes_sendRawTransaction(tx)
```

---

## 🔄 迁移步骤

### 1. Transfer（单笔转账）

**旧代码**:
```go
unsignedTxBytes, err := buildTransferTransaction(ctx, client, from, to, amount, tokenID)
if err != nil {
    return err
}
signedTxBytes, err := wallet.SignTransaction(unsignedTxBytes)
if err != nil {
    return err
}
signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
result, err := client.SendRawTransaction(ctx, signedTxHex)
```

**新代码**:
```go
draftJSON, inputIndex, err := buildTransferDraft(ctx, client, from, to, amount, tokenID)
if err != nil {
    return err
}

hashParams := map[string]interface{}{
    "draft":        json.RawMessage(draftJSON),
    "input_index":  inputIndex,
    "sighash_type": "SIGHASH_ALL",
}
hashResult, err := client.Call(ctx, "wes_computeSignatureHashFromDraft", hashParams)
// ... 解析 hash 和 unsignedTx ...

sigBytes, err := wallet.SignHash(hashBytes)
pubCompressed := ethcrypto.CompressPubkey(&wallet.PrivateKey().PublicKey)

finalizeParams := map[string]interface{}{
    "draft":       json.RawMessage(draftJSON),
    "unsignedTx":  unsignedTxHex,
    "input_index": inputIndex,
    "sighash_type": "SIGHASH_ALL",
    "pubkey":      "0x" + hex.EncodeToString(pubCompressed),
    "signature":   "0x" + hex.EncodeToString(sigBytes),
}
finalResult, err := client.Call(ctx, "wes_finalizeTransactionFromDraft", finalizeParams)
// ... 解析 tx ...

result, err := client.SendRawTransaction(ctx, txHex)
```

**或者直接使用 Token Service**:
```go
tokenService := token.NewService(client)
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:    from,
    To:      to,
    Amount:  amount,
    TokenID: tokenID,
}, wallet)
```

### 2. BatchTransfer（批量转账）

**旧代码**:
```go
unsignedTxBytes, err := buildBatchTransferTransaction(ctx, client, from, transfers)
// ... 签名和提交 ...
```

**新代码**:
```go
tokenService := token.NewService(client)
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From:     from,
    Transfers: transfers,
}, wallet)
```

### 3. Burn（销毁）

**旧代码**:
```go
unsignedTxBytes, err := buildBurnTransaction(ctx, client, from, amount, tokenID)
// ... 签名和提交 ...
```

**新代码**:
```go
tokenService := token.NewService(client)
result, err := tokenService.Burn(ctx, &token.BurnRequest{
    From:    from,
    Amount:  amount,
    TokenID: tokenID,
}, wallet)
```

---

## 📅 迁移时间表

- **当前版本 (v1.0.0-alpha)**: 旧路径仍可用，但已标记为废弃
- **v1.1.0**: 旧路径将产生警告日志
- **v2.0.0**: 旧路径将被完全移除

**建议**: 尽快迁移到新路径，以获得更好的稳定性和功能支持。

---

## 🔍 检查清单

迁移前请确认：

- [ ] 所有 `buildTransferTransaction` 调用已迁移
- [ ] 所有 `buildBatchTransferTransaction` 调用已迁移
- [ ] 所有 `buildBurnTransaction` 调用已迁移
- [ ] 所有 `Wallet.SignTransaction(unsignedTxBytes)` 调用已改为 `Wallet.SignHash(hashBytes)`
- [ ] 测试已更新并通过

---

## 📚 相关文档

- [架构边界文档](./ARCHITECTURE_BOUNDARY.md)
- [Token Service 文档](./services/token/README.md)
- [WES JSON-RPC API 文档](https://github.com/weisyn/go-weisyn/blob/main/docs/api/jsonrpc/README.md)

---

## ❓ 常见问题

### Q: 为什么需要迁移？

A: 新路径提供了更好的架构分离，SDK 不需要知道 WES 内部 protobuf 格式，签名逻辑由节点处理，更安全可靠。

### Q: 迁移会影响性能吗？

A: 不会。新路径实际上可能更快，因为减少了 SDK 端的序列化/反序列化操作。

### Q: 如果我的代码直接调用 `buildXXXTransaction` 怎么办？

A: 这些函数仍然可用，但已标记为废弃。建议迁移到新的 `buildXXXDraft` + `wes_computeSignatureHashFromDraft` + `wes_finalizeTransactionFromDraft` 路径，或直接使用 Token Service 的高级 API。

### Q: 合约调用（Mint、Swap 等）需要迁移吗？

A: 不需要。合约调用使用 `wes_callContract` + `return_unsigned_tx=true`，这是合理的路径，不需要迁移。

---

## 📞 支持

如有问题，请参考：
- [GitHub Issues](https://github.com/weisyn/client-sdk-go/issues)
- [文档仓库](https://github.com/weisyn/go-weisyn/tree/main/docs)

