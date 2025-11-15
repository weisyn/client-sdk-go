package governance

import (
	"bytes"
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// propose 创建提案实现
//
// ⚠️ **当前实现说明**：
// 当前节点没有提供专门的创建提案 JSON-RPC 方法（如 `wes_propose`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点业务服务API构建创建提案交易（MultiKeyLock/ThresholdLock）
//    - 需要节点提供 `wes_propose` JSON-RPC 方法
//    - 或通过合约调用实现（需要 Governance 合约地址）
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/governance/service.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供 `wes_propose` API
// - 需要确认是否通过合约调用实现（需要 Governance 合约地址）
func (s *governanceService) propose(ctx context.Context, req *ProposeRequest, wallets ...wallet.Wallet) (*ProposeResult, error) {
	// 1. 参数验证
	if err := s.validateProposeRequest(req); err != nil {
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

	// 4. TODO: 调用节点API构建创建提案交易
	// 当前节点可能没有提供创建提案相关的 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_propose`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者通过合约调用实现（需要 Governance 合约地址）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("propose not implemented yet: requires node API support (wes_propose) or contract call")
}

// validateProposeRequest 验证提案请求
func (s *governanceService) validateProposeRequest(req *ProposeRequest) error {
	// 1. 验证地址
	if len(req.Proposer) != 20 {
		return fmt.Errorf("proposer address must be 20 bytes")
	}

	// 2. 验证标题
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	// 3. 验证投票期限
	if req.VotingPeriod == 0 {
		return fmt.Errorf("voting period must be greater than 0")
	}

	return nil
}

