package governance

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

// propose 创建提案实现
//
// **架构说明**：
// Propose 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 提案使用 StateOutput + MultiKeyLock/ThresholdLock 锁定条件。
//
// **流程**：
// 1. 调用 `buildProposeTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
// - 不需要节点提供 `wes_propose` API（业务语义在 SDK 层实现）
func (s *governanceService) propose(ctx context.Context, req *ProposeRequest, wallets ...wallet.Wallet) (*ProposeResult, error) {
	// 1. 参数验证
	if err := s.validateProposeRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.Proposer) {
		return nil, fmt.Errorf("wallet address does not match proposer address")
	}

	// 4. 在 SDK 层构建 DraftJSON（不直接构建交易）
	// TODO: 需要从配置或参数获取验证者地址列表
	// 当前简化：使用提案者地址作为验证者（实际应该查询验证者列表）
	validatorAddresses := [][]byte{req.Proposer} // 临时：使用提案者地址
	threshold := uint32(1)                        // 临时：需要1个签名

	draftJSON, inputIndex, err := buildProposeDraft(
		ctx,
		s.client,
		req.Proposer,
		req.Title,
		req.Description,
		req.VotingPeriod,
		validatorAddresses,
		threshold,
	)
	if err != nil {
		return nil, fmt.Errorf("build propose draft failed: %w", err)
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

	// 10. 解析交易结果，提取 ProposalID
	proposalID := ""
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找 StateOutput（提案通常使用 StateOutput）
		stateOutputs := utils.FindStateOutputs(parsedTx.Outputs)
		if len(stateOutputs) > 0 {
			// 使用第一个 StateOutput 的 stateID 或 outpoint 作为 ProposalID
			if len(stateOutputs[0].StateID) > 0 {
				proposalID = hex.EncodeToString(stateOutputs[0].StateID)
			} else {
				proposalID = stateOutputs[0].Outpoint
			}
		} else {
			// 如果没有 StateOutput，使用第一个输出的 outpoint
			if len(parsedTx.Outputs) > 0 {
				proposalID = parsedTx.Outputs[0].Outpoint
			}
		}
	}

	return &ProposeResult{
		ProposalID: proposalID,
		TxHash:     sendResult.TxHash,
		Success:    true,
	}, nil
}

// validateProposeRequest 验证提案请求
func (s *governanceService) validateProposeRequest(req *ProposeRequest) error {
	// 1. 验证地址
	if len(req.Proposer) != 20 {
		return fmt.Errorf("proposer address must be 20 bytes")
	}

	// 2. 验证标题
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	// 3. 验证投票期限
	if req.VotingPeriod == 0 {
		return fmt.Errorf("voting period must be greater than 0")
	}

	return nil
}

