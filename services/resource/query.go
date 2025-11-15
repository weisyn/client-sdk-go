package resource

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
)

// getResource 获取资源信息实现
//
// ⚠️ **当前实现说明**：
// 当前节点可能没有提供专门的资源查询 JSON-RPC 方法（如 `wes_getResource`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点资源查询 API
//    - 需要节点提供 `wes_getResource` JSON-RPC 方法
//    - 或使用 `wes_getContract` API（如果资源是合约）
//    - 或通过状态查询 API 查询 ResourceOutput
//
// **参考实现**：
// - `internal/api/jsonrpc/methods/tx.go` - `wes_getContract` 实现（参考参数格式）
// - `internal/api/jsonrpc/methods/state.go` - 状态查询实现
//
// **当前限制**：
// - 节点可能没有提供专门的资源查询 API
// - 需要确认是否可以通过其他 API 查询资源信息
func (s *resourceService) getResource(ctx context.Context, contentHash []byte) (*ResourceInfo, error) {
	// 1. 验证内容哈希
	if len(contentHash) == 0 {
		return nil, fmt.Errorf("content hash is required")
	}

	// 2. 构建查询参数
	params := []interface{}{
		hex.EncodeToString(contentHash),
	}

	// 3. 调用JSON-RPC方法
	// 注意：当前节点可能没有提供资源查询的JSON-RPC方法
	// 需要确认是否有对应的API，或者使用适配层
	// TODO: 如果节点没有提供 `wes_getResource`，可能需要：
	//   a) 使用 `wes_getContract` API（如果资源是合约）
	//   b) 或通过状态查询 API 查询 ResourceOutput
	//   c) 或在节点中添加 `wes_getResource` API
	result, err := s.client.Call(ctx, "wes_getResource", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_getResource failed: %w", err)
	}

	// 4. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// 5. 提取资源信息
	contentHashStr, _ := resultMap["content_hash"].(string)
	resourceType, _ := resultMap["type"].(string)
	sizeStr, _ := resultMap["size"].(string)
	mimeType, _ := resultMap["mime_type"].(string)
	ownerStr, _ := resultMap["owner"].(string)

	size, _ := strconv.ParseInt(sizeStr, 10, 64)
	owner, _ := hex.DecodeString(ownerStr)

	return &ResourceInfo{
		ContentHash: contentHashStr,
		Type:        resourceType,
		Size:        size,
		MimeType:    mimeType,
		Owner:       owner,
	}, nil
}

