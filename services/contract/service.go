package contract

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Contract 业务服务接口
type Service interface {
	// CallContract 调用合约方法（写操作）
	CallContract(ctx context.Context, req *CallContractRequest, wallets ...wallet.Wallet) (*CallContractResult, error)

	// QueryContract 查询合约方法（只读操作）
	QueryContract(ctx context.Context, req *QueryContractRequest) (interface{}, error)
}

// contractService Contract 服务实现
type contractService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建 Contract 服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &contractService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的 Contract 服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &contractService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *contractService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// CallContractRequest 合约调用请求
type CallContractRequest struct {
	ContractAddress []byte        // 合约地址（contentHash，32字节）
	Method          string        // 方法名
	Args            []interface{} // 方法参数
	From            []byte        // 调用者地址（20字节）
	Amount          *uint64       // 可选：金额（如果需要转账）
	TokenID         []byte        // 可选：代币 ID（如果需要转账代币）
}

// CallContractResult 合约调用结果
type CallContractResult struct {
	TxHash      string  // 交易哈希
	Success     bool    // 是否成功
	BlockHeight *uint64 // 区块高度（如果已确认）
}

// QueryContractRequest 合约查询请求（只读）
type QueryContractRequest struct {
	ContractAddress []byte        // 合约地址（contentHash，32字节）
	Method          string        // 方法名
	Args            []interface{} // 方法参数
}

// CallContract 调用合约方法
func (s *contractService) CallContract(ctx context.Context, req *CallContractRequest, wallets ...wallet.Wallet) (*CallContractResult, error) {
	// 1. 参数验证
	if len(req.ContractAddress) != 32 {
		return nil, fmt.Errorf("contract address must be 32 bytes")
	}
	if len(req.From) != 20 {
		return nil, fmt.Errorf("from address must be 20 bytes")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("method name is required")
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required for contract invocation")
	}

	// 3. 构建 payload（使用统一的 ABI helper）
	payloadBase64, err := buildCallPayload(req)
	if err != nil {
		return nil, fmt.Errorf("build payload failed: %w", err)
	}

	// 4. 调用 wes_callContract，设置 return_unsigned_tx=true
	contractAddressHex := "0x" + hex.EncodeToString(req.ContractAddress)
	callParams := map[string]interface{}{
		"content_hash":       contractAddressHex,
		"method":             req.Method,
		"params":             req.Args,
		"payload":            payloadBase64,
		"return_unsigned_tx": true,
	}

	callResult, err := s.client.Call(ctx, "wes_callContract", []interface{}{callParams})
	if err != nil {
		return nil, fmt.Errorf("call contract failed: %w", err)
	}

	callMap, ok := callResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid call contract response format")
	}

	// 5. 获取未签名交易
	unsignedTxHex, ok := callMap["unsigned_tx"].(string)
	if !ok {
		return nil, fmt.Errorf("missing unsigned_tx in call contract response")
	}

	// 6. 计算签名哈希（简化：直接签名交易）
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsigned tx failed: %w", err)
	}

	// 7. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 8. 提交交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	return &CallContractResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// QueryContract 查询合约方法（只读）
func (s *contractService) QueryContract(ctx context.Context, req *QueryContractRequest) (interface{}, error) {
	// 1. 参数验证
	if len(req.ContractAddress) != 32 {
		return nil, fmt.Errorf("contract address must be 32 bytes")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("method name is required")
	}

	// 2. 构建 payload（使用统一的 ABI helper）
	payloadBase64, err := buildQueryPayload(req)
	if err != nil {
		return nil, fmt.Errorf("build payload failed: %w", err)
	}

	// 3. 调用 wes_callContract，不设置 return_unsigned_tx（只读查询）
	contractAddressHex := "0x" + hex.EncodeToString(req.ContractAddress)
	callParams := map[string]interface{}{
		"content_hash": contractAddressHex,
		"method":       req.Method,
		"params":       req.Args,
		"payload":      payloadBase64,
	}

	callResult, err := s.client.Call(ctx, "wes_callContract", []interface{}{callParams})
	if err != nil {
		return nil, fmt.Errorf("query contract failed: %w", err)
	}

	// 4. 返回查询结果
	callMap, ok := callResult.(map[string]interface{})
	if !ok {
		return callResult, nil
	}

	// 尝试提取 result 字段
	if result, ok := callMap["result"]; ok {
		return result, nil
	}

	return callResult, nil
}
