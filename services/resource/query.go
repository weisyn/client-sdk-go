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

// getResources 获取资源列表实现
//
// **架构说明**：
// GetResources 通过调用节点的 `wes_getResources` API 查询资源列表。
//
// **流程**：
// 1. 构建过滤器参数
// 2. 调用 `wes_getResources` API
// 3. 解析返回的资源数组
// 4. 返回 ResourceInfo 列表
func (s *resourceService) getResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error) {
	// 1. 构建过滤器参数
	filterMap := make(map[string]interface{})
	if filters != nil {
		if filters.ResourceType != "" {
			filterMap["resourceType"] = filters.ResourceType
		}
		if len(filters.Owner) > 0 {
			// owner 需要转换为 hex 字符串（带 0x 前缀）
			filterMap["owner"] = "0x" + hex.EncodeToString(filters.Owner)
		}
		if filters.Limit > 0 {
			filterMap["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			filterMap["offset"] = filters.Offset
		}
	}

	// 2. 构建请求参数
	params := []interface{}{
		map[string]interface{}{
			"filters": filterMap,
		},
	}

	// 3. 调用 wes_getResources API
	result, err := s.client.Call(ctx, "wes_getResources", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_getResources failed: %w", err)
	}

	// 4. 解析结果数组
	resultArray, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: expected array")
	}

	// 5. 转换每个资源对象
	resources := make([]*ResourceInfo, 0, len(resultArray))
	for _, item := range resultArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue // 跳过无效项
		}

		// 提取资源信息（复用 getResource 的解析逻辑）
		contentHashStr, _ := itemMap["content_hash"].(string)
		
		// 优先使用 resourceType 字段（标准化字段），否则回退到 type
		resourceType, _ := itemMap["resourceType"].(string)
		if resourceType == "" {
			resourceType, _ = itemMap["type"].(string)
		}
		
		// 处理 size
		var size int64
		if sizeStr, ok := itemMap["size"].(string); ok {
			size, _ = strconv.ParseInt(sizeStr, 10, 64)
		} else if sizeNum, ok := itemMap["size"].(float64); ok {
			size = int64(sizeNum)
		}
		
		mimeType, _ := itemMap["mime_type"].(string)
		
		// 处理 owner（优先使用 owner 字段，否则从 creator_address 解析）
		var owner []byte
		ownerStr, _ := itemMap["owner"].(string)
		if ownerStr == "" {
			ownerStr, _ = itemMap["creator_address"].(string)
		}
		if ownerStr != "" {
			ownerStr = strings.TrimPrefix(ownerStr, "0x")
			if ownerBytes, err := hex.DecodeString(ownerStr); err == nil {
				owner = ownerBytes
			}
		}

		resources = append(resources, &ResourceInfo{
			ContentHash: contentHashStr,
			Type:        resourceType,
			Size:        size,
			MimeType:    mimeType,
			Owner:       owner,
		})
	}

	return resources, nil
}
