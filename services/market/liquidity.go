package market

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// addLiquidity 添加流动性实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的添加流动性 JSON-RPC 方法（如 `wes_addLiquidity`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建添加流动性交易
//    - 需要节点提供 `wes_addLiquidity` JSON-RPC 方法
//    - 或通过合约调用实现（需要 AMM 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/market/liquidity.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_addLiquidity` API
// - 需要确认是否通过合约调用实现（需要 AMM 合约地址）
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

	// 4. TODO: 调用节点API构建添加流动性交易
	// 当前节点可能没有提供添加流动性相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_addLiquidity`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 AMM 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("add liquidity not implemented yet: requires node API support (wes_addLiquidity) or contract call")
}

// validateAddLiquidityRequest 验证添加流动性请求
func (s *marketService) validateAddLiquidityRequest(req *AddLiquidityRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证金额
	if req.AmountA == 0 || req.AmountB == 0 {
		return fmt.Errorf("both amounts must be greater than 0")
	}

	return nil
}

// removeLiquidity 移除流动性实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的移除流动性 JSON-RPC 方法（如 `wes_removeLiquidity`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建移除流动性交易
//    - 需要节点提供 `wes_removeLiquidity` JSON-RPC 方法
//    - 或通过合约调用实现（需要 AMM 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/market/liquidity.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_removeLiquidity` API
// - 需要确认是否通过合约调用实现（需要 AMM 合约地址）
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

	// 4. TODO: 调用节点API构建移除流动性交易
	// 当前节点可能没有提供移除流动性相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_removeLiquidity`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 AMM 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("remove liquidity not implemented yet: requires node API support (wes_removeLiquidity) or contract call")
}

// validateRemoveLiquidityRequest 验证移除流动性请求
func (s *marketService) validateRemoveLiquidityRequest(req *RemoveLiquidityRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证流动性ID
	if len(req.LiquidityID) == 0 {
		return fmt.Errorf("liquidity ID is required")
	}

	// 3. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	return nil
}

