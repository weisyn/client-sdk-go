package utils

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
)

// TestParseOwnerAddress_EdgeCases 测试地址解析的边界情况
func TestParseOwnerAddress_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		ownerStr string
		wantLen int
	}{
		{
			name:    "Base64 with padding",
			ownerStr: base64.StdEncoding.EncodeToString(make([]byte, 20)),
			wantLen: 20,
		},
		{
			name:    "Base64 without padding",
			ownerStr: base64.RawStdEncoding.EncodeToString(make([]byte, 20)),
			wantLen: 20,
		},
		{
			name:    "hex with uppercase",
			ownerStr: "0x" + hex.EncodeToString(make([]byte, 20)),
			wantLen: 20,
		},
		{
			name:    "hex with mixed case",
			ownerStr: "0xAaBbCcDdEeFf1122334455667788990011223344",
			wantLen: 20,
		},
		{
			name:    "Base64 wrong length (19 bytes)",
			ownerStr: base64.StdEncoding.EncodeToString(make([]byte, 19)),
			wantLen: 0,
		},
		{
			name:    "Base64 wrong length (21 bytes)",
			ownerStr: base64.StdEncoding.EncodeToString(make([]byte, 21)),
			wantLen: 0,
		},
		{
			name:    "hex wrong length (19 bytes)",
			ownerStr: hex.EncodeToString(make([]byte, 19)),
			wantLen: 0,
		},
		{
			name:    "hex wrong length (21 bytes)",
			ownerStr: hex.EncodeToString(make([]byte, 21)),
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOwnerAddress(tt.ownerStr)
			if len(result) != tt.wantLen {
				t.Errorf("parseOwnerAddress() length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestFindOutputsByOwner_EdgeCases 测试按所有者查找输出的边界情况
func TestFindOutputsByOwner_EdgeCases(t *testing.T) {
	ownerAddr := make([]byte, 20)
	ownerAddr[0] = 0x01

	tests := []struct {
		name    string
		outputs []ParsedOutput
		owner   []byte
		wantLen int
	}{
		{
			name:    "empty outputs",
			outputs: []ParsedOutput{},
			owner:   ownerAddr,
			wantLen: 0,
		},
		{
			name: "outputs with nil owner",
			outputs: []ParsedOutput{
				{Index: 0, Owner: nil},
				{Index: 1, Owner: ownerAddr},
			},
			owner:   ownerAddr,
			wantLen: 1,
		},
		{
			name: "outputs with wrong length owner",
			outputs: []ParsedOutput{
				{Index: 0, Owner: make([]byte, 19)},
				{Index: 1, Owner: make([]byte, 21)},
				{Index: 2, Owner: ownerAddr},
			},
			owner:   ownerAddr,
			wantLen: 1,
		},
		{
			name: "owner with wrong length",
			outputs: []ParsedOutput{
				{Index: 0, Owner: ownerAddr},
			},
			owner:   make([]byte, 19),
			wantLen: 0,
		},
		{
			name: "all outputs match",
			outputs: []ParsedOutput{
				{Index: 0, Owner: ownerAddr},
				{Index: 1, Owner: ownerAddr},
				{Index: 2, Owner: ownerAddr},
			},
			owner:   ownerAddr,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindOutputsByOwner(tt.outputs, tt.owner)
			if len(result) != tt.wantLen {
				t.Errorf("FindOutputsByOwner() found %d outputs, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestSumAmountsByToken_EdgeCases 测试按代币汇总金额的边界情况
func TestSumAmountsByToken_EdgeCases(t *testing.T) {
	tokenID := make([]byte, 32)
	tokenID[0] = 0x01

	tests := []struct {
		name    string
		outputs []ParsedOutput
		tokenID []byte
		want    string
	}{
		{
			name:    "empty outputs",
			outputs: []ParsedOutput{},
			tokenID: nil,
			want:    "0",
		},
		{
			name: "outputs with nil amounts",
			outputs: []ParsedOutput{
				{Index: 0, Amount: nil},
				{Index: 1, Amount: bigIntFromString("1000000")},
			},
			tokenID: nil,
			want:    "1000000",
		},
		{
			name: "mixed native and token outputs",
			outputs: []ParsedOutput{
				{Index: 0, Amount: bigIntFromString("1000000"), TokenID: nil},
				{Index: 1, Amount: bigIntFromString("2000000"), TokenID: tokenID},
				{Index: 2, Amount: bigIntFromString("3000000"), TokenID: nil},
			},
			tokenID: nil,
			want:    "4000000",
		},
		{
			name: "very large amounts",
			outputs: []ParsedOutput{
				{Index: 0, Amount: bigIntFromString("999999999999999999999999")},
				{Index: 1, Amount: bigIntFromString("1")},
			},
			tokenID: nil,
			want:    "1000000000000000000000000",
		},
		{
			name: "zero amounts",
			outputs: []ParsedOutput{
				{Index: 0, Amount: bigIntFromString("0")},
				{Index: 1, Amount: bigIntFromString("0")},
			},
			tokenID: nil,
			want:    "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SumAmountsByToken(tt.outputs, tt.tokenID)
			if result.String() != tt.want {
				t.Errorf("SumAmountsByToken() = %s, want %s", result.String(), tt.want)
			}
		})
	}
}

// TestFindOutputsByType_EdgeCases 测试按类型查找输出的边界情况
func TestFindOutputsByType_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		outputs   []ParsedOutput
		outputType string
		wantLen   int
	}{
		{
			name:      "empty outputs",
			outputs:   []ParsedOutput{},
			outputType: "asset",
			wantLen:   0,
		},
		{
			name: "no matching type",
			outputs: []ParsedOutput{
				{Index: 0, Type: "asset"},
				{Index: 1, Type: "state"},
			},
			outputType: "resource",
			wantLen:   0,
		},
		{
			name: "all matching type",
			outputs: []ParsedOutput{
				{Index: 0, Type: "asset"},
				{Index: 1, Type: "asset"},
				{Index: 2, Type: "asset"},
			},
			outputType: "asset",
			wantLen:   3,
		},
		{
			name: "empty type string",
			outputs: []ParsedOutput{
				{Index: 0, Type: ""},
				{Index: 1, Type: "asset"},
			},
			outputType: "",
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindOutputsByType(tt.outputs, tt.outputType)
			if len(result) != tt.wantLen {
				t.Errorf("FindOutputsByType() found %d outputs, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestGetOutpoint_EdgeCases 测试生成 outpoint 的边界情况
func TestGetOutpoint_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		txHash string
		index  uint32
		want   string
	}{
		{
			name:   "hash with 0x prefix",
			txHash: "0x1234567890abcdef",
			index:  0,
			want:   "1234567890abcdef:0",
		},
		{
			name:   "hash without 0x prefix",
			txHash: "1234567890abcdef",
			index:  0,
			want:   "1234567890abcdef:0",
		},
		{
			name:   "large index",
			txHash: "0x1234567890abcdef",
			index:  4294967295, // max uint32
			want:   "1234567890abcdef:4294967295",
		},
		{
			name:   "empty hash",
			txHash: "",
			index:  0,
			want:   ":0",
		},
		{
			name:   "hash with uppercase",
			txHash: "0xABCDEF1234567890",
			index:  5,
			want:   "ABCDEF1234567890:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOutpoint(tt.txHash, tt.index)
			if result != tt.want {
				t.Errorf("GetOutpoint() = %s, want %s", result, tt.want)
			}
		})
	}
}

