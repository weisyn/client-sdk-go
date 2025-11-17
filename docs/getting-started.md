# 快速开始指南

本指南将帮助你快速上手 WES Client SDK for Go。

## 📦 安装

### 当前开发阶段

SDK 在主仓库 `_sdks/` 下孵化，使用本地路径：

```go
// go.mod
module your-app

go 1.24

replace github.com/weisyn/client-sdk-go => ../path/to/_sdks/client-sdk-go

require github.com/weisyn/client-sdk-go v0.0.0
```

### 未来正式发布后

```bash
go get github.com/weisyn/client-sdk-go@latest
```

## 🚀 第一个应用

### 1. 初始化客户端

```go
import "github.com/weisyn/client-sdk-go/client"

cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Timeout:  30,
}
cli, err := client.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer cli.Close()
```

### 2. 创建钱包

```go
import "github.com/weisyn/client-sdk-go/wallet"

// 创建新钱包
wallet, err := wallet.NewWallet()
if err != nil {
    log.Fatal(err)
}

// 或从私钥创建
wallet, err := wallet.NewWalletFromPrivateKey("0x...")
```

### 3. 执行转账

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/services/token"
)

tokenService := token.NewService(cli)

result, err := tokenService.Transfer(context.Background(), &token.TransferRequest{
    From:    wallet.Address(),
    To:      toAddr,
    Amount:  1000000, // 1 WES (假设 6 位小数)
    TokenID: nil,     // nil 表示原生币
}, wallet)

if err != nil {
    log.Fatalf("转账失败: %v", err)
}

fmt.Printf("转账成功！交易哈希: %s\n", result.TxHash)
```

## 📚 下一步

- [架构文档](architecture.md) - 了解 SDK 架构设计
- [业务服务文档](modules/services.md) - 学习各种业务服务
- [钱包文档](modules/wallet.md) - 深入了解钱包功能
- [API 参考](reference/api.md) - 查看完整 API 文档

---

**最后更新**: 2025-11-17

