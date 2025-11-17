package token

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/services/token"
	"github.com/weisyn/client-sdk-go/test/integration"
)

// TestTokenBatchTransfer_Basic 测试批量转账功能
func TestTokenBatchTransfer_Basic(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户
	fromWallet := integration.CreateTestWallet(t)
	toWallet1 := integration.CreateTestWallet(t)
	toWallet2 := integration.CreateTestWallet(t)
	toWallet3 := integration.CreateTestWallet(t)

	fromAddr := fromWallet.Address()
	toAddr1 := toWallet1.Address()
	toAddr2 := toWallet2.Address()
	toAddr3 := toWallet3.Address()

	// 为 from 账户充值
	integration.FundTestAccount(t, c, fromAddr, 1000000)

	// 查询初始余额
	initialBalance1 := integration.GetTestAccountBalance(t, c, toAddr1, nil)
	initialBalance2 := integration.GetTestAccountBalance(t, c, toAddr2, nil)
	initialBalance3 := integration.GetTestAccountBalance(t, c, toAddr3, nil)

	// 创建 Token 服务
	tokenService := token.NewService(c)

	// 准备批量转账参数
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	transferAmount1 := uint64(1000)
	transferAmount2 := uint64(2000)
	transferAmount3 := uint64(3000)

	// 执行批量转账
	result, err := tokenService.BatchTransfer(ctx, &token.BatchTransferRequest{
		From: fromAddr,
		Transfers: []token.TransferItem{
			{To: toAddr1, Amount: transferAmount1, TokenID: nil},
			{To: toAddr2, Amount: transferAmount2, TokenID: nil},
			{To: toAddr3, Amount: transferAmount3, TokenID: nil},
		},
	}, fromWallet)

	require.NoError(t, err, "批量转账失败")
	require.NotNil(t, result, "批量转账结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")
	assert.True(t, result.Success, "批量转账未成功")

	t.Logf("批量转账成功，交易哈希: %s", result.TxHash)

	// 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, fromAddr)
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证余额变化
	finalBalance1 := integration.GetTestAccountBalance(t, c, toAddr1, nil)
	finalBalance2 := integration.GetTestAccountBalance(t, c, toAddr2, nil)
	finalBalance3 := integration.GetTestAccountBalance(t, c, toAddr3, nil)

	assert.Equal(t, initialBalance1+transferAmount1, finalBalance1, "账户1余额不正确")
	assert.Equal(t, initialBalance2+transferAmount2, finalBalance2, "账户2余额不正确")
	assert.Equal(t, initialBalance3+transferAmount3, finalBalance3, "账户3余额不正确")
}

