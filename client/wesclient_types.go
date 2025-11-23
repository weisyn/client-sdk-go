package client

import "time"

// OutPoint UTXO 引用点
type OutPoint struct {
	TxID        string
	OutputIndex uint32
}

// LockingCondition 锁定条件（协议级）
type LockingCondition map[string]interface{}

// TxOutput 交易输出（协议级）
type TxOutput map[string]interface{}

// UTXO 未花费交易输出
type UTXO struct {
	OutPoint         OutPoint
	Output           TxOutput
	LockingCondition LockingCondition
}

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeContract ResourceType = "contract"
	ResourceTypeModel    ResourceType = "model"
	ResourceTypeStatic   ResourceType = "static"
)

// ResourceInfo 资源信息（协议级）
type ResourceInfo struct {
	ResourceID        [32]byte            // 32 字节哈希（资源 ID）
	ResourceType      ResourceType         // 'contract' | 'model' | 'static'
	ContentHash       [32]byte            // 32 字节哈希
	Size              int64               // 字节数
	MimeType          string              // 静态资源的 MIME 类型
	LockingConditions []LockingCondition // 原始锁定条件（协议模型）
	CreatedAt         time.Time           // 创建时间（从 TX 推导）

	// 元数据字段（来自链上，可能为空）
	Name             string                 // 资源名称（来自链上 metadata）
	Version          string                 // 版本号（来自链上 metadata）
	Description      string                 // 描述（来自链上 metadata）
	CreatorAddress   string                 // 创建者地址（来自链上 metadata）
	Tags             []string               // 标签（来自链上 custom_attributes 或 metadata）
	CustomAttributes map[string]interface{} // 自定义属性（来自链上 custom_attributes）
}

// ResourceFilters 资源查询过滤器
type ResourceFilters struct {
	ResourceType *ResourceType
	Owner        *[20]byte // 地址字节（前端一般以 base58/hex 表达）
	Limit        int
	Offset       int
}

// TransactionStatus 交易状态
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// TxInput 交易输入（协议级）
type TxInput map[string]interface{}

// TransactionInfo 交易信息（协议级）
type TransactionInfo struct {
	TxID        string
	Status      TransactionStatus
	Inputs      []TxInput
	Outputs     []TxOutput
	BlockHeight *uint64
	Timestamp   time.Time
}

// TransactionFilters 交易查询过滤器
type TransactionFilters struct {
	ResourceID *[32]byte
	TxID       *string
	Limit      int
	Offset     int
}

// EventInfo 事件信息（协议级）
type EventInfo struct {
	EventName   string
	ResourceID  [32]byte
	Data        []byte
	TxID        string
	BlockHeight *uint64
	Timestamp   time.Time
}

// EventFilters 事件查询过滤器
type EventFilters struct {
	ResourceID *[32]byte
	EventName  *string
	Limit      int
	Offset     int
}

// NodeInfo 节点信息
type NodeInfo struct {
	RPCVersion string
	ChainID    string
	BlockHeight uint64
}

// Transaction 交易（序列化前）
// 可以是 hex 字符串或对象
type Transaction interface{}

// SubmitTxResult 交易提交结果
type SubmitTxResult struct {
	TxHash   string
	Accepted bool
	Reason   string
}

