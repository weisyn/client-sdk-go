package market

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/weisyn/client-sdk-go/utils"
	"github.com/weisyn/client-sdk-go/wallet"
)

// createEscrow 创建托管实现
//
// **架构说明**：
// CreateEscrow 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 托管使用 MultiKeyLock 锁定条件（买方和卖方都需要签名才能解锁）。
//
// **流程**：
// 1. 调用 `buildEscrowTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_createEscrow` API（业务语义在 SDK 层实现）
func (s *marketService) createEscrow(ctx context.Context, req *CreateEscrowRequest, wallets ...wallet.Wallet) (*CreateEscrowResult, error) {
	// 1. 参数验证
	if err := s.validateCreateEscrowRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配（买方创建托管）
	if !bytes.Equal(w.Address(), req.Buyer) {
		return nil, fmt.Errorf("wallet address does not match buyer address")
	}

	// 4. 在 SDK 层构建 DraftJSON（不直接构建交易）
	// 注意：EscrowContractAddr 可以从配置或参数中获取，当前先设为 nil（使用 MultiKeyLock）
	var escrowContractAddr []byte // TODO: 从配置或参数获取 Escrow 合约地址
	draftJSON, inputIndex, err := buildEscrowDraft(
		ctx,
		s.client,
		req.Buyer,
		req.Seller,
		req.Amount,
		req.TokenID,
		req.Expiry,
		escrowContractAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("build escrow draft failed: %w", err)
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

	// 7. 解析交易结果，提取 EscrowID
	var escrowID []byte
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找托管输出（通常是第一个资产输出，且 owner 是买方或卖方地址）
		// 托管输出通常带有 MultiKeyLock
		for _, output := range parsedTx.Outputs {
			if output.Type == "asset" {
				// 托管输出可能是买方或卖方地址（取决于具体实现）
				if bytes.Equal(output.Owner, req.Buyer) || bytes.Equal(output.Owner, req.Seller) {
					escrowID = []byte(output.Outpoint)
					break
				}
			}
		}
	}

	return &CreateEscrowResult{
		EscrowID: escrowID,
		TxHash:   sendResult.TxHash,
		Success:  true,
	}, nil
}

// validateCreateEscrowRequest 验证创建托管请求
func (s *marketService) validateCreateEscrowRequest(req *CreateEscrowRequest) error {
	// 1. 验证地址
	if len(req.Buyer) != 20 {
		return fmt.Errorf("buyer address must be 20 bytes")
	}
	if len(req.Seller) != 20 {
		return fmt.Errorf("seller address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 3. 验证过期时间
	if req.Expiry == 0 {
		return fmt.Errorf("expiry time is required")
	}

	return nil
}

// releaseEscrow 释放托管实现
//
// **架构说明**：
// ReleaseEscrow 业务语义在 SDK 层，通过查询托管 UTXO、构建交易实现。
// 释放托管需要消费带有 MultiKeyLock 的托管 UTXO（需要买方和卖方签名）。
//
// **流程**：
// 1. 调用 `buildReleaseEscrowTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_releaseEscrow` API（业务语义在 SDK 层实现）
// - 释放托管需要买方和卖方都签名（MultiKeyLock），当前实现只处理买方签名部分
func (s *marketService) releaseEscrow(ctx context.Context, req *ReleaseEscrowRequest, wallets ...wallet.Wallet) (*ReleaseEscrowResult, error) {
	// 1. 参数验证
	if err := s.validateReleaseEscrowRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildReleaseEscrowDraft(
		ctx,
		s.client,
		req.From,
		req.SellerAddress,
		req.EscrowID,
	)
	if err != nil {
		return nil, fmt.Errorf("build release escrow draft failed: %w", err)
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
	// 注意：MultiKeyLock 需要买方和卖方都签名，当前实现只处理买方签名
	// 实际使用中，可能需要多方签名流程
	sendResult, err := s.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 7. 返回结果
	return &ReleaseEscrowResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateReleaseEscrowRequest 验证释放托管请求
func (s *marketService) validateReleaseEscrowRequest(req *ReleaseEscrowRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}
	if len(req.SellerAddress) != 20 {
		return fmt.Errorf("seller address must be 20 bytes")
	}

	// 2. 验证托管ID
	if len(req.EscrowID) == 0 {
		return fmt.Errorf("escrow ID is required")
	}

	return nil
}

// refundEscrow 退款托管实现
//
// **架构说明**：
// RefundEscrow 业务语义在 SDK 层，通过查询托管 UTXO、构建交易实现。
// 退款托管需要消费带有 MultiKeyLock 的托管 UTXO（过期后可以退款给买方）。
//
// **流程**：
// 1. 调用 `buildRefundEscrowTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_refundEscrow` API（业务语义在 SDK 层实现）
// - 退款托管在过期后可以退款给买方（TimeLock + MultiKeyLock）
func (s *marketService) refundEscrow(ctx context.Context, req *RefundEscrowRequest, wallets ...wallet.Wallet) (*RefundEscrowResult, error) {
	// 1. 参数验证
	if err := s.validateRefundEscrowRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildRefundEscrowDraft(
		ctx,
		s.client,
		req.From,
		req.BuyerAddress,
		req.EscrowID,
	)
	if err != nil {
		return nil, fmt.Errorf("build refund escrow draft failed: %w", err)
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

	// 7. 返回结果
	return &RefundEscrowResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateRefundEscrowRequest 验证退款托管请求
func (s *marketService) validateRefundEscrowRequest(req *RefundEscrowRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}
	if len(req.BuyerAddress) != 20 {
		return fmt.Errorf("buyer address must be 20 bytes")
	}

	// 2. 验证托管ID
	if len(req.EscrowID) == 0 {
		return fmt.Errorf("escrow ID is required")
	}

	return nil
}
