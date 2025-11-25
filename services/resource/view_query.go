// Package resource 提供资源视图查询实现
package resource

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
)

// ListResources 列出资源列表（新版本，使用 ResourceView）
//
// **架构说明**：
// ListResources 通过调用节点的 `wes_listResources` API 查询资源列表。
//
// **流程**：
// 1. 构建过滤器参数
// 2. 调用 `wes_listResources` API
// 3. 解析返回的 ResourceView 数组
// 4. 返回 ResourceView 列表
func (s *resourceService) ListResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceView, error) {
	// 1. 构建过滤器参数
	filterMap := make(map[string]interface{})
	if filters != nil {
		if filters.ResourceType != "" {
			filterMap["resourceType"] = filters.ResourceType
		}
		if len(filters.Owner) > 0 {
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

	// 3. 调用 wes_listResources API
	result, err := s.client.Call(ctx, "wes_listResources", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_listResources failed: %w", err)
	}

	// 4. 解析结果数组
	resultArray, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: expected array")
	}

	// 5. 转换每个 ResourceView 对象
	views := make([]*ResourceView, 0, len(resultArray))
	for _, item := range resultArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue // 跳过无效项
		}

		view, err := s.parseResourceView(itemMap)
		if err != nil {
			continue // 跳过解析失败的项
		}
		views = append(views, view)
	}

	return views, nil
}

// GetResourceView 获取资源视图（新版本，使用 ResourceView）
//
// **架构说明**：
// GetResourceView 通过调用节点的 `wes_getResource` API 查询资源视图。
func (s *resourceService) GetResourceView(ctx context.Context, contentHash []byte) (*ResourceView, error) {
	// 1. 验证内容哈希
	if len(contentHash) != 32 {
		return nil, fmt.Errorf("content hash must be 32 bytes")
	}

	// 2. 构建查询参数
	contentHashHex := hex.EncodeToString(contentHash)
	params := []interface{}{contentHashHex}

	// 3. 调用 wes_getResource API
	result, err := s.client.Call(ctx, "wes_getResource", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_getResource failed: %w", err)
	}

	// 4. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// 5. 解析 ResourceView
	return s.parseResourceView(resultMap)
}

// GetResourceHistory 获取资源历史
//
// **架构说明**：
// GetResourceHistory 通过调用节点的 `wes_getResourceHistory` API 查询资源历史。
func (s *resourceService) GetResourceHistory(ctx context.Context, contentHash []byte, offset, limit int) (*ResourceHistory, error) {
	// 1. 验证内容哈希
	if len(contentHash) != 32 {
		return nil, fmt.Errorf("content hash must be 32 bytes")
	}

	// 2. 构建查询参数
	contentHashHex := hex.EncodeToString(contentHash)
	params := map[string]interface{}{
		"resourceId": "0x" + contentHashHex,
		"offset":     offset,
		"limit":      limit,
	}

	// 3. 调用 wes_getResourceHistory API
	result, err := s.client.Call(ctx, "wes_getResourceHistory", []interface{}{params})
	if err != nil {
		return nil, fmt.Errorf("call wes_getResourceHistory failed: %w", err)
	}

	// 4. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// 5. 解析 ResourceHistory
	history := &ResourceHistory{}

	// 解析部署交易
	if deployTxMap, ok := resultMap["deployTx"].(map[string]interface{}); ok {
		history.DeployTx = s.parseTxSummary(deployTxMap)
	}

	// 解析升级交易
	if upgradesArray, ok := resultMap["upgrades"].([]interface{}); ok {
		history.Upgrades = make([]*TxSummary, 0, len(upgradesArray))
		for _, upgradeItem := range upgradesArray {
			if upgradeMap, ok := upgradeItem.(map[string]interface{}); ok {
				history.Upgrades = append(history.Upgrades, s.parseTxSummary(upgradeMap))
			}
		}
	}

	// 解析引用统计
	if refSummaryMap, ok := resultMap["referencesSummary"].(map[string]interface{}); ok {
		history.ReferencesSummary = &ReferenceSummary{}
		if total, ok := refSummaryMap["totalReferences"].(float64); ok {
			history.ReferencesSummary.TotalReferences = uint64(total)
		}
		if unique, ok := refSummaryMap["uniqueCallers"].(float64); ok {
			history.ReferencesSummary.UniqueCallers = uint64(unique)
		}
		if lastTime, ok := refSummaryMap["lastReferenceTime"].(float64); ok {
			history.ReferencesSummary.LastReferenceTime = uint64(lastTime)
		}
	}

	return history, nil
}

// parseResourceView 解析 ResourceView
func (s *resourceService) parseResourceView(itemMap map[string]interface{}) (*ResourceView, error) {
	view := &ResourceView{}

	// 解析基础字段
	view.ContentHash, _ = itemMap["contentHash"].(string)
	view.Category, _ = itemMap["category"].(string)
	view.ExecutableType, _ = itemMap["executableType"].(string)
	view.MimeType, _ = itemMap["mimeType"].(string)
	view.Status, _ = itemMap["status"].(string)
	view.Owner, _ = itemMap["owner"].(string)
	view.IsImmutable, _ = itemMap["isImmutable"].(bool)

	// 解析 size
	if sizeStr, ok := itemMap["size"].(string); ok {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			view.Size = size
		}
	} else if sizeNum, ok := itemMap["size"].(float64); ok {
		view.Size = int64(sizeNum)
	}

	// 解析时间戳
	if ts, ok := itemMap["creationTimestamp"].(float64); ok {
		view.CreationTimestamp = uint64(ts)
	}
	if expiryTs, ok := itemMap["expiryTimestamp"].(float64); ok {
		expiry := uint64(expiryTs)
		view.ExpiryTimestamp = &expiry
	}

	// 解析引用计数
	if count, ok := itemMap["currentReferenceCount"].(float64); ok {
		view.CurrentReferenceCount = uint64(count)
	}
	if total, ok := itemMap["totalReferenceTimes"].(float64); ok {
		view.TotalReferenceTimes = uint64(total)
	}

	// 解析区块信息
	view.DeployTxId, _ = itemMap["deployTxId"].(string)
	view.DeployBlockHash, _ = itemMap["deployBlockHash"].(string)
	if height, ok := itemMap["deployBlockHeight"].(float64); ok {
		view.DeployBlockHeight = uint64(height)
	}

	// 解析 OutPoint
	if outPointMap, ok := itemMap["outPoint"].(map[string]interface{}); ok {
		view.OutPoint = &OutPoint{}
		view.OutPoint.TxId, _ = outPointMap["txId"].(string)
		if idx, ok := outPointMap["outputIndex"].(float64); ok {
			view.OutPoint.OutputIndex = uint32(idx)
		}
	}

	return view, nil
}

// parseTxSummary 解析交易摘要
func (s *resourceService) parseTxSummary(txMap map[string]interface{}) *TxSummary {
	summary := &TxSummary{}
	summary.TxId, _ = txMap["txId"].(string)
	summary.BlockHash, _ = txMap["blockHash"].(string)
	if height, ok := txMap["blockHeight"].(float64); ok {
		summary.BlockHeight = uint64(height)
	}
	if ts, ok := txMap["timestamp"].(float64); ok {
		summary.Timestamp = uint64(ts)
	}
	return summary
}

