package token

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

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

	// 4. 在 SDK 层构建 DraftJSON（不直接构建交易）
	draftJSON, inputIndex, err := buildTransferDraft(ctx, s.client, req.From, req.To, req.Amount, req.TokenID)
	if err != nil {
		return nil, fmt.Errorf("build transfer draft failed: %w", err)
	}

	// 5. 调用 wes_computeSignatureHashFromDraft 获取签名哈希
	hashParams := map[string]interface{}{
		"draft":        json.RawMessage(draftJSON),
		"input_index":  inputIndex,
		"sighash_type": "SIGHASH_ALL",
	}
	hashResult, err := s.client.Call(ctx, "wes_computeSignatureHashFromDraft", hashParams)
	if err != nil {
		return nil, fmt.Errorf("compute signature hash failed: %w", err)
	}

	hashMap, ok := hashResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_computeSignatureHashFromDraft")
	}
	hashHex, ok := hashMap["hash"].(string)
	if !ok || hashHex == "" {
		return nil, fmt.Errorf("missing hash in wes_computeSignatureHashFromDraft response")
	}

	// 同时获取对应的 unsignedTx，确保后续 finalize 使用同一份交易
	unsignedTxHex, _ := hashMap["unsignedTx"].(string)

	hashBytes, err := hex.DecodeString(strings.TrimPrefix(hashHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode signature hash failed: %w", err)
	}

	// 6. 使用 Wallet 对哈希进行签名
	sigBytes, err := w.SignHash(hashBytes)
	if err != nil {
		return nil, fmt.Errorf("sign hash failed: %w", err)
	}

	// 7. 获取压缩公钥
	priv := w.PrivateKey()
	if priv == nil {
		return nil, fmt.Errorf("wallet private key is nil")
	}
	pubCompressed := ethcrypto.CompressPubkey(&priv.PublicKey)

	// 8. 调用 wes_finalizeTransactionFromDraft 生成带 SingleKeyProof 的交易
	finalizeParams := map[string]interface{}{
		// draft 依然传递，便于节点在需要时重建；但 unsignedTx 才是签名所对应的交易
		"draft":        json.RawMessage(draftJSON),
		"unsignedTx":   unsignedTxHex,
		"input_index":  inputIndex,
		"sighash_type": "SIGHASH_ALL",
		"pubkey":       "0x" + hex.EncodeToString(pubCompressed),
		"signature":    "0x" + hex.EncodeToString(sigBytes),
	}
	finalResult, err := s.client.Call(ctx, "wes_finalizeTransactionFromDraft", finalizeParams)
	if err != nil {
		return nil, fmt.Errorf("finalize transaction from draft failed: %w", err)
	}

	finalMap, ok := finalResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_finalizeTransactionFromDraft")
	}

	txHex, ok := finalMap["tx"].(string)
	if !ok || txHex == "" {
		return nil, fmt.Errorf("missing tx in wes_finalizeTransactionFromDraft response")
	}

	// 9. 提交交易
	sendResult, err := s.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 10. 返回结果
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
// 1. 调用 `buildBatchTransferDraft` 构建 DraftJSON
// 2. 为每个输入调用 `wes_computeSignatureHashFromDraft` 获取签名哈希
// 3. 使用 Wallet 签名每个哈希
// 4. 调用 `wes_finalizeTransactionFromDraft` 使用多输入签名模式生成带 SingleKeyProof 的交易
// 5. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO
// - 批量转账需要按 tokenID 分组 UTXO，为每个转账选择足够的 UTXO
// - 每个输入都需要单独签名
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

	// 4. 构建 DraftJSON
	draftJSON, inputIndices, err := buildBatchTransferDraft(ctx, s.client, req.From, req.Transfers)
	if err != nil {
		return nil, fmt.Errorf("build batch transfer draft failed: %w", err)
	}

	if len(inputIndices) == 0 {
		return nil, fmt.Errorf("no inputs to sign")
	}

	// 5. 获取压缩公钥（所有输入使用同一个公钥）
	priv := w.PrivateKey()
	if priv == nil {
		return nil, fmt.Errorf("wallet private key is nil")
	}
	pubCompressed := ethcrypto.CompressPubkey(&priv.PublicKey)
	pubKeyHex := "0x" + hex.EncodeToString(pubCompressed)

	// 6. 为每个输入计算签名哈希并签名
	type signatureItem struct {
		InputIndex uint32
		SigHex     string
	}
	var signatures []signatureItem
	var unsignedTxHex string

	for i, inputIndex := range inputIndices {
		// 调用 wes_computeSignatureHashFromDraft 获取签名哈希
		hashParams := map[string]interface{}{
			"draft":        json.RawMessage(draftJSON),
			"input_index":  inputIndex,
			"sighash_type": "SIGHASH_ALL",
		}
		hashResult, err := s.client.Call(ctx, "wes_computeSignatureHashFromDraft", hashParams)
		if err != nil {
			return nil, fmt.Errorf("compute signature hash for input %d failed: %w", inputIndex, err)
		}

		hashMap, ok := hashResult.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid response format from wes_computeSignatureHashFromDraft for input %d", inputIndex)
		}
		hashHex, ok := hashMap["hash"].(string)
		if !ok || hashHex == "" {
			return nil, fmt.Errorf("missing hash in wes_computeSignatureHashFromDraft response for input %d", inputIndex)
		}

		// 第一次调用时获取 unsignedTx，后续调用应该返回相同的 unsignedTx
		if i == 0 {
			unsignedTxHex, _ = hashMap["unsignedTx"].(string)
		}

		hashBytes, err := hex.DecodeString(strings.TrimPrefix(hashHex, "0x"))
		if err != nil {
			return nil, fmt.Errorf("decode signature hash for input %d failed: %w", inputIndex, err)
		}

		// 使用 Wallet 对哈希进行签名
		sigBytes, err := w.SignHash(hashBytes)
		if err != nil {
			return nil, fmt.Errorf("sign hash for input %d failed: %w", inputIndex, err)
		}

		signatures = append(signatures, signatureItem{
			InputIndex: inputIndex,
			SigHex:     "0x" + hex.EncodeToString(sigBytes),
		})
	}

	if unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx from wes_computeSignatureHashFromDraft")
	}

	// 7. 构建多输入签名数组
	signatureArray := make([]map[string]interface{}, len(signatures))
	for i, sig := range signatures {
		signatureArray[i] = map[string]interface{}{
			"input_index":  sig.InputIndex,
			"sighash_type": "SIGHASH_ALL",
			"pubkey":       pubKeyHex,
			"signature":    sig.SigHex,
		}
	}

	// 8. 调用 wes_finalizeTransactionFromDraft 使用多输入签名模式
	finalizeParams := map[string]interface{}{
		"draft":      json.RawMessage(draftJSON),
		"unsignedTx": unsignedTxHex,
		"signatures": signatureArray,
	}
	finalResult, err := s.client.Call(ctx, "wes_finalizeTransactionFromDraft", finalizeParams)
	if err != nil {
		return nil, fmt.Errorf("finalize transaction from draft failed: %w", err)
	}

	finalMap, ok := finalResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_finalizeTransactionFromDraft")
	}

	txHex, ok := finalMap["tx"].(string)
	if !ok || txHex == "" {
		return nil, fmt.Errorf("missing tx in wes_finalizeTransactionFromDraft response")
	}

	// 9. 提交交易
	sendResult, err := s.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 10. 返回结果
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
