module github.com/weisyn/client-sdk-go

go 1.24.0

// WES Client SDK for Go
//
// 这是一个独立的SDK模块，用于链下应用开发
// 使用 Go 1.24 以兼容最新特性
//
// 特点：
// 1. 封装节点 API 调用
// 2. 提供业务语义服务
// 3. 支持多协议（HTTP/gRPC/WebSocket）
// 4. 完整的交易构建与签名支持
//
// ⚠️ 重要：SDK 将来要移出 WES 项目独立发布
// - 禁止依赖任何 WES 包（pkg/interfaces, pkg/types, internal/*）
// - 只能通过 API（JSON-RPC/HTTP/gRPC/WebSocket）与节点交互
// - SDK 应该定义自己的类型和接口（参考节点 API，但不依赖）

// SDK 不依赖任何 WES 包，只依赖 Go 标准库和第三方通用库

require (
	github.com/btcsuite/btcutil v1.0.2
	github.com/gorilla/websocket v1.5.0
	google.golang.org/grpc v1.60.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.16.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
