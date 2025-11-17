package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// httpClient HTTP客户端实现
type httpClient struct {
	endpoint string
	client   *http.Client
	logger   Logger
	debug    bool
	nextID   atomic.Uint64
	retry    *RetryConfig
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 创建HTTP客户端
	httpCli := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	// 配置TLS（如果需要）
	if config.TLS != nil && !config.TLS.Insecure {
		// TODO: 实现TLS配置
	}

	retryConfig := config.Retry
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
		// 如果配置了重试，添加日志回调
		if config.Debug && config.Logger != nil {
			retryConfig.OnRetry = func(attempt int, err error) {
				config.Logger.Warn("Retrying request", "attempt", attempt, "error", err)
			}
		}
	}

	return &httpClient{
		endpoint: config.Endpoint,
		client:   httpCli,
		logger:   config.Logger,
		debug:    config.Debug,
		nextID:   atomic.Uint64{},
		retry:    retryConfig,
	}, nil
}

// Call 调用JSON-RPC方法
func (c *httpClient) Call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	// 构建JSON-RPC请求
	// 使用原子计数器生成唯一ID
	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.nextID.Add(1),
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 调试日志
	if c.debug && c.logger != nil {
		c.logger.Debug("JSON-RPC request", "method", method, "body", string(reqBody))
	}

	// 发送请求（带重试）
	var resp *http.Response
	var respErr error
	
	if c.retry != nil {
		// 使用重试机制
		respErr = withRetry(ctx, func() error {
			// 每次重试都创建新的请求（因为 Body 只能读取一次）
			httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
			if reqErr != nil {
				return fmt.Errorf("create request failed: %w", reqErr)
			}

			// 设置请求头
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Accept", "application/json")

			// 发送请求
			httpResp, reqErr := c.client.Do(httpReq)
			if reqErr != nil {
				return reqErr
			}

			// 检查 HTTP 状态码
			if isRetryableHTTPError(httpResp.StatusCode) {
				httpResp.Body.Close()
				return fmt.Errorf("HTTP error: %d", httpResp.StatusCode)
			}

			// 成功，保存响应
			resp = httpResp
			return nil
		}, c.retry)
		if respErr != nil {
			return nil, fmt.Errorf("send request failed: %w", respErr)
		}
	} else {
		// 不使用重试
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		resp, err = c.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("send request failed: %w", err)
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			if c.logger != nil {
				c.logger.Warn("Failed to close response body", "error", err)
			}
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 调试日志
	if c.debug && c.logger != nil {
		c.logger.Debug("JSON-RPC response", "status", resp.StatusCode, "body", string(respBody))
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// 解析JSON-RPC响应
	var jsonResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	// 检查JSON-RPC错误
	if jsonResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: code=%d, message=%s, data=%v",
			jsonResp.Error.Code, jsonResp.Error.Message, jsonResp.Error.Data)
	}

	return jsonResp.Result, nil
}

// SendRawTransaction 发送已签名的原始交易
func (c *httpClient) SendRawTransaction(ctx context.Context, signedTxHex string) (*SendTxResult, error) {
	// 调用 wes_sendRawTransaction 方法
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
		// 如果结果是字符串，可能是交易哈希
		if txHash, ok := result.(string); ok {
			return &SendTxResult{
				TxHash:   txHash,
				Accepted: true,
			}, nil
		}
		return &SendTxResult{
			Accepted: false,
			Reason:   "invalid response format",
		}, nil
	}

	// 提取交易哈希和接受状态
	txHash, _ := resultMap["tx_hash"].(string)
	accepted, _ := resultMap["accepted"].(bool)
	reason, _ := resultMap["reason"].(string)

	return &SendTxResult{
		TxHash:   txHash,
		Accepted: accepted,
		Reason:   reason,
	}, nil
}

// Subscribe 订阅事件（HTTP不支持，需要使用WebSocket）
func (c *httpClient) Subscribe(ctx context.Context, filter *EventFilter) (<-chan *Event, error) {
	return nil, fmt.Errorf("HTTP client does not support event subscription, use WebSocket client instead")
}

// Close 关闭连接（HTTP客户端无需特殊处理）
func (c *httpClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// jsonRPCRequest JSON-RPC请求结构
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      uint64      `json:"id"`
}

// jsonRPCResponse JSON-RPC响应结构
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      uint64          `json:"id"`
}

// jsonRPCError JSON-RPC错误结构
type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

