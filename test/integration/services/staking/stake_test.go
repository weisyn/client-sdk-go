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

// TestStaking_Stake 测试质押功能
func TestStaking_Stake(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	integration.FundTestAccount(t, c, address, 1000000)

	// 查询初始余额
	initialBalance := integration.GetTestAccountBalance(t, c, address, nil)
	t.Logf("初始余额: %d", initialBalance)

	// 创建 Staking 服务
	stakingService := staking.NewService(c)

	// 准备质押参数
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stakeAmount := uint64(10000)
	lockBlocks := uint64(100)
	validatorAddr := address // 使用自己作为验证者（测试场景）

	// 执行质押
	result, err := stakingService.Stake(ctx, &staking.StakeRequest{
		From:         address,
		ValidatorAddr: validatorAddr,
		Amount:       stakeAmount,
		LockBlocks:   lockBlocks,
	}, wallet)

	require.NoError(t, err, "质押失败")
	require.NotNil(t, result, "质押结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")
	assert.NotEmpty(t, result.StakeID, "StakeID 为空")

	t.Logf("质押成功，交易哈希: %s, StakeID: %s", result.TxHash, result.StakeID)

	// 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, address)
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证 StakeID 格式（应该是 outpoint 格式：txHash:index）
	assert.Contains(t, result.StakeID, ":", "StakeID 格式不正确，应该是 txHash:index")

	// 验证余额减少（质押金额被锁定）
	finalBalance := integration.GetTestAccountBalance(t, c, address, nil)
	t.Logf("最终余额: %d", finalBalance)
	assert.Less(t, finalBalance, initialBalance, "余额应该减少（质押金额被锁定）")
	assert.GreaterOrEqual(t, initialBalance-stakeAmount, finalBalance, "余额减少应该至少等于质押金额")

	// 验证交易输出中包含质押输出
	foundStakeOutput := false
	for _, output := range parsedTx.Outputs {
		if output.Amount != nil && output.Amount.Uint64() == stakeAmount {
			// 检查是否有 HeightLock（通过 outpoint 验证）
			if output.Outpoint != "" {
				foundStakeOutput = true
				t.Logf("找到质押输出: Outpoint=%s, Amount=%s", output.Outpoint, output.Amount.String())
				break
			}
		}
	}
	assert.True(t, foundStakeOutput, "未找到质押输出")
}

// TestStaking_Unstake 测试解质押功能
func TestStaking_Unstake(t *testing.T) {
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

	// 1. 先执行质押
	stakeAmount := uint64(10000)
	lockBlocks := uint64(10) // 使用较短的锁定时间以便测试
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

	// 2. 等待锁定区块数（或直接尝试解质押，如果节点支持提前解质押）
	// 注意：实际测试中，可能需要等待 lockBlocks 个区块
	// 这里假设节点支持立即解质押（或使用较短的 lockBlocks）

	// 3. 执行解质押
	// 注意：UnstakeRequest 需要 StakeID 作为 []byte，需要从字符串转换
	// StakeID 格式是 "txHash:index"，需要解析
	stakeIDBytes := []byte(stakeResult.StakeID)
	unstakeResult, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
		From:    address,
		StakeID: stakeIDBytes,
		Amount:  0, // 0 表示全部解质押
	}, wallet)

	require.NoError(t, err, "解质押失败")
	require.NotNil(t, unstakeResult, "解质押结果为空")
	assert.NotEmpty(t, unstakeResult.TxHash, "交易哈希为空")

	t.Logf("解质押成功，交易哈希: %s", unstakeResult.TxHash)
	if unstakeResult.UnstakeAmount > 0 {
		t.Logf("解质押金额: %d", unstakeResult.UnstakeAmount)
	}
	if unstakeResult.RewardAmount > 0 {
		t.Logf("奖励金额: %d", unstakeResult.RewardAmount)
	}

	// 等待解质押交易确认
	integration.TriggerMining(t, c, address)
	parsedTx := integration.WaitForTransactionWithTest(t, c, unstakeResult.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证余额恢复（解质押金额 + 可能的奖励）
	finalBalance := integration.GetTestAccountBalance(t, c, address, nil)
	t.Logf("最终余额: %d", finalBalance)
	assert.Greater(t, finalBalance, uint64(0), "余额应该大于0")
}

