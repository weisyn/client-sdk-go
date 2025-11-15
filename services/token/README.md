# Token Service - 代币服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 核心功能已完成  
**最后更新**: 2025-01-23

---

## 📋 概述

Token Service 提供完整的代币操作功能，包括转账、批量转账、代币铸造、代币销毁和余额查询。所有操作都遵循 SDK 架构原则，业务语义在 SDK 层实现，通过通用 API 与节点交互。

---

## 🏗️ 服务架构

### 模块结构

```
┌─────────────────────────────────────────────────────────┐
│              Token Service 模块结构                      │
└─────────────────────────────────────────────────────────┘

services/token/
  │
  ├─> service.go          # Service 接口和实现
  ├─> transfer.go         # Transfer 和 BatchTransfer 实现
  ├─> mint.go             # Mint 实现
  ├─> balance.go          # GetBalance 实现
  └─> tx_builder.go       # 交易构建逻辑
      │
      ├─> buildTransferTransaction()      ✅
      ├─> buildBatchTransferTransaction() ✅
      └─> buildBurnTransaction()          ✅
```

### 服务调用流程

```
┌─────────────────────────────────────────────────────────┐
│         Token Service 调用流程                           │
└─────────────────────────────────────────────────────────┘

应用层
  │
  ├─> tokenService.Transfer()
  ├─> tokenService.BatchTransfer()
  ├─> tokenService.Mint()
  ├─> tokenService.Burn()
  └─> tokenService.GetBalance()
      │
      ↓
Token Service (services/token/)
  │
  ├─> 1. 参数验证
  ├─> 2. Wallet 验证
  ├─> 3. 业务逻辑
  │   │
  │   ├─> Transfer: SDK 层构建交易
  │   │   └─> buildTransferTransaction()
  │   │
  │   ├─> BatchTransfer: SDK 层构建交易
  │   │   └─> buildBatchTransferTransaction()
  │   │
  │   ├─> Mint: 调用合约
  │   │   └─> wes_callContract(return_unsigned_tx=true)
  │   │
  │   └─> Burn: SDK 层构建交易
  │       └─> buildBurnTransaction()
  │
  ├─> 4. Wallet 签名
  └─> 5. 提交交易 (wes_sendRawTransaction)
```

---

## 🔧 核心功能

### 1. Transfer - 单笔转账 ✅

**功能**: 单笔代币转账（支持原生币和合约代币）

**实现方式**: SDK 层构建交易

**流程**:
```
1. 查询 UTXO (wes_getUTXO)
   │
   ├─> 过滤匹配 tokenID 的 UTXO
   └─> 选择足够的 UTXO
   
2. 计算手续费和找零
   │
   ├─> 手续费 = 金额 × 0.03%
   └─> 找零 = UTXO金额 - 转账金额 - 手续费
   
3. 构建交易草稿 (DraftJSON)
   │
   ├─> inputs: [选中的 UTXO]
   ├─> outputs: [转账输出, 找零输出(如果有)]
   └─> sign_mode: "defer_sign"
   
4. 调用 wes_buildTransaction
   │
   └─> 获取未签名交易
   
5. Wallet 签名
   │
   └─> wallet.SignTransaction()
   
6. 提交交易
   │
   └─> wes_sendRawTransaction()
```

**使用示例**:
```go
tokenService := token.NewService(client)

result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   fromAddr,
    To:     toAddr,
    Amount: 1000000, // 1 WES (假设 6 位小数)
    TokenID: nil,    // nil = 原生币
}, wallet)
```

---

### 2. BatchTransfer - 批量转账 ✅

**功能**: 批量代币转账

**重要限制**: ⚠️ **所有转账必须使用同一个 tokenID**

**实现方式**: SDK 层构建交易

**流程**:
```
1. 验证所有转账使用同一个 tokenID
   │
   └─> 如果不同，返回错误

2. 查询 UTXO (wes_getUTXO)
   │
   ├─> 过滤匹配 tokenID 的 UTXO
   └─> 为每个转账选择 UTXO

3. 累计总输入和总输出
   │
   ├─> totalInputAmount
   └─> totalOutputAmount

4. 计算手续费和找零
   │
   ├─> 手续费 = 总输出 × 0.03%
   └─> 找零 = 总输入 - 总输出 - 手续费

5. 构建交易草稿 (DraftJSON)
   │
   ├─> inputs: [所有选中的 UTXO]
   ├─> outputs: [所有转账输出, 找零输出(如果有)]
   └─> sign_mode: "defer_sign"

6. 调用 wes_buildTransaction
7. Wallet 签名
8. 提交交易
```

**使用示例**:
```go
// ✅ 正确：所有转账使用同一个 tokenID
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: fromAddr,
    Transfers: []token.TransferItem{
        {To: addr1, Amount: 100, TokenID: tokenID},
        {To: addr2, Amount: 200, TokenID: tokenID}, // 相同
    },
}, wallet)

// ❌ 错误：不同 tokenID
result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
    From: fromAddr,
    Transfers: []token.TransferItem{
        {To: addr1, Amount: 100, TokenID: tokenID1},
        {To: addr2, Amount: 200, TokenID: tokenID2}, // 不同！
    },
}, wallet)
// 返回错误: "all transfers must use the same tokenID"
```

---

### 3. Mint - 代币铸造 ✅

**功能**: 通过智能合约铸造代币

**实现方式**: 调用合约方法

**流程**:
```
1. 构建合约调用参数
   │
   ├─> method: "mint"
   ├─> params: [to, amount, tokenID]
   └─> contractAddr: 代币合约地址

2. 调用 wes_callContract
   │
   ├─> return_unsigned_tx: true
   └─> 获取未签名交易

3. Wallet 签名
4. 提交交易
```

**使用示例**:
```go
result, err := tokenService.Mint(ctx, &token.MintRequest{
    To:           recipientAddr,
    Amount:       10000,
    TokenID:      tokenID,
    ContractAddr: contractAddr,
}, wallet)
```

---

### 4. Burn - 代币销毁 ✅

**功能**: 销毁代币（通过消费 UTXO 但不创建输出）

**实现方式**: SDK 层构建交易

**流程**:
```
1. 查询 UTXO (wes_getUTXO)
   │
   ├─> 过滤匹配 tokenID 的 UTXO
   └─> 选择足够的 UTXO

2. 计算手续费和找零
   │
   ├─> 手续费 = 销毁金额 × 0.03%
   └─> 找零 = UTXO金额 - 销毁金额 - 手续费

3. 构建交易草稿 (DraftJSON)
   │
   ├─> inputs: [选中的 UTXO]
   ├─> outputs: [找零输出(如果有)]
   └─> sign_mode: "defer_sign"

4. 调用 wes_buildTransaction
5. Wallet 签名
6. 提交交易
```

**使用示例**:
```go
result, err := tokenService.Burn(ctx, &token.BurnRequest{
    From:   fromAddr,
    Amount: 500,
    TokenID: tokenID,
}, wallet)
```

---

### 5. GetBalance - 余额查询 ✅

**功能**: 查询地址的代币余额

**实现方式**: 直接调用节点 API

**使用示例**:
```go
// 查询原生币余额
balance, err := tokenService.GetBalance(ctx, address, nil)

// 查询合约代币余额
balance, err := tokenService.GetBalance(ctx, address, tokenID)
```

---

## 🎯 关键特性

### 1. TokenID 处理

```
┌─────────────────────────────────────────┐
│        TokenID 处理规则                  │
└─────────────────────────────────────────┘

原生币:
  TokenID = nil 或 []
  └─> 匹配没有 tokenID 的 UTXO

合约代币:
  TokenID = [32字节]
  └─> 匹配相同 tokenID 的 UTXO
```

### 2. UTXO 选择策略

```
┌─────────────────────────────────────────┐
│        UTXO 选择策略                     │
└─────────────────────────────────────────┘

1. 查询所有 UTXO (wes_getUTXO)
2. 按 tokenID 过滤
3. 选择第一个足够的 UTXO
   └─> UTXO金额 >= 所需金额
```

### 3. 手续费计算

```
┌─────────────────────────────────────────┐
│        手续费计算规则                    │
└─────────────────────────────────────────┘

手续费率: 0.03% (万分之三)

计算方式:
  手续费 = 金额 × 3 / 10000

示例:
  转账 1000000 → 手续费 = 300
  转账 10000   → 手续费 = 3
```

### 4. 找零处理

```
┌─────────────────────────────────────────┐
│        找零处理规则                      │
└─────────────────────────────────────────┘

计算:
  找零 = UTXO金额 - 转账金额 - 手续费

规则:
  - 如果找零 > 0，创建找零输出
  - 找零使用相同的 tokenID
  - 找零地址 = 发送方地址
```

---

## 📊 交易构建详解

### DraftJSON 格式

```json
{
  "sign_mode": "defer_sign",
  "inputs": [
    {
      "tx_hash": "0x...",
      "output_index": 0,
      "is_reference_only": false
    }
  ],
  "outputs": [
    {
      "type": "asset",
      "owner": "0x...",
      "amount": "1000",
      "token_id": "0x..." // 可选
    }
  ],
  "metadata": {
    "caller_address": "0x..."
  }
}
```

### 交易构建函数

#### buildTransferTransaction

**功能**: 构建单笔转账交易

**参数**:
- `ctx`: 上下文
- `client`: 客户端
- `fromAddress`: 发送方地址
- `toAddress`: 接收方地址
- `amount`: 转账金额
- `tokenID`: 代币ID（可选）

**返回**: 未签名交易（字节数组）

#### buildBatchTransferTransaction

**功能**: 构建批量转账交易

**参数**:
- `ctx`: 上下文
- `client`: 客户端
- `fromAddress`: 发送方地址
- `transfers`: 转账列表（**必须使用同一个 tokenID**）

**返回**: 未签名交易（字节数组）

#### buildBurnTransaction

**功能**: 构建销毁交易

**参数**:
- `ctx`: 上下文
- `client`: 客户端
- `fromAddress`: 发送方地址
- `amount`: 销毁金额
- `tokenID`: 代币ID

**返回**: 未签名交易（字节数组）

---

## 🔒 安全考虑

### 1. 地址验证

```go
// 自动验证 Wallet 地址与请求地址匹配
if !bytes.Equal(w.Address(), req.From) {
    return nil, fmt.Errorf("wallet address does not match from address")
}
```

### 2. 余额检查

```go
// 选择 UTXO 时检查余额
if utxoAmount.Cmp(requiredAmount) < 0 {
    return nil, fmt.Errorf("insufficient balance")
}
```

### 3. TokenID 验证

```go
// 批量转账验证所有转账使用同一个 tokenID
if currentTokenIDHex != commonTokenIDHex {
    return nil, fmt.Errorf("all transfers must use the same tokenID")
}
```

---

## 📚 API 参考

### Service 接口

```go
type Service interface {
    // Transfer 单笔转账
    Transfer(ctx context.Context, req *TransferRequest, wallets ...wallet.Wallet) (*TransferResult, error)
    
    // BatchTransfer 批量转账（所有转账必须使用同一个 tokenID）
    BatchTransfer(ctx context.Context, req *BatchTransferRequest, wallets ...wallet.Wallet) (*BatchTransferResult, error)
    
    // Mint 代币铸造
    Mint(ctx context.Context, req *MintRequest, wallets ...wallet.Wallet) (*MintResult, error)
    
    // Burn 代币销毁
    Burn(ctx context.Context, req *BurnRequest, wallets ...wallet.Wallet) (*BurnResult, error)
    
    // GetBalance 查询余额（不需要 Wallet）
    GetBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error)
}
```

### 请求结构

```go
// TransferRequest 转账请求
type TransferRequest struct {
    From    []byte // 发送方地址（20字节）
    To      []byte // 接收方地址（20字节）
    Amount  uint64 // 转账金额
    TokenID []byte // 代币ID（32字节，nil 表示原生币）
}

// BatchTransferRequest 批量转账请求
type BatchTransferRequest {
    From      []byte         // 发送方地址（20字节）
    Transfers []TransferItem // 转账列表（必须使用同一个 tokenID）
}

// TransferItem 转账项
type TransferItem struct {
    To      []byte // 接收方地址（20字节）
    Amount  uint64 // 转账金额
    TokenID []byte // 代币ID（32字节，必须相同）
}
```

---

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档
- [Wallet 文档](../../wallet/README.md) - 钱包功能文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

