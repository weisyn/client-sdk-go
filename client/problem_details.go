package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ProblemDetails WES Problem Details 结构（基于 RFC7807 + WES 扩展）
// Deprecated: 使用 types.WesProblemDetails 代替
type ProblemDetails struct {
	// RFC7807 标准字段
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"`
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

// Error 实现 error 接口
func (p *ProblemDetails) Error() string {
	if p.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", p.Code, p.Title, p.Detail)
	}
	return fmt.Sprintf("[%s] %s", p.Code, p.UserMessage)
}

// ParseProblemDetails 从 HTTP 响应中解析 Problem Details
// Deprecated: 使用 types.ParseProblemDetailsFromRPCError 代替
// 注意：此函数已不再被使用，HTTP 客户端现在直接使用 types.WesProblemDetails
func ParseProblemDetails(resp *http.Response) (*ProblemDetails, error) {
	contentType := resp.Header.Get("Content-Type")
	
	// 检查是否是 Problem Details 格式
	if contentType != "application/problem+json" && 
	   contentType != "application/json" {
		return nil, fmt.Errorf("not a problem details response")
	}

	// 读取响应体
	var problem ProblemDetails
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&problem); err != nil {
		return nil, fmt.Errorf("failed to decode problem details: %w", err)
	}

	// 验证必填字段
	if problem.Code == "" || problem.Layer == "" || 
	   problem.UserMessage == "" || problem.TraceID == "" {
		return nil, fmt.Errorf("invalid problem details: missing required fields")
	}

	return &problem, nil
}

// NewProblemDetails 创建新的 Problem Details
// Deprecated: 使用 types.CreateDefaultWesError 代替
func NewProblemDetails(
	code string,
	layer string,
	userMessage string,
	detail string,
	status int,
	details map[string]interface{},
) *ProblemDetails {
	traceID := uuid.New().String()
	if details == nil {
		details = make(map[string]interface{})
	}

	return &ProblemDetails{
		Code:        code,
		Layer:       layer,
		UserMessage: userMessage,
		Detail:      detail,
		Status:      status,
		Details:     details,
		TraceID:     traceID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// 错误码常量
const (
	// SDK 错误
	CodeSDKHTTPError                    = "SDK_HTTP_ERROR"
	CodeSDKGRPCError                    = "SDK_GRPC_ERROR"
	CodeSDKRPCError                     = "SDK_GRPC_ERROR" // JSON-RPC 错误也使用 GRPC_ERROR 码
	CodeSDKRequestSerializationError   = "SDK_REQUEST_SERIALIZATION_ERROR"
	CodeSDKResponseDeserializationError = "SDK_RESPONSE_DESERIALIZATION_ERROR"
	CodeSDKConnectionError              = "SDK_CONNECTION_ERROR"

	// 通用错误
	CodeCommonValidationError = "COMMON_VALIDATION_ERROR"
	CodeCommonInternalError   = "COMMON_INTERNAL_ERROR"
	CodeCommonTimeout         = "COMMON_TIMEOUT"
)

// Layer 常量
const (
	LayerClientSDKGo = "client-sdk-go"
)

