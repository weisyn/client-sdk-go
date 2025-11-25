// Package contract provides contract service implementation tests.
//
// ABI Helper 规范一致性测试
// 验证 client-sdk-go 的 ABI helper 实现是否符合 WES ABI 规范
// 规范来源：weisyn.git/docs/components/core/ispc/abi-and-payload.md

package contract

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/weisyn/client-sdk-go/utils"
)

// TestBuildPayload_ReservedFields 测试保留字段构建
func TestBuildPayload_ReservedFields(t *testing.T) {
	tests := []struct {
		name    string
		options utils.BuildPayloadOptions
		want    map[string]interface{}
	}{
		{
			name: "包含 from 字段",
			options: utils.BuildPayloadOptions{
				IncludeFrom: true,
				From:        make([]byte, 20),
			},
			want: map[string]interface{}{
				"from": "0x0000000000000000000000000000000000000000",
			},
		},
		{
			name: "包含所有保留字段",
			options: utils.BuildPayloadOptions{
				IncludeFrom:    true,
				From:           make([]byte, 20),
				IncludeTo:      true,
				To:             make([]byte, 20),
				IncludeAmount:  true,
				Amount:         1000000,
				IncludeTokenID: true,
				TokenID:        make([]byte, 32),
			},
			want: map[string]interface{}{
				"from":     "0x0000000000000000000000000000000000000000",
				"to":       "0x0000000000000000000000000000000000000000",
				"amount":   "1000000",
				"token_id": "0x0000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := utils.BuildPayload(tt.options)
			if err != nil {
				t.Fatalf("BuildPayload() error = %v", err)
			}

			// 验证字段存在
			for key, wantValue := range tt.want {
				if got, ok := payload[key]; !ok {
					t.Errorf("BuildPayload() missing field: %s", key)
				} else if got != wantValue {
					t.Errorf("BuildPayload() field %s = %v, want %v", key, got, wantValue)
				}
			}
		})
	}
}

// TestBuildPayload_ExtensionFields 测试扩展字段构建
func TestBuildPayload_ExtensionFields(t *testing.T) {
	tests := []struct {
		name    string
		options utils.BuildPayloadOptions
		want    map[string]interface{}
	}{
		{
			name: "包含扩展字段",
			options: utils.BuildPayloadOptions{
				MethodParams: map[string]interface{}{
					"to":     "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					"amount": "1000",
				},
			},
			want: map[string]interface{}{
				"to":     "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"amount": "1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := utils.BuildPayload(tt.options)
			if err != nil {
				t.Fatalf("BuildPayload() error = %v", err)
			}

			// 验证扩展字段存在
			for key, wantValue := range tt.want {
				if got, ok := payload[key]; !ok {
					t.Errorf("BuildPayload() missing extension field: %s", key)
				} else if got != wantValue {
					t.Errorf("BuildPayload() extension field %s = %v, want %v", key, got, wantValue)
				}
			}
		})
	}
}

// TestBuildPayload_FieldConflict 测试字段冲突检测
func TestBuildPayload_FieldConflict(t *testing.T) {
	tests := []struct {
		name    string
		options utils.BuildPayloadOptions
		wantErr bool
	}{
		{
			name: "扩展字段与保留字段冲突",
			options: utils.BuildPayloadOptions{
				MethodParams: map[string]interface{}{
					"from": "0x1234", // 与保留字段冲突
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := utils.BuildPayload(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBuildAndEncodePayload_Base64Encoding 测试 Base64 编码
func TestBuildAndEncodePayload_Base64Encoding(t *testing.T) {
	options := utils.BuildPayloadOptions{
		IncludeFrom: true,
		From:        make([]byte, 20),
		IncludeAmount: true,
		Amount:     1000000,
	}

	encoded, err := utils.BuildAndEncodePayload(options)
	if err != nil {
		t.Fatalf("BuildAndEncodePayload() error = %v", err)
	}

	// 验证 Base64 编码
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Base64 decode error = %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		t.Fatalf("JSON unmarshal error = %v", err)
	}

	// 验证 payload 内容
	if payload["from"] == nil {
		t.Error("payload missing 'from' field")
	}
	if payload["amount"] == nil {
		t.Error("payload missing 'amount' field")
	}
}

// TestBuildPayload_FieldNaming 测试字段命名规范
func TestBuildPayload_FieldNaming(t *testing.T) {
	options := utils.BuildPayloadOptions{
		IncludeTokenID: true,
		TokenID:        make([]byte, 32),
	}

	payload, err := utils.BuildPayload(options)
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}

	// 验证使用 token_id（下划线）而不是 tokenID（驼峰）
	if _, ok := payload["token_id"]; !ok {
		t.Error("payload missing 'token_id' field")
	}
	if _, ok := payload["tokenID"]; ok {
		t.Error("payload should not have 'tokenID' field (should use 'token_id')")
	}
}

// TestBuildPayload_ConformanceWithSpec 测试与规范示例的一致性
func TestBuildPayload_ConformanceWithSpec(t *testing.T) {
	// 规范示例：
	// {
	//   "from": "0x1234567890123456789012345678901234567890",
	//   "to": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
	//   "amount": "1000000",
	//   "token_id": "0x0000000000000000000000000000000000000000000000000000000000000000"
	// }

	fromHex := "1234567890123456789012345678901234567890"
	toHex := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	tokenIdHex := "0000000000000000000000000000000000000000000000000000000000000000"

	options := utils.BuildPayloadOptions{
		IncludeFrom:    true,
		From:           hexToBytes(fromHex),
		IncludeTo:      true,
		To:             hexToBytes(toHex),
		IncludeAmount:  true,
		Amount:         1000000,
		IncludeTokenID: true,
		TokenID:        hexToBytes(tokenIdHex),
	}

	payload, err := utils.BuildPayload(options)
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}

	// 验证与规范示例一致
	if got := payload["from"]; got != "0x"+fromHex {
		t.Errorf("payload['from'] = %v, want 0x%s", got, fromHex)
	}
	if got := payload["to"]; got != "0x"+toHex {
		t.Errorf("payload['to'] = %v, want 0x%s", got, toHex)
	}
	if got := payload["amount"]; got != "1000000" {
		t.Errorf("payload['amount'] = %v, want 1000000", got)
	}
	if got := payload["token_id"]; got != "0x"+tokenIdHex {
		t.Errorf("payload['token_id'] = %v, want 0x%s", got, tokenIdHex)
	}
}

// hexToBytes 将十六进制字符串转换为字节数组
func hexToBytes(hex string) []byte {
	bytes := make([]byte, len(hex)/2)
	for i := 0; i < len(bytes); i++ {
		bytes[i] = hexCharToByte(hex[i*2])<<4 | hexCharToByte(hex[i*2+1])
	}
	return bytes
}

// hexCharToByte 将十六进制字符转换为字节
func hexCharToByte(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 0
}

