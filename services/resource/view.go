// Package resource 提供资源视图类型定义
package resource

// OutPoint UTXO 位置引用
type OutPoint struct {
	TxId        string `json:"txId"`
	OutputIndex uint32 `json:"outputIndex"`
}

// ResourceView 资源视图（完整的资源信息）
//
// 🎯 **核心职责**：
// 统一的资源视图，包含 UTXO 信息、状态、引用计数等完整信息。
//
// 💡 **设计理念**：
// - 整合 UTXO 视角和元数据视角
// - 包含完整的资源信息
// - 支持前端直接使用
// - 统一使用 camelCase 命名
type ResourceView struct {
	// 资源身份
	ContentHash string `json:"contentHash"`

	// 资源分类
	Category       string `json:"category"`       // EXECUTABLE | STATIC
	ExecutableType string `json:"executableType"` // CONTRACT | AI_MODEL | ...

	// 资源元信息
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`

	// UTXO 视角
	OutPoint          *OutPoint `json:"outPoint"`
	Owner             string    `json:"owner"`
	Status            string    `json:"status"` // ACTIVE | CONSUMED | EXPIRED
	CreationTimestamp uint64    `json:"creationTimestamp"`
	ExpiryTimestamp   *uint64   `json:"expiryTimestamp,omitempty"`
	IsImmutable       bool      `json:"isImmutable"`

	// 使用统计
	CurrentReferenceCount uint64 `json:"currentReferenceCount"`
	TotalReferenceTimes   uint64 `json:"totalReferenceTimes"`

	// 区块信息
	DeployTxId       string `json:"deployTxId"`
	DeployBlockHeight uint64 `json:"deployBlockHeight"`
	DeployBlockHash   string `json:"deployBlockHash"`
}

// ResourceHistory 资源历史记录
type ResourceHistory struct {
	DeployTx *TxSummary          `json:"deployTx"`
	Upgrades []*TxSummary        `json:"upgrades"`
	ReferencesSummary *ReferenceSummary `json:"referencesSummary"`
}

// TxSummary 交易摘要
type TxSummary struct {
	TxId        string `json:"txId"`
	BlockHash   string `json:"blockHash"`
	BlockHeight uint64 `json:"blockHeight"`
	Timestamp   uint64 `json:"timestamp"`
}

// ReferenceSummary 引用统计摘要
type ReferenceSummary struct {
	TotalReferences   uint64 `json:"totalReferences"`
	UniqueCallers     uint64 `json:"uniqueCallers"`
	LastReferenceTime uint64 `json:"lastReferenceTime"`
}

