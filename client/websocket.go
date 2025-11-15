package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// websocketClient WebSocket 客户端实现
type websocketClient struct {
	endpoint string
	conn     *websocket.Conn
	mu       sync.RWMutex
	closed   int32
	nextID   uint64
	requests map[uint64]chan *jsonrpcResponse
	muReq    sync.RWMutex
}

// jsonrpcRequest JSON-RPC 请求
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

// jsonrpcResponse JSON-RPC 响应
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	ID      uint64          `json:"id"`
}

// jsonrpcError JSON-RPC 错误
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	endpoint := config.Endpoint
	// 将 http:// 或 https:// 转换为 ws:// 或 wss://
	if len(endpoint) >= 7 && endpoint[:7] == "http://" {
		endpoint = "ws://" + endpoint[7:]
	} else if len(endpoint) >= 8 && endpoint[:8] == "https://" {
		endpoint = "wss://" + endpoint[8:]
	} else if len(endpoint) < 5 || (endpoint[:5] != "ws://" && endpoint[:5] != "wss://") {
		endpoint = "ws://" + endpoint
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	client := &websocketClient{
		endpoint: endpoint,
		conn:     conn,
		requests: make(map[uint64]chan *jsonrpcResponse),
	}

	// 启动消息读取循环
	go client.readLoop()

	return client, nil
}

// readLoop 消息读取循环
func (c *websocketClient) readLoop() {
	defer func() {
		atomic.StoreInt32(&c.closed, 1)
		c.muReq.Lock()
		for _, ch := range c.requests {
			close(ch)
		}
		c.muReq.Unlock()
	}()

	for {
		if atomic.LoadInt32(&c.closed) == 1 {
			return
		}

		var resp jsonrpcResponse
		if err := c.conn.ReadJSON(&resp); err != nil {
			// 连接关闭或错误
			c.muReq.Lock()
			for _, ch := range c.requests {
				select {
				case ch <- &jsonrpcResponse{
					Error: &jsonrpcError{
						Code:    -1,
						Message: fmt.Sprintf("websocket read error: %v", err),
					},
				}:
				default:
				}
				close(ch)
			}
			c.requests = make(map[uint64]chan *jsonrpcResponse)
			c.muReq.Unlock()
			return
		}

		// 查找对应的请求通道
		c.muReq.Lock()
		ch, exists := c.requests[resp.ID]
		if exists {
			delete(c.requests, resp.ID)
		}
		c.muReq.Unlock()

		if exists && ch != nil {
			select {
			case ch <- &resp:
			default:
			}
		}
	}
}

// Call 调用 JSON-RPC 方法
func (c *websocketClient) Call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, fmt.Errorf("websocket client is closed")
	}

	// 生成请求 ID
	reqID := atomic.AddUint64(&c.nextID, 1)

	// 创建请求
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      reqID,
	}

	// 创建响应通道
	respCh := make(chan *jsonrpcResponse, 1)
	c.muReq.Lock()
	c.requests[reqID] = respCh
	c.muReq.Unlock()

	// 发送请求
	c.mu.RLock()
	err := c.conn.WriteJSON(req)
	c.mu.RUnlock()
	if err != nil {
		c.muReq.Lock()
		delete(c.requests, reqID)
		c.muReq.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	// 等待响应
	select {
	case resp := <-respCh:
		if resp == nil {
			return nil, fmt.Errorf("response channel closed")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		// 解析结果
		var result interface{}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}
		return result, nil

	case <-ctx.Done():
		c.muReq.Lock()
		delete(c.requests, reqID)
		c.muReq.Unlock()
		return nil, ctx.Err()

	case <-time.After(30 * time.Second):
		c.muReq.Lock()
		delete(c.requests, reqID)
		c.muReq.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// SendRawTransaction 发送已签名的原始交易
func (c *websocketClient) SendRawTransaction(ctx context.Context, signedTxHex string) (*SendTxResult, error) {
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
func (c *websocketClient) Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error) {
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

	// TODO: 实现事件订阅处理逻辑
	// 当前简化实现，实际需要处理订阅消息
	go func() {
		<-ctx.Done()
		close(eventCh)
	}()

	return eventCh, nil
}

// Close 关闭连接
func (c *websocketClient) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}
	return nil
}

