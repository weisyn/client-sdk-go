package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
)

// AddressBytesToBase58 将 20 字节地址转换为 Base58Check 编码
//
// **格式**：
// - 版本字节（1字节）+ 地址哈希（20字节）+ 校验和（4字节）
// - 使用 Base58Check 编码
//
// **注意**：
// - SDK 独立实现，不依赖 WES 内部包
// - 使用标准 Base58Check 编码（与 Bitcoin 兼容）
func AddressBytesToBase58(addressBytes []byte) (string, error) {
	if len(addressBytes) != 20 {
		return "", fmt.Errorf("invalid address length: expected 20 bytes, got %d", len(addressBytes))
	}

	// WES 地址版本字节
	// WESP2PKHVersion = 0x1C (WES P2PKH 地址版本)
	// 参考：internal/core/infrastructure/crypto/address/address.go
	versionByte := byte(0x1C)

	// 构建版本字节 + 地址哈希
	versionedAddress := append([]byte{versionByte}, addressBytes...)

	// 计算校验和（双重 SHA256，取前4字节）
	hash1 := sha256.Sum256(versionedAddress)
	hash2 := sha256.Sum256(hash1[:])
	checksum := hash2[:4]

	// 构建完整地址：版本字节 + 地址哈希 + 校验和
	fullAddress := append(versionedAddress, checksum...)

	// Base58 编码
	base58Addr := base58.Encode(fullAddress)

	return base58Addr, nil
}

// AddressBase58ToBytes 将 Base58Check 编码地址转换为 20 字节地址哈希
//
// **格式**：
// - Base58Check 解码后：版本字节（1字节）+ 地址哈希（20字节）+ 校验和（4字节）
// - 返回地址哈希（20字节）
func AddressBase58ToBytes(base58Addr string) ([]byte, error) {
	// Base58 解码
	decoded := base58.Decode(base58Addr)

	// 验证长度：版本字节（1）+ 地址哈希（20）+ 校验和（4）= 25 字节
	if len(decoded) != 25 {
		return nil, fmt.Errorf("invalid address length: expected 25 bytes after Base58 decode, got %d", len(decoded))
	}

	// 验证校验和
	versionedAddress := decoded[:21] // 版本字节 + 地址哈希
	checksum := decoded[21:]         // 校验和

	hash1 := sha256.Sum256(versionedAddress)
	hash2 := sha256.Sum256(hash1[:])
	expectedChecksum := hash2[:4]

	// 比较校验和
	if !equalBytes(checksum, expectedChecksum) {
		return nil, fmt.Errorf("invalid checksum")
	}

	// 返回地址哈希（跳过版本字节）
	addressBytes := decoded[1:21]

	return addressBytes, nil
}

// equalBytes 比较两个字节数组是否相等
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// AddressHexToBase58 将十六进制地址转换为 Base58Check 编码
// 注意：此函数仅用于兼容性，WES SDK 推荐使用 Base58 格式
// hexAddr 可以是带 0x 前缀或不带前缀的十六进制字符串（40个字符，20字节）
func AddressHexToBase58(hexAddr string) (string, error) {
	// 移除 0x 前缀（如果存在）
	if len(hexAddr) >= 2 && hexAddr[:2] == "0x" {
		hexAddr = hexAddr[2:]
	}

	// 解析十六进制字符串为字节数组
	addressBytes, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}

	// 验证长度：20 字节
	if len(addressBytes) != 20 {
		return "", fmt.Errorf("invalid hex address length: expected 40 hex characters (20 bytes), got %d bytes", len(addressBytes))
	}

	// 转换为 Base58
	return AddressBytesToBase58(addressBytes)
}

// AddressBase58ToHex 将 Base58Check 编码地址转换为十六进制格式（带 0x 前缀）
// 注意：此函数仅用于兼容性，WES SDK 推荐使用 Base58 格式
func AddressBase58ToHex(base58Addr string) (string, error) {
	addressBytes, err := AddressBase58ToBytes(base58Addr)
	if err != nil {
		return "", err
	}
	// 转换为十六进制，带 0x 前缀
	return fmt.Sprintf("0x%x", addressBytes), nil
}

