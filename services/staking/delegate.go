package staking

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// delegate 委托实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的委托 JSON-RPC 方法（如 `wes_delegate`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建委托交易（DelegationLock）
//    - 需要节点提供 `wes_delegate` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/delegate.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_delegate` API
// - 需要确认是否通过合约调用实现（需要合约地址）
func (s *stakingService) delegate(ctx context.Context, req *DelegateRequest, wallets ...wallet.Wallet) (*DelegateResult, error) {
	// 1. 参数验证
	if err := s.validateDelegateRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建委托交易
	// 当前节点可能没有提供委托相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_delegate`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("delegate not implemented yet: requires node API support (wes_delegate) or contract call")
}

// validateDelegateRequest 验证委托请求
func (s *stakingService) validateDelegateRequest(req *DelegateRequest) error {
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

	return nil
}

// undelegate 取消委托实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的取消委托 JSON-RPC 方法（如 `wes_undelegate`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建取消委托交易
//    - 需要节点提供 `wes_undelegate` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/delegate.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_undelegate` API
// - 需要确认是否通过合约调用实现（需要合约地址）
func (s *stakingService) undelegate(ctx context.Context, req *UndelegateRequest, wallets ...wallet.Wallet) (*UndelegateResult, error) {
	// 1. 参数验证
	if err := s.validateUndelegateRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建取消委托交易
	// 当前节点可能没有提供取消委托相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_undelegate`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("undelegate not implemented yet: requires node API support (wes_undelegate) or contract call")
}

// validateUndelegateRequest 验证取消委托请求
func (s *stakingService) validateUndelegateRequest(req *UndelegateRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证委托ID
	if len(req.DelegateID) == 0 {
		return fmt.Errorf("delegate ID is required")
	}

	return nil
}

// claimReward 领取奖励实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的领取奖励 JSON-RPC 方法（如 `wes_claimReward`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建领取奖励交易
//    - 需要节点提供 `wes_claimReward` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/delegate.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_claimReward` API
// - 需要确认是否通过合约调用实现（需要合约地址）
func (s *stakingService) claimReward(ctx context.Context, req *ClaimRewardRequest, wallets ...wallet.Wallet) (*ClaimRewardResult, error) {
	// 1. 参数验证
	if err := s.validateClaimRewardRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建领取奖励交易
	// 当前节点可能没有提供领取奖励相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_claimReward`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("claim reward not implemented yet: requires node API support (wes_claimReward) or contract call")
}

// validateClaimRewardRequest 验证领取奖励请求
func (s *stakingService) validateClaimRewardRequest(req *ClaimRewardRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证至少提供一个ID
	if len(req.StakeID) == 0 && len(req.DelegateID) == 0 {
		return fmt.Errorf("either stake ID or delegate ID is required")
	}

	return nil
}

