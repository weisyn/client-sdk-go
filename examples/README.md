# Examples - 示例代码

**版本**: 1.0.0-alpha  
**状态**: ✅ 基础示例已完成  
**最后更新**: 2025-01-23

---

## 📋 概述

Examples 目录包含使用 WES Client SDK 的完整示例代码，帮助开发者快速上手和理解 SDK 的使用方法。

---

## 📦 示例列表

### 1. Simple Transfer ✅

**路径**: `examples/simple-transfer/`  
**描述**: 简单的转账示例，演示如何使用 SDK 进行单笔转账

**功能**:
- ✅ 创建 HTTP 客户端
- ✅ 创建钱包
- ✅ 创建 Token 服务
- ✅ 执行转账
- ✅ 处理结果

**运行**:
```bash
cd examples/simple-transfer
go run main.go
```

**代码结构**:
```
simple-transfer/
  └─> main.go
      ├─> 1. 创建客户端
      ├─> 2. 创建钱包
      ├─> 3. 创建 Token 服务
      ├─> 4. 准备转账参数
      └─> 5. 执行转账
```

---

## 🚀 快速开始

### 运行示例

```bash
# 进入示例目录
cd examples/simple-transfer

# 运行示例
go run main.go
```

### 配置节点地址

修改 `main.go` 中的节点地址：

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545", // 修改为你的节点地址
    Protocol: client.ProtocolHTTP,
    Timeout:  30,
    Debug:    true, // 启用调试日志
}
```

### 配置私钥

修改 `main.go` 中的私钥：

```go
// 注意：实际应用中应该从 Keystore 加载
privateKeyHex := "0x..." // 替换为你的私钥
wallet, err := wallet.NewWalletFromPrivateKey(privateKeyHex)
```

---

## 📝 示例代码说明

### Simple Transfer 示例

```go
┌─────────────────────────────────────────┐
│         Simple Transfer 流程            │
└─────────────────────────────────────────┘

1. 创建客户端
   client.NewClient(config)
   │
   └─> HTTP 客户端连接到节点

2. 创建钱包
   wallet.NewWalletFromPrivateKey(privateKeyHex)
   │
   └─> 从私钥创建钱包实例

3. 创建 Token 服务
   token.NewService(client)
   │
   └─> 创建业务服务实例

4. 准备转账参数
   TransferRequest{
       From:   wallet.Address(),
       To:     toAddr,
       Amount: 1000,
       TokenID: nil, // 原生币
   }

5. 执行转账
   tokenService.Transfer(ctx, req, wallet)
   │
   ├─> SDK 层构建交易
   ├─> Wallet 签名
   └─> 提交交易

6. 处理结果
   result.TxHash
```

---

## 🔧 扩展示例

### 批量转账示例

```go
// 创建 Token 服务
tokenService := token.NewService(client)

// 批量转账（所有转账必须使用同一个 tokenID）
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: wallet.Address(),
    Transfers: []token.TransferItem{
        {To: addr1, Amount: 100, TokenID: tokenID},
        {To: addr2, Amount: 200, TokenID: tokenID}, // 必须相同
    },
}, wallet)
```

### 代币铸造示例

```go
// 代币铸造
result, err := tokenService.Mint(ctx, &token.MintRequest{
    To:           recipientAddr,
    Amount:       10000,
    TokenID:      tokenID,
    ContractAddr: contractAddr,
}, wallet)
```

### 余额查询示例

```go
// 查询余额（不需要 Wallet）
balance, err := tokenService.GetBalance(ctx, address, tokenID)
fmt.Printf("余额: %d\n", balance)
```

---

## 📚 相关文档

- [主 README](../README.md) - SDK 总体文档
- [Services 文档](../services/README.md) - 业务服务文档
- [Wallet 文档](../wallet/README.md) - 钱包功能文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

