package utils

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// BuildPayloadOptions Payload 构建选项
// 遵循 WES ABI 规范：weisyn.git/docs/components/core/ispc/abi-and-payload.md
type BuildPayloadOptions struct {
	// 保留字段（Reserved Fields）
	IncludeFrom    bool   // 是否包含调用者地址（from）
	From           []byte // 调用者地址（20字节）
	IncludeTo      bool   // 是否包含接收者地址（to）
	To             []byte // 接收者地址（20字节）
	IncludeAmount  bool   // 是否包含金额（amount）
	Amount         uint64 // 转账金额
	IncludeTokenID bool   // 是否包含代币ID（token_id）
	TokenID        []byte // 代币ID（32字节）
	
	// 扩展字段（Extension Fields）- 方法参数
	MethodParams map[string]interface{} // 方法参数（键值对）
}

// BuildPayload 构建合约调用 payload JSON 对象
// 遵循 WES ABI 规范：保留字段 + 扩展字段
func BuildPayload(options BuildPayloadOptions) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	// 1. 添加保留字段（根据选项）
	if options.IncludeFrom && len(options.From) > 0 {
		if len(options.From) != 20 {
			return nil, fmt.Errorf("from address must be 20 bytes, got %d", len(options.From))
		}
		payload["from"] = "0x" + hex.EncodeToString(options.From)
	}

	if options.IncludeTo && len(options.To) > 0 {
		if len(options.To) != 20 {
			return nil, fmt.Errorf("to address must be 20 bytes, got %d", len(options.To))
		}
		payload["to"] = "0x" + hex.EncodeToString(options.To)
	}

	if options.IncludeAmount {
		payload["amount"] = fmt.Sprintf("%d", options.Amount)
	}

	if options.IncludeTokenID && len(options.TokenID) > 0 {
		if len(options.TokenID) != 32 {
			return nil, fmt.Errorf("token_id must be 32 bytes, got %d", len(options.TokenID))
		}
		// 根据 WES ABI 规范，使用下划线命名：token_id
		payload["token_id"] = "0x" + hex.EncodeToString(options.TokenID)
	}

	// 2. 添加扩展字段（方法参数）
	// 根据 WES ABI 规范，方法参数作为扩展字段，键名不得与保留字段冲突
	for key, value := range options.MethodParams {
		// 检查是否与保留字段冲突
		reservedFields := []string{"from", "to", "amount", "token_id"}
		for _, reserved := range reservedFields {
			if key == reserved {
				return nil, fmt.Errorf("parameter name '%s' conflicts with reserved field", key)
			}
		}
		payload[key] = value
	}

	return payload, nil
}

// BuildAndEncodePayload 构建并编码 payload（JSON + Base64）
// 这是主要入口函数，用于构建符合 WES ABI 规范的 payload
func BuildAndEncodePayload(options BuildPayloadOptions) (string, error) {
	// 1. 构建 payload JSON 对象
	payload, err := BuildPayload(options)
	if err != nil {
		return "", fmt.Errorf("build payload failed: %w", err)
	}

	// 2. 序列化为 JSON 字符串
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload failed: %w", err)
	}

	// 3. Base64 编码
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	return payloadBase64, nil
}

