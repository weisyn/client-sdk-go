package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// grpcClient gRPC 客户端实现
type grpcClient struct {
	conn     *grpc.ClientConn
	endpoint string
}

// NewGRPCClient 创建 gRPC 客户端
func NewGRPCClient(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	endpoint := config.Endpoint
	// 如果 endpoint 包含 http:// 或 https://，移除协议前缀
	if len(endpoint) >= 7 && endpoint[:7] == "http://" {
		endpoint = endpoint[7:]
	} else if len(endpoint) >= 8 && endpoint[:8] == "https://" {
		endpoint = endpoint[8:]
	}

	// 设置超时
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建 gRPC 连接
	// 注意：当前使用 insecure 连接，生产环境应该使用 TLS
	conn, err := grpc.DialContext(ctx, endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial gRPC: %w", err)
	}

	client := &grpcClient{
		conn:     conn,
		endpoint: endpoint,
	}

	return client, nil
}

// Call 调用 JSON-RPC 方法（通过 gRPC）
//
// 注意：当前实现假设节点提供 gRPC 接口，如果节点只提供 JSON-RPC over HTTP，
// 则 gRPC 客户端需要通过 HTTP 适配器实现。
func (c *grpcClient) Call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	// TODO: 实现 gRPC 调用
	// 当前简化实现，返回错误提示
	// 实际实现需要：
	// 1. 定义 gRPC 服务接口（如果节点提供）
	// 2. 或者通过 HTTP 适配器调用 JSON-RPC API
	return nil, fmt.Errorf("gRPC client not fully implemented yet. " +
		"Please use HTTP or WebSocket client, or implement gRPC service interface")
}

// SendRawTransaction 发送已签名的原始交易
func (c *grpcClient) SendRawTransaction(ctx context.Context, signedTxHex string) (*SendTxResult, error) {
	// 通过 Call 方法调用
	result, err := c.Call(ctx, "wes_sendRawTransaction", []interface{}{signedTxHex})
	if err != nil {
		return &SendTxResult{
			Accepted: false,
			Reason:   err.Error(),
		}, nil
	}

	// 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return &SendTxResult{
			Accepted: false,
			Reason:   "invalid response format",
		}, nil
	}

	txHash, _ := resultMap["tx_hash"].(string)
	accepted, _ := resultMap["accepted"].(bool)
	reason, _ := resultMap["reason"].(string)

	return &SendTxResult{
		TxHash:   txHash,
		Accepted: accepted,
		Reason:   reason,
	}, nil
}

// Subscribe 订阅事件
func (c *grpcClient) Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error) {
	// 构建订阅参数
	params := map[string]interface{}{}
	if filter != nil {
		if len(filter.Topics) > 0 {
			params["topics"] = filter.Topics
		}
		if len(filter.From) > 0 {
			params["from"] = "0x" + hex.EncodeToString(filter.From)
		}
		if len(filter.To) > 0 {
			params["to"] = "0x" + hex.EncodeToString(filter.To)
		}
	}

	// 调用订阅方法
	result, err := c.Call(ctx, "wes_subscribe", []interface{}{params})
	if err != nil {
		return nil, fmt.Errorf("subscribe failed: %w", err)
	}

	// 解析订阅 ID
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid subscription response")
	}

	subscriptionID, _ := resultMap["subscription"].(string)
	if subscriptionID == "" {
		return nil, fmt.Errorf("missing subscription ID")
	}

	// 创建事件通道
	eventCh := make(chan *Event, 100)

	// TODO: 实现 gRPC 流式订阅
	// 当前简化实现
	go func() {
		<-ctx.Done()
		close(eventCh)
	}()

	return eventCh, nil
}

// Close 关闭连接
func (c *grpcClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

