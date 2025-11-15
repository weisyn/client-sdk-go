package governance

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// vote 投票实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的投票 JSON-RPC 方法（如 `wes_vote`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建投票交易
//    - 需要节点提供 `wes_vote` JSON-RPC 方法
//    - 或通过合约调用实现（需要 Governance 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/governance/service.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_vote` API
// - 需要确认是否通过合约调用实现（需要 Governance 合约地址）
func (s *governanceService) vote(ctx context.Context, req *VoteRequest, wallets ...wallet.Wallet) (*VoteResult, error) {
	// 1. 参数验证
	if err := s.validateVoteRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.Voter) {
		return nil, fmt.Errorf("wallet address does not match voter address")
	}

	// 4. TODO: 调用节点API构建投票交易
	// 当前节点可能没有提供投票相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_vote`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 Governance 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("vote not implemented yet: requires node API support (wes_vote) or contract call")
}

// validateVoteRequest 验证投票请求
func (s *governanceService) validateVoteRequest(req *VoteRequest) error {
	// 1. 验证地址
	if len(req.Voter) != 20 {
		return fmt.Errorf("voter address must be 20 bytes")
	}

	// 2. 验证提案ID
	if len(req.ProposalID) == 0 {
		return fmt.Errorf("proposal ID is required")
	}

	// 3. 验证投票选择
	if req.Choice < -1 || req.Choice > 1 {
		return fmt.Errorf("choice must be -1 (abstain), 0 (against), or 1 (for)")
	}

	// 4. 验证投票权重
	if req.VoteWeight == 0 {
		return fmt.Errorf("vote weight must be greater than 0")
	}

	return nil
}

// updateParam 更新参数实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的更新参数 JSON-RPC 方法（如 `wes_updateParam`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建更新参数交易
//    - 需要节点提供 `wes_updateParam` JSON-RPC 方法
//    - 或通过合约调用实现（需要 Governance 合约地址）
// 2. 使用钱包签名交易（可能需要多方签名）
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/governance/service.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_updateParam` API
// - 更新参数通常需要治理投票通过，可能需要先创建提案
// - 需要确认是否通过合约调用实现（需要 Governance 合约地址）
func (s *governanceService) updateParam(ctx context.Context, req *UpdateParamRequest, wallets ...wallet.Wallet) (*UpdateParamResult, error) {
	// 1. 参数验证
	if err := s.validateUpdateParamRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.Proposer) {
		return nil, fmt.Errorf("wallet address does not match proposer address")
	}

	// 4. TODO: 调用节点API构建更新参数交易
	// 当前节点可能没有提供更新参数相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_updateParam`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 Governance 合约地址）
	//   e) 注意：更新参数通常需要治理投票通过，可能需要先创建提案

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("update param not implemented yet: requires node API support (wes_updateParam) or contract call")
}

// validateUpdateParamRequest 验证更新参数请求
func (s *governanceService) validateUpdateParamRequest(req *UpdateParamRequest) error {
	// 1. 验证地址
	if len(req.Proposer) != 20 {
		return fmt.Errorf("proposer address must be 20 bytes")
	}

	// 2. 验证参数键
	if req.ParamKey == "" {
		return fmt.Errorf("param key is required")
	}

	// 3. 验证参数值
	if req.ParamValue == "" {
		return fmt.Errorf("param value is required")
	}

	return nil
}

