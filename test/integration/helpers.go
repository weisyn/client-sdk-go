package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/utils"
)

// waitForTransaction 等待交易确认
//
// **功能**：
// - 轮询查询交易状态
// - 等待交易被确认（status == "confirmed"）
// - 超时后返回错误
//
// **参数**：
// - ctx: 上下文
// - c: 客户端
// - txHash: 交易哈希
// - timeout: 超时时间
//
// **返回**：
// - *utils.ParsedTx: 解析后的交易信息
// - error: 错误信息
func waitForTransaction(ctx context.Context, c client.Client, txHash string, timeout time.Duration) (*utils.ParsedTx, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			parsedTx, err := utils.FetchAndParseTx(ctx, c, txHash)
			if err == nil && parsedTx != nil {
				if parsedTx.Status == "confirmed" {
					return parsedTx, nil
				}
			}

			if time.Now().After(deadline) {
				return nil, fmt.Errorf("交易确认超时: %s (超时时间: %v)", txHash, timeout)
			}
		}
	}
}

// WaitForTransactionWithTest 等待交易确认（导出函数）
func WaitForTransactionWithTest(t *testing.T, c client.Client, txHash string) *utils.ParsedTx {
	return waitForTransactionWithTest(t, c, txHash)
}

// waitForTransactionWithTest 等待交易确认（内部实现）
func waitForTransactionWithTest(t *testing.T, c client.Client, txHash string) *utils.ParsedTx {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	parsedTx, err := waitForTransaction(ctx, c, txHash, 60*time.Second)
	require.NoError(t, err, "等待交易确认失败: %s", txHash)
	require.NotNil(t, parsedTx, "交易解析结果为空: %s", txHash)

	return parsedTx
}

// VerifyTransactionSuccess 验证交易成功（导出函数）
func VerifyTransactionSuccess(t *testing.T, parsedTx *utils.ParsedTx) {
	verifyTransactionSuccess(t, parsedTx)
}

// verifyTransactionSuccess 验证交易成功（内部实现）
func verifyTransactionSuccess(t *testing.T, parsedTx *utils.ParsedTx) {
	require.NotNil(t, parsedTx, "交易解析结果为空")
	assert.NotEmpty(t, parsedTx.Hash, "交易哈希为空")
	assert.Equal(t, "confirmed", parsedTx.Status, "交易状态不是 confirmed")
	assert.Greater(t, parsedTx.BlockHeight, uint64(0), "交易未打包进区块")
}

// verifyBalanceChange 验证余额变化
//
// **功能**：
// - 验证账户余额变化符合预期
// - 支持原生币和代币余额验证
func verifyBalanceChange(t *testing.T, c client.Client, address []byte, tokenID []byte, expectedBalance uint64, tolerance uint64) {
	// 注意：getTestAccountBalance 在 setup.go 中定义，需要导入 integration 包才能使用
	// 这里暂时使用直接调用，因为都在同一个包中
	actualBalance := getTestAccountBalance(t, c, address, tokenID)

	if tolerance == 0 {
		assert.Equal(t, expectedBalance, actualBalance, "余额不匹配")
	} else {
		diff := uint64(0)
		if actualBalance > expectedBalance {
			diff = actualBalance - expectedBalance
		} else {
			diff = expectedBalance - actualBalance
		}
		assert.LessOrEqual(t, diff, tolerance, "余额差异超出容差范围: 预期=%d, 实际=%d, 差异=%d", expectedBalance, actualBalance, diff)
	}
}

// TriggerMining 触发挖矿（导出函数）
func TriggerMining(t *testing.T, c client.Client, minerAddress []byte) {
	triggerMining(t, c, minerAddress)
}

// triggerMining 触发挖矿（内部实现）
func triggerMining(t *testing.T, c client.Client, minerAddress []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 将地址转换为 Base58 格式
	addressBase58, err := utils.AddressBytesToBase58(minerAddress)
	if err != nil {
		t.Fatalf("地址转换失败: %v", err)
	}

	// 启动挖矿
	_, err = c.Call(ctx, "wes_startMining", []interface{}{addressBase58})
	if err != nil {
		t.Logf("启动挖矿失败（可能已在运行）: %v", err)
	}

	// 等待区块生成
	time.Sleep(3 * time.Second)

	// 停止挖矿
	_, _ = c.Call(ctx, "wes_stopMining", []interface{}{})

	t.Logf("已触发挖矿，区块已生成")
}

// getBlockHeight 获取当前区块高度
func getBlockHeight(t *testing.T, c client.Client) uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.Call(ctx, "wes_blockNumber", []interface{}{})
	require.NoError(t, err, "获取区块高度失败")

	// 解析结果 - wes_blockNumber 返回十六进制字符串
	heightStr, ok := result.(string)
	if !ok {
		t.Fatalf("无效的响应格式")
	}

	// 解析十六进制高度
	var heightValue uint64
	if len(heightStr) > 2 && heightStr[:2] == "0x" {
		heightStr = heightStr[2:]
	}
	_, err = fmt.Sscanf(heightStr, "%x", &heightValue)
	if err != nil {
		return 0
	}

	return heightValue
}

// waitForBlockHeight 等待区块高度达到指定值
func waitForBlockHeight(t *testing.T, c client.Client, targetHeight uint64, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentHeight := getBlockHeight(t, c)
			if currentHeight >= targetHeight {
				return
			}

			if time.Now().After(deadline) {
				t.Fatalf("等待区块高度超时: 当前=%d, 目标=%d", currentHeight, targetHeight)
			}
		}
	}
}
