package client

import (
	"context"
	"fmt"
	"net"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int
	// InitialDelay 初始延迟（毫秒）
	InitialDelay int
	// MaxDelay 最大延迟（毫秒）
	MaxDelay int
	// BackoffMultiplier 退避倍数
	BackoffMultiplier float64
	// Retryable 判断错误是否可重试的函数
	Retryable func(error) bool
	// OnRetry 重试前的回调函数
	OnRetry func(attempt int, err error)
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialDelay:      1000,
		MaxDelay:          10000,
		BackoffMultiplier: 2.0,
		Retryable:         isRetryableError,
		OnRetry:           nil,
	}
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 网络错误（连接失败、超时等）
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
	}

	// DNS 错误
	if _, ok := err.(*net.DNSError); ok {
		return true
	}

	// HTTP 错误（通过错误消息判断）
	errMsg := err.Error()
	if containsAny(errMsg, []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"timeout",
		"ECONNREFUSED",
		"ENOTFOUND",
	}) {
		return true
	}

	return false
}

// isRetryableHTTPError 判断 HTTP 响应错误是否可重试
func isRetryableHTTPError(statusCode int) bool {
	// HTTP 5xx 错误（服务器错误）
	if statusCode >= 500 && statusCode < 600 {
		return true
	}
	// HTTP 429 错误（请求过多）
	if statusCode == 429 {
		return true
	}
	return false
}

// containsAny 检查字符串是否包含任意一个子串
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// calculateBackoffDelay 计算退避延迟
func calculateBackoffDelay(attempt int, config *RetryConfig) time.Duration {
	delay := float64(config.InitialDelay) * pow(config.BackoffMultiplier, float64(attempt))
	maxDelay := float64(config.MaxDelay)
	if delay > maxDelay {
		delay = maxDelay
	}
	return time.Duration(delay) * time.Millisecond
}

// pow 计算 x 的 y 次方
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

// withRetry 带重试的函数执行器
func withRetry(ctx context.Context, fn func() error, config *RetryConfig) error {
	if config == nil {
		return fn()
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，直接返回错误
		if attempt >= config.MaxRetries {
			break
		}

		// 判断是否可重试
		retryable := config.Retryable
		if retryable == nil {
			retryable = isRetryableError
		}
		if !retryable(err) {
			return err
		}

		// 计算延迟时间
		delay := calculateBackoffDelay(attempt, config)

		// 调用重试回调
		if config.OnRetry != nil {
			config.OnRetry(attempt+1, err)
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	// 所有重试都失败，返回最后一个错误
	return fmt.Errorf("retry failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// withRetryHTTP 已废弃，使用 withRetry 直接处理 HTTP 请求
// 保留此函数以避免编译错误，但实际不再使用
