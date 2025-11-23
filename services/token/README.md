# Token Service - 代币服务

Token Service 提供完整的代币操作功能：转账、批量转账、代币铸造、代币销毁和余额查询。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/token"

tokenService := token.NewService(client)

// 单笔转账
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   fromAddr,
    To:     toAddr,
    Amount: 1000,
}, wallet)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#1-token-服务-)**

---

**最后更新**: 2025-11-17
