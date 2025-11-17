package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160"
)

// Wallet 钱包接口
type Wallet interface {
	// Address 获取钱包地址
	Address() []byte
	
	// SignTransaction 签名交易
	SignTransaction(tx []byte) ([]byte, error)
	
	// SignMessage 签名消息
	SignMessage(msg []byte) ([]byte, error)

	// SignHash 签名给定哈希（供高级调用方使用）
	SignHash(hash []byte) ([]byte, error)
	
	// PrivateKey 获取私钥（谨慎使用）
	PrivateKey() *ecdsa.PrivateKey
}

// SimpleWallet 简单钱包实现（用于测试和开发）
type SimpleWallet struct {
	privateKey *ecdsa.PrivateKey
	address    []byte
	createdAt  time.Time
}

// NewWallet 创建新钱包
func NewWallet() (Wallet, error) {
	// 生成 secp256k1 私钥（与链上使用的曲线保持一致）
	privateKey, err := ethcrypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	
	// 从私钥派生地址（简化实现，实际应该使用AddressManager）
	address := deriveAddress(privateKey)
	
	return &SimpleWallet{
		privateKey: privateKey,
		address:    address,
		createdAt:  time.Now(),
	}, nil
}

// NewWalletFromPrivateKey 从私钥创建钱包
func NewWalletFromPrivateKey(privateKeyHex string) (Wallet, error) {
	// 移除0x前缀（如果有）
	privateKeyHex = hexRemovePrefix(privateKeyHex)
	
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	
	// 验证私钥长度（ECDSA私钥应该是32字节）
	if len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}
	
	// 解析私钥（参考 client/core/transfer/service.go）
	privateKey, err := parsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	
	address := deriveAddress(privateKey)
	
	return &SimpleWallet{
		privateKey: privateKey,
		address:    address,
		createdAt:  time.Now(),
	}, nil
}

// Address 获取钱包地址
func (w *SimpleWallet) Address() []byte {
	return w.address
}

// SignTransaction 签名交易
func (w *SimpleWallet) SignTransaction(tx []byte) ([]byte, error) {
	// 1. 计算交易哈希
	hash := sha256.Sum256(tx)
	
	// 2. 签名哈希
	return w.SignHash(hash[:])
}

// SignHash 签名哈希值（参考 client/core/wallet/keystore.go）
func (w *SimpleWallet) SignHash(hash []byte) ([]byte, error) {
	// 使用ECDSA签名
	r, s, err := ecdsa.Sign(rand.Reader, w.privateKey, hash)
	if err != nil {
		return nil, fmt.Errorf("ecdsa sign: %w", err)
	}
	
	// 序列化签名: r || s (64字节)
	// 确保r和s都是32字节（补齐前导零）
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	s.FillBytes(sBytes)
	
	signature := append(rBytes, sBytes...)
	return signature, nil
}

// SignMessage 签名消息
func (w *SimpleWallet) SignMessage(msg []byte) ([]byte, error) {
	// 1. 计算消息哈希
	hash := sha256.Sum256(msg)
	
	// 2. 签名哈希
	return w.SignHash(hash[:])
}

// PrivateKey 获取私钥
func (w *SimpleWallet) PrivateKey() *ecdsa.PrivateKey {
	return w.privateKey
}

// deriveAddress 从私钥派生地址
// 使用 secp256k1 公钥的 HASH160(compressed_pubkey) 作为 20 字节地址
// 与链上 AddressManager 的语义保持一致
func deriveAddress(privateKey *ecdsa.PrivateKey) []byte {
	// 压缩公钥
	compressed := ethcrypto.CompressPubkey(&privateKey.PublicKey)

	// 计算 HASH160(compressed_pubkey)
	sha := sha256.Sum256(compressed)
	r := ripemd160.New()
	_, _ = r.Write(sha[:])
	return r.Sum(nil) // 20 字节
}

// parsePrivateKey 解析私钥（参考 client/core/transfer/service.go）
// 使用 go-ethereum/crypto 解析 secp256k1 私钥
func parsePrivateKey(privateKeyBytes []byte) (*ecdsa.PrivateKey, error) {
	privateKey, err := ethcrypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse secp256k1 private key failed: %w", err)
	}
	return privateKey, nil
}

// hexRemovePrefix 移除十六进制字符串的0x前缀
func hexRemovePrefix(hexStr string) string {
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		return hexStr[2:]
	}
	return hexStr
}

