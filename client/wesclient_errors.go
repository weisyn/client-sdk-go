package client

import (
	"fmt"
)

// WESClientErrorCode WESClient 错误码
type WESClientErrorCode string

const (
	WESErrCodeNetwork        WESClientErrorCode = "NETWORK_ERROR"
	WESErrCodeRPC            WESClientErrorCode = "RPC_ERROR"
	WESErrCodeInvalidParams  WESClientErrorCode = "INVALID_PARAMS"
	WESErrCodeNotImplemented WESClientErrorCode = "RPC_NOT_IMPLEMENTED"
	WESErrCodeNotFound       WESClientErrorCode = "NOT_FOUND"
	WESErrCodeDecodeFailed   WESClientErrorCode = "DECODE_FAILED"
)

// WESClientError WESClient 统一错误类型
type WESClientError struct {
	Code    WESClientErrorCode
	Message string
	Cause   error
}

func (e *WESClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause=%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *WESClientError) Unwrap() error {
	return e.Cause
}

// wrapRPCError 包装 RPC 错误为 WESClientError
func wrapRPCError(method string, err error) error {
	if err == nil {
		return nil
	}

	// 网络错误
	if netErr, ok := err.(*Error); ok && netErr.Code == ErrCodeNetwork {
		return &WESClientError{
			Code:    WESErrCodeNetwork,
			Message: fmt.Sprintf("network error calling %s", method),
			Cause:   err,
		}
	}

	// JSON-RPC 错误（根据错误码进一步分类）
	if rpcErr, ok := err.(*Error); ok && rpcErr.Code == ErrCodeRPCError {
		// 尝试从 RPC 错误中提取 JSON-RPC 错误码
		// 这里需要根据实际的 JSONRPCError 结构调整
		// 假设 Error 结构中有额外的字段来存储 JSON-RPC 错误码
		return &WESClientError{
			Code:    WESErrCodeRPC,
			Message: fmt.Sprintf("RPC error calling %s: %s", method, rpcErr.Message),
			Cause:   err,
		}
	}

	// 检查是否是 WESClientError（避免重复包装）
	if wesErr, ok := err.(*WESClientError); ok {
		return wesErr
	}

	// 其他错误保持原状
	return err
}
