package market

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/weisyn/client-sdk-go/utils"
	"github.com/weisyn/client-sdk-go/wallet"
)

// swapAMM AMM交换实现
//
// **架构说明**：
// SwapAMM 业务语义在 SDK 层，通过调用 AMM 合约的 swap 方法实现。
//
// **流程**：
// 1. 确定 AMM 合约地址（需要从配置或参数获取）
// 2. 构建 swap 方法参数（通过 payload）
// 3. 调用 `wes_callContract` API，设置 `return_unsigned_tx=true` 获取未签名交易
// 4. 使用 Wallet 签名未签名交易
// 5. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **注意**：
// - 需要提供 AMM 合约地址（contentHash）
// - 合约必须实现 swap 方法
func (s *marketService) swapAMM(ctx context.Context, req *SwapRequest, wallets ...wallet.Wallet) (*SwapResult, error) {
	// 1. 参数验证
	if err := s.validateSwapRequest(req); err != nil {
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

	// 5. 构建 payload（遵循 WES ABI 规范）
	// 规范来源：weisyn.git/docs/components/core/ispc/abi-and-payload.md
	payloadOptions := utils.BuildPayloadOptions{
		IncludeFrom: true,
		From:        req.From,
		MethodParams: map[string]interface{}{
			"tokenIn":      hex.EncodeToString(req.TokenIn),
			"tokenOut":     hex.EncodeToString(req.TokenOut),
			"amountIn":     req.AmountIn,
			"amountOutMin": req.AmountOutMin,
		},
	}

	payloadBase64, err := utils.BuildAndEncodePayload(payloadOptions)
	if err != nil {
		return nil, fmt.Errorf("build payload failed: %w", err)
	}

	// 6. 调用 wes_callContract API，设置 return_unsigned_tx=true
	callContractParams := map[string]interface{}{
		"content_hash":       hex.EncodeToString(ammContractHash),
		"method":             "swap",
		"params":             []uint64{}, // WASM 原生参数（空，使用 payload）
		"payload":            payloadBase64,
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

	// 11. 解析交易结果，提取实际输出金额
	amountOut := req.AmountOutMin

	parsedTx, err := utils.FetchAndParseTx(ctx, s.client, sendResult.TxHash)
	if err == nil && parsedTx != nil {
		// 查找返回给用户的输出（owner 是交换者地址）
		userOutputs := utils.FindOutputsByOwner(parsedTx.Outputs, req.From)

		// 汇总 tokenOut 金额
		if len(req.TokenOut) > 0 {
			totalAmount := utils.SumAmountsByToken(userOutputs, req.TokenOut)
			if totalAmount != nil {
				amountOut = totalAmount.Uint64()
			}
		} else {
			// 原生币
			totalAmount := utils.SumAmountsByToken(userOutputs, nil)
			if totalAmount != nil {
				amountOut = totalAmount.Uint64()
			}
		}
	}

	return &SwapResult{
		TxHash:    sendResult.TxHash,
		AmountOut: amountOut,
		Success:   true,
	}, nil
}

// validateSwapRequest 验证交换请求
func (s *marketService) validateSwapRequest(req *SwapRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证 AMM 合约地址（contentHash，32字节）
	if len(req.AMMContractAddr) != 32 {
		return fmt.Errorf("AMM contract address must be 32 bytes (contentHash)")
	}

	// 3. 验证金额
	if req.AmountIn == 0 {
		return fmt.Errorf("amount in must be greater than 0")
	}
	if req.AmountOutMin == 0 {
		return fmt.Errorf("minimum amount out must be greater than 0")
	}

	// 4. 验证代币不同
	if string(req.TokenIn) == string(req.TokenOut) {
		return fmt.Errorf("token in and token out must be different")
	}

	return nil
}
