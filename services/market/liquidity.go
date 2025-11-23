package market

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/weisyn/client-sdk-go/utils"
	"github.com/weisyn/client-sdk-go/wallet"
)

// addLiquidity 添加流动性实现
//
// **架构说明**：
// AddLiquidity 业务语义在 SDK 层，通过调用 AMM 合约的 addLiquidity 方法实现。
// 
// **流程**：
// 1. 确定 AMM 合约地址（需要从配置或参数获取）
// 2. 构建 addLiquidity 方法参数（通过 payload）
// 3. 调用 `wes_callContract` API，设置 `return_unsigned_tx=true` 获取未签名交易
// 4. 使用 Wallet 签名未签名交易
// 5. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - 需要提供 AMM 合约地址（contentHash）
// - 合约必须实现 addLiquidity 方法
func (s *marketService) addLiquidity(ctx context.Context, req *AddLiquidityRequest, wallets ...wallet.Wallet) (*AddLiquidityResult, error) {
	// 1. 参数验证
	if err := s.validateAddLiquidityRequest(req); err != nil {
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

	// 4. 使用请求中的 AMM 合约地址（contentHash）
	ammContractHash := req.AMMContractAddr
	if len(ammContractHash) != 32 {
		return nil, fmt.Errorf("invalid AMM contract address: expected 32 bytes (contentHash), got %d bytes", len(ammContractHash))
	}

	// 5. 构建 addLiquidity 方法的参数（通过 payload）
	addLiquidityParams := map[string]interface{}{
		"from":    hex.EncodeToString(req.From),
		"tokenA":  hex.EncodeToString(req.TokenA),
		"tokenB":  hex.EncodeToString(req.TokenB),
		"amountA": req.AmountA,
		"amountB": req.AmountB,
	}

	// 将参数编码为 JSON，然后 Base64 编码
	payloadJSON, err := json.Marshal(addLiquidityParams)
	if err != nil {
		return nil, fmt.Errorf("marshal addLiquidity params failed: %w", err)
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// 6. 调用 wes_callContract API，设置 return_unsigned_tx=true
	callContractParams := map[string]interface{}{
		"content_hash":      hex.EncodeToString(ammContractHash),
		"method":            "addLiquidity",
		"params":            []uint64{}, // WASM 原生参数（空，使用 payload）
		"payload":           payloadBase64,
		"return_unsigned_tx": true,
	}

	result, err := s.client.Call(ctx, "wes_callContract", []interface{}{callContractParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_callContract failed: %w", err)
	}

	// 7. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in response")
	}

	// 8. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	// 9. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 10. 调用 wes_sendRawTransaction 提交已签名交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 11. 解析交易结果，提取 LiquidityID
	var liquidityID []byte
	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找流动性输出（通常是第一个资产输出，且 owner 是流动性提供者地址）
		// 流动性输出可能带有特殊的锁定条件或 metadata
		for _, output := range parsedTx.Outputs {
			if output.Type == "asset" && bytes.Equal(output.Owner, req.From) {
				liquidityID = []byte(output.Outpoint)
				break
			}
		}
	}

	return &AddLiquidityResult{
		LiquidityID: liquidityID,
		TxHash:      sendResult.TxHash,
		Success:     true,
	}, nil
}

// validateAddLiquidityRequest 验证添加流动性请求
func (s *marketService) validateAddLiquidityRequest(req *AddLiquidityRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证 AMM 合约地址（contentHash，32字节）
	if len(req.AMMContractAddr) != 32 {
		return fmt.Errorf("AMM contract address must be 32 bytes (contentHash)")
	}

	// 3. 验证金额
	if req.AmountA == 0 || req.AmountB == 0 {
		return fmt.Errorf("both amounts must be greater than 0")
	}

	return nil
}

// removeLiquidity 移除流动性实现
//
// **架构说明**：
// RemoveLiquidity 业务语义在 SDK 层，通过调用 AMM 合约的 removeLiquidity 方法实现。
// 
// **流程**：
// 1. 确定 AMM 合约地址（需要从配置或参数获取）
// 2. 构建 removeLiquidity 方法参数（通过 payload）
// 3. 调用 `wes_callContract` API，设置 `return_unsigned_tx=true` 获取未签名交易
// 4. 使用 Wallet 签名未签名交易
// 5. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - 需要提供 AMM 合约地址（contentHash）
// - 合约必须实现 removeLiquidity 方法
func (s *marketService) removeLiquidity(ctx context.Context, req *RemoveLiquidityRequest, wallets ...wallet.Wallet) (*RemoveLiquidityResult, error) {
	// 1. 参数验证
	if err := s.validateRemoveLiquidityRequest(req); err != nil {
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

	// 4. 使用请求中的 AMM 合约地址（contentHash）
	ammContractHash := req.AMMContractAddr
	if len(ammContractHash) != 32 {
		return nil, fmt.Errorf("invalid AMM contract address: expected 32 bytes (contentHash), got %d bytes", len(ammContractHash))
	}

	// 5. 构建 removeLiquidity 方法的参数（通过 payload）
	removeLiquidityParams := map[string]interface{}{
		"from":        hex.EncodeToString(req.From),
		"liquidityID": hex.EncodeToString(req.LiquidityID),
		"amount":      req.Amount,
	}

	// 将参数编码为 JSON，然后 Base64 编码
	payloadJSON, err := json.Marshal(removeLiquidityParams)
	if err != nil {
		return nil, fmt.Errorf("marshal removeLiquidity params failed: %w", err)
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// 6. 调用 wes_callContract API，设置 return_unsigned_tx=true
	callContractParams := map[string]interface{}{
		"content_hash":      hex.EncodeToString(ammContractHash),
		"method":            "removeLiquidity",
		"params":            []uint64{}, // WASM 原生参数（空，使用 payload）
		"payload":           payloadBase64,
		"return_unsigned_tx": true,
	}

	result, err := s.client.Call(ctx, "wes_callContract", []interface{}{callContractParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_callContract failed: %w", err)
	}

	// 7. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in response")
	}

	// 8. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	// 9. 使用 Wallet 签名交易
	signedTxBytes, err := w.SignTransaction(unsignedTxBytes)
	if err != nil {
		return nil, fmt.Errorf("sign transaction failed: %w", err)
	}

	// 10. 调用 wes_sendRawTransaction 提交已签名交易
	signedTxHex := "0x" + hex.EncodeToString(signedTxBytes)
	sendResult, err := s.client.SendRawTransaction(ctx, signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	// 11. 解析交易结果，提取实际获得的代币金额
	amountA := uint64(0)
	amountB := uint64(0)

	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找返回给用户的输出（owner 是流动性提供者地址）
		userOutputs := utils.FindOutputsByOwner(parsedTx.Outputs, req.From)
		
		// 分别汇总 TokenA 和 TokenB 的金额
		// 注意：需要从请求中获取 TokenA 和 TokenB，但当前 RemoveLiquidityRequest 没有这些字段
		// 简化处理：汇总所有代币输出
		for _, output := range userOutputs {
			if output.Amount != nil {
				if output.TokenID == nil {
					// 原生币，可能是 TokenA 或 TokenB（简化处理）
					if amountA == 0 {
						amountA = output.Amount.Uint64()
					} else {
						amountB = output.Amount.Uint64()
					}
				} else {
					// 代币，根据 TokenID 判断
					if amountA == 0 {
						amountA = output.Amount.Uint64()
					} else {
						amountB = output.Amount.Uint64()
					}
				}
			}
		}
	}

	return &RemoveLiquidityResult{
		TxHash:  sendResult.TxHash,
		AmountA: amountA,
		AmountB: amountB,
		Success: true,
	}, nil
}

// validateRemoveLiquidityRequest 验证移除流动性请求
func (s *marketService) validateRemoveLiquidityRequest(req *RemoveLiquidityRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证 AMM 合约地址（contentHash，32字节）
	if len(req.AMMContractAddr) != 32 {
		return fmt.Errorf("AMM contract address must be 32 bytes (contentHash)")
	}

	// 3. 验证流动性ID
	if len(req.LiquidityID) == 0 {
		return fmt.Errorf("liquidity ID is required")
	}

	// 4. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	return nil
}

