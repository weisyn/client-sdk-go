package token

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weisyn/client-sdk-go/services/token"
	integration "github.com/weisyn/client-sdk-go/test/integration"
)

// TestTokenGetBalance_Basic 测试余额查询功能
func TestTokenGetBalance_Basic(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建测试账户并充值
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	integration.FundTestAccount(t, c, address, 1000000)

	// 创建 Token 服务
	tokenService := token.NewService(c)

	// 查询余额
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := tokenService.GetBalance(ctx, address, nil)
	require.NoError(t, err, "查询余额失败")
	assert.Greater(t, balance, uint64(0), "余额应该大于0")

	t.Logf("账户余额: %d", balance)

	// 验证余额与直接查询一致
	directBalance := integration.GetTestAccountBalance(t, c, address, nil)
	assert.Equal(t, directBalance, balance, "余额查询结果不一致")
}

// TestTokenGetBalance_ZeroBalance 测试零余额账户
func TestTokenGetBalance_ZeroBalance(t *testing.T) {
	integration.EnsureNodeRunning(t)

	c := integration.SetupTestClient(t)
	defer integration.TeardownTestClient(t, c)

	// 创建新账户（不充值）
	wallet := integration.CreateTestWallet(t)
	address := wallet.Address()

	// 创建 Token 服务
	tokenService := token.NewService(c)

	// 查询余额
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := tokenService.GetBalance(ctx, address, nil)
	require.NoError(t, err, "查询余额失败")
	assert.Equal(t, uint64(0), balance, "新账户余额应该为0")
}

