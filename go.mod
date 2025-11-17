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
	github.com/ethereum/go-ethereum v1.15.11
	github.com/gorilla/websocket v1.5.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.40.0
	google.golang.org/grpc v1.76.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
