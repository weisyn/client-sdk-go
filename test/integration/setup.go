package integration

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/utils"
	"github.com/weisyn/client-sdk-go/wallet"
)

const (
	// DefaultNodeEndpoint 默认节点端点
	DefaultNodeEndpoint = "http://localhost:8080/jsonrpc"
	// DefaultTimeout 默认超时时间
	DefaultTimeout = 30 * time.Second
	// TransactionConfirmTimeout 交易确认超时时间
	TransactionConfirmTimeout = 60 * time.Second
	// TransactionConfirmInterval 交易确认轮询间隔
	TransactionConfirmInterval = 2 * time.Second
)

// TestConfig 测试配置
type TestConfig struct {
	NodeEndpoint string
	Timeout      time.Duration
}

// DefaultTestConfig 返回默认测试配置
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		NodeEndpoint: DefaultNodeEndpoint,
		Timeout:      DefaultTimeout,
	}
}

// SetupTestClient 设置测试客户端（导出函数）
//
// **功能**：
// - 创建 HTTP 客户端连接到 WES 节点
// - 验证节点是否运行（通过调用 wes_blockNumber）
// - 如果节点未运行，测试会失败
func SetupTestClient(t *testing.T) client.Client {
	return setupTestClient(t)
}

// setupTestClient 设置测试客户端（内部实现）
func setupTestClient(t *testing.T) client.Client {
	return setupTestClientWithConfig(t, DefaultTestConfig())
}

// setupTestClientWithConfig 使用配置设置测试客户端
func setupTestClientWithConfig(t *testing.T, cfg *TestConfig) client.Client {
	if cfg == nil {
		cfg = DefaultTestConfig()
	}

	clientCfg := &client.Config{
		Endpoint: cfg.NodeEndpoint,
		Protocol: client.ProtocolHTTP,
		Timeout:  int(cfg.Timeout.Seconds()),
		Debug:    false,
	}

	c, err := client.NewClient(clientCfg)
	require.NoError(t, err, "创建客户端失败")

	// 验证节点是否运行
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = c.Call(ctx, "wes_blockNumber", []interface{}{})
	require.NoError(t, err, "节点未运行，请先启动节点: %s", cfg.NodeEndpoint)

	return c
}

// TeardownTestClient 清理测试客户端（导出函数）
func TeardownTestClient(t *testing.T, c client.Client) {
	teardownTestClient(t, c)
}

// teardownTestClient 清理测试客户端（内部实现）
func teardownTestClient(t *testing.T, c client.Client) {
	if c != nil {
		err := c.Close()
		if err != nil {
			t.Logf("关闭客户端时出现警告: %v", err)
		}
	}
}

// CreateTestWallet 创建测试钱包（导出函数）
func CreateTestWallet(t *testing.T) wallet.Wallet {
	return createTestWallet(t)
}

// createTestWallet 创建测试钱包（内部实现）
func createTestWallet(t *testing.T) wallet.Wallet {
	w, err := wallet.NewWallet()
	require.NoError(t, err, "创建测试钱包失败")
	return w
}

// createTestWalletFromPrivateKey 从私钥创建测试钱包
//
// **用途**：
// - 使用固定的测试私钥创建钱包
// - 便于测试用例之间共享账户
func createTestWalletFromPrivateKey(t *testing.T, privateKeyHex string) wallet.Wallet {
	w, err := wallet.NewWalletFromPrivateKey(privateKeyHex)
	require.NoError(t, err, "从私钥创建测试钱包失败")
	return w
}

// FundTestAccount 为测试账户充值（导出函数）
func FundTestAccount(t *testing.T, c client.Client, address []byte, amount uint64) {
	fundTestAccount(t, c, address, amount)
}

// fundTestAccount 为测试账户充值（内部实现）
func fundTestAccount(t *testing.T, c client.Client, address []byte, amount uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 将地址转换为 Base58 格式（WES 使用 Base58 地址）
	addressBase58, err := utils.AddressBytesToBase58(address)
	if err != nil {
		t.Fatalf("地址转换失败: %v", err)
	}

	// 为确保每次充值都使用当前测试地址作为矿工地址，这里采取：
	// 1. 尝试停止当前挖矿（如果未在挖矿会返回错误，忽略即可）
	// 2. 使用当前地址调用 wes_startMining
	// 3. 等待一个短暂时间让新区块产出
	// 4. 再次停止挖矿，避免影响后续测试

	// 1. 尝试停止当前挖矿（忽略错误）
	_, _ = c.Call(ctx, "wes_stopMining", []interface{}{})

	// 2. 使用当前地址启动挖矿
	_, err = c.Call(ctx, "wes_startMining", []interface{}{addressBase58})
	if err != nil {
		t.Fatalf("启动挖矿失败: %v", err)
	}

	// 3. 等待区块生成并确认 UTXO 可用
	// 轮询检查 UTXO，最多等待 10 秒
	maxWait := 10 * time.Second
	checkInterval := 500 * time.Millisecond
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		time.Sleep(checkInterval)
		// 查询 UTXO 确认是否已有余额
		result, err := c.Call(ctx, "wes_getUTXO", []interface{}{addressBase58})
		if err == nil {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if utxos, ok := resultMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
					break // UTXO 已可用
				}
			}
		}
	}

	// 4. 停止挖矿（忽略错误）
	_, _ = c.Call(ctx, "wes_stopMining", []interface{}{})

	t.Logf("已为账户充值（通过挖矿）: %s", addressBase58)
}

// GetTestAccountBalance 查询测试账户余额（导出函数）
func GetTestAccountBalance(t *testing.T, c client.Client, address []byte, tokenID []byte) uint64 {
	return getTestAccountBalance(t, c, address, tokenID)
}

// getTestAccountBalance 查询测试账户余额（内部实现）
func getTestAccountBalance(t *testing.T, c client.Client, address []byte, tokenID []byte) uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 将地址转换为 Base58 格式
	addressBase58, err := utils.AddressBytesToBase58(address)
	if err != nil {
		t.Fatalf("地址转换失败: %v", err)
	}

	// 调用 wes_getUTXO
	result, err := c.Call(ctx, "wes_getUTXO", []interface{}{addressBase58})
	require.NoError(t, err, "查询余额失败")

	// 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("无效的响应格式")
	}

	utxos, ok := resultMap["utxos"].([]interface{})
	if !ok {
		return 0
	}

	// 汇总金额
	var totalAmount uint64
	for _, utxo := range utxos {
		utxoMap, ok := utxo.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查 tokenID 是否匹配
		if tokenID != nil {
			utxoTokenID, ok := utxoMap["token_id"].(string)
			if !ok {
				continue
			}
			expectedTokenIDHex := fmt.Sprintf("0x%x", tokenID)
			if utxoTokenID != expectedTokenIDHex {
				continue
			}
		} else {
			// nil tokenID 表示原生币
			utxoTokenID, ok := utxoMap["token_id"].(string)
			if ok && utxoTokenID != "0x" && utxoTokenID != "" {
				continue
			}
		}

		// 提取金额（amount 为十进制字符串，例如 "5000000000"）
		amountStr, ok := utxoMap["amount"].(string)
		if !ok {
			continue
		}

		// 解析十进制金额
		var amountValue uint64
		amountValue, err := strconv.ParseUint(amountStr, 10, 64)
		if err != nil {
			t.Logf("解析金额失败，忽略该UTXO: %v", err)
			continue
		}
			totalAmount += amountValue
	}

	return totalAmount
}

// EnsureNodeRunning 确保节点运行（导出函数）
func EnsureNodeRunning(t *testing.T) {
	ensureNodeRunning(t)
}

// ensureNodeRunning 确保节点运行（内部实现）
func ensureNodeRunning(t *testing.T) {
	cfg := DefaultTestConfig()
	clientCfg := &client.Config{
		Endpoint: cfg.NodeEndpoint,
		Protocol: client.ProtocolHTTP,
		Timeout:  5,
		Debug:    false,
	}

	c, err := client.NewClient(clientCfg)
	if err != nil {
		t.Fatalf("无法创建客户端: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = c.Call(ctx, "wes_blockNumber", []interface{}{})
	if err != nil {
		t.Fatalf("节点未运行，请先启动节点: %s\n启动命令: cd /Users/qinglong/go/src/chaincodes/WES/weisyn.git && bash scripts/testing/common/test_init.sh", cfg.NodeEndpoint)
	}
}
