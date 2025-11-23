package transaction

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service Transaction 业务服务接口
type Service interface {
	// GetTransaction 获取交易信息
	GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error)

	// GetTransactionHistory 获取交易历史（支持按交易ID或资源ID查询）
	GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error)

	// SubmitTransaction 提交交易
	SubmitTransaction(ctx context.Context, tx interface{}, wallets ...wallet.Wallet) (*SubmitTxResult, error)
}

// transactionService Transaction 服务实现
type transactionService struct {
	client client.Client
}

// NewService 创建 Transaction 服务
func NewService(client client.Client) Service {
	return &transactionService{
		client: client,
	}
}

// TransactionFilters 交易查询过滤器
type TransactionFilters struct {
	ResourceID *[32]byte // 资源内容哈希（32字节）
	TxID       *string   // 交易哈希（hex 字符串，可带或不带 0x 前缀）
	Limit      int       // 返回数量限制（默认1）
	Offset     int       // 偏移量（默认0）
}

// TransactionInfo 交易信息
type TransactionInfo struct {
	TxID        string                 // 交易哈希
	BlockHeight *uint64                // 区块高度（如果已确认）
	BlockHash   string                 // 区块哈希（如果已确认）
	Status      string                 // 交易状态："pending" | "confirmed" | "failed"
	Inputs      []interface{}          // 交易输入（协议级）
	Outputs     []interface{}          // 交易输出（协议级）
	Timestamp   time.Time              // 时间戳
}

// SubmitTxResult 交易提交结果
type SubmitTxResult struct {
	TxHash   string
	Accepted bool
	Reason   string
}

// GetTransaction 获取交易信息
func (s *transactionService) GetTransaction(ctx context.Context, txID string) (*TransactionInfo, error) {
	params := map[string]interface{}{
		"txId": txID,
	}

	raw, err := s.client.Call(ctx, "wes_getTransactionByHash", []interface{}{params})
	if err != nil {
		return nil, fmt.Errorf("get transaction failed: %w", err)
	}

	// 解码交易信息
	tx, err := decodeTransactionInfo(raw)
	if err != nil {
		return nil, fmt.Errorf("decode transaction info failed: %w", err)
	}

	return tx, nil
}

// GetTransactionHistory 获取交易历史
func (s *transactionService) GetTransactionHistory(ctx context.Context, filters *TransactionFilters) ([]*TransactionInfo, error) {
	// 1. 构建过滤器参数
	filterMap := make(map[string]interface{})
	if filters != nil {
		if filters.ResourceID != nil {
			filterMap["resourceId"] = "0x" + hex.EncodeToString(filters.ResourceID[:])
		}
		if filters.TxID != nil {
			txID := strings.TrimSpace(*filters.TxID)
			if !strings.HasPrefix(txID, "0x") {
				txID = "0x" + txID
			}
			filterMap["txId"] = txID
		}
		if filters.Limit > 0 {
			filterMap["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			filterMap["offset"] = filters.Offset
		}
	}

	// 2. 验证：至少需要 txId 或 resourceId 之一
	hasResourceID := filters != nil && filters.ResourceID != nil
	hasTxID := filters != nil && filters.TxID != nil && *filters.TxID != ""
	if !hasResourceID && !hasTxID {
		return nil, fmt.Errorf("at least one of txId or resourceId is required")
	}

	// 3. 构建请求参数
	params := []interface{}{
		map[string]interface{}{
			"filters": filterMap,
		},
	}

	// 4. 调用 wes_getTransactionHistory API
	result, err := s.client.Call(ctx, "wes_getTransactionHistory", params)
	if err != nil {
		return nil, fmt.Errorf("call wes_getTransactionHistory failed: %w", err)
	}

	// 5. 解析结果数组
	return decodeTransactionArray(result)
}

// SubmitTransaction 提交交易
func (s *transactionService) SubmitTransaction(ctx context.Context, tx interface{}, wallets ...wallet.Wallet) (*SubmitTxResult, error) {
	// 将交易序列化为 hex 字符串
	txHex, err := encodeTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("encode transaction failed: %w", err)
	}

	// 如果提供了 wallet，可能需要先签名（当前实现假设 tx 已经是签名后的交易）
	// TODO: 如果 tx 是未签名交易且提供了 wallet，需要先签名

	result, err := s.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	return &SubmitTxResult{
		TxHash:   result.TxHash,
		Accepted: result.Accepted,
		Reason:   result.Reason,
	}, nil
}

// decodeTransactionInfo 解码交易信息
func decodeTransactionInfo(raw interface{}) (*TransactionInfo, error) {
	itemMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid transaction response format")
	}

	// 提取交易信息
	txID, _ := itemMap["hash"].(string)
	if txID == "" {
		txID, _ = itemMap["txId"].(string)
	}
	if txID == "" {
		txID, _ = itemMap["transactionHash"].(string)
	}

	// 解析 blockHeight
	var blockHeight *uint64
	if bhStr, ok := itemMap["blockHeight"].(string); ok {
		bhStr = strings.TrimPrefix(bhStr, "0x")
		if bh, err := strconv.ParseUint(bhStr, 16, 64); err == nil {
			blockHeight = &bh
		}
	} else if bhNum, ok := itemMap["blockHeight"].(float64); ok {
		bh := uint64(bhNum)
		blockHeight = &bh
	}

	// 解析 blockHash
	blockHash, _ := itemMap["blockHash"].(string)

	// 解析 status
	status := "confirmed"
	if statusStr, ok := itemMap["status"].(string); ok {
		status = statusStr
	} else if blockHeight != nil && *blockHeight > 0 {
		status = "confirmed"
	} else {
		status = "pending"
	}

	// 解析 inputs 和 outputs
	inputs, _ := itemMap["inputs"].([]interface{})
	outputs, _ := itemMap["outputs"].([]interface{})

	// 解析 timestamp
	var timestamp time.Time
	if tsStr, ok := itemMap["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}
	} else if tsNum, ok := itemMap["timestamp"].(float64); ok {
		timestamp = time.Unix(int64(tsNum), 0)
	} else {
		timestamp = time.Now()
	}

	return &TransactionInfo{
		TxID:        txID,
		BlockHeight: blockHeight,
		BlockHash:   blockHash,
		Status:      status,
		Inputs:      inputs,
		Outputs:     outputs,
		Timestamp:   timestamp,
	}, nil
}

// decodeTransactionArray 解码交易数组
func decodeTransactionArray(raw interface{}) ([]*TransactionInfo, error) {
	if raw == nil {
		return []*TransactionInfo{}, nil
	}

	resultArray, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: expected array")
	}

	transactions := make([]*TransactionInfo, 0, len(resultArray))
	for _, item := range resultArray {
		tx, err := decodeTransactionInfo(item)
		if err != nil {
			continue // 跳过无效项
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// encodeTransaction 编码交易为 hex 字符串
func encodeTransaction(tx interface{}) (string, error) {
	// 如果已经是字符串格式
	if txStr, ok := tx.(string); ok {
		if !strings.HasPrefix(txStr, "0x") {
			return "0x" + txStr, nil
		}
		return txStr, nil
	}

	// 对象格式暂不支持（需要 protobuf 序列化）
	return "", fmt.Errorf("transaction object encoding not supported. Please provide a signed transaction hex string")
}


