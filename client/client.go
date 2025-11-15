package client

import (
	"context"
	"fmt"
)

// Client WES 客户端接口
type Client interface {
	// Call 调用 JSON-RPC 方法
	Call(ctx context.Context, method string, params interface{}) (interface{}, error)
	
	// SendRawTransaction 发送已签名的原始交易
	SendRawTransaction(ctx context.Context, signedTxHex string) (*SendTxResult, error)
	
	// Subscribe 订阅事件
	Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error)
	
	// Close 关闭连接
	Close() error
}

// EventFilter 事件过滤器
type EventFilter struct {
	Topics []string
	From   []byte
	To     []byte
}

// Event 事件
type Event struct {
	Topic string
	Data  []byte
}

// SendTxResult 交易提交结果
type SendTxResult struct {
	TxHash   string `json:"tx_hash"`
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"` // 拒绝原因
}

// NewClient 创建新的客户端
func NewClient(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	switch config.Protocol {
	case ProtocolHTTP:
		return NewHTTPClient(config)
	case ProtocolGRPC:
		return NewGRPCClient(config)
	case ProtocolWebSocket:
		return NewWebSocketClient(config)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", config.Protocol)
	}
}

// NewHTTPClient 创建 HTTP 客户端（已在http.go中实现）
// 这里只是占位，实际实现在http.go中

// NewGRPCClient 创建 gRPC 客户端（实现在 grpc.go 中）
// NewWebSocketClient 创建 WebSocket 客户端（实现在 websocket.go 中）

