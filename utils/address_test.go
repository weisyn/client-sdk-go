package utils

import (
	"testing"
)

func TestAddressBytesToBase58(t *testing.T) {
	tests := []struct {
		name        string
		addressBytes []byte
		wantErr     bool
	}{
		{
			name:        "valid 20-byte address",
			addressBytes: make([]byte, 20),
			wantErr:     false,
		},
		{
			name:        "invalid length - 19 bytes",
			addressBytes: make([]byte, 19),
			wantErr:     true,
		},
		{
			name:        "invalid length - 21 bytes",
			addressBytes: make([]byte, 21),
			wantErr:     true,
		},
		{
			name:        "empty address",
			addressBytes: []byte{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestAddressBase58ToBytes(t *testing.T) {
	// 先创建一个有效的 Base58 地址
	validAddressBytes := make([]byte, 20)
	for i := range validAddressBytes {
		validAddressBytes[i] = byte(i)
	}

	validBase58, err := AddressBytesToBase58(validAddressBytes)
	if err != nil {
		t.Fatalf("AddressBytesToBase58() failed: %v", err)
	}

	tests := []struct {
		name      string
		base58Addr string
		wantErr   bool
	}{
		{
			name:      "valid Base58 address",
			base58Addr: validBase58,
			wantErr:   false,
		},
		{
			name:      "invalid Base58 - empty string",
			base58Addr: "",
			wantErr:   true,
		},
		{
			name:      "invalid Base58 - invalid characters",
			base58Addr: "0OIl",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AddressBase58ToBytes(tt.base58Addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddressBase58ToBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result) != 20 {
					t.Errorf("AddressBase58ToBytes() result length = %d, want 20", len(result))
				}
			}
		})
	}
}

func TestAddressRoundTrip(t *testing.T) {
	// 测试地址转换的往返一致性
	originalBytes := make([]byte, 20)
	for i := range originalBytes {
		originalBytes[i] = byte(i)
	}

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

	for i := range originalBytes {
		if convertedBytes[i] != originalBytes[i] {
			t.Errorf("Round trip: byte mismatch at index %d, got %d, want %d", i, convertedBytes[i], originalBytes[i])
		}
	}
}

func TestAddressHexToBase58(t *testing.T) {
	tests := []struct {
		name    string
		hexAddr string
		wantErr bool
	}{
		{
			name:    "valid hex address with 0x prefix",
			hexAddr: "0x" + makeHexString(20),
			wantErr: false,
		},
		{
			name:    "valid hex address without 0x prefix",
			hexAddr: makeHexString(20),
			wantErr: false,
		},
		{
			name:    "invalid hex - wrong length",
			hexAddr: "0x1234",
			wantErr: true,
		},
		{
			name:    "invalid hex - invalid characters",
			hexAddr: "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
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

func TestAddressBase58ToHex(t *testing.T) {
	// 先创建一个有效的 Base58 地址
	validAddressBytes := make([]byte, 20)
	validBase58, err := AddressBytesToBase58(validAddressBytes)
	if err != nil {
		t.Fatalf("AddressBytesToBase58() failed: %v", err)
	}

	tests := []struct {
		name      string
		base58Addr string
		wantErr   bool
	}{
		{
			name:      "valid Base58 address",
			base58Addr: validBase58,
			wantErr:   false,
		},
		{
			name:      "invalid Base58 - empty string",
			base58Addr: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AddressBase58ToHex(tt.base58Addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddressBase58ToHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == "" {
					t.Error("AddressBase58ToHex() result is empty")
				}
				// 验证结果以 0x 开头
				if len(result) < 2 || result[:2] != "0x" {
					t.Errorf("AddressBase58ToHex() result should start with 0x, got %s", result)
				}
			}
		})
	}
}

func TestAddressHexRoundTrip(t *testing.T) {
	// 测试十六进制地址转换的往返一致性
	originalHex := "0x" + makeHexString(20)

	base58, err := AddressHexToBase58(originalHex)
	if err != nil {
		t.Fatalf("AddressHexToBase58() failed: %v", err)
	}

	convertedHex, err := AddressBase58ToHex(base58)
	if err != nil {
		t.Fatalf("AddressBase58ToHex() failed: %v", err)
	}

	// 标准化十六进制格式（都转换为小写，带 0x 前缀）
	originalNormalized := normalizeHex(originalHex)
	convertedNormalized := normalizeHex(convertedHex)

	if originalNormalized != convertedNormalized {
		t.Errorf("Hex round trip: got %s, want %s", convertedNormalized, originalNormalized)
	}
}

// makeHexString 生成指定字节数的十六进制字符串
func makeHexString(bytes int) string {
	result := ""
	for i := 0; i < bytes; i++ {
		result += "01"
	}
	return result
}

// normalizeHex 标准化十六进制字符串（小写，带 0x 前缀）
func normalizeHex(hex string) string {
	if len(hex) >= 2 && hex[:2] == "0x" {
		hex = hex[2:]
	}
	return "0x" + hex
}

