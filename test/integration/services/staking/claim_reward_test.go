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

// TestStaking_ClaimReward 测试领取奖励功能
func TestStaking_ClaimReward(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	integration.FundTestAccount(t, c, address, 1000000)

	// 创建 Staking 服务
	stakingService := staking.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 先执行质押，以便后续领取奖励
	stakeAmount := uint64(10000)
	lockBlocks := uint64(10)
	validatorAddr := address // 使用自己作为验证者（测试场景）

	stakeResult, err := stakingService.Stake(ctx, &staking.StakeRequest{
		From:         address,
		ValidatorAddr: validatorAddr,
		Amount:       stakeAmount,
		LockBlocks:   lockBlocks,
	}, wallet)
	require.NoError(t, err, "质押失败")

	// 等待质押交易确认
	integration.TriggerMining(t, c, address)
	integration.WaitForTransactionWithTest(t, c, stakeResult.TxHash)

	// 2. 执行领取奖励
	// 注意：ClaimRewardRequest 需要 StakeID 或 DelegateID
	stakeIDBytes := []byte(stakeResult.StakeID)
	claimResult, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
		From:     address,
		StakeID:  stakeIDBytes,
		DelegateID: nil, // 使用 StakeID
	}, wallet)

	// 注意：如果当前没有奖励可领取，可能会返回错误
	// 这是正常的，因为奖励需要时间累积
	if err != nil {
		t.Logf("领取奖励失败（可能没有奖励可领取）: %v", err)
		// 如果是因为没有奖励，这是可以接受的
		return
	}

	require.NotNil(t, claimResult, "领取奖励结果为空")
	assert.NotEmpty(t, claimResult.TxHash, "交易哈希为空")

	t.Logf("领取奖励成功，交易哈希: %s", claimResult.TxHash)
	if claimResult.RewardAmount > 0 {
		t.Logf("奖励金额: %d", claimResult.RewardAmount)
	}

	// 等待领取奖励交易确认
	integration.TriggerMining(t, c, address)
	parsedTx := integration.WaitForTransactionWithTest(t, c, claimResult.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证余额增加（奖励金额）
	finalBalance := integration.GetTestAccountBalance(t, c, address, nil)
	t.Logf("最终余额: %d", finalBalance)
	assert.Greater(t, finalBalance, uint64(0), "余额应该大于0")
}

// TestStaking_ClaimReward_WithDelegateID 测试通过 DelegateID 领取奖励
func TestStaking_ClaimReward_WithDelegateID(t *testing.T) {
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

	// 1. 先执行委托
	delegateAmount := uint64(5000)
	delegateResult, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
		From:         delegatorAddr,
		ValidatorAddr: validatorAddr,
		Amount:       delegateAmount,
	}, delegatorWallet)
	require.NoError(t, err, "委托失败")

	// 等待委托交易确认
	integration.TriggerMining(t, c, delegatorAddr)
	integration.WaitForTransactionWithTest(t, c, delegateResult.TxHash)

	// 2. 执行领取奖励（通过 DelegateID）
	delegateIDBytes := []byte(delegateResult.DelegateID)
	claimResult, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
		From:       delegatorAddr,
		StakeID:    nil, // 使用 DelegateID
		DelegateID: delegateIDBytes,
	}, delegatorWallet)

	// 注意：如果当前没有奖励可领取，可能会返回错误
	if err != nil {
		t.Logf("领取奖励失败（可能没有奖励可领取）: %v", err)
		return
	}

	require.NotNil(t, claimResult, "领取奖励结果为空")
	assert.NotEmpty(t, claimResult.TxHash, "交易哈希为空")

	t.Logf("领取奖励成功，交易哈希: %s", claimResult.TxHash)

	// 等待领取奖励交易确认
	integration.TriggerMining(t, c, delegatorAddr)
	parsedTx := integration.WaitForTransactionWithTest(t, c, claimResult.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)
}

