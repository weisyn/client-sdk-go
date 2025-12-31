package client

import (
	"fmt"

	"github.com/weisyn/client-sdk-go/types"
)

// Error 客户端错误
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("client error [%d]: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("client error [%d]: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsWesError 检查错误是否为 WES Error
// 推荐使用：优先检查 types.WesError
func IsWesError(err error) (*types.WesError, bool) {
	return types.IsWesError(err)
}

// IsProblemDetails 检查错误是否为旧的 Problem Details 格式（已废弃）
// 保留此函数以保持向后兼容，但推荐使用 IsWesError
// Deprecated: 使用 types.IsWesError 代替
func IsProblemDetails(err error) (*ProblemDetails, bool) {
	if pd, ok := err.(*ProblemDetails); ok {
		return pd, true
	}
	return nil, false
}

// 错误码定义
const (
	ErrCodeNetwork         = 1000 // 网络错误
	ErrCodeTimeout         = 1001 // 超时错误
	ErrCodeInvalidResponse = 1002 // 无效响应
	ErrCodeRPCError        = 1003 // JSON-RPC错误
	ErrCodeNotSupported    = 1004 // 不支持的操作
)

// NewNetworkError 创建网络错误
func NewNetworkError(err error) *Error {
	return &Error{
		Code:    ErrCodeNetwork,
		Message: "network error",
		Err:     err,
	}
}

// NewTimeoutError 创建超时错误
func NewTimeoutError() *Error {
	return &Error{
		Code:    ErrCodeTimeout,
		Message: "request timeout",
	}
}

// NewInvalidResponseError 创建无效响应错误
func NewInvalidResponseError(message string) *Error {
	return &Error{
		Code:    ErrCodeInvalidResponse,
		Message: message,
	}
}

// NewRPCError 创建JSON-RPC错误
func NewRPCError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    ErrCodeRPCError,
		Message: fmt.Sprintf("RPC error [%d]: %s, data: %v", code, message, data),
	}
}

// NewNotSupportedError 创建不支持的操作错误
func NewNotSupportedError(operation string) *Error {
	return &Error{
		Code:    ErrCodeNotSupported,
		Message: fmt.Sprintf("operation not supported: %s", operation),
	}
}
