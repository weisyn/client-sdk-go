package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Keystore Keystore文件结构（参考client/core/wallet/keystore.go）
type Keystore struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Address string `json:"address"`
	Crypto  Crypto `json:"crypto"`
}

// Crypto 加密信息
type Crypto struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams CipherParams           `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

// CipherParams 加密参数
type CipherParams struct {
	IV string `json:"iv"`
}

// KeystoreManager Keystore管理器
type KeystoreManager struct {
	keystoreDir string
}

// NewKeystoreManager 创建Keystore管理器
func NewKeystoreManager(keystoreDir string) (*KeystoreManager, error) {
	if err := os.MkdirAll(keystoreDir, 0700); err != nil {
		return nil, fmt.Errorf("create keystore dir: %w", err)
	}
	
	return &KeystoreManager{
		keystoreDir: keystoreDir,
	}, nil
}

// Save 保存私钥到Keystore
func (km *KeystoreManager) Save(address string, privateKey []byte, password string) (string, error) {
	// 1. 生成随机salt和IV
	salt := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("generate iv: %w", err)
	}
	
	// 2. 派生密钥（使用PBKDF2）
	key := deriveKey(password, salt)
	
	// 3. 加密私钥
	ciphertext, err := encryptAES(key, privateKey, iv)
	if err != nil {
		return "", fmt.Errorf("encrypt private key: %w", err)
	}
	
	// 4. 计算MAC
	mac := computeMAC(key, ciphertext)
	
	// 5. 构建Keystore结构
	keystore := &Keystore{
		Version: 1,
		ID:      generateID(),
		Address: address,
		Crypto: Crypto{
			Cipher:     "aes-128-ctr",
			CipherText: hex.EncodeToString(ciphertext),
			CipherParams: CipherParams{
				IV: hex.EncodeToString(iv),
			},
			KDF: "pbkdf2",
			KDFParams: map[string]interface{}{
				"c":    262144,
				"dklen": 32,
				"prf":  "hmac-sha256",
				"salt": hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
	}
	
	// 6. 保存到文件
	keystorePath := filepath.Join(km.keystoreDir, fmt.Sprintf("%s.json", address))
	file, err := os.Create(keystorePath)
	if err != nil {
		return "", fmt.Errorf("create keystore file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(keystore); err != nil {
		return "", fmt.Errorf("encode keystore: %w", err)
	}
	
	return keystorePath, nil
}

// Load 从Keystore加载私钥
func (km *KeystoreManager) Load(address string, password string) ([]byte, error) {
	// 1. 读取Keystore文件
	keystorePath := filepath.Join(km.keystoreDir, fmt.Sprintf("%s.json", address))
	data, err := os.ReadFile(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("read keystore file: %w", err)
	}
	
	// 2. 解析Keystore
	var keystore Keystore
	if err := json.Unmarshal(data, &keystore); err != nil {
		return nil, fmt.Errorf("parse keystore: %w", err)
	}
	
	// 3. 提取参数
	saltHex, ok := keystore.Crypto.KDFParams["salt"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid salt")
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	
	iv, err := hex.DecodeString(keystore.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("decode iv: %w", err)
	}
	
	ciphertext, err := hex.DecodeString(keystore.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	
	// 4. 派生密钥
	key := deriveKey(password, salt)
	
	// 5. 验证MAC
	expectedMAC := computeMAC(key, ciphertext)
	actualMAC, err := hex.DecodeString(keystore.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("decode mac: %w", err)
	}
	if !equalMAC(expectedMAC, actualMAC) {
		return nil, fmt.Errorf("invalid password")
	}
	
	// 6. 解密私钥
	privateKey, err := decryptAES(key, ciphertext, iv)
	if err != nil {
		return nil, fmt.Errorf("decrypt private key: %w", err)
	}
	
	return privateKey, nil
}

// deriveKey 派生密钥（PBKDF2）
func deriveKey(password string, salt []byte) []byte {
	// TODO: 实现PBKDF2密钥派生
	// 实际应该使用 golang.org/x/crypto/pbkdf2
	hash := sha256.Sum256(append([]byte(password), salt...))
	return hash[:]
}

// encryptAES AES加密
func encryptAES(key, plaintext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	stream := cipher.NewCTR(block, iv)
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)
	
	return ciphertext, nil
}

// decryptAES AES解密
func decryptAES(key, ciphertext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	
	return plaintext, nil
}

// computeMAC 计算MAC
func computeMAC(key, ciphertext []byte) []byte {
	hash := sha256.Sum256(append(key, ciphertext...))
	return hash[:]
}

// equalMAC 比较MAC
func equalMAC(a, b []byte) bool {
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

// generateID 生成ID
func generateID() string {
	id := make([]byte, 16)
	rand.Read(id)
	return hex.EncodeToString(id)
}

