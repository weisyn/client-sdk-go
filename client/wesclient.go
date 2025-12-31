package client

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

	// ========== 新增 API 方法 ==========

	// 区块查询
	GetBlockByHeight(ctx context.Context, height uint64, fullTx bool) (*BlockInfo, error)
	GetBlockByHash(ctx context.Context, hash []byte, fullTx bool) (*BlockInfo, error)

	// 交易收据
	GetTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error)

	// 费用估算
	EstimateFee(ctx context.Context, tx Transaction) (*FeeEstimate, error)

	// 同步状态
	GetSyncStatus(ctx context.Context) (*SyncStatus, error)

	// 只读合约调用
	ContractCall(ctx context.Context, contractHash []byte, method string, params []uint64, payload []byte) ([]byte, error)

	// 订阅管理
	Unsubscribe(ctx context.Context, subscriptionID string) (bool, error)

	// 合约代币余额
	GetContractTokenBalance(ctx context.Context, address []byte, contractHash []byte, tokenID string) (*TokenBalance, error)

	// AI 模型推理
	CallAIModel(ctx context.Context, req *AIModelCallRequest) (*AIModelCallResult, error)
}

// wesClientImpl WESClient 实现类
type wesClientImpl struct {
	client             Client
	supportsBatchQuery bool
}

// NewWESClient 创建 WESClient 实例
func NewWESClient(config *Config) (WESClient, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &wesClientImpl{
		client:             client,
		supportsBatchQuery: false, // 当前节点不支持批量 RPC，使用并发模拟
	}, nil
}

// NewWESClientFromClient 从现有 Client 创建 WESClient
func NewWESClientFromClient(client Client) WESClient {
	return &wesClientImpl{
		client:             client,
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
	checksum := decoded[21:]         // 校验和

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
		if filters.Offset >= 0 { // 允许 offset 为 0（与 client-sdk-js 保持一致）
			req["offset"] = filters.Offset
		}
	}

	// ✅ 已迁移到 wes_listResources（基于 UTXO 视图）
	raw, err := c.client.Call(ctx, "wes_listResources", []interface{}{map[string]interface{}{"filters": req}})
	if err != nil {
		return nil, wrapRPCError("wes_listResources", err)
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
		if filters.Offset >= 0 { // 允许 offset 为 0（与 client-sdk-js 保持一致）
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
		if filters.Offset >= 0 { // 允许 offset 为 0（与 client-sdk-js 保持一致）
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
		RPCVersion:  "1.0.0", // TODO: 从节点获取实际版本
		ChainID:     chainIDRes.chainID,
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
			sem <- struct{}{}        // 获取信号量
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

// ========== 新增 API 方法实现 ==========

// GetBlockByHeight 按高度查询区块
func (c *wesClientImpl) GetBlockByHeight(ctx context.Context, height uint64, fullTx bool) (*BlockInfo, error) {
	// 构建参数：高度使用十六进制格式
	params := []interface{}{fmt.Sprintf("0x%x", height), fullTx}

	raw, err := c.client.Call(ctx, "wes_getBlockByHeight", params)
	if err != nil {
		return nil, wrapRPCError("wes_getBlockByHeight", err)
	}

	if raw == nil {
		return nil, nil // 区块不存在
	}

	return decodeBlockInfo(raw, fullTx)
}

// GetBlockByHash 按哈希查询区块
func (c *wesClientImpl) GetBlockByHash(ctx context.Context, hash []byte, fullTx bool) (*BlockInfo, error) {
	if len(hash) != 32 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "block hash must be 32 bytes",
		}
	}

	// 构建参数：哈希使用十六进制格式
	hashHex := "0x" + hex.EncodeToString(hash)
	params := []interface{}{hashHex, fullTx}

	raw, err := c.client.Call(ctx, "wes_getBlockByHash", params)
	if err != nil {
		return nil, wrapRPCError("wes_getBlockByHash", err)
	}

	if raw == nil {
		return nil, nil // 区块不存在
	}

	return decodeBlockInfo(raw, fullTx)
}

// decodeBlockInfo 解码区块信息
func decodeBlockInfo(raw interface{}, fullTx bool) (*BlockInfo, error) {
	blockMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid block response format",
		}
	}

	block := &BlockInfo{}

	// 解析高度
	if h, ok := blockMap["height"].(float64); ok {
		block.Height = uint64(h)
	} else if hStr, ok := blockMap["height"].(string); ok {
		cleanHex := strings.TrimPrefix(hStr, "0x")
		if parsed, err := strconv.ParseUint(cleanHex, 16, 64); err == nil {
			block.Height = parsed
		}
	}

	// 解析哈希
	if hashStr, ok := blockMap["hash"].(string); ok {
		if hashBytes, err := hexStringToBytes(hashStr); err == nil {
			block.Hash = hashBytes
		}
	} else if hashStr, ok := blockMap["block_hash"].(string); ok {
		if hashBytes, err := hexStringToBytes(hashStr); err == nil {
			block.Hash = hashBytes
		}
	}

	// 解析父哈希
	if parentStr, ok := blockMap["parent_hash"].(string); ok {
		if parentBytes, err := hexStringToBytes(parentStr); err == nil {
			block.ParentHash = parentBytes
		}
	} else if parentStr, ok := blockMap["parentHash"].(string); ok {
		if parentBytes, err := hexStringToBytes(parentStr); err == nil {
			block.ParentHash = parentBytes
		}
	}

	// 解析时间戳
	if tsStr, ok := blockMap["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			block.Timestamp = t
		}
	} else if ts, ok := blockMap["timestamp"].(float64); ok {
		block.Timestamp = time.Unix(int64(ts), 0)
	}

	// 解析状态根
	if stateRootStr, ok := blockMap["state_root"].(string); ok {
		if stateRootBytes, err := hexStringToBytes(stateRootStr); err == nil {
			block.StateRoot = stateRootBytes
		}
	} else if stateRootStr, ok := blockMap["stateRoot"].(string); ok {
		if stateRootBytes, err := hexStringToBytes(stateRootStr); err == nil {
			block.StateRoot = stateRootBytes
		}
	}

	// 解析难度
	if diff, ok := blockMap["difficulty"].(string); ok {
		block.Difficulty = diff
	}

	// 解析矿工
	if miner, ok := blockMap["miner"].(string); ok {
		block.Miner = miner
	}

	// 解析大小
	if size, ok := blockMap["size"].(float64); ok {
		block.Size = int(size)
	}

	// 解析交易数量
	if txCount, ok := blockMap["tx_count"].(float64); ok {
		block.TxCount = int(txCount)
	}

	// 解析交易列表
	if fullTx {
		if txs, ok := blockMap["transactions"].([]interface{}); ok {
			block.Transactions = txs
			block.TxCount = len(txs)
		}
	} else {
		if txHashes, ok := blockMap["tx_hashes"].([]interface{}); ok {
			for _, h := range txHashes {
				if hStr, ok := h.(string); ok {
					block.TxHashes = append(block.TxHashes, hStr)
				}
			}
			block.TxCount = len(block.TxHashes)
		}
	}

	return block, nil
}

// GetTransactionReceipt 获取交易收据
func (c *wesClientImpl) GetTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error) {
	if txHash == "" {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "transaction hash is required",
		}
	}

	// 确保哈希有 0x 前缀
	if !strings.HasPrefix(txHash, "0x") {
		txHash = "0x" + txHash
	}

	params := []interface{}{txHash}
	raw, err := c.client.Call(ctx, "wes_getTransactionReceipt", params)
	if err != nil {
		return nil, wrapRPCError("wes_getTransactionReceipt", err)
	}

	if raw == nil {
		return nil, nil // 收据不存在
	}

	return decodeTransactionReceipt(raw)
}

// decodeTransactionReceipt 解码交易收据
func decodeTransactionReceipt(raw interface{}) (*TransactionReceipt, error) {
	receiptMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid transaction receipt format",
		}
	}

	receipt := &TransactionReceipt{}

	// 节点真实返回字段（internal/api/jsonrpc/methods/tx.go）：
	// tx_hash, tx_index, block_height, block_hash, status("0x1"/"0x0"), state_root, timestamp, execution_result_hash, statusReason

	if txHash, ok := receiptMap["tx_hash"].(string); ok {
		receipt.TxHash = txHash
	}

	if idx, ok := receiptMap["tx_index"].(float64); ok {
		receipt.TxIndex = uint32(idx)
	}

	if bh, ok := receiptMap["block_height"].(float64); ok {
		receipt.BlockHeight = uint64(bh)
	} else if bhStr, ok := receiptMap["block_height"].(string); ok {
		cleanHex := strings.TrimPrefix(bhStr, "0x")
		if parsed, err := strconv.ParseUint(cleanHex, 16, 64); err == nil {
			receipt.BlockHeight = parsed
		}
	}

	if bhStr, ok := receiptMap["block_hash"].(string); ok && bhStr != "" {
		if bhBytes, err := hexStringToBytes(bhStr); err == nil {
			receipt.BlockHash = bhBytes
		}
	}

	if status, ok := receiptMap["status"].(string); ok {
		receipt.Status = status
	}
	if reason, ok := receiptMap["statusReason"].(string); ok {
		receipt.StatusReason = reason
	}

	if execHash, ok := receiptMap["execution_result_hash"].(string); ok && execHash != "" {
		if b, err := hexStringToBytes(execHash); err == nil {
			receipt.ExecutionResultHash = b
		}
	}

	if sr, ok := receiptMap["state_root"].(string); ok && sr != "" {
		if b, err := hexStringToBytes(sr); err == nil {
			receipt.StateRoot = b
		}
	}

	if ts, ok := receiptMap["timestamp"].(float64); ok {
		receipt.Timestamp = uint64(ts)
	} else if ts, ok := receiptMap["timestamp"].(uint64); ok {
		receipt.Timestamp = ts
	}

	return receipt, nil
}

// EstimateFee 估算交易费用
func (c *wesClientImpl) EstimateFee(ctx context.Context, tx Transaction) (*FeeEstimate, error) {
	// 节点端 wes_estimateFee 要求 params[0] 为“交易草稿对象”而不是已签名 hex
	// 允许调用方传 map[string]interface{} 作为草稿；如果传 string(已签名hex)，直接报错避免误用
	if _, ok := tx.(string); ok {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "wes_estimateFee expects a transaction draft object, not signed tx hex string",
		}
	}

	params := []interface{}{tx}
	raw, err := c.client.Call(ctx, "wes_estimateFee", params)
	if err != nil {
		return nil, wrapRPCError("wes_estimateFee", err)
	}

	return decodeFeeEstimate(raw)
}

// decodeFeeEstimate 解码费用估算结果
func decodeFeeEstimate(raw interface{}) (*FeeEstimate, error) {
	feeMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid fee estimate format",
		}
	}

	fee := &FeeEstimate{}

	// 节点真实返回字段：estimated_fee, fee_rate, num_inputs, num_outputs
	if v, ok := feeMap["estimated_fee"].(float64); ok {
		fee.EstimatedFee = uint64(v)
	} else if v, ok := feeMap["estimated_fee"].(uint64); ok {
		fee.EstimatedFee = v
	}

	if v, ok := feeMap["fee_rate"].(string); ok {
		fee.FeeRate = v
	}

	if v, ok := feeMap["num_inputs"].(float64); ok {
		fee.NumInputs = int(v)
	}
	if v, ok := feeMap["num_outputs"].(float64); ok {
		fee.NumOutputs = int(v)
	}

	return fee, nil
}

// GetSyncStatus 获取节点同步状态
func (c *wesClientImpl) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	raw, err := c.client.Call(ctx, "wes_syncing", nil)
	if err != nil {
		return nil, wrapRPCError("wes_syncing", err)
	}

	// 如果返回 false，表示已同步
	if syncing, ok := raw.(bool); ok && !syncing {
		return &SyncStatus{
			Syncing:  false,
			Progress: 1.0,
		}, nil
	}

	// 否则解析同步状态对象
	return decodeSyncStatus(raw)
}

// decodeSyncStatus 解码同步状态
func decodeSyncStatus(raw interface{}) (*SyncStatus, error) {
	syncMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid sync status format",
		}
	}

	status := &SyncStatus{
		Syncing: true,
	}

	// 解析起始区块
	if startBlock, ok := syncMap["startingBlock"].(string); ok {
		if parsed, err := strconv.ParseUint(strings.TrimPrefix(startBlock, "0x"), 16, 64); err == nil {
			status.StartingBlock = parsed
		}
	}

	// 解析当前区块
	if currentBlock, ok := syncMap["currentBlock"].(string); ok {
		if parsed, err := strconv.ParseUint(strings.TrimPrefix(currentBlock, "0x"), 16, 64); err == nil {
			status.CurrentHeight = parsed
		}
	}

	// 解析最高区块
	if highestBlock, ok := syncMap["highestBlock"].(string); ok {
		if parsed, err := strconv.ParseUint(strings.TrimPrefix(highestBlock, "0x"), 16, 64); err == nil {
			status.HighestHeight = parsed
		}
	}

	// 计算进度
	if status.HighestHeight > 0 {
		status.Progress = float64(status.CurrentHeight) / float64(status.HighestHeight)
	}

	return status, nil
}

// ContractCall 只读合约调用
func (c *wesClientImpl) ContractCall(ctx context.Context, contractHash []byte, method string, params []uint64, payload []byte) ([]byte, error) {
	if len(contractHash) != 32 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "contract hash must be 32 bytes",
		}
	}

	if method == "" {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "method name is required",
		}
	}

	// 节点端只解析 callData["data"]（支持 JSON string / 0xhex(json bytes) / 直接方法名）
	// 为了携带 params/payload，这里统一用 JSON string 形式：{"method":"...","params":[...],"payload":"0x..."}
	callSpec := map[string]interface{}{
		"method": method,
	}
	if len(params) > 0 {
		callSpec["params"] = params
	}
	if len(payload) > 0 {
		callSpec["payload"] = "0x" + hex.EncodeToString(payload)
	}
	specBytes, _ := json.Marshal(callSpec)

	callData := map[string]interface{}{
		"to":   "0x" + hex.EncodeToString(contractHash),
		"data": string(specBytes),
	}

	raw, err := c.client.Call(ctx, "wes_call", []interface{}{callData})
	if err != nil {
		return nil, wrapRPCError("wes_call", err)
	}

	// 解析返回数据
	resultMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid call result format",
		}
	}

	// 提取返回数据
	if returnData, ok := resultMap["return_data"].(string); ok {
		return hexStringToBytes(returnData)
	}

	if returnData, ok := resultMap["returnData"].(string); ok {
		return hexStringToBytes(returnData)
	}

	return nil, nil
}

// Unsubscribe 取消订阅
func (c *wesClientImpl) Unsubscribe(ctx context.Context, subscriptionID string) (bool, error) {
	if subscriptionID == "" {
		return false, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "subscription ID is required",
		}
	}

	params := []interface{}{subscriptionID}
	raw, err := c.client.Call(ctx, "wes_unsubscribe", params)
	if err != nil {
		return false, wrapRPCError("wes_unsubscribe", err)
	}

	// 解析结果
	if result, ok := raw.(bool); ok {
		return result, nil
	}

	return false, nil
}

// GetContractTokenBalance 查询合约代币余额
func (c *wesClientImpl) GetContractTokenBalance(ctx context.Context, address []byte, contractHash []byte, tokenID string) (*TokenBalance, error) {
	if len(address) != 20 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "address must be 20 bytes",
		}
	}

	if len(contractHash) != 32 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "contract hash must be 32 bytes",
		}
	}

	// 将地址转换为 Base58 格式
	addressBase58, err := addressBytesToBase58(address)
	if err != nil {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: fmt.Sprintf("convert address to Base58 failed: %v", err),
			Cause:   err,
		}
	}

	// 构建请求参数
	reqParams := map[string]interface{}{
		"address":      addressBase58,
		"content_hash": hex.EncodeToString(contractHash),
	}

	if tokenID != "" {
		reqParams["token_id"] = tokenID
	}

	raw, err := c.client.Call(ctx, "wes_getContractTokenBalance", []interface{}{reqParams})
	if err != nil {
		return nil, wrapRPCError("wes_getContractTokenBalance", err)
	}

	return decodeTokenBalance(raw)
}

// decodeTokenBalance 解码代币余额
func decodeTokenBalance(raw interface{}) (*TokenBalance, error) {
	balanceMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid token balance format",
		}
	}

	balance := &TokenBalance{}

	// 解析地址
	if addr, ok := balanceMap["address"].(string); ok {
		balance.Address = addr
	}

	// 解析合约哈希
	if ch, ok := balanceMap["content_hash"].(string); ok {
		balance.ContractHash = ch
	}

	// 解析合约地址
	if ca, ok := balanceMap["contract_address"].(string); ok {
		balance.ContractAddress = ca
	}

	// 解析代币 ID
	if tid, ok := balanceMap["token_id"].(string); ok {
		balance.TokenID = tid
	}

	// 解析余额（字符串格式）
	if bal, ok := balanceMap["balance"].(string); ok {
		balance.Balance = bal
	}

	// 解析余额（uint64 格式）
	if balUint64, ok := balanceMap["balance_uint64"].(float64); ok {
		balance.BalanceUint64 = uint64(balUint64)
	}

	// 解析 UTXO 数量
	if utxoCount, ok := balanceMap["utxo_count"].(float64); ok {
		balance.UTXOCount = int(utxoCount)
	}

	// 解析区块高度
	if height, ok := balanceMap["height"].(float64); ok {
		balance.Height = uint64(height)
	}

	return balance, nil
}

// CallAIModel 调用 AI 模型
func (c *wesClientImpl) CallAIModel(ctx context.Context, req *AIModelCallRequest) (*AIModelCallResult, error) {
	if req == nil {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "request is required",
		}
	}

	if len(req.ModelHash) != 32 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "model hash must be 32 bytes",
		}
	}
	if len(req.Inputs) == 0 {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "inputs is required and cannot be empty",
		}
	}
	if !req.ReturnUnsignedTx && strings.TrimSpace(req.PrivateKey) == "" {
		return nil, &WESClientError{
			Code:    WESErrCodeInvalidParams,
			Message: "private_key is required when return_unsigned_tx is false",
		}
	}

	// 构建请求参数
	reqParams := map[string]interface{}{
		"model_hash":         "0x" + hex.EncodeToString(req.ModelHash),
		"inputs":             req.Inputs,
		"return_unsigned_tx": req.ReturnUnsignedTx,
	}

	if strings.TrimSpace(req.PrivateKey) != "" {
		reqParams["private_key"] = req.PrivateKey
	}

	if strings.TrimSpace(req.PaymentToken) != "" {
		reqParams["payment_token"] = req.PaymentToken
	}

	raw, err := c.client.Call(ctx, "wes_callAIModel", []interface{}{reqParams})
	if err != nil {
		return nil, wrapRPCError("wes_callAIModel", err)
	}

	return decodeAIModelCallResult(raw)
}

// decodeAIModelCallResult 解码 AI 模型调用结果
func decodeAIModelCallResult(raw interface{}) (*AIModelCallResult, error) {
	resultMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &WESClientError{
			Code:    WESErrCodeDecodeFailed,
			Message: "invalid AI model call result format",
		}
	}

	result := &AIModelCallResult{}

	// 解析成功标志
	if success, ok := resultMap["success"].(bool); ok {
		result.Success = success
	}

	// tx_hash
	if txHash, ok := resultMap["tx_hash"].(string); ok {
		result.TxHash = txHash
	}
	// unsigned_tx（当 return_unsigned_tx=true）
	if utx, ok := resultMap["unsigned_tx"].(string); ok {
		result.UnsignedTx = utx
	}
	// outputs
	if outputs, exists := resultMap["outputs"]; exists {
		result.Outputs = outputs
	}
	// message
	if msg, ok := resultMap["message"].(string); ok {
		result.Message = msg
	}
	// compute_info（可选）
	if ci, exists := resultMap["compute_info"]; exists {
		result.ComputeInfo = ci
	}

	return result, nil
}
