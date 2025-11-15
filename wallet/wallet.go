package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// Wallet 钱包接口
type Wallet interface {
	// Address 获取钱包地址
	Address() []byte
	
	// SignTransaction 签名交易
	SignTransaction(tx []byte) ([]byte, error)
	
	// SignMessage 签名消息
	SignMessage(msg []byte) ([]byte, error)
	
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
	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(nil, rand.Reader)
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

// deriveAddress 从私钥派生地址（简化实现）
// 注意：这是一个占位实现，实际应该使用 AddressManager
// 参考：client/core/wallet/account_manager.go 中的地址派生逻辑
func deriveAddress(privateKey *ecdsa.PrivateKey) []byte {
	// 简化实现：使用私钥的D值的前20字节作为地址
	// 实际应该：
	// 1. 从私钥计算公钥
	// 2. 对公钥进行哈希（SHA256 + RIPEMD160）
	// 3. 添加版本号和校验和
	// 4. Base58编码
	
	// 临时实现：使用私钥D值的前20字节
	address := make([]byte, 20)
	dBytes := privateKey.D.Bytes()
	if len(dBytes) >= 20 {
		copy(address, dBytes[:20])
	} else {
		// 如果D值不足20字节，用0补齐
		copy(address[20-len(dBytes):], dBytes)
	}
	return address
}

// parsePrivateKey 解析私钥（参考 client/core/transfer/service.go）
// 使用标准库的 crypto/ecdsa
func parsePrivateKey(privateKeyBytes []byte) (*ecdsa.PrivateKey, error) {
	// 验证长度
	if len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}
	
	// 转换为big.Int
	d := new(big.Int).SetBytes(privateKeyBytes)
	
	// 验证私钥范围（必须在[1, n-1]之间，其中n是曲线的阶）
	curve := elliptic.P256()
	n := curve.Params().N
	if d.Cmp(big.NewInt(1)) < 0 || d.Cmp(new(big.Int).Sub(n, big.NewInt(1))) > 0 {
		return nil, fmt.Errorf("private key out of range")
	}
	
	// 计算公钥点
	x, y := curve.ScalarBaseMult(privateKeyBytes)
	
	// 创建私钥结构
	privateKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: d,
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

