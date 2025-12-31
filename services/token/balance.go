package token

import (
	"context"
	"fmt"
	"strconv"

	"github.com/weisyn/client-sdk-go/utils"
)

// getBalance 查询余额实现
func (s *tokenService) getBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error) {
	// 1. 验证地址
	if len(address) != 20 {
		return 0, fmt.Errorf("address must be 20 bytes")
	}

	// 2. 将地址转换为 Base58 格式
	addressBase58, err := utils.AddressBytesToBase58(address)
	if err != nil {
		return 0, fmt.Errorf("address conversion failed: %w", err)
	}

	// 3. 构建查询参数
	params := []interface{}{
		addressBase58,
		"latest", // blockParameter: "latest" | "pending" | blockNumber
	}

	// 4. 调用JSON-RPC方法
	result, err := s.client.Call(ctx, "wes_getBalance", params)
	if err != nil {
		return 0, fmt.Errorf("call wes_getBalance failed: %w", err)
	}

	// 5. 解析结果 - wes_getBalance 返回包含 balance 字段的对象
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid response format: expected map, got %T", result)
	}

	balanceStr, ok := resultMap["balance"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid response format: balance field not found or not a string")
	}

	// 6. 转换为uint64（balance 是十六进制字符串，如 "0x4a817c800"）
	// 移除 0x 前缀
	if len(balanceStr) > 2 && balanceStr[:2] == "0x" {
		balanceStr = balanceStr[2:]
	}
	balance, err := strconv.ParseUint(balanceStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse balance failed: %w", err)
	}

	return balance, nil
}
