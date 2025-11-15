package market

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// createVesting 创建归属计划实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的创建归属计划 JSON-RPC 方法（如 `wes_createVesting`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建创建归属计划交易（TimeLock + HeightLock）
//    - 需要节点提供 `wes_createVesting` JSON-RPC 方法
//    - 或通过合约调用实现（需要 Vesting 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/market/vesting.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_createVesting` API
// - 需要确认是否通过合约调用实现（需要 Vesting 合约地址）
func (s *marketService) createVesting(ctx context.Context, req *CreateVestingRequest, wallets ...wallet.Wallet) (*CreateVestingResult, error) {
	// 1. 参数验证
	if err := s.validateCreateVestingRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建创建归属计划交易
	// 当前节点可能没有提供创建归属计划相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_createVesting`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 Vesting 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("create vesting not implemented yet: requires node API support (wes_createVesting) or contract call")
}

// validateCreateVestingRequest 验证创建归属计划请求
func (s *marketService) validateCreateVestingRequest(req *CreateVestingRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}
	if len(req.To) != 20 {
		return fmt.Errorf("to address must be 20 bytes")
	}

	// 2. 验证金额
	if req.Amount == 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	// 3. 验证时间
	if req.Duration == 0 {
		return fmt.Errorf("duration must be greater than 0")
	}

	return nil
}

// claimVesting 领取归属代币实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的领取归属代币 JSON-RPC 方法（如 `wes_claimVesting`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建领取归属代币交易
//    - 需要节点提供 `wes_claimVesting` JSON-RPC 方法
//    - 或通过合约调用实现（需要 Vesting 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/market/vesting.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_claimVesting` API
// - 需要确认是否通过合约调用实现（需要 Vesting 合约地址）
func (s *marketService) claimVesting(ctx context.Context, req *ClaimVestingRequest, wallets ...wallet.Wallet) (*ClaimVestingResult, error) {
	// 1. 参数验证
	if err := s.validateClaimVestingRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建领取归属代币交易
	// 当前节点可能没有提供领取归属代币相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_claimVesting`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 Vesting 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("claim vesting not implemented yet: requires node API support (wes_claimVesting) or contract call")
}

// validateClaimVestingRequest 验证领取归属代币请求
func (s *marketService) validateClaimVestingRequest(req *ClaimVestingRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证归属计划ID
	if len(req.VestingID) == 0 {
		return fmt.Errorf("vesting ID is required")
	}

	return nil
}

