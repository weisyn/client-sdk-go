package types

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WesProblemDetails WES Problem Details 结构（基于 RFC7807 + WES 扩展）
// 与 JS WesProblemDetails 对齐
type WesProblemDetails struct {
	// RFC7807 标准字段
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   *int   `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`

	// WES 扩展字段（必填）
	Code        string                 `json:"code"`
	Layer       string                 `json:"layer"`
	UserMessage string                 `json:"userMessage"`
	Details     map[string]interface{} `json:"details,omitempty"`
	TraceID     string                 `json:"traceId"`
	Timestamp   string                 `json:"timestamp"`
}

// WesError WES 错误类型
// 与 JS WesError 对齐
type WesError struct {
	Code        string
	Layer       string
	UserMessage string
	Detail      string
	Status      *int
	Details     map[string]interface{}
	TraceID     string
	Timestamp   string
}

func (e *WesError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.UserMessage, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.UserMessage)
}

// ToProblemDetails 转换为 Problem Details
func (e *WesError) ToProblemDetails() *WesProblemDetails {
	return &WesProblemDetails{
		Code:        e.Code,
		Layer:       e.Layer,
		UserMessage: e.UserMessage,
		Detail:      e.Detail,
		Status:      e.Status,
		Details:     e.Details,
		TraceID:     e.TraceID,
		Timestamp:   e.Timestamp,
		Type:        "", // 可选字段
		Title:       "", // 可选字段
		Instance:    "", // 可选字段
	}
}

// NewWesErrorFromProblemDetails 从 Problem Details 创建 WesError
func NewWesErrorFromProblemDetails(pd *WesProblemDetails) *WesError {
	return &WesError{
		Code:        pd.Code,
		Layer:       pd.Layer,
		UserMessage: pd.UserMessage,
		Detail:      pd.Detail,
		Status:      pd.Status,
		Details:     pd.Details,
		TraceID:     pd.TraceID,
		Timestamp:   pd.Timestamp,
	}
}

// IsWesError 检查错误是否为 WesError
func IsWesError(err error) (*WesError, bool) {
	if wesErr, ok := err.(*WesError); ok {
		return wesErr, true
	}
	return nil, false
}

// Layer 常量
const (
	LayerClientSDKGo         = "client-sdk-go"
	LayerBlockchainService   = "blockchain-service"
	LayerContractCompiler    = "contract-compiler"
	LayerContractWorkbenchUI = "contract-workbench-ui"
)

// ErrorCode 错误码常量
const (
	// SDK 错误
	ErrorCodeSDKHTTPError                    = "SDK_HTTP_ERROR"
	ErrorCodeSDKGRPCError                    = "SDK_GRPC_ERROR"
	ErrorCodeSDKRequestSerializationError    = "SDK_REQUEST_SERIALIZATION_ERROR"
	ErrorCodeSDKResponseDeserializationError = "SDK_RESPONSE_DESERIALIZATION_ERROR"
	ErrorCodeSDKConnectionError              = "SDK_CONNECTION_ERROR"

	// 通用错误
	ErrorCodeCommonValidationError    = "COMMON_VALIDATION_ERROR"
	ErrorCodeCommonInternalError      = "COMMON_INTERNAL_ERROR"
	ErrorCodeCommonTimeout            = "COMMON_TIMEOUT"
	ErrorCodeCommonServiceUnavailable = "COMMON_SERVICE_UNAVAILABLE"
)

// ParseProblemDetailsFromRPCError 从 JSON-RPC 错误响应解析 Problem Details
// 与 JS parseProblemDetailsFromRPCError 对齐
func ParseProblemDetailsFromRPCError(rpcError interface{}) (*WesProblemDetails, error) {
	// 尝试转换为 map[string]interface{}
	rpcMap, ok := rpcError.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid RPC error format")
	}

	// 检查 data 字段是否包含 Problem Details
	data, ok := rpcMap["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no data field in RPC error")
	}

	// 检查是否包含 Problem Details 必填字段
	code, _ := data["code"].(string)
	layer, _ := data["layer"].(string)
	userMessage, _ := data["userMessage"].(string)
	traceID, _ := data["traceId"].(string)

	if code == "" || layer == "" || userMessage == "" || traceID == "" {
		return nil, fmt.Errorf("missing required fields in problem details")
	}

	// 提取可选字段
	detail, _ := data["detail"].(string)
	if detail == "" {
		if msg, ok := rpcMap["message"].(string); ok {
			detail = msg
		}
	}

	var status *int
	if statusVal, ok := data["status"].(float64); ok {
		s := int(statusVal)
		status = &s
	} else if statusVal, ok := rpcMap["code"].(float64); ok {
		s := int(statusVal)
		status = &s
	}

	details, _ := data["details"].(map[string]interface{})

	timestamp, _ := data["timestamp"].(string)
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	typeVal, _ := data["type"].(string)
	title, _ := data["title"].(string)
	instance, _ := data["instance"].(string)

	return &WesProblemDetails{
		Code:        code,
		Layer:       layer,
		UserMessage: userMessage,
		Detail:      detail,
		Status:      status,
		Details:     details,
		TraceID:     traceID,
		Timestamp:   timestamp,
		Type:        typeVal,
		Title:       title,
		Instance:    instance,
	}, nil
}

// CreateDefaultWesError 创建默认的 WesError（用于 fallback）
// 与 JS createDefaultWesError 对齐
func CreateDefaultWesError(
	code string,
	userMessage string,
	detail string,
	status int,
	details map[string]interface{},
) *WesError {
	if details == nil {
		details = make(map[string]interface{})
	}

	traceID := uuid.New().String()
	statusPtr := &status

	return &WesError{
		Code:        code,
		Layer:       LayerClientSDKGo,
		UserMessage: userMessage,
		Detail:      detail,
		Status:      statusPtr,
		Details:     details,
		TraceID:     traceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}
