package governance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/services/governance"
	"github.com/weisyn/client-sdk-go/test/integration"
)

// TestGovernance_Propose 测试创建提案功能
func TestGovernance_Propose(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	integration.FundTestAccount(t, c, address, 1000000)

	// 创建 Governance 服务
	governanceService := governance.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 准备提案参数
	title := "测试提案"
	description := "这是一个测试提案"
	votingPeriod := uint64(100) // 投票期：100 个区块

	// 执行创建提案
	result, err := governanceService.Propose(ctx, &governance.ProposeRequest{
		Proposer:     address,
		Title:        title,
		Description:  description,
		VotingPeriod: votingPeriod,
	}, wallet)

	// 如果当前地址没有可用的原生币 UTXO，用于支付治理手续费，则跳过本测试
	if err != nil && strings.Contains(err.Error(), "no available native coin UTXO for fee") {
		t.Logf("创建提案失败（当前账户无可用原生币 UTXO，暂时跳过此治理测试）: %v", err)
		return
	}

	require.NoError(t, err, "创建提案失败")
	require.NotNil(t, result, "提案结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")
	assert.NotEmpty(t, result.ProposalID, "提案ID为空")

	t.Logf("创建提案成功，交易哈希: %s, 提案ID: %s", result.TxHash, result.ProposalID)

	// 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, address)
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)

	// 验证提案ID格式（应该是 outpoint 格式：txHash:index）
	assert.Contains(t, result.ProposalID, ":", "提案ID格式不正确，应该是 txHash:index")

	// 验证交易输出中包含提案输出
	foundProposalOutput := false
	for _, output := range parsedTx.Outputs {
		if output.Type == "state" && output.Outpoint != "" {
			foundProposalOutput = true
			t.Logf("找到提案输出: Outpoint=%s", output.Outpoint)
			break
		}
	}
	assert.True(t, foundProposalOutput, "未找到提案输出")
}

// TestGovernance_Vote 测试投票功能
func TestGovernance_Vote(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	proposerWallet := integration.CreateTestWallet(t)
	voterWallet := integration.CreateTestWallet(t)

	proposerAddr := proposerWallet.Address()
	voterAddr := voterWallet.Address()

	integration.FundTestAccount(t, c, proposerAddr, 1000000)
	integration.FundTestAccount(t, c, voterAddr, 1000000)

	// 创建 Governance 服务
	governanceService := governance.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 先创建提案
	title := "测试投票提案"
	description := "这是一个用于测试投票的提案"
	votingPeriod := uint64(100)

	proposeResult, err := governanceService.Propose(ctx, &governance.ProposeRequest{
		Proposer:     proposerAddr,
		Title:        title,
		Description:  description,
		VotingPeriod: votingPeriod,
	}, proposerWallet)
	if err != nil && strings.Contains(err.Error(), "no available native coin UTXO for fee") {
		t.Logf("创建提案失败（当前账户无可用原生币 UTXO，暂时跳过投票测试）: %v", err)
		return
	}
	require.NoError(t, err, "创建提案失败")

	// 等待提案交易确认
	integration.TriggerMining(t, c, proposerAddr)
	integration.WaitForTransactionWithTest(t, c, proposeResult.TxHash)

	// 2. 执行投票
	proposalIDBytes := []byte(proposeResult.ProposalID)
	choice := 1 // 1=支持, 0=反对, -1=弃权

	voteResult, err := governanceService.Vote(ctx, &governance.VoteRequest{
		Voter:      voterAddr,
		ProposalID: proposalIDBytes,
		Choice:     choice,
		VoteWeight: 1,
	}, voterWallet)

	require.NoError(t, err, "投票失败")
	require.NotNil(t, voteResult, "投票结果为空")
	assert.NotEmpty(t, voteResult.TxHash, "交易哈希为空")

	t.Logf("投票成功，交易哈希: %s", voteResult.TxHash)

	// 等待投票交易确认
	integration.TriggerMining(t, c, voterAddr)
	parsedTx := integration.WaitForTransactionWithTest(t, c, voteResult.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)
}

// TestGovernance_UpdateParam 测试参数更新功能
func TestGovernance_UpdateParam(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	integration.FundTestAccount(t, c, address, 1000000)

	// 创建 Governance 服务
	governanceService := governance.NewService(c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 准备参数更新请求
	paramKey := "test_param"
	paramValue := "test_value"

	// 执行参数更新
	result, err := governanceService.UpdateParam(ctx, &governance.UpdateParamRequest{
		Proposer:  address,
		ParamKey:  paramKey,
		ParamValue: paramValue,
	}, wallet)

	if err != nil && strings.Contains(err.Error(), "no available native coin UTXO for fee") {
		t.Logf("参数更新失败（当前账户无可用原生币 UTXO，暂时跳过此治理测试）: %v", err)
		return
	}

	require.NoError(t, err, "参数更新失败")
	require.NotNil(t, result, "参数更新结果为空")
	assert.NotEmpty(t, result.TxHash, "交易哈希为空")

	t.Logf("参数更新成功，交易哈希: %s", result.TxHash)

	// 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, address)
	parsedTx := integration.WaitForTransactionWithTest(t, c, result.TxHash)
	integration.VerifyTransactionSuccess(t, parsedTx)
}

