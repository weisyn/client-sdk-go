# Market Service - 市场服务

Market Service 提供 AMM 交换、流动性管理、归属计划、托管等功能。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/market"

marketService := market.NewService(client)

// AMM 交换
result, err := marketService.SwapAMM(ctx, &market.SwapAMMRequest{
    ContractAddr: ammContractAddr,
    TokenIn:      tokenIn,
    AmountIn:     1000,
}, wallet)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#3-market-服务-)**

---

**最后更新**: 2025-11-17
