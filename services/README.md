# Services - 业务服务层

业务服务层提供面向业务场景的高层 API，将底层交易复杂性抽象为直观的业务操作。

## 📦 服务列表

- **Token** - 代币转账、批量转账、铸造、销毁、余额查询
- **Staking** - 质押、解质押、委托、取消委托、领取奖励
- **Market** - AMM 交换、流动性管理、归属计划、托管
- **Governance** - 提案、投票、参数更新
- **Resource** - 合约部署、AI 模型部署、资源查询

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/token"

tokenService := token.NewService(client)
result, err := tokenService.Transfer(ctx, &token.TransferRequest{
    From:   fromAddr,
    To:     toAddr,
    Amount: 1000,
}, wallet)
```

## 📚 完整文档

👉 **详细设计与能力说明请见：[`docs/modules/services.md`](../docs/modules/services.md)**

---

**最后更新**: 2025-11-17
