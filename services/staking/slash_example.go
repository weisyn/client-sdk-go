package staking

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// slashViaContract 通过 Slash 合约实现罚没（示例）
//
// **前提条件**：
// - 链上已部署 Slash 合约
// - 合约地址已知
//
// **流程**：
// 1. 构建 Slash 方法参数
// 2. 调用 `wes_callContract` API
// 3. 签名并提交交易
func slashViaContract(
	ctx context.Context,
	client client.Client,
	request *SlashRequest,
	slashContractAddr []byte,
	w wallet.Wallet,
) (*SlashResult, error) {
	// 1. 参数验证
	if err := validateSlashRequest(request); err != nil {
		return nil, err
	}

	// 2. 构建 Slash 方法参数
	slashParams := map[string]interface{}{
		"validator_addr": hex.EncodeToString(request.ValidatorAddr),
		"amount":         request.Amount,
		"reason":         request.Reason,
	}

	payloadJSON, err := json.Marshal(slashParams)
	if err != nil {
		return nil, fmt.Errorf("marshal slash params failed: %w", err)
	}

	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// 3. 调用 `wes_callContract` API
	callContractParams := map[string]interface{}{
		"content_hash":      hex.EncodeToString(slashContractAddr),
		"method":            "slash",
		"params":            []interface{}{},
		"payload":           payloadBase64,
		"return_unsigned_tx": true,
	}

	result, err := client.Call(ctx, "wes_callContract", []interface{}{callContractParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_callContract failed: %w", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_callContract")
	}

	unsignedTxHex, ok := resultMap["unsigned_tx"].(string)
	if !ok {
		return nil, fmt.Errorf("missing unsigned_tx in response")
	}

	// 4. 签名交易
	unsignedTx, err := hex.DecodeString(unsignedTxHex)
	if err != nil {
		return nil, fmt.Errorf("decode unsigned tx failed: %w", err)
	}

	signature, err := w.SignTransaction(unsignedTx)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 5. 完成交易
	// 从私钥获取公钥
	privateKey := w.PrivateKey()
	publicKeyBytes := ethcrypto.FromECDSAPub(&privateKey.PublicKey)
	
	finalizeParams := map[string]interface{}{
		"draft":        unsignedTxHex,
		"signatures":   []string{hex.EncodeToString(signature)},
		"input_index":  0,
		"public_key":   hex.EncodeToString(publicKeyBytes),
	}

	finalResult, err := client.Call(ctx, "wes_finalizeTransactionFromDraft", []interface{}{finalizeParams})
	if err != nil {
		return nil, fmt.Errorf("finalize transaction failed: %w", err)
	}

	finalResultMap, ok := finalResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid finalize response format")
	}

	signedTxHex, ok := finalResultMap["signed_tx"].(string)
	if !ok {
		return nil, fmt.Errorf("missing signed_tx in response")
	}

	// 6. 提交交易
	sendResult, err := client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	return &SlashResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// slashViaGovernance 通过治理提案实现罚没（示例）
//
// **前提条件**：
// - Governance Service 已配置
// - 治理合约支持 Slash 提案
//
// **流程**：
// 1. 创建 Slash 提案
// 2. 等待投票
// 3. 执行提案（自动执行 Slash）
func slashViaGovernance(
	ctx context.Context,
	governanceService interface { // Governance Service 接口
		Propose(ctx context.Context, req interface{}, wallets ...wallet.Wallet) (interface{}, error)
	},
	request *SlashRequest,
	proposerWallet wallet.Wallet,
	votingPeriod uint64,
) (*SlashResult, error) {
	// 1. 构建提案内容
	proposalTitle := fmt.Sprintf("Slash Validator: %s", hex.EncodeToString(request.ValidatorAddr))
	proposalDescription := fmt.Sprintf(
		"Slash Request:\n- Validator: %s\n- Amount: %d\n- Reason: %s",
		hex.EncodeToString(request.ValidatorAddr),
		request.Amount,
		request.Reason,
	)

	// 2. 创建治理提案
	// 注意：这里需要根据实际的 Governance Service 接口调整
	proposeReq := map[string]interface{}{
		"proposer":     proposerWallet.Address(),
		"title":        proposalTitle,
		"description":  proposalDescription,
		"voting_period": votingPeriod,
	}

	proposeResult, err := governanceService.Propose(ctx, proposeReq, proposerWallet)
	if err != nil {
		return nil, fmt.Errorf("create slash proposal failed: %w", err)
	}

	// 3. 解析提案结果
	// 注意：实际实现需要根据 Governance Service 的返回类型调整
	proposeResultMap, ok := proposeResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid propose result format")
	}

	txHash, _ := proposeResultMap["tx_hash"].(string)
	success, _ := proposeResultMap["success"].(bool)

	if !success {
		return nil, fmt.Errorf("failed to create slash proposal")
	}

	// 4. 返回提案结果
	// 注意：实际的 Slash 执行会在提案通过后自动执行
	return &SlashResult{
		TxHash:  txHash,
		Success: true,
	}, nil
}

// validateSlashRequest 验证罚没请求（辅助函数）
func validateSlashRequest(req *SlashRequest) error {
	// 1. 验证验证者地址
	if len(req.ValidatorAddr) != 20 {
		return fmt.Errorf("validator address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 3. 验证原因
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}

	return nil
}

