package staking

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/services/staking"
	"github.com/weisyn/client-sdk-go/test/integration"
)

// TestStaking_Delegate 测试委托功能
func TestStaking_Delegate(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	delegatorWallet := integration.CreateTestWallet(t)
	validatorWallet := integration.CreateTestWallet(t)

	delegatorAddr := delegatorWallet.Address()
	validatorAddr := validatorWallet.Address()

	integration.FundTestAccount(t, c, delegatorAddr, 1000000)

	// 创建 Staking 服务
	stakingService := staking.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	delegateAmount := uint64(5000)

	// 执行委托
	result, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
		From:         delegatorAddr,
		ValidatorAddr: validatorAddr,
		Amount:       delegateAmount,
	}, delegatorWallet)

	require.NoError(t, err, "委托失败")
	require.NotNil(t, result, "委托结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")
	assert.NotEmpty(t, result.DelegateID, "DelegateID 为空")

	t.Logf("委托成功，交易哈希: %s, DelegateID: %s", result.TxHash, result.DelegateID)

	// 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, delegatorAddr)
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证 DelegateID 格式
	assert.Contains(t, result.DelegateID, ":", "DelegateID 格式不正确，应该是 txHash:index")
}

// TestStaking_Undelegate 测试取消委托功能
func TestStaking_Undelegate(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	delegatorWallet := integration.CreateTestWallet(t)
	validatorWallet := integration.CreateTestWallet(t)

	delegatorAddr := delegatorWallet.Address()
	validatorAddr := validatorWallet.Address()

	integration.FundTestAccount(t, c, delegatorAddr, 1000000)

	// 创建 Staking 服务
	stakingService := staking.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	delegateAmount := uint64(5000)

	// 1. 先执行委托
	delegateResult, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
		From:         delegatorAddr,
		ValidatorAddr: validatorAddr,
		Amount:       delegateAmount,
	}, delegatorWallet)
	require.NoError(t, err, "委托失败")

	// 等待委托交易确认
	integration.TriggerMining(t, c, delegatorAddr)
	integration.WaitForTransactionWithTest(t, c, delegateResult.TxHash)

	// 2. 执行取消委托
	// 注意：UndelegateRequest 需要 DelegateID 作为 []byte，需要从字符串转换
	delegateIDBytes := []byte(delegateResult.DelegateID)
	undelegateResult, err := stakingService.Undelegate(ctx, &staking.UndelegateRequest{
		From:       delegatorAddr,
		DelegateID: delegateIDBytes,
		Amount:     0, // 0 表示全部取消委托
	}, delegatorWallet)

	require.NoError(t, err, "取消委托失败")
	require.NotNil(t, undelegateResult, "取消委托结果为空")
	assert.NotEmpty(t, undelegateResult.TxHash, "交易哈希为空")

	t.Logf("取消委托成功，交易哈希: %s", undelegateResult.TxHash)

	// 等待取消委托交易确认
	integration.TriggerMining(t, c, delegatorAddr)
	parsedTx := integration.WaitForTransactionWithTest(t, c, undelegateResult.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)
}

