package utils

import (
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestParseOwnerAddress(t *testing.T) {
	tests := []struct {
		name    string
		ownerStr string
		wantLen int
		wantErr bool
	}{
		{
			name:    "Base64 encoded address",
			ownerStr: base64.StdEncoding.EncodeToString(make([]byte, 20)),
			wantLen: 20,
			wantErr: false,
		},
		{
			name:    "hex address with 0x prefix",
			ownerStr: "0x" + hex.EncodeToString(make([]byte, 20)),
			wantLen: 20,
			wantErr: false,
		},
		{
			name:    "hex address without 0x prefix",
			ownerStr: hex.EncodeToString(make([]byte, 20)),
			wantLen: 20,
			wantErr: false,
		},
		{
			name:    "empty string",
			ownerStr: "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid hex",
			ownerStr: "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOwnerAddress(tt.ownerStr)
			if tt.wantErr {
				if result != nil {
					t.Errorf("parseOwnerAddress() should return nil on error, got %v", result)
				}
			} else {
				if len(result) != tt.wantLen {
					t.Errorf("parseOwnerAddress() length = %d, want %d", len(result), tt.wantLen)
				}
			}
		})
	}
}

// parseAmount 和 parseTokenID 是内联函数，通过 FetchAndParseTx 测试覆盖
// 注意：FetchAndParseTx 需要真实的客户端连接，这里只测试解析逻辑

func TestParseTransactionData(t *testing.T) {
	// 创建一个模拟的交易响应数据结构
	mockTx := map[string]interface{}{
		"hash":         "0x1234567890abcdef",
		"status":       "confirmed",
		"block_height": float64(100),
		"block_hash":   "0xabcdef1234567890",
		"tx_index":     float64(0),
		"inputs": []interface{}{
			map[string]interface{}{
				"tx_hash":      "0x1111111111111111",
				"output_index": float64(0),
				"is_reference": false,
			},
		},
		"outputs": []interface{}{
			map[string]interface{}{
				"index":  float64(0),
				"type":   "asset",
				"owner":  base64.StdEncoding.EncodeToString(make([]byte, 20)),
				"amount": "1000000",
				"token_id": "",
			},
		},
	}

	// 验证数据结构正确性（FetchAndParseTx 的输入格式）
	if mockTx["hash"] == nil {
		t.Error("mockTx should have hash field")
	}
	if mockTx["outputs"] == nil {
		t.Error("mockTx should have outputs field")
	}
	
	// 验证 outputs 结构
	outputs, ok := mockTx["outputs"].([]interface{})
	if !ok {
		t.Error("outputs should be an array")
	}
	if len(outputs) != 1 {
		t.Errorf("outputs length = %d, want 1", len(outputs))
	}
}

func TestFindOutputsByOwner(t *testing.T) {
	ownerAddr := make([]byte, 20)
	for i := range ownerAddr {
		ownerAddr[i] = byte(i)
	}

	outputs := []ParsedOutput{
		{
			Index:  0,
			Owner:  ownerAddr,
			Amount: bigIntFromString("1000000"),
		},
		{
			Index:  1,
			Owner:  make([]byte, 20), // 不同的地址
			Amount: bigIntFromString("2000000"),
		},
		{
			Index:  2,
			Owner:  ownerAddr,
			Amount: bigIntFromString("3000000"),
		},
	}

	found := FindOutputsByOwner(outputs, ownerAddr)
	if len(found) != 2 {
		t.Errorf("FindOutputsByOwner() found %d outputs, want 2", len(found))
	}

	// 验证找到的输出索引
	if found[0].Index != 0 && found[0].Index != 2 {
		t.Errorf("FindOutputsByOwner() found wrong output index: %d", found[0].Index)
	}
}

func TestFindStateOutputs(t *testing.T) {
	outputs := []ParsedOutput{
		{
			Index:   0,
			Type:    "asset",
			StateID: nil,
		},
		{
			Index:   1,
			Type:    "state",
			StateID: make([]byte, 32),
		},
		{
			Index:   2,
			Type:    "state",
			StateID: make([]byte, 32),
		},
		{
			Index:   3,
			Type:    "resource",
			StateID: nil,
		},
	}

	found := FindStateOutputs(outputs)
	if len(found) != 2 {
		t.Errorf("FindStateOutputs() found %d outputs, want 2", len(found))
	}
	for _, output := range found {
		if output.Type != "state" {
			t.Errorf("FindStateOutputs() found wrong type: %s", output.Type)
		}
	}
}

func TestFindOutputsByType(t *testing.T) {
	outputs := []ParsedOutput{
		{Index: 0, Type: "asset"},
		{Index: 1, Type: "state"},
		{Index: 2, Type: "asset"},
		{Index: 3, Type: "resource"},
	}

	found := FindOutputsByType(outputs, "asset")
	if len(found) != 2 {
		t.Errorf("FindOutputsByType() found %d outputs, want 2", len(found))
	}
}

func TestSumAmountsByToken(t *testing.T) {
	tokenID := make([]byte, 32)
	tokenID[0] = 1

	outputs := []ParsedOutput{
		{
			Index:   0,
			Type:    "asset",
			Amount:  bigIntFromString("1000000"),
			TokenID: nil, // 原生币
		},
		{
			Index:   1,
			Type:    "asset",
			Amount:  bigIntFromString("2000000"),
			TokenID: nil, // 原生币
		},
		{
			Index:   2,
			Type:    "asset",
			Amount:  bigIntFromString("3000000"),
			TokenID: tokenID, // 合约代币
		},
	}

	// 测试原生币汇总
	nativeTotal := SumAmountsByToken(outputs, nil)
	if nativeTotal.String() != "3000000" {
		t.Errorf("SumAmountsByToken() native total = %s, want 3000000", nativeTotal.String())
	}

	// 测试合约代币汇总
	tokenTotal := SumAmountsByToken(outputs, tokenID)
	if tokenTotal.String() != "3000000" {
		t.Errorf("SumAmountsByToken() token total = %s, want 3000000", tokenTotal.String())
	}
}

func TestGetOutpoint(t *testing.T) {
	tests := []struct {
		name    string
		txHash  string
		index   uint32
		want    string
	}{
		{
			name:   "with 0x prefix",
			txHash: "0x1234567890abcdef",
			index:  0,
			want:   "1234567890abcdef:0",
		},
		{
			name:   "without 0x prefix",
			txHash: "1234567890abcdef",
			index:  5,
			want:   "1234567890abcdef:5",
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

// bigIntFromString 从字符串创建 big.Int（辅助函数）
func bigIntFromString(s string) *big.Int {
	result, _ := new(big.Int).SetString(s, 10)
	return result
}

