# 测试指南

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

本文档说明 SDK 的测试结构、如何运行测试，以及与 WES 节点测试的关系。

---

## 🔗 关联文档

- **WES 测试策略**：[WES 测试文档](https://github.com/weisyn/weisyn/blob/main/docs/testing/README.md)（待确认）
- **快速开始**：[快速开始指南](./getting-started.md)

---

## 🏗️ 测试结构

### 目录结构

```
test/
├── unit/              # 单元测试
│   ├── client/       # Client 测试
│   ├── wallet/       # Wallet 测试
│   ├── services/     # Services 测试
│   └── utils/        # Utils 测试
└── integration/      # 集成测试
    ├── setup.go      # 集成测试工具函数
    ├── helpers.go    # 测试辅助函数
    └── services/     # 各服务测试
        ├── token/
        ├── staking/
        ├── market/
        ├── governance/
        └── resource/
```

---

## 🧪 单元测试

### 运行单元测试

```bash
# 运行所有单元测试
go test ./...

# 运行特定包的单元测试
go test ./utils/... -v

# 运行特定测试函数
go test -run TestFunctionName ./utils/...

# 显示覆盖率
go test -cover ./...
```

### 单元测试示例

```go
// utils/address_test.go
package utils

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAddressConversion(t *testing.T) {
    addr := make([]byte, 20)
    addr[0] = 0x01
    
    base58 := AddressToBase58(addr)
    assert.NotEmpty(t, base58)
    
    decoded, err := AddressFromBase58(base58)
    assert.NoError(t, err)
    assert.Equal(t, addr, decoded)
}
```

### 测试覆盖范围

- ✅ **Client**：连接、重试、错误处理
- ✅ **Wallet**：密钥生成、签名、Keystore
- ✅ **Services**：业务逻辑、参数验证
- ✅ **Utils**：地址转换、批量操作、文件处理

---

## 🔗 集成测试

### 运行集成测试

```bash
# 运行所有集成测试（需要本地节点运行）
go test ./test/integration/... -v

# 运行特定服务的集成测试
go test ./test/integration/services/token/... -v

# 设置环境变量指定节点端点
WES_NODE_ENDPOINT=http://localhost:8545 go test ./test/integration/... -v
```

### 集成测试设置

```go
// test/integration/setup.go
package integration

import (
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/wallet"
)

func SetupTestClient(t *testing.T) client.Client {
    endpoint := os.Getenv("WES_NODE_ENDPOINT")
    if endpoint == "" {
        endpoint = "http://localhost:8545"
    }
    
    cfg := &client.Config{
        Endpoint: endpoint,
        Protocol: client.ProtocolHTTP,
    }
    
    c, err := client.NewClient(cfg)
    require.NoError(t, err)
    return c
}

func CreateTestWallet(t *testing.T) wallet.Wallet {
    w, err := wallet.NewWallet()
    require.NoError(t, err)
    return w
}
```

### 集成测试示例

```go
// test/integration/services/token/transfer_test.go
package token

import (
    "context"
    "testing"
    "github.com/weisyn/client-sdk-go/services/token"
    "github.com/weisyn/client-sdk-go/test/integration"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/assert"
)

func TestTokenTransfer_Integration(t *testing.T) {
    // 1. 设置测试客户端
    c := integration.SetupTestClient(t)
    defer c.Close()
    
    // 2. 创建测试账户
    wallet := integration.CreateTestWallet(t)
    
    // 3. 为账户充值（如需要）
    integration.FundTestAccount(t, c, wallet.Address(), 1000000)
    
    // 4. 创建服务实例
    tokenService := token.NewService(c)
    
    // 5. 执行转账
    ctx := context.Background()
    result, err := tokenService.Transfer(ctx, &token.TransferRequest{
        From:   wallet.Address(),
        To:     recipientAddr,
        Amount: 1000,
    }, wallet)
    
    // 6. 验证结果
    require.NoError(t, err)
    assert.NotEmpty(t, result.TxHash)
    assert.True(t, result.Success)
}
```

---

## 🎯 测试最佳实践

### 1. 测试命名

- 使用 `Test` 前缀
- 使用下划线分隔测试对象和方法
- 示例：`TestTokenTransfer_Basic`、`TestStaking_Stake`

### 2. 测试结构

- 使用 `require` 进行必须通过的断言
- 使用 `assert` 进行可继续的断言
- 对于依赖问题，使用 `t.Skip()` 跳过测试

### 3. 测试数据

- 使用有意义的测试数据
- 避免硬编码，使用常量或配置
- 确保测试数据不会相互干扰

---

## 🔗 相关文档

- **[快速开始](./getting-started.md)** - 安装和配置
- **[故障排查](./troubleshooting.md)** - 常见问题

---

