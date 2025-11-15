package token

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

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

	// 3. 确定合约 contentHash
	// TODO: 需要实现从 contractAddr 或 tokenID 查询合约 contentHash 的接口
	// 当前简化：假设 contractAddr 或 tokenID 可以作为 contentHash（32字节）
	var contentHash []byte
	if len(req.ContractAddr) == 20 {
		// 简化：假设 contractAddr 可以用于查询 contentHash
		// 实际应该通过 contractAddr 查询合约的 contentHash
		// 临时方案：如果 tokenID 不为空，使用 tokenID 作为 contentHash
		if len(req.TokenID) == 32 {
			contentHash = req.TokenID
		} else {
			return nil, fmt.Errorf("cannot determine contract contentHash: need contractAddr to contentHash lookup or provide tokenID")
		}
	} else if len(req.TokenID) == 32 {
		contentHash = req.TokenID
	} else {
		return nil, fmt.Errorf("contractAddr or tokenID is required")
	}

	// 4. 构建 mint 方法的参数（通过 payload）
	mintParams := map[string]interface{}{
		"to":     hex.EncodeToString(req.To),
		"amount": req.Amount,
	}
	if len(req.TokenID) > 0 {
		mintParams["tokenID"] = hex.EncodeToString(req.TokenID)
	}
	if len(req.ContractAddr) > 0 {
		mintParams["contractAddr"] = hex.EncodeToString(req.ContractAddr)
	}

	// 将参数编码为 JSON，然后 Base64 编码
	payloadJSON, err := json.Marshal(mintParams)
	if err != nil {
		return nil, fmt.Errorf("marshal mint params failed: %w", err)
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// 5. 调用 wes_callContract API，设置 return_unsigned_tx=true
	callContractParams := map[string]interface{}{
		"content_hash":      hex.EncodeToString(contentHash),
		"method":            "mint",
		"params":            []uint64{}, // WASM 原生参数（空，使用 payload）
		"payload":           payloadBase64,
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

	// 3. 验证TokenID（如果提供）
	if req.TokenID != nil && len(req.TokenID) != 32 {
		return fmt.Errorf("tokenID must be 32 bytes if provided")
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

	// 4. 在 SDK 层构建未签名交易
	unsignedTxBytes, err := buildBurnTransaction(ctx, s.client, req.From, req.Amount, req.TokenID)
	if err != nil {
		return nil, fmt.Errorf("build burn transaction failed: %w", err)
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

	// 3. 验证TokenID（必需）
	if req.TokenID == nil {
		return fmt.Errorf("tokenID is required")
	}
	if len(req.TokenID) != 32 {
		return fmt.Errorf("tokenID must be 32 bytes")
	}

	return nil
}

