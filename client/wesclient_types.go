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
	ResourceID        [32]byte           // 32 字节哈希（资源 ID）
	ResourceType      ResourceType       // 'contract' | 'model' | 'static'
	ContentHash       [32]byte           // 32 字节哈希
	Size              int64              // 字节数
	MimeType          string             // 静态资源的 MIME 类型
	LockingConditions []LockingCondition // 原始锁定条件（协议模型）
	CreatedAt         time.Time          // 创建时间（从 TX 推导）

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
	RPCVersion  string
	ChainID     string
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

// ========== 新增类型定义（API 补齐） ==========

// BlockInfo 区块信息
type BlockInfo struct {
	Height       uint64        // 区块高度
	Hash         []byte        // 区块哈希（32字节）
	ParentHash   []byte        // 父区块哈希（32字节）
	Timestamp    time.Time     // 区块时间戳
	StateRoot    []byte        // 状态根（32字节）
	Difficulty   string        // 难度
	Miner        string        // 矿工地址
	Size         int           // 区块大小（字节）
	TxHashes     []string      // 交易哈希列表（fullTx=false 时）
	Transactions []interface{} // 完整交易列表（fullTx=true 时）
	TxCount      int           // 交易数量
}

// TransactionReceipt 交易收据
type TransactionReceipt struct {
	TxHash              string // 交易哈希（0x + 64hex）
	TxIndex             uint32 // tx_index
	BlockHeight         uint64 // block_height
	BlockHash           []byte // block_hash（32字节）
	Status              string // status："0x1" | "0x0"
	StatusReason        string // statusReason（可选）
	ExecutionResultHash []byte // execution_result_hash（可选，32字节）
	StateRoot           []byte // state_root（可选，32字节）
	Timestamp           uint64 // timestamp（秒）
}

// FeeEstimate 费用估算结果
type FeeEstimate struct {
	EstimatedFee uint64 // estimated_fee
	FeeRate      string // fee_rate（例如 "3 bps (0.03%)"）
	NumInputs    int    // num_inputs
	NumOutputs   int    // num_outputs
}

// SyncStatus 同步状态
type SyncStatus struct {
	Syncing       bool    // 是否正在同步
	CurrentHeight uint64  // 当前高度
	HighestHeight uint64  // 网络最高高度
	StartingBlock uint64  // 同步起始区块
	Progress      float64 // 同步进度（0-1）
}

// TokenBalance 代币余额
type TokenBalance struct {
	Address         string // 查询的地址
	ContractHash    string // 合约内容哈希
	ContractAddress string // 合约地址
	TokenID         string // 代币 ID
	Balance         string // 余额（字符串格式，支持大数）
	BalanceUint64   uint64 // 余额（uint64 格式，可能溢出）
	UTXOCount       int    // UTXO 数量
	Height          uint64 // 查询时的区块高度
}

// AIModelCallRequest AI 模型调用请求
type AIModelCallRequest struct {
	PrivateKey       string                   // 可选：return_unsigned_tx=false 时必需
	ModelHash        []byte                   // 模型内容哈希（32字节）
	Inputs           []map[string]interface{} // 张量输入列表（与节点 API 对齐）
	ReturnUnsignedTx bool                     // true 时仅返回 unsigned_tx，不提交
	PaymentToken     string                   // 可选：支付代币（Phase 3）
}

// AIModelCallResult AI 模型调用结果
type AIModelCallResult struct {
	Success     bool        // success
	TxHash      string      // tx_hash（注意：节点在不同分支可能返回 0x 前缀或不带前缀）
	UnsignedTx  string      // unsigned_tx（hex，不带 0x）
	Outputs     interface{} // outputs（推理结果张量列表）
	Message     string      // message
	ComputeInfo interface{} // compute_info（可选）
}
