package resource

import (
	"context"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Resource 业务服务接口
type Service interface {
	// DeployStaticResource 部署静态资源
	DeployStaticResource(ctx context.Context, req *DeployStaticResourceRequest, wallet ...wallet.Wallet) (*DeployStaticResourceResult, error)

	// DeployContract 部署智能合约
	DeployContract(ctx context.Context, req *DeployContractRequest, wallet ...wallet.Wallet) (*DeployContractResult, error)

	// DeployAIModel 部署AI模型
	DeployAIModel(ctx context.Context, req *DeployAIModelRequest, wallet ...wallet.Wallet) (*DeployAIModelResult, error)

	// GetResource 获取资源信息（不需要 Wallet）
	GetResource(ctx context.Context, contentHash []byte) (*ResourceInfo, error)
}

// resourceService Resource 服务实现
type resourceService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Resource 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &resourceService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Resource 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &resourceService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *resourceService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// DeployStaticResourceRequest 部署静态资源请求
type DeployStaticResourceRequest struct {
	From     []byte // 部署者地址（20字节）
	FilePath string // 文件路径
	MimeType string // MIME类型
}

// DeployStaticResourceResult 部署静态资源结果
type DeployStaticResourceResult struct {
	ContentHash []byte // 内容哈希
	TxHash      string // 交易哈希
	Success     bool   // 是否成功
}

// DeployContractRequest 部署合约请求
type DeployContractRequest struct {
	From         []byte // 部署者地址（20字节）
	WasmPath     string // WASM文件路径
	ContractName string // 合约名称
	InitArgs     []byte // 初始化参数
}

// DeployContractResult 部署合约结果
type DeployContractResult struct {
	ContractAddress []byte // 合约地址
	ContentHash     []byte // 内容哈希
	TxHash          string // 交易哈希
	Success         bool   // 是否成功
}

// DeployAIModelRequest 部署AI模型请求
type DeployAIModelRequest struct {
	From      []byte // 部署者地址（20字节）
	ModelPath string // 模型文件路径
	ModelName string // 模型名称
}

// DeployAIModelResult 部署AI模型结果
type DeployAIModelResult struct {
	ContentHash []byte // 内容哈希
	TxHash      string // 交易哈希
	Success     bool   // 是否成功
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	ContentHash string // 内容哈希
	Type        string // 资源类型（static/contract/aimodel）
	Size        int64  // 文件大小
	MimeType    string // MIME类型
	Owner       []byte // 所有者地址
}

// DeployStaticResource 部署静态资源（实现在deploy.go）
func (s *resourceService) DeployStaticResource(ctx context.Context, req *DeployStaticResourceRequest, wallets ...wallet.Wallet) (*DeployStaticResourceResult, error) {
	return s.deployStaticResource(ctx, req, wallets...)
}

// DeployContract 部署智能合约（实现在deploy.go）
func (s *resourceService) DeployContract(ctx context.Context, req *DeployContractRequest, wallets ...wallet.Wallet) (*DeployContractResult, error) {
	return s.deployContract(ctx, req, wallets...)
}

// DeployAIModel 部署AI模型（实现在deploy.go）
func (s *resourceService) DeployAIModel(ctx context.Context, req *DeployAIModelRequest, wallets ...wallet.Wallet) (*DeployAIModelResult, error) {
	return s.deployAIModel(ctx, req, wallets...)
}

// GetResource 获取资源信息（实现在query.go）
func (s *resourceService) GetResource(ctx context.Context, contentHash []byte) (*ResourceInfo, error) {
	return s.getResource(ctx, contentHash)
}
