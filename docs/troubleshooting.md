# 故障排查指南

---

## 📌 版本信息

- **版本**：0.1.0-alpha
- **状态**：draft
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK

---

## 📖 概述

本文档提供常见错误的排查方法和解决方案。

---

## 🔗 关联文档

- **WES 故障排查**：[WES 节点故障排查](https://github.com/weisyn/weisyn/blob/main/docs/troubleshooting/README.md)（待确认）
- **快速开始**：[快速开始指南](./getting-started.md)

---

## 🔌 连接问题

### 连接失败

**错误信息**：
```
NetworkError: Failed to connect to node
```

**可能原因**：
1. 节点未启动
2. 端点地址错误
3. 网络不可达

**解决方案**：
```go
// 1. 检查节点是否运行
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
}
c, err := client.NewClient(cfg)
if err != nil {
    log.Fatal("节点连接失败，请检查：")
    log.Fatal("1. 节点是否已启动？")
    log.Fatal("2. 端点地址是否正确？")
    log.Fatal("3. 防火墙是否阻止连接？")
}

// 2. 测试连接
ctx := context.Background()
_, err = c.Call(ctx, "wes_blockNumber", nil)
if err != nil {
    log.Fatal("节点连接失败:", err)
}
```

---

### 连接超时

**错误信息**：
```
NetworkError: Request timeout
```

**可能原因**：
1. 节点响应慢
2. 网络延迟高
3. 超时设置过短

**解决方案**：
```go
// 增加超时时间
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Timeout:  60, // 60 秒
}
c, err := client.NewClient(cfg)
```

---

## 💰 交易问题

### 余额不足

**错误信息**：
```
TransactionError: Insufficient balance
```

**可能原因**：
1. 账户余额不足
2. 未考虑交易手续费

**解决方案**：
```go
import "github.com/weisyn/client-sdk-go/services/token"

tokenService := token.NewService(client)

// 查询余额
balance, err := tokenService.GetBalance(ctx, wallet.Address(), nil)
if err != nil {
    log.Fatal(err)
}

// 检查余额是否足够（包括手续费）
requiredAmount := transferAmount + estimatedFee
if balance < requiredAmount {
    log.Fatal("余额不足")
}
```

---

### 交易失败

**错误信息**：
```
TransactionError: Transaction failed
```

**可能原因**：
1. 交易参数错误
2. 锁定条件未满足
3. 节点拒绝交易

**解决方案**：
```go
result, err := tokenService.Transfer(ctx, req, wallet)
if err != nil {
    // 检查错误类型
    if strings.Contains(err.Error(), "insufficient balance") {
        log.Fatal("余额不足")
    } else if strings.Contains(err.Error(), "invalid address") {
        log.Fatal("地址无效")
    } else {
        log.Fatal("交易失败:", err)
    }
}
```

---

## 🔐 密钥问题

### 私钥格式错误

**错误信息**：
```
WalletError: Invalid private key format
```

**解决方案**：
```go
// 确保私钥是 32 字节
privateKeyHex := "0x1234..."
privateKeyBytes, err := hex.DecodeString(strings.TrimPrefix(privateKeyHex, "0x"))
if err != nil {
    log.Fatal("私钥格式错误")
}

if len(privateKeyBytes) != 32 {
    log.Fatal("私钥长度必须为 32 字节")
}

wallet, err := wallet.FromPrivateKey(privateKeyBytes)
```

---

## 📝 常见错误码

| 错误类型 | 说明 | 解决方案 |
|---------|------|---------|
| `NetworkError` | 网络连接错误 | 检查节点是否运行、网络是否可达 |
| `TransactionError` | 交易错误 | 检查余额、参数、锁定条件 |
| `WalletError` | 钱包错误 | 检查私钥格式、地址格式 |
| `ValidationError` | 参数验证错误 | 检查请求参数 |

---

## 🔗 相关文档

- **[快速开始](./getting-started.md)** - 安装和配置
- **[API 参考](./api/client.md)** - 完整 API 文档

---

