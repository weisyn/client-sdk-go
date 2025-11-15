package staking

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// stake 质押实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的质押 JSON-RPC 方法（如 `wes_stake`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建质押交易（ContractLock + HeightLock）
//    - 需要节点提供 `wes_stake` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/stake.go` - 业务逻辑实现
// - `internal/api/jsonrpc/methods/tx.go` - JSON-RPC 方法实现（参考参数格式）
//
// **当前限制**：
// - 节点可能没有提供 `wes_stake` API
// - 需要确认是否通过合约调用实现（需要合约地址）
//
// **后续工作**：
// - 在节点中添加 `wes_stake` JSON-RPC 方法（推荐）
// - 或通过合约调用实现（需要合约地址和合约签名）
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

	// 4. TODO: 调用节点API构建质押交易
	// 当前节点可能没有提供质押相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_stake`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("staking not implemented yet: requires node API support (wes_stake) or contract call")
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
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的解除质押 JSON-RPC 方法（如 `wes_unstake`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建解除质押交易
//    - 需要节点提供 `wes_unstake` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/stake.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_unstake` API
// - 需要确认是否通过合约调用实现（需要合约地址）
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

	// 4. TODO: 调用节点API构建解除质押交易
	// 当前节点可能没有提供解除质押相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_unstake`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("unstake not implemented yet: requires node API support (wes_unstake) or contract call")
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

