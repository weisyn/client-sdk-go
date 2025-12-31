package staking

import (
	"context"
	"fmt"

	"github.com/weisyn/client-sdk-go/wallet"
)

// slash 罚没实现
//
// **架构说明**：
// Slash（罚没）是 Staking 系统的风控机制，需要明确的业务规则和合约支持。
//
// **当前状态**：
// - Slash 功能需要治理规则 / Slash 合约支持，属于后续阶段能力
// - 当前 SDK 保留 Slash 接口，但实现为"架构预留、业务未定义"
//
// **未来实现路径**（需要治理规则 / Slash 合约确定后）：
// 1. 如果链上部署了 Slash 合约：
//   - SDK 调用 `wes_callContract` → `method: "slash"`
//   - 参数包括：被罚验证者地址、罚没金额、证据/理由
//   - 合约内部根据治理逻辑构建具体消费的 UTXO & 接收方
//
// 2. 如果通过治理系统实现：
//   - 需要多方签名（ThresholdLock）
//   - 通过 Governance 服务创建 Slash 提案
//
// **参考**：
// - `contract-sdk-go/helpers/staking/slash.go` - 业务逻辑实现（待确定）
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

	// 3. 当前实现：架构预留，业务未定义
	// Slash 需要治理规则 / Slash 合约支持，属于后续阶段能力
	// 当前返回明确的错误，提示需要治理规则 / Slash 合约
	return nil, fmt.Errorf("slash not implemented: requires governance rules or slash contract (architecture reserved, business logic undefined)")
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
