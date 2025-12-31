package services

// Config 统一的业务服务配置结构，用于为各个具体 Service 提供合约地址、治理验证者等运行时参数。
//
// **设计目的**：
// - 避免在各个 service 内部硬编码合约地址 / 验证者集合
// - 保持与 WES 协议的解耦：协议层只关心输入输出，业务配置由 SDK 使用方提供
//
// **说明**：
// - 所有字段均为可选，未提供时各 service 可以采用合理的默认行为或返回错误
// - 地址类字段统一使用原始字节切片（20 字节地址 / 32 字节 contentHash）
//
// **使用场景**：
// - v1.0：当前各 service 使用默认/约定方式获取合约地址（功能可用）
// - v1.1+：通过 NewServiceWithConfig 传入配置，支持多合约、多环境场景
type Config struct {
	// Staking 合约 contentHash（32 字节）
	StakingContractHash []byte

	// Vesting 合约 contentHash（32 字节）
	VestingContractHash []byte

	// Escrow 合约 contentHash（32 字节）
	EscrowContractHash []byte

	// 治理相关配置
	Governance GovernanceConfig
}

// GovernanceConfig 治理相关配置
type GovernanceConfig struct {
	// 验证者地址列表（20 字节地址）
	ValidatorAddresses [][]byte

	// ThresholdLock 的门限值（需要多少个验证者签名）
	Threshold uint32
}
