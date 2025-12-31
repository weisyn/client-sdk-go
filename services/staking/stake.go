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

// stake 质押实现
//
// **架构说明**：
// Stake 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
// 质押通过组合 HeightLock + ContractLock 实现：
// - HeightLock：锁定指定区块数
// - ContractLock：由 Staking 合约控制解锁逻辑（可选）
//
// **流程**：
// 1. 调用 `buildStakeTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_stake` API（业务语义在 SDK 层实现）
func (s *stakingService) stake(ctx context.Context, req *StakeRequest, wallets ...wallet.Wallet) (*StakeResult, error) {
	// 1. 参数验证
	if err := s.validateStakeRequest(req); err != nil {
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
	// 注意：StakingContractAddr 可以从配置或参数中获取，当前先设为 nil（只使用 HeightLock）
	var stakingContractAddr []byte // TODO: 从配置或参数获取 Staking 合约地址
	draftJSON, inputIndex, err := buildStakeDraft(
		ctx,
		s.client,
		req.From,
		req.ValidatorAddr,
		req.Amount,
		req.LockBlocks,
		stakingContractAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("build stake draft failed: %w", err)
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

	// 10. 解析交易结果，提取 StakeID
	stakeID := ""
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找质押输出（通常是第一个资产输出，且 owner 是质押者地址）
		// 质押输出通常带有 HeightLock 或 ContractLock
		for _, output := range parsedTx.Outputs {
			if output.Type == "asset" && bytes.Equal(output.Owner, req.From) {
				stakeID = output.Outpoint
				break
			}
		}
	}

	return &StakeResult{
		StakeID: stakeID,
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateStakeRequest 验证质押请求
func (s *stakingService) validateStakeRequest(req *StakeRequest) error {
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

	// 3. 验证锁定期
	if req.LockBlocks == 0 {
		return fmt.Errorf("lock blocks must be greater than 0")
	}

	return nil
}

// unstake 解除质押实现
//
// **架构说明**：
// Unstake 业务语义在 SDK 层，通过查询质押 UTXO、构建交易实现。
// 解质押需要消费带有 HeightLock 的质押 UTXO。
//
// **流程**：
// 1. 调用 `buildUnstakeTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_unstake` API（业务语义在 SDK 层实现）
func (s *stakingService) unstake(ctx context.Context, req *UnstakeRequest, wallets ...wallet.Wallet) (*UnstakeResult, error) {
	// 1. 参数验证
	if err := s.validateUnstakeRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildUnstakeDraft(
		ctx,
		s.client,
		req.From,
		req.StakeID,
		req.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("build unstake draft failed: %w", err)
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

	// 10. 解析交易结果，提取解质押金额和奖励金额
	unstakeAmount := req.Amount
	rewardAmount := uint64(0)

	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找返回给用户的输出（owner 是解质押者地址）
		userOutputs := utils.FindOutputsByOwner(parsedTx.Outputs, req.From)

		// 汇总原生币金额（解质押金额 + 奖励）
		totalAmount := utils.SumAmountsByToken(userOutputs, nil)
		if totalAmount != nil {
			unstakeAmount = totalAmount.Uint64()
			// 奖励金额 = 总金额 - 请求的解质押金额（简化处理）
			if unstakeAmount > req.Amount {
				rewardAmount = unstakeAmount - req.Amount
			}
		}
	}

	return &UnstakeResult{
		TxHash:        sendResult.TxHash,
		UnstakeAmount: unstakeAmount,
		RewardAmount:  rewardAmount,
		Success:       true,
	}, nil
}

// validateUnstakeRequest 验证解除质押请求
func (s *stakingService) validateUnstakeRequest(req *UnstakeRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证质押ID
	if len(req.StakeID) == 0 {
		return fmt.Errorf("stake ID is required")
	}

	return nil
}
