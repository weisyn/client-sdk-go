package staking

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

// delegate 委托实现
//
// **架构说明**：
// Delegate 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 委托使用 DelegationLock 锁定条件。
//
// **流程**：
// 1. 调用 `buildDelegateTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_delegate` API（业务语义在 SDK 层实现）
func (s *stakingService) delegate(ctx context.Context, req *DelegateRequest, wallets ...wallet.Wallet) (*DelegateResult, error) {
	// 1. 参数验证
	if err := s.validateDelegateRequest(req); err != nil {
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
	// 默认参数：有效期 0（永不过期），单次操作最大价值等于委托金额
	draftJSON, inputIndex, err := buildDelegateDraft(
		ctx,
		s.client,
		req.From,
		req.ValidatorAddr,
		req.Amount,
		0,          // expiryDurationBlocks: 0 = 永不过期
		req.Amount, // maxValuePerOperation: 等于委托金额
	)
	if err != nil {
		return nil, fmt.Errorf("build delegate draft failed: %w", err)
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

	// 10. 解析交易结果，提取 DelegateID
	delegateID := ""
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找委托输出（通常是第一个资产输出，且 owner 是委托者地址）
		// 委托输出通常带有 DelegationLock
		for _, output := range parsedTx.Outputs {
			if output.Type == "asset" && bytes.Equal(output.Owner, req.From) {
				delegateID = output.Outpoint
				break
			}
		}
	}

	return &DelegateResult{
		DelegateID: delegateID,
		TxHash:     sendResult.TxHash,
		Success:    true,
	}, nil
}

// validateDelegateRequest 验证委托请求
func (s *stakingService) validateDelegateRequest(req *DelegateRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}
	if len(req.ValidatorAddr) != 20 {
		return fmt.Errorf("validator address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	return nil
}

// undelegate 取消委托实现
//
// **架构说明**：
// Undelegate 业务语义在 SDK 层，通过查询委托 UTXO、构建交易实现。
// 取消委托需要消费带有 DelegationLock 的委托 UTXO。
//
// **流程**：
// 1. 调用 `buildUndelegateTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_undelegate` API（业务语义在 SDK 层实现）
func (s *stakingService) undelegate(ctx context.Context, req *UndelegateRequest, wallets ...wallet.Wallet) (*UndelegateResult, error) {
	// 1. 参数验证
	if err := s.validateUndelegateRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildUndelegateDraft(
		ctx,
		s.client,
		req.From,
		req.DelegateID,
		req.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("build undelegate draft failed: %w", err)
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

	// 10. 返回结果
	return &UndelegateResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateUndelegateRequest 验证取消委托请求
func (s *stakingService) validateUndelegateRequest(req *UndelegateRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证委托ID
	if len(req.DelegateID) == 0 {
		return fmt.Errorf("delegate ID is required")
	}

	return nil
}

// claimReward 领取奖励实现
//
// **架构说明**：
// ClaimReward 业务语义在 SDK 层，通过查询奖励 UTXO、构建交易实现。
// 奖励可能由合约产生，需要通过查询链上状态获取奖励 UTXO。
//
// **流程**：
// 1. 查询奖励 UTXO（通过 StakeID 或 DelegateID）
// 2. 构建交易草稿（消费奖励 UTXO，返回给用户）
// 3. 调用 `wes_buildTransaction` API 获取未签名交易
// 4. 签名并提交
//
// **注意**：
// - 奖励 UTXO 可能由合约产生，需要查询链上状态或通过合约调用获取
// - 当前实现假设奖励 UTXO 可以通过查询用户的 UTXO 列表找到
func (s *stakingService) claimReward(ctx context.Context, req *ClaimRewardRequest, wallets ...wallet.Wallet) (*ClaimRewardResult, error) {
	// 1. 参数验证
	if err := s.validateClaimRewardRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildClaimRewardDraft(
		ctx,
		s.client,
		req.From,
		req.StakeID,
		req.DelegateID,
	)
	if err != nil {
		return nil, fmt.Errorf("build claim reward draft failed: %w", err)
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

	// 10. 解析交易结果，提取奖励金额
	rewardAmount := uint64(0)

	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找返回给用户的输出（owner 是领取者地址）
		userOutputs := utils.FindOutputsByOwner(parsedTx.Outputs, req.From)

		// 汇总原生币金额（奖励通常是原生币）
		totalAmount := utils.SumAmountsByToken(userOutputs, nil)
		if totalAmount != nil {
			rewardAmount = totalAmount.Uint64()
		}
	}

	return &ClaimRewardResult{
		TxHash:       sendResult.TxHash,
		RewardAmount: rewardAmount,
		Success:      true,
	}, nil
}

// validateClaimRewardRequest 验证领取奖励请求
func (s *stakingService) validateClaimRewardRequest(req *ClaimRewardRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证至少提供一个ID
	if len(req.StakeID) == 0 && len(req.DelegateID) == 0 {
		return fmt.Errorf("either stake ID or delegate ID is required")
	}

	return nil
}
