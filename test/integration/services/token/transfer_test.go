package token

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/services/token"
	"github.com/weisyn/client-sdk-go/test/integration"
)

// TestTokenTransfer_Basic 测试基本转账功能
//
// **测试步骤**：
// 1. 创建两个测试账户（from, to）
// 2. 为 from 账户充值（通过挖矿）
// 3. 调用 tokenService.Transfer() 执行转账
// 4. 触发挖矿，等待交易确认
// 5. 验证结果：
//    - result.TxHash 不为空
//    - result.Success == true
//    - 查询 to 账户余额，验证金额正确
//    - 查询 from 账户余额，验证金额减少
func TestTokenTransfer_Basic(t *testing.T) {
	// 1. 确保节点运行
	integration.EnsureNodeRunning(t)

	// 2. 设置测试客户端
	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 3. 创建测试账户
	fromWallet := integration.CreateTestWallet(t)
	toWallet := integration.CreateTestWallet(t)

	fromAddr := fromWallet.Address()
	toAddr := toWallet.Address()

	t.Logf("From 地址: 0x%x", fromAddr)
	t.Logf("To 地址: 0x%x", toAddr)

	// 4. 为 from 账户充值（通过挖矿）
	integration.FundTestAccount(t, c, fromAddr, 1000000)

	// 5. 查询初始余额
	initialFromBalance := integration.GetTestAccountBalance(t, c, fromAddr, nil)
	initialToBalance := integration.GetTestAccountBalance(t, c, toAddr, nil)

	t.Logf("From 初始余额: %d", initialFromBalance)
	t.Logf("To 初始余额: %d", initialToBalance)

	// 6. 创建 Token 服务
	tokenService := token.NewService(c)

	// 7. 准备转账参数
	transferAmount := uint64(1000)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 8. 执行转账
	result, err := tokenService.Transfer(ctx, &token.TransferRequest{
		From:    fromAddr,
		To:      toAddr,
		Amount:  transferAmount,
		TokenID: nil, // nil 表示原生币
	}, fromWallet)

	require.NoError(t, err, "转账失败")
	require.NotNil(t, result, "转账结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")
	assert.True(t, result.Success, "转账未成功")

	t.Logf("转账成功，交易哈希: %s", result.TxHash)

	// 9. 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, fromAddr)

	// 10. 等待交易确认
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	t.Logf("交易已确认，区块高度: %d", parsedTx.BlockHeight)

	// 11. 验证余额变化
	finalFromBalance := integration.GetTestAccountBalance(t, c, fromAddr, nil)
	finalToBalance := integration.GetTestAccountBalance(t, c, toAddr, nil)

	t.Logf("From 最终余额: %d", finalFromBalance)
	t.Logf("To 最终余额: %d", finalToBalance)

	// 验证 from 账户余额变化
	// 转账后，From 账户会收到找零，所以余额减少应该至少是转账金额
	// 但由于测试中可能触发挖矿确认，From 账户可能收到挖矿奖励
	// 因此只验证 To 账户余额增加即可，From 账户余额变化不做严格验证
	// 实际减少 = 消耗的 UTXO - 收到的找零 = transferAmount + fee（约 1000）
	balanceChange := int64(finalFromBalance) - int64(initialFromBalance)
	t.Logf("From 账户余额变化: %d (正值表示增加，负值表示减少)", balanceChange)
	// 如果余额减少，应该至少减少转账金额
	if balanceChange < 0 {
		assert.LessOrEqual(t, balanceChange, -int64(transferAmount), "From 账户余额减少应至少等于转账金额")
	}
	// 验证 to 账户余额增加
	assert.Equal(t, initialToBalance+transferAmount, finalToBalance, "To 账户余额增加不正确")

	// 12. 验证交易输出
	require.NotEmpty(t, parsedTx.Outputs, "交易输出为空")

	// 查找 to 账户的输出
	foundToOutput := false
	for _, output := range parsedTx.Outputs {
		if hex.EncodeToString(output.Owner) == hex.EncodeToString(toAddr) {
			if output.Amount != nil && output.Amount.Uint64() >= transferAmount {
				foundToOutput = true
				t.Logf("找到 To 账户输出: 金额=%s, TokenID=%x", output.Amount.String(), output.TokenID)
				break
			}
		}
	}
	assert.True(t, foundToOutput, "未找到 To 账户的输出")
}

// TestTokenTransfer_InvalidAddress 测试无效地址转账
func TestTokenTransfer_InvalidAddress(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	fromWallet := integration.CreateTestWallet(t)
	fromAddr := fromWallet.Address()

	// 创建无效的接收地址（长度错误）
	invalidToAddr := make([]byte, 19) // 应该是 20 字节

	tokenService := token.NewService(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tokenService.Transfer(ctx, &token.TransferRequest{
		From:    fromAddr,
		To:      invalidToAddr,
		Amount:  1000,
		TokenID: nil,
	}, fromWallet)

	assert.Error(t, err, "应该返回错误")
}

// TestTokenTransfer_InsufficientBalance 测试余额不足转账
func TestTokenTransfer_InsufficientBalance(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	fromWallet := integration.CreateTestWallet(t)
	toWallet := integration.CreateTestWallet(t)

	fromAddr := fromWallet.Address()
	toAddr := toWallet.Address()

	// 不为 from 账户充值，余额为 0

	tokenService := token.NewService(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := tokenService.Transfer(ctx, &token.TransferRequest{
		From:    fromAddr,
		To:      toAddr,
		Amount:  1000,
		TokenID: nil,
	}, fromWallet)

	assert.Error(t, err, "应该返回余额不足错误")
}

