# Resource Service - 资源服务

Resource Service 提供合约部署、AI 模型部署、静态资源部署和资源查询功能。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/services/resource"

resourceService := resource.NewService(client)

// 部署合约
result, err := resourceService.DeployContract(ctx, &resource.DeployContractRequest{
    WasmBytes: wasmBytes,
    Name:      "My Contract",
}, wallet)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../../docs/modules/services.md#5-resource-服务-)**

---

**最后更新**: 2025-11-17
