# Wallet API 参考

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

`Wallet` 提供密钥管理、交易签名、地址派生等功能。它支持从私钥导入、Keystore 加密存储等场景。

---

## 🔗 关联文档

- **架构说明**：[SDK 架构设计](../architecture.md)
- **安全指南**：[最佳实践](../reference/security.md)（待创建）

---

## 📦 导入

```go
import "github.com/weisyn/client-sdk-go/wallet"
```

---

## 🏗️ Wallet 接口

### Wallet Interface

```go
type Wallet interface {
    // Address 返回钱包地址（20 字节）
    Address() []byte
    
    // SignTransaction 签名交易
    SignTransaction(tx []byte) ([]byte, error)
    
    // SignMessage 签名消息
    SignMessage(msg []byte) ([]byte, error)
    
    // SignHash 签名哈希值
    SignHash(hash []byte) ([]byte, error)
    
    // PrivateKey 返回私钥（谨慎使用）
    PrivateKey() *ecdsa.PrivateKey
}
```

---

## 🚀 使用示例

### 创建新钱包

```go
import "github.com/weisyn/client-sdk-go/wallet"

// 创建新钱包（生成随机私钥）
w, err := wallet.NewWallet()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("地址: %x\n", w.Address())
```

### 从私钥导入

```go
// 从十六进制私钥导入
privateKeyHex := "0x1234..."
w, err := wallet.NewWalletFromPrivateKey(privateKeyHex)
if err != nil {
    log.Fatal(err)
}

// 或使用不带 0x 前缀的格式
w, err := wallet.NewWalletFromPrivateKey("1234...")
```

### 签名交易

```go
// 1. 获取未签名交易（通过 Client）
unsignedTx, err := client.Call(ctx, "wes_buildTransaction", []interface{}{draft})

// 2. Wallet 签名
unsignedTxBytes := []byte(unsignedTx.(string)) // 假设返回的是 hex 字符串
signature, err := w.SignTransaction(unsignedTxBytes)
if err != nil {
    log.Fatal(err)
}

// 3. 提交交易
signedTxHex := hex.EncodeToString(signature)
result, err := client.SendRawTransaction(ctx, signedTxHex)
```

### 签名消息

```go
message := []byte("Hello, WES!")
signature, err := w.SignMessage(message)
if err != nil {
    log.Fatal(err)
}

// 签名可用于身份验证等场景
```

---

## 🔐 Keystore 加密存储

### 导出到 Keystore

```go
import "github.com/weisyn/client-sdk-go/wallet"

// 导出钱包到 Keystore（加密存储）
keystoreData, err := wallet.EncryptKeystore(w, "password123")
if err != nil {
    log.Fatal(err)
}

// 保存到文件
keystoreJSON, _ := json.Marshal(keystoreData)
err = os.WriteFile("keystore.json", keystoreJSON, 0600)
if err != nil {
    log.Fatal(err)
}
```

### 从 Keystore 导入

```go
// 从文件加载
keystoreJSON, err := os.ReadFile("keystore.json")
if err != nil {
    log.Fatal(err)
}

var keystoreData wallet.KeystoreData
json.Unmarshal(keystoreJSON, &keystoreData)

// 解密并导入钱包
w, err := wallet.DecryptKeystore(&keystoreData, "password123")
if err != nil {
    log.Fatal(err)
}
```

---

## 🔑 地址操作

### 获取地址

```go
// 获取 20 字节地址
addressBytes := w.Address() // []byte (20 bytes)

// 转换为 Base58 格式
import "github.com/weisyn/client-sdk-go/utils"
addressBase58, err := utils.AddressBytesToBase58(addressBytes)

// 转换为十六进制格式
addressHex := hex.EncodeToString(addressBytes) // "0x..."
```

### 地址验证

```go
import "github.com/weisyn/client-sdk-go/utils"

addressBytes, err := utils.AddressBase58ToBytes(addressBase58)
if err != nil {
    log.Fatal("地址无效:", err)
}
```

---

## 🔒 安全考虑

### 私钥安全

```go
// ✅ 推荐：使用 Keystore 加密存储
keystoreData, err := wallet.EncryptKeystore(w, strongPassword)
saveToSecureStorage(keystoreData)

// ❌ 不推荐：明文存储私钥
privateKey := w.PrivateKey() // 仅用于调试
// 不要将私钥保存到文件或发送到服务器
```

### 密码管理

```go
// ✅ 推荐：使用强密码
password := generateStrongPassword() // 至少 12 位，包含大小写字母、数字、特殊字符

// ✅ 推荐：使用密码管理器
// 让用户使用密码管理器生成和存储密码
```

---

## 📚 方法参考

### NewWallet()

创建新钱包（生成随机私钥）。

```go
func NewWallet() (Wallet, error)
```

**返回**：`(Wallet, error)` - 新创建的钱包

**示例**：
```go
w, err := wallet.NewWallet()
```

---

### NewWalletFromPrivateKey()

从私钥创建钱包。

```go
func NewWalletFromPrivateKey(privateKeyHex string) (Wallet, error)
```

**参数**：
- `privateKeyHex`: 私钥（十六进制字符串，可带或不带 `0x` 前缀）

**返回**：`(Wallet, error)` - 钱包实例

**示例**：
```go
w, err := wallet.NewWalletFromPrivateKey("0x1234...")
```

---

### SignTransaction()

签名交易。

```go
func (w *SimpleWallet) SignTransaction(tx []byte) ([]byte, error)
```

**参数**：
- `tx`: 未签名交易（`[]byte`）

**返回**：`([]byte, error)` - 签名（64 字节）

**流程**：
1. 计算交易哈希（SHA-256）
2. 使用 ECDSA 签名哈希
3. 返回紧凑格式签名（r || s）

---

### SignMessage()

签名消息。

```go
func (w *SimpleWallet) SignMessage(msg []byte) ([]byte, error)
```

**参数**：
- `msg`: 消息（`[]byte`）

**返回**：`([]byte, error)` - 签名（64 字节）

**用途**：身份验证、消息认证等

---

### SignHash()

签名哈希值（同步方法）。

```go
func (w *SimpleWallet) SignHash(hash []byte) ([]byte, error)
```

**参数**：
- `hash`: 哈希值（32 字节）

**返回**：`([]byte, error)` - 签名（64 字节）

**注意**：这是同步方法，适用于已计算好哈希的场景

---

## 🔗 相关文档

- **[Client API](./client.md)** - 客户端接口
- **[Services API](./services.md)** - 业务服务
- **[故障排查](../troubleshooting.md)** - 常见问题

---

**最后更新**: 2025-11-17

