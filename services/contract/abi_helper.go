// Package contract provides contract service implementation.
//
// ABI Helper 统一封装 payload 构建逻辑
// 规范来源：weisyn.git/docs/components/core/ispc/abi-and-payload.md

package contract

import (
	"fmt"

	"github.com/weisyn/client-sdk-go/utils"
)

// buildCallPayload 构建合约调用 payload
//
// 统一封装 payload 构建逻辑，确保符合 WES ABI 规范
// 字段命名严格对齐 abi-and-payload.md 与 JS helper（from/to/amount/token_id）
func buildCallPayload(req *CallContractRequest) (string, error) {
	payloadOptions := utils.BuildPayloadOptions{
		IncludeFrom: true,
		From:        req.From,
	}

	if req.Amount != nil {
		payloadOptions.IncludeAmount = true
		payloadOptions.Amount = *req.Amount
	}

	if len(req.TokenID) > 0 {
		payloadOptions.IncludeTokenID = true
		payloadOptions.TokenID = req.TokenID
	}

	// 方法参数作为扩展字段（根据 WES ABI 规范，参数通过 payload 传递）
	// 注意：这里简化处理，将 args 数组转换为键值对
	// 如果有 ABI 信息，应该使用参数名作为键
	methodParams := make(map[string]interface{})
	for i, arg := range req.Args {
		methodParams[fmt.Sprintf("arg%d", i)] = arg
	}
	payloadOptions.MethodParams = methodParams

	return utils.BuildAndEncodePayload(payloadOptions)
}

// buildQueryPayload 构建合约查询 payload
//
// 查询操作不需要保留字段，只需要方法参数作为扩展字段
func buildQueryPayload(req *QueryContractRequest) (string, error) {
	payloadOptions := utils.BuildPayloadOptions{}

	// 方法参数作为扩展字段
	methodParams := make(map[string]interface{})
	for i, arg := range req.Args {
		methodParams[fmt.Sprintf("arg%d", i)] = arg
	}
	payloadOptions.MethodParams = methodParams

	return utils.BuildAndEncodePayload(payloadOptions)
}
