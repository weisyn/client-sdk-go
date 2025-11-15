package token

import (
	"context"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Token 业务服务接口
type Service interface {
	// Transfer 单笔转账
	// wallet 参数可选：如果提供则使用，否则使用服务实例的默认 Wallet
	Transfer(ctx context.Context, req *TransferRequest, wallet ...wallet.Wallet) (*TransferResult, error)

	// BatchTransfer 批量转账
	BatchTransfer(ctx context.Context, req *BatchTransferRequest, wallet ...wallet.Wallet) (*BatchTransferResult, error)

	// Mint 代币铸造
	Mint(ctx context.Context, req *MintRequest, wallet ...wallet.Wallet) (*MintResult, error)

	// Burn 代币销毁
	Burn(ctx context.Context, req *BurnRequest, wallet ...wallet.Wallet) (*BurnResult, error)

	// GetBalance 查询余额（不需要 Wallet）
	GetBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error)
}

// tokenService Token 服务实现
type tokenService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Token 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &tokenService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Token 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &tokenService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *tokenService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// TransferRequest 转账请求
type TransferRequest struct {
	From    []byte // 发送方地址（20字节）
	To      []byte // 接收方地址（20字节）
	Amount  uint64 // 转账金额
	TokenID []byte // 代币ID（32字节，nil 表示原生币）
}

// TransferResult 转账结果
type TransferResult struct {
	TxHash  string
	Success bool
}

// Transfer 单笔转账（实现在transfer.go）
func (s *tokenService) Transfer(ctx context.Context, req *TransferRequest, wallets ...wallet.Wallet) (*TransferResult, error) {
	return s.transfer(ctx, req, wallets...)
}

// transfer 单笔转账实现（在transfer.go中）
// 实际实现在transfer.go文件中

// BatchTransferRequest 批量转账请求
type BatchTransferRequest struct {
	Transfers []TransferItem // 转账列表
	From      []byte         // 发送方地址（20字节，所有转账的发送方）
}

// TransferItem 转账项
type TransferItem struct {
	To      []byte // 接收方地址（20字节）
	Amount  uint64 // 转账金额
	TokenID []byte // 代币ID（32字节，可选，nil表示原生币）
}

// BatchTransferResult 批量转账结果
type BatchTransferResult struct {
	TxHash  string
	Success bool
}

// BatchTransfer 批量转账（实现在transfer.go）
func (s *tokenService) BatchTransfer(ctx context.Context, req *BatchTransferRequest, wallets ...wallet.Wallet) (*BatchTransferResult, error) {
	return s.batchTransfer(ctx, req, wallets...)
}

// batchTransfer 批量转账实现（在transfer.go中）
// 实际实现在transfer.go文件中

// MintRequest 铸造请求
type MintRequest struct {
	To           []byte // 接收者地址（20字节）
	Amount       uint64 // 铸造数量
	TokenID      []byte // 代币ID（32字节，可选，nil表示原生币）
	ContractAddr []byte // 发行合约地址（20字节，用于权限校验，可选）
}

// MintResult 铸造结果
type MintResult struct {
	TxHash  string
	Success bool
}

// Mint 代币铸造（实现在mint.go）
func (s *tokenService) Mint(ctx context.Context, req *MintRequest, wallets ...wallet.Wallet) (*MintResult, error) {
	return s.mint(ctx, req, wallets...)
}

// mint 代币铸造实现（在mint.go中）
// 实际实现在mint.go文件中

// BurnRequest 销毁请求
type BurnRequest struct {
	From      []byte // 销毁者地址（20字节）
	Amount    uint64 // 销毁数量
	TokenID   []byte // 代币ID（32字节，必需）
	BurnProof []byte // 销毁证明（可选）
}

// BurnResult 销毁结果
type BurnResult struct {
	TxHash  string
	Success bool
}

// Burn 代币销毁（实现在mint.go）
func (s *tokenService) Burn(ctx context.Context, req *BurnRequest, wallets ...wallet.Wallet) (*BurnResult, error) {
	return s.burn(ctx, req, wallets...)
}

// burn 代币销毁实现（在mint.go中）
// 实际实现在mint.go文件中

// GetBalance 查询余额（实现在balance.go）
func (s *tokenService) GetBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error) {
	return s.getBalance(ctx, address, tokenID)
}

// getBalance 查询余额实现（在balance.go中）
// 实际实现在balance.go文件中
