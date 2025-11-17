package market

import (
	"context"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Market 业务服务接口
type Service interface {
	// SwapAMM AMM代币交换
	SwapAMM(ctx context.Context, req *SwapRequest, wallet ...wallet.Wallet) (*SwapResult, error)

	// AddLiquidity 添加流动性
	AddLiquidity(ctx context.Context, req *AddLiquidityRequest, wallet ...wallet.Wallet) (*AddLiquidityResult, error)

	// RemoveLiquidity 移除流动性
	RemoveLiquidity(ctx context.Context, req *RemoveLiquidityRequest, wallet ...wallet.Wallet) (*RemoveLiquidityResult, error)

	// CreateVesting 创建归属计划
	CreateVesting(ctx context.Context, req *CreateVestingRequest, wallet ...wallet.Wallet) (*CreateVestingResult, error)

	// ClaimVesting 领取归属代币
	ClaimVesting(ctx context.Context, req *ClaimVestingRequest, wallet ...wallet.Wallet) (*ClaimVestingResult, error)

	// CreateEscrow 创建托管
	CreateEscrow(ctx context.Context, req *CreateEscrowRequest, wallet ...wallet.Wallet) (*CreateEscrowResult, error)

	// ReleaseEscrow 释放托管给卖方
	ReleaseEscrow(ctx context.Context, req *ReleaseEscrowRequest, wallet ...wallet.Wallet) (*ReleaseEscrowResult, error)

	// RefundEscrow 退款托管给买方
	RefundEscrow(ctx context.Context, req *RefundEscrowRequest, wallet ...wallet.Wallet) (*RefundEscrowResult, error)
}

// marketService Market 服务实现
type marketService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Market 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &marketService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Market 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &marketService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *marketService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// SwapRequest AMM交换请求
type SwapRequest struct {
	From           []byte // 交换者地址（20字节）
	AMMContractAddr []byte // AMM 合约地址（contentHash，32字节）
	TokenIn        []byte // 输入代币ID（nil表示原生币）
	TokenOut       []byte // 输出代币ID（nil表示原生币）
	AmountIn       uint64 // 输入金额
	AmountOutMin   uint64 // 最小输出金额（滑点保护）
}

// SwapResult AMM交换结果
type SwapResult struct {
	TxHash    string // 交易哈希
	AmountOut uint64 // 实际输出金额
	Success   bool   // 是否成功
}

// AddLiquidityRequest 添加流动性请求
type AddLiquidityRequest struct {
	From           []byte // 流动性提供者地址（20字节）
	AMMContractAddr []byte // AMM 合约地址（contentHash，32字节）
	TokenA         []byte // 代币A ID
	TokenB         []byte // 代币B ID
	AmountA        uint64 // 代币A金额
	AmountB         uint64 // 代币B金额
}

// AddLiquidityResult 添加流动性结果
type AddLiquidityResult struct {
	TxHash      string // 交易哈希
	LiquidityID []byte // 流动性ID
	Success     bool   // 是否成功
}

// RemoveLiquidityRequest 移除流动性请求
type RemoveLiquidityRequest struct {
	From           []byte // 流动性提供者地址（20字节）
	AMMContractAddr []byte // AMM 合约地址（contentHash，32字节）
	LiquidityID    []byte // 流动性ID
	Amount         uint64 // 移除金额
}

// RemoveLiquidityResult 移除流动性结果
type RemoveLiquidityResult struct {
	TxHash    string // 交易哈希
	AmountA   uint64 // 获得的代币A金额
	AmountB   uint64 // 获得的代币B金额
	Success   bool   // 是否成功
}

// CreateVestingRequest 创建归属计划请求
type CreateVestingRequest struct {
	From     []byte // 创建者地址（20字节）
	To       []byte // 受益人地址（20字节）
	TokenID  []byte // 代币ID
	Amount   uint64 // 总金额
	StartTime uint64 // 开始时间（Unix时间戳）
	Duration uint64 // 持续时间（秒）
}

// CreateVestingResult 创建归属计划结果
type CreateVestingResult struct {
	TxHash     string // 交易哈希
	VestingID  []byte // 归属计划ID
	Success    bool   // 是否成功
}

// ClaimVestingRequest 领取归属代币请求
type ClaimVestingRequest struct {
	From      []byte // 领取者地址（20字节）
	VestingID []byte // 归属计划ID
}

// ClaimVestingResult 领取归属代币结果
type ClaimVestingResult struct {
	TxHash      string // 交易哈希
	ClaimAmount uint64 // 领取金额
	Success     bool   // 是否成功
}

// CreateEscrowRequest 创建托管请求
type CreateEscrowRequest struct {
	Buyer   []byte // 买方地址（20字节）
	Seller  []byte // 卖方地址（20字节）
	TokenID []byte // 代币ID
	Amount  uint64 // 托管金额
	Expiry  uint64 // 过期时间（Unix时间戳）
}

// CreateEscrowResult 创建托管结果
type CreateEscrowResult struct {
	TxHash    string // 交易哈希
	EscrowID  []byte // 托管ID
	Success   bool   // 是否成功
}

// ReleaseEscrowRequest 释放托管请求
type ReleaseEscrowRequest struct {
	From         []byte // 释放者地址（通常是买方，20字节）
	SellerAddress []byte // 卖方地址（20字节）
	EscrowID     []byte // 托管ID
}

// ReleaseEscrowResult 释放托管结果
type ReleaseEscrowResult struct {
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// RefundEscrowRequest 退款托管请求
type RefundEscrowRequest struct {
	From         []byte // 退款者地址（通常是买方或卖方，20字节）
	BuyerAddress []byte // 买方地址（20字节）
	EscrowID     []byte // 托管ID
}

// RefundEscrowResult 退款托管结果
type RefundEscrowResult struct {
	TxHash  string // 交易哈希
	Success bool   // 是否成功
}

// SwapAMM AMM代币交换（实现在swap.go）
func (s *marketService) SwapAMM(ctx context.Context, req *SwapRequest, wallets ...wallet.Wallet) (*SwapResult, error) {
	return s.swapAMM(ctx, req, wallets...)
}

// AddLiquidity 添加流动性（实现在liquidity.go）
func (s *marketService) AddLiquidity(ctx context.Context, req *AddLiquidityRequest, wallets ...wallet.Wallet) (*AddLiquidityResult, error) {
	return s.addLiquidity(ctx, req, wallets...)
}

// RemoveLiquidity 移除流动性（实现在liquidity.go）
func (s *marketService) RemoveLiquidity(ctx context.Context, req *RemoveLiquidityRequest, wallets ...wallet.Wallet) (*RemoveLiquidityResult, error) {
	return s.removeLiquidity(ctx, req, wallets...)
}

// CreateVesting 创建归属计划（实现在vesting.go）
func (s *marketService) CreateVesting(ctx context.Context, req *CreateVestingRequest, wallets ...wallet.Wallet) (*CreateVestingResult, error) {
	return s.createVesting(ctx, req, wallets...)
}

// ClaimVesting 领取归属代币（实现在vesting.go）
func (s *marketService) ClaimVesting(ctx context.Context, req *ClaimVestingRequest, wallets ...wallet.Wallet) (*ClaimVestingResult, error) {
	return s.claimVesting(ctx, req, wallets...)
}

// CreateEscrow 创建托管（实现在escrow.go）
func (s *marketService) CreateEscrow(ctx context.Context, req *CreateEscrowRequest, wallets ...wallet.Wallet) (*CreateEscrowResult, error) {
	return s.createEscrow(ctx, req, wallets...)
}

// ReleaseEscrow 释放托管（实现在escrow.go）
func (s *marketService) ReleaseEscrow(ctx context.Context, req *ReleaseEscrowRequest, wallets ...wallet.Wallet) (*ReleaseEscrowResult, error) {
	return s.releaseEscrow(ctx, req, wallets...)
}

// RefundEscrow 退款托管（实现在escrow.go）
func (s *marketService) RefundEscrow(ctx context.Context, req *RefundEscrowRequest, wallets ...wallet.Wallet) (*RefundEscrowResult, error) {
	return s.refundEscrow(ctx, req, wallets...)
}

