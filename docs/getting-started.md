# 快速开始指南

本指南将帮助你快速上手 WES Client SDK for Go。

## 📦 安装

### 安装方式

SDK 已独立发布，直接使用 Go 模块：

```bash
go get github.com/weisyn/client-sdk-go@latest
```

或使用 `go.mod`：

```go
// go.mod
module your-app

go 1.24

require github.com/weisyn/client-sdk-go v0.0.0
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

### 3. 使用业务服务

SDK 提供5个核心业务服务：

#### Token 服务 - 代币操作

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/services/token"
)

tokenService := token.NewService(cli)

// 转账
result, err := tokenService.Transfer(context.Background(), &token.TransferRequest{
    From:    wallet.Address(),
    To:      toAddr,
    Amount:  1000000, // 1 WES (假设 6 位小数)
    TokenID: nil,     // nil 表示原生币
}, wallet)

// 查询余额
balance, err := tokenService.GetBalance(context.Background(), wallet.Address(), nil)
```

#### Staking 服务 - 质押与委托

```go
import "github.com/weisyn/client-sdk-go/services/staking"

stakingService := staking.NewService(cli)

// 质押
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:     wallet.Address(),
    Amount:   10000,
    Validator: validatorAddr,
}, wallet)
```

#### Market 服务 - 市场与流动性

```go
import "github.com/weisyn/client-sdk-go/services/market"

marketService := market.NewService(cli)

// AMM 交换
result, err := marketService.SwapAMM(ctx, &market.SwapAMMRequest{
    ContractAddr: ammContractAddr,
    TokenIn:      tokenIn,
    AmountIn:     1000,
}, wallet)
```

#### Governance 服务 - 治理

```go
import "github.com/weisyn/client-sdk-go/services/governance"

governanceService := governance.NewService(cli)

// 创建提案
result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
    Title:   "提案标题",
    Content: "提案内容",
}, wallet)
```

#### Resource 服务 - 资源部署

```go
import "github.com/weisyn/client-sdk-go/services/resource"

resourceService := resource.NewService(cli)

// 部署合约
result, err := resourceService.DeployContract(ctx, &resource.DeployContractRequest{
    WasmBytes: wasmBytes,
    Name:      "My Contract",
}, wallet)
```

## 📚 下一步

- [架构文档](architecture.md) - 了解 SDK 架构设计
- [业务服务文档](modules/services.md) - 学习各种业务服务
- [钱包文档](modules/wallet.md) - 深入了解钱包功能
- [API 参考](reference/api.md) - 查看完整 API 文档

---

**最后更新**: 2025-11-17

