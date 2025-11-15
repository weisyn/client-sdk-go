package token

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// transfer 单笔转账实现
//
// **架构说明**：
// Transfer 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
// 1. 调用 `buildTransferTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 支持原生币和合约代币转账
func (s *tokenService) transfer(ctx context.Context, req *TransferRequest, wallets ...wallet.Wallet) (*TransferResult, error) {
	// 1. 参数验证
	if err := s.validateTransferRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.From) {
		return nil, fmt.Errorf("wallet address does not match from address")
	}

	// 4. 在 SDK 层构建未签名交易
	unsignedTxBytes, err := buildTransferTransaction(ctx, s.client, req.From, req.To, req.Amount, req.TokenID)
	if err != nil {
		return nil, fmt.Errorf("build transfer transaction failed: %w", err)
	}

	// 5. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 6. 调用 wes_sendRawTransaction 提交已签名交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 7. 返回结果
	return &TransferResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateTransferRequest 验证转账请求
func (s *tokenService) validateTransferRequest(req *TransferRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}
	if len(req.To) != 20 {
		return fmt.Errorf("to address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	return nil
}

// batchTransfer 批量转账实现
//
// **架构说明**：
// BatchTransfer 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
// 1. 调用 `buildBatchTransferTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 批量转账需要按 tokenID 分组 UTXO，为每个转账选择足够的 UTXO
func (s *tokenService) batchTransfer(ctx context.Context, req *BatchTransferRequest, wallets ...wallet.Wallet) (*BatchTransferResult, error) {
	// 1. 参数验证
	if err := s.validateBatchTransferRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.From) {
		return nil, fmt.Errorf("wallet address does not match from address")
	}

	// 4. 在 SDK 层构建未签名交易
	unsignedTxBytes, err := buildBatchTransferTransaction(ctx, s.client, req.From, req.Transfers)
	if err != nil {
		return nil, fmt.Errorf("build batch transfer transaction failed: %w", err)
	}

	// 5. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 6. 调用 wes_sendRawTransaction 提交已签名交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 7. 返回结果
	return &BatchTransferResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateBatchTransferRequest 验证批量转账请求
func (s *tokenService) validateBatchTransferRequest(req *BatchTransferRequest) error {
	// 1. 验证转账列表
	if len(req.Transfers) == 0 {
		return fmt.Errorf("transfers list cannot be empty")
	}

	// 2. 验证每个转账项
	for i, transfer := range req.Transfers {
		if len(transfer.To) != 20 {
			return fmt.Errorf("transfer %d: to address must be 20 bytes", i)
		}
		if transfer.Amount == 0 {
			return fmt.Errorf("transfer %d: amount must be greater than 0", i)
		}
		// TokenID可选，但如果提供必须是32字节
		if transfer.TokenID != nil && len(transfer.TokenID) != 32 {
			return fmt.Errorf("transfer %d: tokenID must be 32 bytes if provided", i)
		}
	}

	return nil
}

