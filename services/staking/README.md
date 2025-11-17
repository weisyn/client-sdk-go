# Staking Service - 质押服务

Staking Service 提供质押、解质押、委托、取消委托、领取奖励等功能。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/staking"

stakingService := staking.NewService(client)

// 质押
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:     stakerAddr,
    Amount:   10000,
    Validator: validatorAddr,
}, wallet)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#2-staking-服务-)**

---

**最后更新**: 2025-11-17
