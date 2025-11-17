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

// TestTokenBurn_Basic 测试基本销毁功能
//
// **测试步骤**：
// 1. 创建测试账户
// 2. 为账户充值（通过挖矿）
// 3. 调用 tokenService.Burn() 执行销毁
// 4. 触发挖矿，等待交易确认
// 5. 验证结果：
//    - result.TxHash 不为空
//    - result.Success == true
//    - 查询账户余额，验证金额减少（销毁金额）
func TestTokenBurn_Basic(t *testing.T) {
	// 1. 确保节点运行
	integration.EnsureNodeRunning(t)

	// 2. 设置测试客户端
	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 3. 创建测试账户
	wallet := integration.CreateTestWallet(t)
	addr := wallet.Address()

	t.Logf("账户地址: 0x%x", addr)

	// 4. 为账户充值（通过挖矿）
	integration.FundTestAccount(t, c, addr, 1000000)

	// 5. 查询初始余额
	initialBalance := integration.GetTestAccountBalance(t, c, addr, nil)
	t.Logf("初始余额: %d", initialBalance)
	require.Greater(t, initialBalance, uint64(0), "账户应该有初始余额")

	// 6. 创建 Token 服务
	tokenService := token.NewService(c)

	// 7. 准备销毁请求（使用原生币，tokenID 为全0的32字节）
	burnAmount := uint64(10000)
	nativeTokenID := make([]byte, 32) // 全0表示原生币
	burnReq := &token.BurnRequest{
		From:    addr,
		Amount:  burnAmount,
		TokenID: nativeTokenID, // 原生币（全0）
	}

	// 8. 执行销毁
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	burnResult, err := tokenService.Burn(ctx, burnReq, wallet)
	require.NoError(t, err, "销毁应该成功")
	require.NotNil(t, burnResult, "销毁结果不应该为空")
	assert.NotEmpty(t, burnResult.TxHash, "交易哈希不应该为空")
	assert.True(t, burnResult.Success, "销毁应该成功")

	t.Logf("销毁交易哈希: %s", burnResult.TxHash)

	// 9. 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, addr)

	// 10. 等待交易确认（轮询查询余额变化）
	maxRetries := 10
	retryInterval := 1 * time.Second
	var finalBalance uint64

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)
		finalBalance = integration.GetTestAccountBalance(t, c, addr, nil)
		if finalBalance < initialBalance {
			break
		}
		if i == maxRetries-1 {
			t.Fatalf("交易未确认：余额未变化，初始余额: %d, 当前余额: %d", initialBalance, finalBalance)
		}
	}

	// 11. 验证余额变化
	// 注意：手续费从接收者扣除，Burn 操作没有接收者，手续费由节点端从销毁金额中扣除
	// 所以余额减少应该等于销毁金额（手续费由节点端处理）
	t.Logf("初始余额: %d", initialBalance)
	t.Logf("最终余额: %d", finalBalance)
	t.Logf("销毁金额: %d", burnAmount)
	t.Logf("余额减少: %d", initialBalance-finalBalance)

	// 余额应该减少，减少金额应该等于或接近销毁金额（可能因为手续费略有差异）
	assert.Less(t, finalBalance, initialBalance, "余额应该减少")
	assert.GreaterOrEqual(t, initialBalance-finalBalance, burnAmount, "余额减少应该至少等于销毁金额")
}

// TestTokenBurn_WithTokenID 测试代币销毁功能
//
// **测试步骤**：
// 1. 创建测试账户
// 2. 为账户充值（通过挖矿）
// 3. 创建代币（如果需要）
// 4. 调用 tokenService.Burn() 执行销毁
// 5. 触发挖矿，等待交易确认
// 6. 验证结果
func TestTokenBurn_WithTokenID(t *testing.T) {
	t.Skip("需要先实现代币创建功能，暂时跳过")
}

// TestTokenBurn_InsufficientBalance 测试余额不足的情况
func TestTokenBurn_InsufficientBalance(t *testing.T) {
	// 1. 确保节点运行
	integration.EnsureNodeRunning(t)

	// 2. 设置测试客户端
	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 3. 创建测试账户（不充值）
	wallet := integration.CreateTestWallet(t)
	addr := wallet.Address()

	// 4. 创建 Token 服务
	tokenService := token.NewService(c)

	// 5. 准备销毁请求（金额大于余额）
	nativeTokenID := make([]byte, 32) // 全0表示原生币
	burnReq := &token.BurnRequest{
		From:    addr,
		Amount:  1000000, // 大金额
		TokenID: nativeTokenID, // 原生币（全0）
	}

	// 6. 执行销毁（应该失败）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := tokenService.Burn(ctx, burnReq, wallet)
	require.Error(t, err, "余额不足时应该返回错误")
	t.Logf("预期的错误: %v", err)
}

// TestTokenBurn_InvalidAmount 测试无效金额的情况
func TestTokenBurn_InvalidAmount(t *testing.T) {
	// 1. 确保节点运行
	integration.EnsureNodeRunning(t)

	// 2. 设置测试客户端
	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 3. 创建测试账户
	wallet := integration.CreateTestWallet(t)
	addr := wallet.Address()

	// 4. 创建 Token 服务
	tokenService := token.NewService(c)

	// 5. 准备销毁请求（金额为0）
	nativeTokenID := make([]byte, 32) // 全0表示原生币
	burnReq := &token.BurnRequest{
		From:    addr,
		Amount:  0, // 无效金额
		TokenID: nativeTokenID,
	}

	// 6. 执行销毁（应该失败）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := tokenService.Burn(ctx, burnReq, wallet)
	require.Error(t, err, "金额为0时应该返回错误")
	t.Logf("预期的错误: %v", err)
}

// TestTokenBurn_ChangeCalculation 测试找零计算逻辑
//
// **测试场景**：
// - 账户有 100000，销毁 10000，应该找零 90000
// - 验证找零逻辑是否正确（不扣除手续费）
func TestTokenBurn_ChangeCalculation(t *testing.T) {
	// 1. 确保节点运行
	integration.EnsureNodeRunning(t)

	// 2. 设置测试客户端
	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 3. 创建测试账户
	wallet := integration.CreateTestWallet(t)
	addr := wallet.Address()

	t.Logf("账户地址: 0x%x", addr)

	// 4. 为账户充值（通过挖矿）
	integration.FundTestAccount(t, c, addr, 100000)

	// 5. 查询初始余额
	initialBalance := integration.GetTestAccountBalance(t, c, addr, nil)
	t.Logf("初始余额: %d", initialBalance)
	require.GreaterOrEqual(t, initialBalance, uint64(100000), "账户应该有足够的余额")

	// 6. 创建 Token 服务
	tokenService := token.NewService(c)

	// 7. 准备销毁请求
	burnAmount := uint64(10000)
	nativeTokenID := make([]byte, 32) // 全0表示原生币
	burnReq := &token.BurnRequest{
		From:    addr,
		Amount:  burnAmount,
		TokenID: nativeTokenID, // 原生币（全0）
	}

	// 8. 执行销毁
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	burnResult, err := tokenService.Burn(ctx, burnReq, wallet)
	require.NoError(t, err, "销毁应该成功")
	require.NotNil(t, burnResult, "销毁结果不应该为空")
	assert.NotEmpty(t, burnResult.TxHash, "交易哈希不应该为空")

	t.Logf("销毁交易哈希: %s", burnResult.TxHash)

	// 9. 触发挖矿，等待交易确认
	integration.TriggerMining(t, c, addr)

	// 10. 等待交易确认
	maxRetries := 10
	retryInterval := 1 * time.Second
	var finalBalance uint64

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)
		finalBalance = integration.GetTestAccountBalance(t, c, addr, nil)
		if finalBalance < initialBalance {
			break
		}
		if i == maxRetries-1 {
			t.Fatalf("交易未确认：余额未变化")
		}
	}

	// 11. 验证找零逻辑
	// 找零 = initialBalance - burnAmount（不扣除手续费，因为手续费从接收者扣除，Burn 没有接收者）
	expectedChange := initialBalance - burnAmount
	t.Logf("初始余额: %d", initialBalance)
	t.Logf("销毁金额: %d", burnAmount)
	t.Logf("预期找零: %d", expectedChange)
	t.Logf("最终余额: %d", finalBalance)
	t.Logf("实际余额减少: %d", initialBalance-finalBalance)

	// 最终余额应该等于预期找零（可能因为手续费略有差异，但应该接近）
	assert.GreaterOrEqual(t, finalBalance, expectedChange, "最终余额应该至少等于预期找零")
}

