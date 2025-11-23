package utils

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/btcsuite/btcutil/base58"
)

// TestAddressBytesToBase58_EdgeCases 测试地址转换的边界情况
func TestAddressBytesToBase58_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		addressBytes []byte
		wantErr     bool
	}{
		{
			name:        "all zeros",
			addressBytes: make([]byte, 20),
			wantErr:     false,
		},
		{
			name:        "all ones",
			addressBytes: make([]byte, 20),
			wantErr:     false,
		},
		{
			name:        "max bytes",
			addressBytes: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr:     false,
		},
		{
			name:        "nil address",
			addressBytes: nil,
			wantErr:     true,
		},
		{
			name:        "very large address (100 bytes)",
			addressBytes: make([]byte, 100),
			wantErr:     true,
		},
		{
			name:        "single byte",
			addressBytes: []byte{0x01},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "all ones" {
				for i := range tt.addressBytes {
					tt.addressBytes[i] = 0xFF
				}
			}
			result, err := AddressBytesToBase58(tt.addressBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddressBytesToBase58() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("AddressBytesToBase58() result is empty")
			}
		})
	}
}

// TestAddressBase58ToBytes_InvalidChecksum 测试无效校验和
func TestAddressBase58ToBytes_InvalidChecksum(t *testing.T) {
	// 创建一个有效的地址
	validBytes := make([]byte, 20)
	validBase58, err := AddressBytesToBase58(validBytes)
	if err != nil {
		t.Fatalf("AddressBytesToBase58() failed: %v", err)
	}

	// 解码并修改校验和
	decoded := base58.Decode(validBase58)
	if len(decoded) != 25 {
		t.Fatalf("decoded length = %d, want 25", len(decoded))
	}

	// 修改校验和（翻转最后一位）
	decoded[24] ^= 0xFF
	invalidBase58 := base58.Encode(decoded)

	// 应该失败
	_, err = AddressBase58ToBytes(invalidBase58)
	if err == nil {
		t.Error("AddressBase58ToBytes() should fail with invalid checksum")
	}
}

// TestAddressBase58ToBytes_WrongVersion 测试错误的版本字节
func TestAddressBase58ToBytes_WrongVersion(t *testing.T) {
	// 创建一个有效的地址
	validBytes := make([]byte, 20)
	validBase58, err := AddressBytesToBase58(validBytes)
	if err != nil {
		t.Fatalf("AddressBytesToBase58() failed: %v", err)
	}

	// 解码并修改版本字节
	decoded := base58.Decode(validBase58)
	if len(decoded) != 25 {
		t.Fatalf("decoded length = %d, want 25", len(decoded))
	}

	// 修改版本字节（WES 使用 0x1C，改为其他值）
	decoded[0] = 0x00
	// 重新计算校验和
	hash1 := sha256.Sum256(decoded[:21])
	hash2 := sha256.Sum256(hash1[:])
	copy(decoded[21:], hash2[:4])
	invalidBase58 := base58.Encode(decoded)

	// 应该失败（校验和不匹配）
	_, err = AddressBase58ToBytes(invalidBase58)
	if err == nil {
		t.Error("AddressBase58ToBytes() should fail with wrong version byte")
	}
}

// TestAddressBase58ToBytes_TooShort 测试过短的 Base58 地址
func TestAddressBase58ToBytes_TooShort(t *testing.T) {
	// 创建一个只有 20 字节的 Base58（缺少版本字节和校验和）
	shortBytes := make([]byte, 20)
	shortBase58 := base58.Encode(shortBytes)

	_, err := AddressBase58ToBytes(shortBase58)
	if err == nil {
		t.Error("AddressBase58ToBytes() should fail with too short address")
	}
}

// TestAddressHexToBase58_EdgeCases 测试十六进制地址转换的边界情况
func TestAddressHexToBase58_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		hexAddr string
		wantErr bool
	}{
		{
			name:    "uppercase hex",
			hexAddr: "0x" + makeHexStringUpper(20),
			wantErr: false,
		},
		{
			name:    "mixed case hex",
			hexAddr: "0xAaBbCcDdEeFf1122334455667788990011223344",
			wantErr: false,
		},
		{
			name:    "hex with many leading zeros",
			hexAddr: "0x0000000000000000000000000000000000000001",
			wantErr: false,
		},
		{
			name:    "hex with spaces",
			hexAddr: "0x 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01 01",
			wantErr: true,
		},
		{
			name:    "hex with newlines",
			hexAddr: "0x01\n01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AddressHexToBase58(tt.hexAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddressHexToBase58() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("AddressHexToBase58() result is empty")
			}
		})
	}
}

// TestAddressRoundTrip_MultipleAddresses 测试多个地址的往返一致性
func TestAddressRoundTrip_MultipleAddresses(t *testing.T) {
	testAddresses := [][]byte{
		make([]byte, 20),                    // 全零
		{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // 全1
		{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC}, // 随机
	}

	for i, originalBytes := range testAddresses {
		t.Run(fmt.Sprintf("address_%d", i), func(t *testing.T) {
			base58, err := AddressBytesToBase58(originalBytes)
			if err != nil {
				t.Fatalf("AddressBytesToBase58() failed: %v", err)
			}

			convertedBytes, err := AddressBase58ToBytes(base58)
			if err != nil {
				t.Fatalf("AddressBase58ToBytes() failed: %v", err)
			}

			if len(convertedBytes) != len(originalBytes) {
				t.Errorf("Round trip: length mismatch, got %d, want %d", len(convertedBytes), len(originalBytes))
			}

			for j := range originalBytes {
				if convertedBytes[j] != originalBytes[j] {
					t.Errorf("Round trip: byte mismatch at index %d, got %d, want %d", j, convertedBytes[j], originalBytes[j])
				}
			}
		})
	}
}

// makeHexStringUpper 生成大写的十六进制字符串
func makeHexStringUpper(bytes int) string {
	result := ""
	for i := 0; i < bytes; i++ {
		result += "AB"
	}
	return result
}

