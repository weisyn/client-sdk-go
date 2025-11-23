# Wallet - 钱包功能模块

Wallet 模块提供密钥管理、交易签名等核心功能。

## 🔑 核心功能

- **密钥管理** - 创建钱包、从私钥导入、Keystore 加密存储
- **交易签名** - 签名交易、签名消息、签名哈希
- **地址派生** - 从私钥派生地址

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/wallet"

// 创建新钱包
wallet, err := wallet.NewWallet()

// 从私钥创建
wallet, err := wallet.NewWalletFromPrivateKey("0x...")

// 签名交易
signedTx, err := wallet.SignHash(hashBytes)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/wallet.md`](../docs/modules/wallet.md)**

---

**最后更新**: 2025-11-17
