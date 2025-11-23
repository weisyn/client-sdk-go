package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

// WESClient WES 客户端接口
// 提供类型化的 RPC 封装，避免直接使用 Call(method, params)
type WESClient interface {
	// UTXO 操作（以地址为中心）
	// ListUTXOs 按地址查询该地址下的所有 UTXO 列表
	ListUTXOs(ctx context.Context, address []byte) ([]*UTXO, error)

	// 资源操作（封装）
	GetResource(ctx context.Context, resourceID [32]byte) (*ResourceInfo, error)
	GetResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error)

	// 交易操作
	GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error)
	GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error)
	SubmitTransaction(ctx context.Context, tx Transaction) (*SubmitTxResult, error)

	// 事件操作
	GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error)
	SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error)

	// 节点信息
	GetNodeInfo(ctx context.Context) (*NodeInfo, error)

	// 批量能力（利用 utils/batch 封装）
	SupportsBatchQuery() bool
	BatchGetResources(ctx context.Context, resourceIDs [][32]byte) ([]*ResourceInfo, error)

	// 底层通道（不推荐上层直接使用）
	Call(ctx context.Context, method string, params interface{}) (interface{}, error)

	// 连接管理
	Close() error
}

// wesClientImpl WESClient 实现类
type wesClientImpl struct {
	client            Client
	supportsBatchQuery bool
}

// NewWESClient 创建 WESClient 实例
func NewWESClient(config *Config) (WESClient, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &wesClientImpl{
		client:            client,
		supportsBatchQuery: false, // 当前节点不支持批量 RPC，使用并发模拟
	}, nil
}

// NewWESClientFromClient 从现有 Client 创建 WESClient
func NewWESClientFromClient(client Client) WESClient {
	return &wesClientImpl{
		client:            client,
		supportsBatchQuery: false,
	}
}

// SupportsBatchQuery 返回是否支持批量查询
func (c *wesClientImpl) SupportsBatchQuery() bool {
	return c.supportsBatchQuery
}

// ListUTXOs 按地址查询该地址下的所有 UTXO 列表
// 这是节点 API wes_getUTXO 的原生用法，直接匹配节点 API 设计
func (c *wesClientImpl) ListUTXOs(ctx context.Context, address []byte) ([]*UTXO, error) {
	if len(address) != 20 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "address must be 20 bytes",
		}
	}

	// 将地址转换为 Base58 格式（避免导入循环，直接实现）
	addressBase58, err := addressBytesToBase58(address)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: fmt.Sprintf("convert address to Base58 failed: %v", err),
			Cause:   err,
		}
	}

	// 调用节点 API wes_getUTXO(address)
	raw, err := c.client.Call(ctx, "wes_getUTXO", []interface{}{addressBase58})
	if err != nil {
		return nil, wrapRPCError("wes_getUTXO", err)
	}

	// 解析返回的 UTXO 列表
	utxoMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid UTXO response format: expected map",
		}
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid UTXOs format: expected array",
		}
	}

	// 解码每个 UTXO
	utxos := make([]*UTXO, 0, len(utxosArray))
	for _, item := range utxosArray {
		utxo, err := decodeUTXO(item, nil)
		if err != nil {
			// 跳过无法解码的 UTXO，记录错误但继续处理其他 UTXO
			continue
		}
		if utxo != nil {
			utxos = append(utxos, utxo)
		}
	}

	return utxos, nil
}

// addressBytesToBase58 将 20 字节地址转换为 Base58Check 编码（避免导入循环）
func addressBytesToBase58(addressBytes []byte) (string, error) {
	if len(addressBytes) != 20 {
		return "", fmt.Errorf("invalid address length: expected 20 bytes, got %d", len(addressBytes))
	}

	// WES 地址版本字节
	versionByte := byte(0x1C)

	// 构建版本字节 + 地址哈希
	versionedAddress := append([]byte{versionByte}, addressBytes...)

	// 计算校验和（双重 SHA256，取前4字节）
	hash1 := sha256.Sum256(versionedAddress)
	hash2 := sha256.Sum256(hash1[:])
	checksum := hash2[:4]

	// 构建完整地址：版本字节 + 地址哈希 + 校验和
	fullAddress := append(versionedAddress, checksum...)

	// Base58 编码
	base58Addr := base58.Encode(fullAddress)

	return base58Addr, nil
}

// addressBase58ToBytes 将 Base58Check 编码地址转换为 20 字节地址哈希（避免导入循环）
func addressBase58ToBytes(base58Addr string) ([]byte, error) {
	// Base58 解码
	decoded := base58.Decode(base58Addr)

	// 验证长度：版本字节（1）+ 地址哈希（20）+ 校验和（4）= 25 字节
	if len(decoded) != 25 {
		return nil, fmt.Errorf("invalid address length: expected 25 bytes after Base58 decode, got %d", len(decoded))
	}

	// 验证校验和
	versionedAddress := decoded[:21] // 版本字节 + 地址哈希
	checksum := decoded[21:]          // 校验和

	hash1 := sha256.Sum256(versionedAddress)
	hash2 := sha256.Sum256(hash1[:])
	expectedChecksum := hash2[:4]

	// 比较校验和
	if !equalBytes(checksum, expectedChecksum) {
		return nil, fmt.Errorf("invalid checksum")
	}

	// 返回地址哈希（跳过版本字节）
	addressBytes := decoded[1:21]

	return addressBytes, nil
}

// equalBytes 比较两个字节数组是否相等
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetResource 查询单个资源
func (c *wesClientImpl) GetResource(ctx context.Context, resourceID [32]byte) (*ResourceInfo, error) {
	// 节点 API wes_getResource 需要 resourceId 参数（字符串格式，十六进制）
	// 将 resourceID 转换为 hex 字符串
	resourceIDHex := "0x" + hex.EncodeToString(resourceID[:])

	// 节点 API 支持字符串数组格式：["resourceId"]
	raw, err := c.client.Call(ctx, "wes_getResource", []interface{}{resourceIDHex})
	if err != nil {
		return nil, wrapRPCError("wes_getResource", err)
	}

	resource, err := mapWireResourceToDomain(raw.(map[string]interface{}))
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("decode resource info failed: %v", err),
			Cause:   err,
		}
	}

	return resource, nil
}

// GetResources 查询资源列表
func (c *wesClientImpl) GetResources(ctx context.Context, filters *ResourceFilters) ([]*ResourceInfo, error) {
	req := make(map[string]interface{})

	if filters != nil {
		if filters.ResourceType != nil {
			req["resourceType"] = string(*filters.ResourceType)
		}
		if filters.Owner != nil {
			// 将 owner 转换为 hex 字符串
			req["owner"] = "0x" + hex.EncodeToString(filters.Owner[:])
		}
		if filters.Limit > 0 {
			req["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			req["offset"] = filters.Offset
		}
	}

	raw, err := c.client.Call(ctx, "wes_getResources", []interface{}{map[string]interface{}{"filters": req}})
	if err != nil {
		return nil, wrapRPCError("wes_getResources", err)
	}

	wire, err := decodeResourceArray(raw)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("decode resource array failed: %v", err),
			Cause:   err,
		}
	}

	resources := make([]*ResourceInfo, 0, len(wire))
	for _, w := range wire {
		resource, err := mapWireResourceToDomain(w)
		if err != nil {
			return nil, &WESClientError{
				Code:    WESErrCodeDecodeFailed,
				Message: fmt.Sprintf("map wire resource to domain failed: %v", err),
				Cause:   err,
			}
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// GetTransaction 查询交易
func (c *wesClientImpl) GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error) {
	// 节点 API wes_getTransactionByHash 需要字符串参数，而不是对象
	raw, err := c.client.Call(ctx, "wes_getTransactionByHash", []interface{}{txID})
	if err != nil {
		return nil, wrapRPCError("wes_getTransactionByHash", err)
	}

	txMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid transaction response format",
		}
	}

	tx, err := mapWireTransactionToDomain(txMap)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("decode transaction info failed: %v", err),
			Cause:   err,
		}
	}

	return tx, nil
}

// GetTransactionHistory 查询交易历史
func (c *wesClientImpl) GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error) {
	req := make(map[string]interface{})

	if filters != nil {
		if filters.ResourceID != nil {
			req["resourceId"] = "0x" + hex.EncodeToString(filters.ResourceID[:])
		}
		if filters.TxID != nil {
			req["txId"] = *filters.TxID
		}
		if filters.Limit > 0 {
			req["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			req["offset"] = filters.Offset
		}
	}

	raw, err := c.client.Call(ctx, "wes_getTransactionHistory", []interface{}{map[string]interface{}{"filters": req}})
	if err != nil {
		return nil, wrapRPCError("wes_getTransactionHistory", err)
	}

	wire, err := decodeTransactionArray(raw)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("decode transaction array failed: %v", err),
			Cause:   err,
		}
	}

	transactions := make([]*TransactionInfo, 0, len(wire))
	for _, w := range wire {
		tx, err := mapWireTransactionToDomain(w)
		if err != nil {
			return nil, &WESClientError{
				Code:    WESErrCodeDecodeFailed,
				Message: fmt.Sprintf("map wire transaction to domain failed: %v", err),
				Cause:   err,
			}
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// SubmitTransaction 提交交易
func (c *wesClientImpl) SubmitTransaction(ctx context.Context, tx Transaction) (*SubmitTxResult, error) {
	// 将交易序列化为 hex 字符串
	txHex, err := encodeTransaction(tx)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("encode transaction failed: %v", err),
			Cause:   err,
		}
	}

	result, err := c.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, wrapRPCError("wes_sendRawTransaction", err)
	}

	return &SubmitTxResult{
		TxHash:   result.TxHash,
		Accepted: result.Accepted,
		Reason:   result.Reason,
	}, nil
}

// GetEvents 查询事件列表
func (c *wesClientImpl) GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error) {
	req := make(map[string]interface{})

	if filters != nil {
		if filters.ResourceID != nil {
			req["resourceId"] = "0x" + hex.EncodeToString(filters.ResourceID[:])
		}
		if filters.EventName != nil {
			req["eventName"] = *filters.EventName
		}
		if filters.Limit > 0 {
			req["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			req["offset"] = filters.Offset
		}
	}

	raw, err := c.client.Call(ctx, "wes_getEvents", []interface{}{map[string]interface{}{"filters": req}})
	if err != nil {
		return nil, wrapRPCError("wes_getEvents", err)
	}

	wire, err := decodeEventArray(raw)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: fmt.Sprintf("decode event array failed: %v", err),
			Cause:   err,
		}
	}

	events := make([]*EventInfo, 0, len(wire))
	for _, w := range wire {
		event, err := mapWireEventToDomain(w)
		if err != nil {
			return nil, &WESClientError{
				Code:    WESErrCodeDecodeFailed,
				Message: fmt.Sprintf("map wire event to domain failed: %v", err),
				Cause:   err,
			}
		}
		events = append(events, event)
	}

	return events, nil
}

// SubscribeEvents 订阅事件
func (c *wesClientImpl) SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error) {
	// 构建事件过滤器
	eventFilter := &EventFilter{}

	if filters != nil {
		if filters.ResourceID != nil {
			// 将 resourceID 转换为字节数组（EventFilter 使用 []byte）
			eventFilter.To = filters.ResourceID[:]
		}
		if filters.EventName != nil {
			eventFilter.Topics = []string{*filters.EventName}
		}
	}

	// 使用底层 Client.Subscribe
	eventChan, err := c.client.Subscribe(ctx, eventFilter)
	if err != nil {
		return nil, wrapRPCError("wes_subscribeEvents", err)
	}

	// 转换 Event 为 EventInfo
	infoChan := make(chan *EventInfo, 10)
	go func() {
		defer close(infoChan)
		for event := range eventChan {
			info := &EventInfo{
				EventName: event.Topic,
				Data:      event.Data,
				Timestamp: time.Now(), // TODO: 从 event 中提取实际时间戳
			}
			if filters != nil && filters.ResourceID != nil {
				copy(info.ResourceID[:], filters.ResourceID[:])
			}
			infoChan <- info
		}
	}()

	return infoChan, nil
}

// GetNodeInfo 获取节点信息
func (c *wesClientImpl) GetNodeInfo(ctx context.Context) (*NodeInfo, error) {
	// 组合多个 RPC 调用获取节点信息
	type chainIDResult struct {
		chainID string
		err     error
	}
	type blockNumberResult struct {
		blockNumber uint64
		err         error
	}

	chainIDChan := make(chan chainIDResult, 1)
	blockNumberChan := make(chan blockNumberResult, 1)

	go func() {
		chainID, err := c.client.Call(ctx, "wes_chainId", nil)
		if err != nil {
			chainIDChan <- chainIDResult{err: err}
			return
		}
		chainIDStr := "0x1"
		if str, ok := chainID.(string); ok {
			chainIDStr = str
		}
		chainIDChan <- chainIDResult{chainID: chainIDStr}
	}()

	go func() {
		blockNumber, err := c.client.Call(ctx, "wes_blockNumber", nil)
		if err != nil {
			blockNumberChan <- blockNumberResult{err: err}
			return
		}
		var blockNum uint64
		switch v := blockNumber.(type) {
		case string:
			// hex 字符串
			cleanHex := strings.TrimPrefix(v, "0x")
			parsed, err := strconv.ParseUint(cleanHex, 16, 64)
			if err == nil {
				blockNum = parsed
			}
		case float64:
			blockNum = uint64(v)
		case uint64:
			blockNum = v
		}
		blockNumberChan <- blockNumberResult{blockNumber: blockNum}
	}()

	chainIDRes := <-chainIDChan
	blockNumberRes := <-blockNumberChan

	if chainIDRes.err != nil {
		return nil, wrapRPCError("wes_chainId", chainIDRes.err)
	}
	if blockNumberRes.err != nil {
		return nil, wrapRPCError("wes_blockNumber", blockNumberRes.err)
	}

	return &NodeInfo{
		RPCVersion: "1.0.0", // TODO: 从节点获取实际版本
		ChainID:    chainIDRes.chainID,
		BlockHeight: blockNumberRes.blockNumber,
	}, nil
}

// BatchGetResources 批量查询资源（并发调用）
func (c *wesClientImpl) BatchGetResources(ctx context.Context, resourceIDs [][32]byte) ([]*ResourceInfo, error) {
	if len(resourceIDs) == 0 {
		return []*ResourceInfo{}, nil
	}

	// 使用并发控制批量查询
	const concurrency = 5
	sem := make(chan struct{}, concurrency)
	results := make([]*ResourceInfo, len(resourceIDs))
	errs := make([]error, len(resourceIDs))
	var wg sync.WaitGroup

	for i, resourceID := range resourceIDs {
		wg.Add(1)
		go func(idx int, rid [32]byte) {
			defer wg.Done()
			sem <- struct{}{} // 获取信号量
			defer func() { <-sem }() // 释放信号量

			resource, err := c.GetResource(ctx, rid)
			results[idx] = resource
			errs[idx] = err
		}(i, resourceID)
	}

	wg.Wait()

	// 检查是否有错误
	for _, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("batch get resources failed: %w", err)
		}
	}

	return results, nil
}

// Call 底层 RPC 调用
func (c *wesClientImpl) Call(ctx context.Context, method string, params interface{}) (interface{}, error) {
	return c.client.Call(ctx, method, params)
}

// Close 关闭连接
func (c *wesClientImpl) Close() error {
	return c.client.Close()
}

// ========== 解码函数 ==========

// decodeUTXO 解码 UTXO
func decodeUTXO(raw interface{}, expectedOutPoint *OutPoint) (*UTXO, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw UTXO is nil")
	}

	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		// 尝试数组格式
		if rawArray, ok := raw.([]interface{}); ok && len(rawArray) > 0 {
			if rawMap, ok = rawArray[0].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid UTXO response format")
			}
		} else {
			return nil, fmt.Errorf("invalid UTXO response format")
		}
	}

	// 检查是否是 { utxos: [...] } 格式
	if utxos, ok := rawMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
		// 查找匹配的 UTXO
		for _, u := range utxos {
			if uMap, ok := u.(map[string]interface{}); ok {
				if outpointStr, ok := uMap["outpoint"].(string); ok && expectedOutPoint != nil {
					parts := strings.Split(outpointStr, ":")
					if len(parts) == 2 {
						if parts[0] == expectedOutPoint.TxID {
							if idx, err := strconv.ParseUint(parts[1], 10, 32); err == nil && uint32(idx) == expectedOutPoint.OutputIndex {
								rawMap = uMap
								break
							}
						}
					}
				}
			}
		}
	}

	// 解析 outpoint
	var outPoint OutPoint
	if expectedOutPoint != nil {
		outPoint = *expectedOutPoint
	} else {
		if outpointStr, ok := rawMap["outpoint"].(string); ok {
			parts := strings.Split(outpointStr, ":")
			if len(parts) == 2 {
				outPoint.TxID = parts[0]
				if idx, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
					outPoint.OutputIndex = uint32(idx)
				}
			}
		} else {
			if txID, ok := rawMap["txId"].(string); ok {
				outPoint.TxID = txID
			}
			if outputIndex, ok := rawMap["outputIndex"].(float64); ok {
				outPoint.OutputIndex = uint32(outputIndex)
			}
		}
	}

	// 解析 output
	output := make(TxOutput)
	if outputRaw, ok := rawMap["output"].(map[string]interface{}); ok {
		for k, v := range outputRaw {
			output[k] = v
		}
	}

	// 解析 lockingCondition
	var lockingCondition LockingCondition
	if lcRaw, ok := rawMap["lockingCondition"].(map[string]interface{}); ok {
		lockingCondition = lcRaw
	} else if lcRaw, ok := rawMap["locking_condition"].(map[string]interface{}); ok {
		lockingCondition = lcRaw
	} else {
		lockingCondition = make(LockingCondition)
	}

	return &UTXO{
		OutPoint:         outPoint,
		Output:           output,
		LockingCondition: lockingCondition,
	}, nil
}

// decodeResourceArray 解码资源数组
func decodeResourceArray(raw interface{}) ([]map[string]interface{}, error) {
	if raw == nil {
		return []map[string]interface{}{}, nil
	}

	if rawArray, ok := raw.([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(rawArray))
		for _, item := range rawArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			}
		}
		return result, nil
	}

	if rawMap, ok := raw.(map[string]interface{}); ok {
		if resources, ok := rawMap["resources"].([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(resources))
			for _, item := range resources {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
	}

	return []map[string]interface{}{}, nil
}

// mapWireResourceToDomain 映射 wire 资源到域模型
func mapWireResourceToDomain(w map[string]interface{}) (*ResourceInfo, error) {
	// 解析 contentHash
	contentHashStr, ok := w["contentHash"].(string)
	if !ok {
		if ch, ok := w["content_hash"].(string); ok {
			contentHashStr = ch
		} else {
			contentHashStr = "0x00"
		}
	}

	contentHashBytes, err := hexStringToBytes(contentHashStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contentHash: %w", err)
	}

	if len(contentHashBytes) != 32 {
		return nil, fmt.Errorf("contentHash must be 32 bytes")
	}

	var resourceID [32]byte
	copy(resourceID[:], contentHashBytes)

	var contentHash [32]byte
	copy(contentHash[:], contentHashBytes)

	// 解析 resourceType
	typeSource, _ := w["resourceType"].(string)
	if typeSource == "" {
		if t, ok := w["resource_type"].(string); ok {
			typeSource = t
		} else if cat, ok := w["category"].(string); ok {
			typeSource = cat
		} else {
			typeSource = "static"
		}
	}

	resourceType := mapResourceType(typeSource)

	// 解析 createdAt
	var createdAt time.Time
	if createdAtStr, ok := w["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		} else {
			createdAt = time.Now()
		}
	} else if ts, ok := w["createdTimestamp"].(float64); ok {
		createdAt = time.Unix(int64(ts), 0)
	} else {
		createdAt = time.Now()
	}

	// 解析元数据字段
	var name, version, description, creatorAddress string
	if n, ok := w["name"].(string); ok && n != "" {
		name = n
	}
	if v, ok := w["version"].(string); ok && v != "" {
		version = v
	}
	if d, ok := w["description"].(string); ok && d != "" {
		description = d
	}
	if ca, ok := w["creatorAddress"].(string); ok && ca != "" {
		creatorAddress = ca
	} else if ca, ok := w["creator_address"].(string); ok && ca != "" {
		creatorAddress = ca
	}

	// 解析 tags
	var tags []string
	if tagArray, ok := w["tags"].([]interface{}); ok {
		for _, tag := range tagArray {
			if tagStr, ok := tag.(string); ok && tagStr != "" {
				tags = append(tags, tagStr)
			}
		}
	}

	// 解析 customAttributes
	var customAttributes map[string]interface{}
	if ca, ok := w["customAttributes"].(map[string]interface{}); ok {
		customAttributes = ca
	} else if ca, ok := w["custom_attributes"].(map[string]interface{}); ok {
		customAttributes = ca
	}

	// 解析 lockingConditions
	var lockingConditions []LockingCondition
	if lcArray, ok := w["lockingConditions"].([]interface{}); ok {
		for _, lc := range lcArray {
			if lcMap, ok := lc.(map[string]interface{}); ok {
				lockingConditions = append(lockingConditions, lcMap)
			}
		}
	} else if lcArray, ok := w["locking_conditions"].([]interface{}); ok {
		for _, lc := range lcArray {
			if lcMap, ok := lc.(map[string]interface{}); ok {
				lockingConditions = append(lockingConditions, lcMap)
			}
		}
	}

	var size int64
	if s, ok := w["size"].(float64); ok {
		size = int64(s)
	}

	var mimeType string
	if mt, ok := w["mimeType"].(string); ok {
		mimeType = mt
	} else if mt, ok := w["mime_type"].(string); ok {
		mimeType = mt
	}

	return &ResourceInfo{
		ResourceID:        resourceID,
		ResourceType:      resourceType,
		ContentHash:       contentHash,
		Size:              size,
		MimeType:          mimeType,
		LockingConditions: lockingConditions,
		CreatedAt:         createdAt,
		Name:              name,
		Version:           version,
		Description:       description,
		CreatorAddress:    creatorAddress,
		Tags:              tags,
		CustomAttributes:  customAttributes,
	}, nil
}

// decodeTransactionArray 解码交易数组
func decodeTransactionArray(raw interface{}) ([]map[string]interface{}, error) {
	if raw == nil {
		return []map[string]interface{}{}, nil
	}

	if rawArray, ok := raw.([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(rawArray))
		for _, item := range rawArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			}
		}
		return result, nil
	}

	if rawMap, ok := raw.(map[string]interface{}); ok {
		if transactions, ok := rawMap["transactions"].([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(transactions))
			for _, item := range transactions {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
	}

	return []map[string]interface{}{}, nil
}

// mapWireTransactionToDomain 映射 wire 交易到域模型
func mapWireTransactionToDomain(w map[string]interface{}) (*TransactionInfo, error) {
	// 解析 txId
	txID := ""
	if hash, ok := w["hash"].(string); ok {
		txID = hash
	} else if txId, ok := w["txId"].(string); ok {
		txID = txId
	} else if txId, ok := w["tx_id"].(string); ok {
		txID = txId
	} else if txHash, ok := w["txHash"].(string); ok {
		txID = txHash
	} else if txHash, ok := w["tx_hash"].(string); ok {
		txID = txHash
	}

	// 解析 blockHeight
	var blockHeight *uint64
	if bh, ok := w["blockHeight"].(float64); ok {
		h := uint64(bh)
		blockHeight = &h
	} else if bhStr, ok := w["blockHeight"].(string); ok {
		cleanHex := strings.TrimPrefix(bhStr, "0x")
		if parsed, err := strconv.ParseUint(cleanHex, 16, 64); err == nil {
			blockHeight = &parsed
		}
	}

	// 解析 status
	status := TransactionStatusPending
	if statusStr, ok := w["status"].(string); ok {
		status = mapTransactionStatus(statusStr)
	} else if blockHeight != nil && *blockHeight > 0 {
		status = TransactionStatusConfirmed
	}

	// 解析 timestamp
	var timestamp time.Time
	if tsStr, ok := w["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}
	} else if ts, ok := w["timestamp"].(float64); ok {
		timestamp = time.Unix(int64(ts), 0)
	} else if ts, ok := w["creationTimestamp"].(float64); ok {
		timestamp = time.Unix(int64(ts), 0)
	} else {
		timestamp = time.Now()
	}

	// 解析 inputs 和 outputs
	var inputs []TxInput
	if inputsArray, ok := w["inputs"].([]interface{}); ok {
		for _, input := range inputsArray {
			if inputMap, ok := input.(map[string]interface{}); ok {
				inputs = append(inputs, inputMap)
			}
		}
	}

	var outputs []TxOutput
	if outputsArray, ok := w["outputs"].([]interface{}); ok {
		for _, output := range outputsArray {
			if outputMap, ok := output.(map[string]interface{}); ok {
				outputs = append(outputs, outputMap)
			}
		}
	}

	return &TransactionInfo{
		TxID:        txID,
		Status:      status,
		Inputs:      inputs,
		Outputs:     outputs,
		BlockHeight: blockHeight,
		Timestamp:   timestamp,
	}, nil
}

// decodeEventArray 解码事件数组
func decodeEventArray(raw interface{}) ([]map[string]interface{}, error) {
	if raw == nil {
		return []map[string]interface{}{}, nil
	}

	if rawArray, ok := raw.([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(rawArray))
		for _, item := range rawArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			}
		}
		return result, nil
	}

	if rawMap, ok := raw.(map[string]interface{}); ok {
		if events, ok := rawMap["events"].([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(events))
			for _, item := range events {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
	}

	return []map[string]interface{}{}, nil
}

// mapWireEventToDomain 映射 wire 事件到域模型
func mapWireEventToDomain(w map[string]interface{}) (*EventInfo, error) {
	// 解析 eventName
	eventName := ""
	if en, ok := w["eventName"].(string); ok {
		eventName = en
	} else if en, ok := w["event_name"].(string); ok {
		eventName = en
	}

	// 解析 resourceId
	resourceIDStr := "0x00"
	if rid, ok := w["resourceId"].(string); ok {
		resourceIDStr = rid
	} else if rid, ok := w["resource_id"].(string); ok {
		resourceIDStr = rid
	}

	resourceIDBytes, err := hexStringToBytes(resourceIDStr)
	if err != nil || len(resourceIDBytes) != 32 {
		resourceIDBytes = make([]byte, 32)
	}

	var resourceID [32]byte
	copy(resourceID[:], resourceIDBytes)

	// 解析 data
	dataStr := "0x00"
	if d, ok := w["data"].(string); ok {
		dataStr = d
	}

	dataBytes, err := hexStringToBytes(dataStr)
	if err != nil {
		dataBytes = []byte{}
	}

	// 解析 txId
	txID := ""
	if txId, ok := w["txId"].(string); ok {
		txID = txId
	} else if txId, ok := w["tx_id"].(string); ok {
		txID = txId
	} else if txHash, ok := w["txHash"].(string); ok {
		txID = txHash
	} else if txHash, ok := w["tx_hash"].(string); ok {
		txID = txHash
	}

	// 解析 blockHeight
	var blockHeight *uint64
	if bh, ok := w["blockHeight"].(float64); ok {
		h := uint64(bh)
		blockHeight = &h
	} else if bhStr, ok := w["blockHeight"].(string); ok {
		cleanHex := strings.TrimPrefix(bhStr, "0x")
		if parsed, err := strconv.ParseUint(cleanHex, 16, 64); err == nil {
			blockHeight = &parsed
		}
	}

	// 解析 timestamp
	var timestamp time.Time
	if tsStr, ok := w["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}
	} else if ts, ok := w["timestamp"].(float64); ok {
		timestamp = time.Unix(int64(ts), 0)
	} else {
		timestamp = time.Now()
	}

	return &EventInfo{
		EventName:   eventName,
		ResourceID:  resourceID,
		Data:        dataBytes,
		TxID:        txID,
		BlockHeight: blockHeight,
		Timestamp:   timestamp,
	}, nil
}

// encodeTransaction 编码交易为 hex 字符串
func encodeTransaction(tx Transaction) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("transaction is nil")
	}

	// 如果已经是字符串格式
	if txStr, ok := tx.(string); ok {
		if strings.HasPrefix(txStr, "0x") {
			return txStr, nil
		}
		return "0x" + txStr, nil
	}

	// 对象格式暂不支持（需要 protobuf 序列化）
	return "", fmt.Errorf("transaction object encoding not supported. Please provide a signed transaction hex string")
}

// mapResourceType 映射资源类型
func mapResourceType(typeStr string) ResourceType {
	t := strings.ToLower(strings.TrimSpace(typeStr))

	if t == "contract" {
		return ResourceTypeContract
	}
	if t == "model" || t == "aimodel" {
		return ResourceTypeModel
	}
	if t == "static" || t == "file" {
		return ResourceTypeStatic
	}

	if strings.Contains(t, "executable") {
		return ResourceTypeContract
	}

	return ResourceTypeStatic
}

// mapTransactionStatus 映射交易状态
func mapTransactionStatus(status string) TransactionStatus {
	switch strings.ToLower(status) {
	case "confirmed", "success":
		return TransactionStatusConfirmed
	case "failed":
		return TransactionStatusFailed
	case "pending":
	default:
		return TransactionStatusPending
	}
	return TransactionStatusPending
}

// hexStringToBytes 将 hex 字符串转换为字节数组
func hexStringToBytes(hexStr string) ([]byte, error) {
	cleanHex := strings.TrimPrefix(hexStr, "0x")
	return hex.DecodeString(cleanHex)
}

