package staking

import (
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// slash 罚没实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的罚没 JSON-RPC 方法（如 `wes_slash`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建罚没交易
//    - 需要节点提供 `wes_slash` JSON-RPC 方法
//    - 或通过合约调用实现（需要合约地址）
// 2. 使用钱包签名交易（可能需要多方签名）
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/staking/slash.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_slash` API
// - 罚没通常需要多方验证，可能需要治理系统支持
// - 需要确认是否通过合约调用实现（需要合约地址）
func (s *stakingService) slash(ctx context.Context, req *SlashRequest, wallets ...wallet.Wallet) (*SlashResult, error) {
	// 1. 参数验证
	if err := s.validateSlashRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet（罚没可能需要多方签名）
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. TODO: 调用节点API构建罚没交易
	// 当前节点可能没有提供罚没相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_slash`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易（可能需要多方签名）
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要合约地址）
	//   e) 注意：罚没通常需要多方验证，可能需要治理系统支持

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("slash not implemented yet: requires node API support (wes_slash) or contract call")
}

// validateSlashRequest 验证罚没请求
func (s *stakingService) validateSlashRequest(req *SlashRequest) error {
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

