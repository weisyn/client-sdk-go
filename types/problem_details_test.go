package types

import (
	"testing"
	"time"
)

func TestParseProblemDetailsFromRPCError(t *testing.T) {
	tests := []struct {
		name      string
		rpcError  interface{}
		wantError bool
		checkFunc func(*testing.T, *WesProblemDetails, error)
	}{
		{
			name: "valid problem details",
			rpcError: map[string]interface{}{
				"code":    -32000,
				"message": "Internal error",
				"data": map[string]interface{}{
					"code":        "BC_TX_NOT_FOUND",
					"layer":       "blockchain-service",
					"userMessage":  "交易不存在",
					"detail":       "Transaction with hash 0x1234 not found",
					"traceId":      "trace-123",
					"timestamp":    "2025-01-23T10:00:00Z",
					"status":       404.0,
					"details": map[string]interface{}{
						"txHash": "0x1234",
					},
				},
			},
			wantError: false,
			checkFunc: func(t *testing.T, pd *WesProblemDetails, err error) {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if pd == nil {
					t.Error("expected problem details, got nil")
					return
				}
				if pd.Code != "BC_TX_NOT_FOUND" {
					t.Errorf("expected code BC_TX_NOT_FOUND, got %s", pd.Code)
				}
				if pd.Layer != "blockchain-service" {
					t.Errorf("expected layer blockchain-service, got %s", pd.Layer)
				}
				if pd.UserMessage != "交易不存在" {
					t.Errorf("expected userMessage 交易不存在, got %s", pd.UserMessage)
				}
				if pd.TraceID != "trace-123" {
					t.Errorf("expected traceId trace-123, got %s", pd.TraceID)
				}
				if pd.Status == nil || *pd.Status != 404 {
					t.Errorf("expected status 404, got %v", pd.Status)
				}
			},
		},
		{
			name: "missing required fields",
			rpcError: map[string]interface{}{
				"code":    -32000,
				"message": "Internal error",
				"data": map[string]interface{}{
					"code": "BC_TX_NOT_FOUND",
					// missing layer, userMessage, traceId
				},
			},
			wantError: true,
			checkFunc: func(t *testing.T, pd *WesProblemDetails, err error) {
				if err == nil {
					t.Error("expected error for missing required fields")
				}
			},
		},
		{
			name:      "invalid RPC error format",
			rpcError:  "not a map",
			wantError: true,
			checkFunc: func(t *testing.T, pd *WesProblemDetails, err error) {
				if err == nil {
					t.Error("expected error for invalid format")
				}
			},
		},
		{
			name: "no data field",
			rpcError: map[string]interface{}{
				"code":    -32000,
				"message": "Internal error",
				// no data field
			},
			wantError: true,
			checkFunc: func(t *testing.T, pd *WesProblemDetails, err error) {
				if err == nil {
					t.Error("expected error for missing data field")
				}
			},
		},
		{
			name: "data field is not a map",
			rpcError: map[string]interface{}{
				"code":    -32000,
				"message": "Internal error",
				"data":    "not a map",
			},
			wantError: true,
			checkFunc: func(t *testing.T, pd *WesProblemDetails, err error) {
				if err == nil {
					t.Error("expected error for invalid data field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd, err := ParseProblemDetailsFromRPCError(tt.rpcError)
			tt.checkFunc(t, pd, err)
		})
	}
}

func TestNewWesErrorFromProblemDetails(t *testing.T) {
	status := 404
	pd := &WesProblemDetails{
		Code:        "BC_TX_NOT_FOUND",
		Layer:       "blockchain-service",
		UserMessage: "交易不存在",
		Detail:      "Transaction not found",
		Status:      &status,
		TraceID:     "trace-123",
		Timestamp:   "2025-01-23T10:00:00Z",
		Details: map[string]interface{}{
			"txHash": "0x1234",
		},
	}

	wesErr := NewWesErrorFromProblemDetails(pd)

	if wesErr.Code != pd.Code {
		t.Errorf("expected code %s, got %s", pd.Code, wesErr.Code)
	}
	if wesErr.Layer != pd.Layer {
		t.Errorf("expected layer %s, got %s", pd.Layer, wesErr.Layer)
	}
	if wesErr.UserMessage != pd.UserMessage {
		t.Errorf("expected userMessage %s, got %s", pd.UserMessage, wesErr.UserMessage)
	}
	if wesErr.Detail != pd.Detail {
		t.Errorf("expected detail %s, got %s", pd.Detail, wesErr.Detail)
	}
	if wesErr.TraceID != pd.TraceID {
		t.Errorf("expected traceId %s, got %s", pd.TraceID, wesErr.TraceID)
	}
	if wesErr.Status == nil || *wesErr.Status != status {
		t.Errorf("expected status %d, got %v", status, wesErr.Status)
	}
}

func TestWesError_Error(t *testing.T) {
	tests := []struct {
		name     string
		wesError *WesError
		want     string
	}{
		{
			name: "with detail",
			wesError: &WesError{
				Code:        "BC_TX_NOT_FOUND",
				UserMessage: "交易不存在",
				Detail:      "Transaction not found",
			},
			want: "[BC_TX_NOT_FOUND] 交易不存在: Transaction not found",
		},
		{
			name: "without detail",
			wesError: &WesError{
				Code:        "BC_TX_NOT_FOUND",
				UserMessage: "交易不存在",
			},
			want: "[BC_TX_NOT_FOUND] 交易不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.wesError.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWesError_ToProblemDetails(t *testing.T) {
	status := 404
	wesErr := &WesError{
		Code:        "BC_TX_NOT_FOUND",
		Layer:       "blockchain-service",
		UserMessage: "交易不存在",
		Detail:      "Transaction not found",
		Status:      &status,
		TraceID:     "trace-123",
		Timestamp:   "2025-01-23T10:00:00Z",
		Details: map[string]interface{}{
			"txHash": "0x1234",
		},
	}

	pd := wesErr.ToProblemDetails()

	if pd.Code != wesErr.Code {
		t.Errorf("expected code %s, got %s", wesErr.Code, pd.Code)
	}
	if pd.Layer != wesErr.Layer {
		t.Errorf("expected layer %s, got %s", wesErr.Layer, pd.Layer)
	}
	if pd.UserMessage != wesErr.UserMessage {
		t.Errorf("expected userMessage %s, got %s", wesErr.UserMessage, pd.UserMessage)
	}
	if pd.Detail != wesErr.Detail {
		t.Errorf("expected detail %s, got %s", wesErr.Detail, pd.Detail)
	}
	if pd.TraceID != wesErr.TraceID {
		t.Errorf("expected traceId %s, got %s", wesErr.TraceID, pd.TraceID)
	}
	if pd.Status == nil || *pd.Status != status {
		t.Errorf("expected status %d, got %v", status, pd.Status)
	}
}

func TestIsWesError(t *testing.T) {
	wesErr := &WesError{
		Code:        "BC_TX_NOT_FOUND",
		UserMessage: "交易不存在",
	}

	// Test with WesError
	gotErr, ok := IsWesError(wesErr)
	if !ok {
		t.Error("expected IsWesError to return true for WesError")
	}
	if gotErr != wesErr {
		t.Error("expected IsWesError to return the same error")
	}

	// Test with regular error
	regularErr := &testError{msg: "regular error"}
	_, ok = IsWesError(regularErr)
	if ok {
		t.Error("expected IsWesError to return false for regular error")
	}
}

func TestCreateDefaultWesError(t *testing.T) {
	details := map[string]interface{}{
		"txHash": "0x1234",
	}

	wesErr := CreateDefaultWesError(
		ErrorCodeSDKHTTPError,
		"网络请求失败",
		"Connection timeout",
		500,
		details,
	)

	if wesErr.Code != ErrorCodeSDKHTTPError {
		t.Errorf("expected code %s, got %s", ErrorCodeSDKHTTPError, wesErr.Code)
	}
	if wesErr.Layer != LayerClientSDKGo {
		t.Errorf("expected layer %s, got %s", LayerClientSDKGo, wesErr.Layer)
	}
	if wesErr.UserMessage != "网络请求失败" {
		t.Errorf("expected userMessage 网络请求失败, got %s", wesErr.UserMessage)
	}
	if wesErr.Detail != "Connection timeout" {
		t.Errorf("expected detail Connection timeout, got %s", wesErr.Detail)
	}
	if wesErr.Status == nil || *wesErr.Status != 500 {
		t.Errorf("expected status 500, got %v", wesErr.Status)
	}
	if wesErr.TraceID == "" {
		t.Error("expected traceId to be generated")
	}
	if wesErr.Timestamp == "" {
		t.Error("expected timestamp to be generated")
	}
	// Verify timestamp format
	_, err := time.Parse(time.RFC3339, wesErr.Timestamp)
	if err != nil {
		t.Errorf("invalid timestamp format: %v", err)
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

