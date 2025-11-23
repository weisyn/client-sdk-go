package staking

import (
	"context"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Staking 业务服务接口
type Service interface {
	// Stake 质押代币
	Stake(ctx context.Context, req *StakeRequest, wallet ...wallet.Wallet) (*StakeResult, error)

	// Unstake 解除质押
	Unstake(ctx context.Context, req *UnstakeRequest, wallet ...wallet.Wallet) (*UnstakeResult, error)

	// Delegate 委托验证
	Delegate(ctx context.Context, req *DelegateRequest, wallet ...wallet.Wallet) (*DelegateResult, error)

	// Undelegate 取消委托
	Undelegate(ctx context.Context, req *UndelegateRequest, wallet ...wallet.Wallet) (*UndelegateResult, error)

	// ClaimReward 领取奖励
	ClaimReward(ctx context.Context, req *ClaimRewardRequest, wallet ...wallet.Wallet) (*ClaimRewardResult, error)

	// Slash 罚没（治理功能）
	Slash(ctx context.Context, req *SlashRequest, wallet ...wallet.Wallet) (*SlashResult, error)
}

// stakingService Staking 服务实现
type stakingService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Staking 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &stakingService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Staking 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &stakingService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *stakingService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// Stake 质押代币（实现在stake.go）
func (s *stakingService) Stake(ctx context.Context, req *StakeRequest, wallets ...wallet.Wallet) (*StakeResult, error) {
	return s.stake(ctx, req, wallets...)
}

// Unstake 解除质押（实现在stake.go）
func (s *stakingService) Unstake(ctx context.Context, req *UnstakeRequest, wallets ...wallet.Wallet) (*UnstakeResult, error) {
	return s.unstake(ctx, req, wallets...)
}

// Delegate 委托验证（实现在delegate.go）
func (s *stakingService) Delegate(ctx context.Context, req *DelegateRequest, wallets ...wallet.Wallet) (*DelegateResult, error) {
	return s.delegate(ctx, req, wallets...)
}

// Undelegate 取消委托（实现在delegate.go）
func (s *stakingService) Undelegate(ctx context.Context, req *UndelegateRequest, wallets ...wallet.Wallet) (*UndelegateResult, error) {
	return s.undelegate(ctx, req, wallets...)
}

// ClaimReward 领取奖励（实现在delegate.go）
func (s *stakingService) ClaimReward(ctx context.Context, req *ClaimRewardRequest, wallets ...wallet.Wallet) (*ClaimRewardResult, error) {
	return s.claimReward(ctx, req, wallets...)
}

// Slash 罚没（实现在slash.go）
func (s *stakingService) Slash(ctx context.Context, req *SlashRequest, wallets ...wallet.Wallet) (*SlashResult, error) {
	return s.slash(ctx, req, wallets...)
}

// StakeRequest 质押请求
type StakeRequest struct {
	From         []byte // 质押者地址（20字节）
	ValidatorAddr []byte // 验证者地址（20字节）
	Amount       uint64 // 质押金额
	LockBlocks   uint64 // 锁定期（区块数）
}

// StakeResult 质押结果
type StakeResult struct {
	StakeID string // 质押ID
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// UnstakeRequest 解除质押请求
type UnstakeRequest struct {
	From    []byte // 质押者地址（20字节）
	StakeID []byte // 质押ID
	Amount  uint64 // 解除质押金额（0表示全部）
}

// UnstakeResult 解除质押结果
type UnstakeResult struct {
	TxHash      string // 交易哈希
	UnstakeAmount uint64 // 解除质押金额
	RewardAmount  uint64 // 奖励金额
	Success     bool   // 是否成功
}

// DelegateRequest 委托请求
type DelegateRequest struct {
	From         []byte // 委托者地址（20字节）
	ValidatorAddr []byte // 验证者地址（20字节）
	Amount       uint64 // 委托金额
}

// DelegateResult 委托结果
type DelegateResult struct {
	DelegateID string // 委托ID
	TxHash     string // 交易哈希
	Success    bool   // 是否成功
}

// UndelegateRequest 取消委托请求
type UndelegateRequest struct {
	From       []byte // 委托者地址（20字节）
	DelegateID []byte // 委托ID
	Amount     uint64 // 取消委托金额（0表示全部）
}

// UndelegateResult 取消委托结果
type UndelegateResult struct {
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// ClaimRewardRequest 领取奖励请求
type ClaimRewardRequest struct {
	From       []byte // 领取者地址（20字节）
	StakeID    []byte // 质押ID（可选）
	DelegateID []byte // 委托ID（可选）
}

// ClaimRewardResult 领取奖励结果
type ClaimRewardResult struct {
	TxHash      string // 交易哈希
	RewardAmount uint64 // 奖励金额
	Success     bool   // 是否成功
}

// SlashRequest 罚没请求
type SlashRequest struct {
	ValidatorAddr []byte // 被罚没的验证者地址
	Amount       uint64 // 罚没金额
	Reason       string // 罚没原因
}

// SlashResult 罚没结果
type SlashResult struct {
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

