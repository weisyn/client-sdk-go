# Utils - 工具函数模块

Utils 模块提供 SDK 内部使用的工具函数，包括地址转换、编码解码等辅助功能。

## 🔧 核心功能

- **地址转换** - Base58Check 编码/解码、十六进制转换
- **交易解析** - 解析交易、查找输出、汇总金额

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/utils"

// 地址转换
base58Addr, err := utils.AddressBytesToBase58(addressBytes)
addressBytes, err := utils.AddressBase58ToBytes(base58Addr)
```

## 📚 完整文档

👉 **详细 API 参考请见：[`docs/modules/utils.md`](../docs/modules/utils.md)**

---

**最后更新**: 2025-11-17
