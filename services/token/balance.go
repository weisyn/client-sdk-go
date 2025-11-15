package token

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
)

// getBalance 查询余额实现
func (s *tokenService) getBalance(ctx context.Context, address []byte, tokenID []byte) (uint64, error) {
	// 1. 验证地址
	if len(address) != 20 {
		return 0, fmt.Errorf("address must be 20 bytes")
	}

	// 2. 构建查询参数
	params := []interface{}{
		hex.EncodeToString(address),
		"latest", // blockParameter: "latest" | "pending" | blockNumber
	}

	// 3. 调用JSON-RPC方法
	result, err := s.client.Call(ctx, "wes_getBalance", params)
	if err != nil {
		return 0, fmt.Errorf("call wes_getBalance failed: %w", err)
	}

	// 4. 解析结果
	balanceStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("invalid response format")
	}

	// 5. 转换为uint64
	balance, err := strconv.ParseUint(balanceStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse balance failed: %w", err)
	}

	return balance, nil
}

