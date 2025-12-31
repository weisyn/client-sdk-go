# Client - 核心客户端层

Client 模块是 SDK 的核心通信层，提供与 WES 节点交互的统一接口。支持 HTTP、gRPC、WebSocket 三种传输协议。

## 🚀 快速开始

```go
import "github.com/weisyn/client-sdk-go/client"

// HTTP 客户端（最常用）
cfg := &client.Config{
    Endpoint: "http://localhost:28680/jsonrpc",
    Protocol: client.ProtocolHTTP,
}
cli, err := client.NewClient(cfg)
```

## 📚 完整文档

👉 **详细设计与 API 参考请见：[`docs/modules/services.md`](../docs/modules/services.md)**（Client 层说明）

---

**最后更新**: 2025-11-17
