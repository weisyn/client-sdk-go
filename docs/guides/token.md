# Token 服务指南

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

Token Service 提供代币操作功能，包括转账、批量转账、铸造、销毁和余额查询。

---

## 🔗 关联文档

- **API 参考**：[Services API - Token](../api/services.md#-token-service)
- **WES 协议**：[WES 系统架构](https://github.com/weisyn/weisyn/blob/main/docs/system/architecture/README.md)

---

## 🚀 快速开始

### 创建服务

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/wallet"
)

cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
}
cli, err := client.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}

w, err := wallet.NewWallet()
if err != nil {
    log.Fatal(err)
}

tokenService := token.NewService(cli)
```

---

## 💸 转账

### 单笔转账

```go
ctx := context.Background()

result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   w.Address(),
    To:     recipientAddr,
    Amount: 1000000, // 1 WES（假设 6 位小数）
    TokenID: nil,    // nil 表示原生币
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("转账成功！交易哈希: %s\n", result.TxHash)
```

### 代币转账

```go
// 创建代币 ID（32 字节）
tokenID := make([]byte, 32)
for i := range tokenID {
    tokenID[i] = 1 // 示例：使用全 1 作为代币 ID
}

result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:    w.Address(),
    To:      recipientAddr,
    Amount:  1000,
    TokenID: tokenID, // 指定代币 ID
}, w)
if err != nil {
    log.Fatal(err)
}
```

### 转账流程说明

SDK 内部流程：
1. **查询 UTXO**：调用 `wes_getUTXO` 查询发送方的可用 UTXO
2. **选择 UTXO**：自动选择足够的 UTXO 覆盖转账金额
3. **构建交易**：调用 `wes_buildTransaction` 构建交易草稿
4. **签名交易**：使用 Wallet 签名
5. **提交交易**：调用 `wes_sendRawTransaction` 提交交易

---

## 📦 批量转账

### 基本使用

```go
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: w.Address(),
    Transfers: []token.TransferItem{
        {To: recipient1Addr, Amount: 100000},
        {To: recipient2Addr, Amount: 200000},
        {To: recipient3Addr, Amount: 300000},
    },
    TokenID: tokenID, // 所有转账必须使用同一个 tokenID
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("批量转账成功！交易哈希: %s\n", result.TxHash)
```

### 注意事项

- ⚠️ **所有转账必须使用同一个 `tokenID`**
- ✅ 批量转账在一个交易中完成，节省 Gas 费
- ✅ 如果任何一笔转账失败，整个交易会回滚

---

## 🪙 代币铸造

### 前提条件

- 需要代币合约已部署
- 需要合约地址和代币 ID

### 铸造代币

```go
result, err := tokenService.Mint(ctx, &token.MintRequest{
    To:          recipientAddr,
    Amount:      10000,
    TokenID:     tokenID,
    ContractAddr: contractAddr, // 代币合约地址
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("铸造成功！交易哈希: %s\n", result.TxHash)
```

### 实现原理

SDK 内部调用 `wes_callContract`，调用代币合约的 `mint` 方法：

```go
// SDK 内部实现（简化）
_, err := client.CallContract(ctx, &client.CallContractRequest{
    ContractAddr: contractAddr,
    Method:       "mint",
    Payload:      payload, // Base64 编码的 JSON
    Options: &client.CallContractOptions{
        ReturnUnsignedTx: true,
    },
})
```

---

## 🔥 代币销毁

### 销毁代币

```go
result, err := tokenService.Burn(ctx, &token.BurnRequest{
    From:        w.Address(),
    Amount:      5000,
    TokenID:     tokenID,
    ContractAddr: contractAddr, // 代币合约地址
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("销毁成功！交易哈希: %s\n", result.TxHash)
```

---

## 💰 查询余额

### 查询原生币余额

```go
balance, err := tokenService.GetBalance(ctx, w.Address(), nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("余额: %d wei\n", balance)
```

### 查询代币余额

```go
tokenBalance, err := tokenService.GetBalance(ctx, w.Address(), tokenID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("代币余额: %d\n", tokenBalance)
```

### 实现原理

SDK 内部：
1. 调用 `wes_getUTXO` 查询地址的所有 UTXO
2. 过滤匹配 `tokenID` 的 UTXO
3. 汇总 UTXO 的金额

---

## 🎯 典型场景

### 场景 1：用户支付

```go
func payForService(
    ctx context.Context,
    userWallet wallet.Wallet,
    serviceProvider []byte,
    amount uint64,
    tokenService token.Service,
) (string, error) {
    result, err := tokenService.Transfer(ctx, &token.TransferRequest{
        From:    userWallet.Address(),
        To:      serviceProvider,
        Amount:  amount,
        TokenID: nil, // 使用原生币
    }, userWallet)
    if err != nil {
        return "", err
    }
    
    return result.TxHash, nil
}
```

### 场景 2：批量发放奖励

```go
type Recipient struct {
    Address []byte
    Amount  uint64
}

func distributeRewards(
    ctx context.Context,
    fromWallet wallet.Wallet,
    recipients []Recipient,
    tokenID []byte,
    tokenService token.Service,
) (string, error) {
    transfers := make([]token.TransferItem, len(recipients))
    for i, r := range recipients {
        transfers[i] = token.TransferItem{
            To:     r.Address,
            Amount: r.Amount,
        }
    }
    
    result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
        From:     fromWallet.Address(),
        Transfers: transfers,
        TokenID:   tokenID,
    }, fromWallet)
    if err != nil {
        return "", err
    }
    
    return result.TxHash, nil
}
```

### 场景 3：检查余额是否足够

```go
func checkBalance(
    ctx context.Context,
    address []byte,
    requiredAmount uint64,
    tokenID []byte,
    tokenService token.Service,
) (bool, error) {
    balance, err := tokenService.GetBalance(ctx, address, tokenID)
    if err != nil {
        return false, err
    }
    
    return balance >= requiredAmount, nil
}
```

---

## ⚠️ 常见错误

### 余额不足

```go
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:    w.Address(),
    To:      recipientAddr,
    Amount:  1000000000, // 非常大的金额
    TokenID: nil,
}, w)
if err != nil {
    if strings.Contains(err.Error(), "insufficient balance") {
        log.Fatal("余额不足")
    }
    log.Fatal(err)
}
```

### 无效地址

```go
invalidAddr := make([]byte, 19) // 错误长度

result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:    w.Address(),
    To:      invalidAddr,
    Amount:  1000,
    TokenID: nil,
}, w)
if err != nil {
    log.Printf("地址无效: %v\n", err)
}
```

---

## 🔗 相关文档

- **[API 参考](../api/services.md#-token-service)** - 完整 API 文档
- **[快速开始](../getting-started.md)** - 安装和配置
- **[故障排查](../troubleshooting.md)** - 常见问题

---

