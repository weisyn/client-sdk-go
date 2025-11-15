package market

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// swapAMM AMM交换实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的 AMM 交换 JSON-RPC 方法（如 `wes_swapAMM`）。
//
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建 AMM 交换交易
//   - 需要节点提供 `wes_swapAMM` JSON-RPC 方法
//   - 或通过合约调用实现（需要 AMM 合约地址）
//
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/market/swap.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_swapAMM` API
// - 需要确认是否通过合约调用实现（需要 AMM 合约地址）
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

	// 4. TODO: 调用节点API构建 AMM 交换交易
	// 当前节点可能没有提供 AMM 交换相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_swapAMM`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 AMM 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("swap AMM not implemented yet: requires node API support (wes_swapAMM) or contract call")
}

// validateSwapRequest 验证交换请求
func (s *marketService) validateSwapRequest(req *SwapRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证金额
	if req.AmountIn == 0 {
		return fmt.Errorf("amount in must be greater than 0")
	}
	if req.AmountOutMin == 0 {
		return fmt.Errorf("minimum amount out must be greater than 0")
	}

	// 3. 验证代币不同
	if string(req.TokenIn) == string(req.TokenOut) {
		return fmt.Errorf("token in and token out must be different")
	}

	return nil
}
