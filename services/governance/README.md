# Governance Service - 治理服务

Governance Service 提供提案、投票、参数更新等治理功能。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/governance"

governanceService := governance.NewService(client)

// 创建提案
result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
    Title:   "提案标题",
    Content: "提案内容",
}, wallet)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#4-governance-服务-)**

---

**最后更新**: 2025-11-17
