package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/weisyn/client-sdk-go/types"
)

func TestHTTPClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		statusCode     int
		contentType    string
		wantWesError   bool
		checkErrorFunc func(*testing.T, error)
	}{
		{
			name: "valid problem details in JSON-RPC error",
			responseBody: `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32000,
					"message": "Internal error",
					"data": {
						"code": "BC_TX_NOT_FOUND",
						"layer": "blockchain-service",
						"userMessage": "交易不存在",
						"detail": "Transaction not found",
						"traceId": "trace-123",
						"timestamp": "2025-11-23T10:00:00Z",
						"status": 404
					}
				},
				"id": 1
			}`,
			statusCode:   200,
			contentType:  "application/json",
			wantWesError: true,
			checkErrorFunc: func(t *testing.T, err error) {
				wesErr, ok := types.IsWesError(err)
				if !ok {
					t.Error("expected WesError, got regular error")
					return
				}
				if wesErr.Code != "BC_TX_NOT_FOUND" {
					t.Errorf("expected code BC_TX_NOT_FOUND, got %s", wesErr.Code)
				}
				if wesErr.UserMessage != "交易不存在" {
					t.Errorf("expected userMessage 交易不存在, got %s", wesErr.UserMessage)
				}
			},
		},
		{
			name: "missing problem details in JSON-RPC error",
			responseBody: `{
				"jsonrpc": "2.0",
				"error": {
					"code": -32000,
					"message": "Internal error"
				},
				"id": 1
			}`,
			statusCode:   200,
			contentType:  "application/json",
			wantWesError: false,
			checkErrorFunc: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				// Should return a clear error message about missing Problem Details
				if err.Error() == "" {
					t.Error("expected error message")
				}
			},
		},
		{
			name: "HTTP error with problem details",
			responseBody: `{
				"code": "BC_TX_NOT_FOUND",
				"layer": "blockchain-service",
				"userMessage": "交易不存在",
				"detail": "Transaction not found",
				"traceId": "trace-123",
				"timestamp": "2025-11-23T10:00:00Z",
				"status": 404
			}`,
			statusCode:   404,
			contentType:  "application/problem+json",
			wantWesError: true,
			checkErrorFunc: func(t *testing.T, err error) {
				wesErr, ok := types.IsWesError(err)
				if !ok {
					t.Error("expected WesError, got regular error")
					return
				}
				if wesErr.Code != "BC_TX_NOT_FOUND" {
					t.Errorf("expected code BC_TX_NOT_FOUND, got %s", wesErr.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create HTTP client
			cfg := &Config{
				Endpoint: server.URL,
				Protocol: ProtocolHTTP,
			}
			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer client.Close()

			// Make a call
			ctx := context.Background()
			_, err = client.Call(ctx, "test_method", []interface{}{})

			// Check error
			if err == nil && tt.wantWesError {
				t.Error("expected error, got nil")
				return
			}
			if err != nil && tt.checkErrorFunc != nil {
				tt.checkErrorFunc(t, err)
			}
		})
	}
}

func TestParseProblemDetailsFromRPCError(t *testing.T) {
	// Test the helper function used in HTTP client
	rpcError := map[string]interface{}{
		"code":    -32000,
		"message": "Internal error",
		"data": map[string]interface{}{
			"code":        "BC_TX_NOT_FOUND",
			"layer":       "blockchain-service",
			"userMessage": "交易不存在",
			"traceId":     "trace-123",
			"timestamp":   "2025-11-23T10:00:00Z",
		},
	}

	// Convert to JSON-RPC error format
	jsonRPCError := &jsonrpcError{
		Code:    -32000,
		Message: "Internal error",
		Data:    rpcError["data"],
	}

	// Convert to map for parsing
	rpcErrorMap := map[string]interface{}{
		"code":    jsonRPCError.Code,
		"message": jsonRPCError.Message,
		"data":    jsonRPCError.Data,
	}

	pd, err := types.ParseProblemDetailsFromRPCError(rpcErrorMap)
	if err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}

	if pd.Code != "BC_TX_NOT_FOUND" {
		t.Errorf("expected code BC_TX_NOT_FOUND, got %s", pd.Code)
	}
	if pd.Layer != "blockchain-service" {
		t.Errorf("expected layer blockchain-service, got %s", pd.Layer)
	}
}
