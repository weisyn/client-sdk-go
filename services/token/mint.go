package token

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

// mint 代币铸造实现
//
// **架构说明**：
// Mint 业务语义在 SDK 层，通过调用合约的 mint 方法实现。
//
// **流程**：
// 1. 确定合约 contentHash（从 contractAddr 或 tokenID）
// 2. 构建 mint 方法参数（通过 payload）
// 3. 调用 `wes_callContract` API，设置 `return_unsigned_tx=true` 获取未签名交易
// 4. 使用 Wallet 签名未签名交易
// 5. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - 合约必须实现 mint 方法
// - 合约内部通过 create_utxo_output 创建代币输出
func (s *tokenService) mint(ctx context.Context, req *MintRequest, wallets ...wallet.Wallet) (*MintResult, error) {
	// 1. 参数验证
	if err := s.validateMintRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 使用请求中的合约 contentHash
	contentHash := req.ContractContentHash
	if len(contentHash) != 32 {
		return nil, fmt.Errorf("contract contentHash must be 32 bytes")
	}

	// 4. 构建 payload（遵循 WES ABI 规范）
	// 规范来源：weisyn.git/docs/components/core/ispc/abi-and-payload.md
	payloadOptions := utils.BuildPayloadOptions{
		IncludeFrom:   true,
		From:          w.Address(),
		IncludeTo:     true,
		To:            req.To,
		IncludeAmount: true,
		Amount:        req.Amount,
	}

	if len(req.TokenID) > 0 {
		payloadOptions.IncludeTokenID = true
		payloadOptions.TokenID = req.TokenID
	}

	payloadBase64, err := utils.BuildAndEncodePayload(payloadOptions)
	if err != nil {
		return nil, fmt.Errorf("build payload failed: %w", err)
	}

	// 5. 调用 wes_callContract API，设置 return_unsigned_tx=true
	callContractParams := map[string]interface{}{
		"content_hash":       hex.EncodeToString(contentHash),
		"method":             "mint",
		"params":             []uint64{}, // WASM 原生参数（空，使用 payload）
		"payload":            payloadBase64,
		"return_unsigned_tx": true,
	}

	result, err := s.client.Call(ctx, "wes_callContract", []interface{}{callContractParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_callContract failed: %w", err)
	}

	// 6. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in response")
	}

	// 7. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	// 8. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 9. 调用 wes_sendRawTransaction 提交已签名交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 10. 返回结果
	return &MintResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateMintRequest 验证铸造请求
func (s *tokenService) validateMintRequest(req *MintRequest) error {
	// 1. 验证接收地址
	if len(req.To) != 20 {
		return fmt.Errorf("to address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 3. 验证合约 contentHash（必需）
	if len(req.ContractContentHash) != 32 {
		return fmt.Errorf("contract contentHash must be 32 bytes")
	}

	return nil
}

// burn 代币销毁实现
//
// **架构说明**：
// Burn 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
// 1. 调用 `buildBurnTransaction` 在 SDK 层构建未签名交易
// 2. 使用 Wallet 签名未签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - Burn 交易通过消费 UTXO 但不创建输出（或只创建找零）来实现销毁
// - SDK 层使用 `wes_getUTXO` 查询 UTXO，使用 `wes_buildTransaction` 构建交易
func (s *tokenService) burn(ctx context.Context, req *BurnRequest, wallets ...wallet.Wallet) (*BurnResult, error) {
	// 1. 参数验证
	if err := s.validateBurnRequest(req); err != nil {
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
	draftJSON, inputIndex, err := buildBurnDraft(ctx, s.client, req.From, req.Amount, req.TokenID)
	if err != nil {
		return nil, fmt.Errorf("build burn draft failed: %w", err)
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
	return &BurnResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// validateBurnRequest 验证销毁请求
func (s *tokenService) validateBurnRequest(req *BurnRequest) error {
	// 1. 验证发送地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 3. 验证TokenID（nil 表示原生币，非 nil 必须是 32 字节）
	if req.TokenID != nil && len(req.TokenID) != 32 {
		return fmt.Errorf("tokenID must be 32 bytes or nil (for native coin)")
	}

	return nil
}
