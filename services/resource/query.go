package resource

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// getResource 获取资源信息实现
//
// **架构说明**：
// GetResource 通过调用节点的 `wes_getResourceByContentHash` API 查询资源元数据。
//
// **流程**：
// 1. 调用 `wes_getResourceByContentHash` API
// 2. 解析返回的资源元数据
// 3. 返回 ResourceInfo
func (s *resourceService) getResource(ctx context.Context, contentHash []byte) (*ResourceInfo, error) {
	// 1. 验证内容哈希
	if len(contentHash) == 0 {
		return nil, fmt.Errorf("content hash is required")
	}

	if len(contentHash) != 32 {
		return nil, fmt.Errorf("content hash must be 32 bytes")
	}

	// 2. 构建查询参数
	contentHashHex := hex.EncodeToString(contentHash)
	params := []interface{}{contentHashHex}

	// 3. 调用 wes_getResourceByContentHash API
	result, err := s.client.Call(ctx, "wes_getResourceByContentHash", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_getResourceByContentHash failed: %w", err)
	}

	// 4. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// 5. 提取资源信息
	contentHashStr, _ := resultMap["content_hash"].(string)
	resourceType, _ := resultMap["type"].(string)
	
	// 处理 size（可能是字符串或数字）
	var size int64
	if sizeStr, ok := resultMap["size"].(string); ok {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	} else if sizeNum, ok := resultMap["size"].(float64); ok {
		size = int64(sizeNum)
	}
	
	mimeType, _ := resultMap["mime_type"].(string)
	ownerStr, _ := resultMap["owner"].(string)
	
	var owner []byte
	if ownerStr != "" {
		ownerStr = strings.TrimPrefix(ownerStr, "0x")
		if ownerBytes, err := hex.DecodeString(ownerStr); err == nil {
			owner = ownerBytes
		}
	}

	return &ResourceInfo{
		ContentHash: contentHashStr,
		Type:        resourceType,
		Size:        size,
		MimeType:    mimeType,
		Owner:       owner,
	}, nil
}
