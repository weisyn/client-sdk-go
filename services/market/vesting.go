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

// createVesting 创建归属计划实现
//
// **架构说明**：
// CreateVesting 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 归属计划使用 TimeLock + ContractLock 锁定条件。
//
// **流程**：
// 1. 调用 `buildVestingTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_createVesting` API（业务语义在 SDK 层实现）
func (s *marketService) createVesting(ctx context.Context, req *CreateVestingRequest, wallets ...wallet.Wallet) (*CreateVestingResult, error) {
	// 1. 参数验证
	if err := s.validateCreateVestingRequest(req); err != nil {
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
	// 注意：VestingContractAddr 可以从配置或参数中获取，当前先设为 nil（只使用 TimeLock）
	var vestingContractAddr []byte // TODO: 从配置或参数获取 Vesting 合约地址
	draftJSON, inputIndex, err := buildVestingDraft(
		ctx,
		s.client,
		req.From,
		req.To,
		req.Amount,
		req.TokenID,
		req.StartTime,
		req.Duration,
		vestingContractAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("build vesting draft failed: %w", err)
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

	// 7. 解析交易结果，提取 VestingID
	var vestingID []byte
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找归属输出（通常是第一个资产输出，且 owner 是受益人地址）
		// 归属输出通常带有 TimeLock
		for _, output := range parsedTx.Outputs {
			if output.Type == "asset" && bytes.Equal(output.Owner, req.To) {
				vestingID = []byte(output.Outpoint)
				break
			}
		}
	}

	return &CreateVestingResult{
		VestingID: vestingID,
		TxHash:    sendResult.TxHash,
		Success:   true,
	}, nil
}

// validateCreateVestingRequest 验证创建归属计划请求
func (s *marketService) validateCreateVestingRequest(req *CreateVestingRequest) error {
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

	// 3. 验证时间
	if req.Duration == 0 {
		return fmt.Errorf("duration must be greater than 0")
	}

	return nil
}

// claimVesting 领取归属代币实现
//
// **架构说明**：
// ClaimVesting 业务语义在 SDK 层，通过查询归属 UTXO、构建交易实现。
// 领取归属代币需要消费带有 TimeLock 的归属 UTXO（需要满足时间条件）。
//
// **流程**：
// 1. 解析 VestingID（outpoint 格式：txHash:index）
// 2. 查询归属 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费归属 UTXO，返回给受益人）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
// 5. 签名并提交
//
// **注意**：
// - 需要满足 TimeLock 的解锁条件（当前时间 >= unlock_timestamp）
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
func (s *marketService) claimVesting(ctx context.Context, req *ClaimVestingRequest, wallets ...wallet.Wallet) (*ClaimVestingResult, error) {
	// 1. 参数验证
	if err := s.validateClaimVestingRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildClaimVestingDraft(
		ctx,
		s.client,
		req.From,
		req.VestingID,
	)
	if err != nil {
		return nil, fmt.Errorf("build claim vesting draft failed: %w", err)
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

	// 7. 解析交易结果，提取实际领取金额
	claimAmount := uint64(0)

	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找返回给用户的输出（owner 是领取者地址）
		userOutputs := utils.FindOutputsByOwner(parsedTx.Outputs, req.From)

		// 汇总金额（归属代币可能是原生币或特定代币）
		totalAmount := utils.SumAmountsByToken(userOutputs, nil)
		if totalAmount != nil {
			claimAmount = totalAmount.Uint64()
		}
	}

	return &ClaimVestingResult{
		TxHash:      sendResult.TxHash,
		ClaimAmount: claimAmount,
		Success:     true,
	}, nil
}

// validateClaimVestingRequest 验证领取归属代币请求
func (s *marketService) validateClaimVestingRequest(req *ClaimVestingRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证归属计划ID
	if len(req.VestingID) == 0 {
		return fmt.Errorf("vesting ID is required")
	}

	return nil
}
