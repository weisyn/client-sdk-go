package permission

// TransferOwnershipIntent 所有权转移意图
type TransferOwnershipIntent struct {
	ResourceID      string // txId:outputIndex
	NewOwnerAddress string // Base58 地址或 hex 地址
	Memo            string // 可选备注
}

// UpdateCollaboratorsIntent 协作者/白名单管理意图
type UpdateCollaboratorsIntent struct {
	ResourceID         string   // txId:outputIndex
	RequiredSignatures uint32   // M
	Collaborators      []string // 授权地址列表（Base58 或 hex）
}

// GrantDelegationIntent 临时授权意图
type GrantDelegationIntent struct {
	ResourceID           string   // txId:outputIndex
	DelegateAddress      string   // 被委托者地址
	Operations           []string // 授权操作类型: "reference", "execute", "query", "consume", "transfer", "stake", "vote"
	ExpiryBlocks         uint64   // 过期区块数（0 = 永不过期）
	MaxValuePerOperation *uint64  // 单次操作最大价值（可选）
}

// SetTimeOrHeightLockIntent 时间/高度锁意图
type SetTimeOrHeightLockIntent struct {
	ResourceID      string  // txId:outputIndex
	UnlockTimestamp *uint64 // Unix 秒（可选）
	UnlockHeight    *uint64 // 区块高度（可选）
}

// UnsignedTransaction 未签名交易（包含 draft 和签名信息）
type UnsignedTransaction struct {
	Draft      map[string]interface{} // 交易草稿（用于签名）
	InputIndex uint32                 // 需要签名的输入索引
}
