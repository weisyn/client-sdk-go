package governance

import (
	"context"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Governance 业务服务接口
type Service interface {
	// Propose 创建提案
	Propose(ctx context.Context, req *ProposeRequest, wallet ...wallet.Wallet) (*ProposeResult, error)

	// Vote 投票
	Vote(ctx context.Context, req *VoteRequest, wallet ...wallet.Wallet) (*VoteResult, error)

	// UpdateParam 更新参数
	UpdateParam(ctx context.Context, req *UpdateParamRequest, wallet ...wallet.Wallet) (*UpdateParamResult, error)
}

// governanceService Governance 服务实现
type governanceService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Governance 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &governanceService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Governance 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &governanceService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *governanceService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// ProposeRequest 提案请求
type ProposeRequest struct {
	Proposer     []byte // 提案者地址（20字节）
	Title        string // 提案标题
	Description  string // 提案描述
	VotingPeriod uint64 // 投票期限（区块数）
}

// ProposeResult 提案结果
type ProposeResult struct {
	ProposalID string // 提案ID
	TxHash     string // 交易哈希
	Success    bool   // 是否成功
}

// VoteRequest 投票请求
type VoteRequest struct {
	Voter      []byte // 投票者地址（20字节）
	ProposalID []byte // 提案ID
	Choice     int    // 投票选择（1=支持, 0=反对, -1=弃权）
	VoteWeight uint64 // 投票权重
}

// VoteResult 投票结果
type VoteResult struct {
	VoteID  string // 投票ID
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// UpdateParamRequest 更新参数请求
type UpdateParamRequest struct {
	Proposer   []byte // 提案者地址（20字节）
	ParamKey   string // 参数键
	ParamValue string // 参数值
}

// UpdateParamResult 更新参数结果
type UpdateParamResult struct {
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// Propose 创建提案（实现在propose.go）
func (s *governanceService) Propose(ctx context.Context, req *ProposeRequest, wallets ...wallet.Wallet) (*ProposeResult, error) {
	return s.propose(ctx, req, wallets...)
}

// Vote 投票（实现在vote.go）
func (s *governanceService) Vote(ctx context.Context, req *VoteRequest, wallets ...wallet.Wallet) (*VoteResult, error) {
	return s.vote(ctx, req, wallets...)
}

// UpdateParam 更新参数（实现在vote.go）
func (s *governanceService) UpdateParam(ctx context.Context, req *UpdateParamRequest, wallets ...wallet.Wallet) (*UpdateParamResult, error) {
	return s.updateParam(ctx, req, wallets...)
}
