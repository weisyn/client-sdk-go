package resource

import (
	"encoding/hex"
	"fmt"
)

// LockingConditionType 锁定条件类型
type LockingConditionType string

const (
	LockingConditionTypeSingleKey   LockingConditionType = "singleKey"
	LockingConditionTypeMultiKey    LockingConditionType = "multiKey"
	LockingConditionTypeContract    LockingConditionType = "contract"
	LockingConditionTypeDelegation  LockingConditionType = "delegation"
	LockingConditionTypeThreshold   LockingConditionType = "threshold"
	LockingConditionTypeTimeLock   LockingConditionType = "timeLock"
	LockingConditionTypeHeightLock LockingConditionType = "heightLock"
)

// LockingCondition 锁定条件接口（Host ABI 层）
type LockingCondition interface {
	Type() LockingConditionType
	ToProto() (map[string]interface{}, error)
	Validate() error
}

// SingleKeyLockCondition 单密钥锁定条件
type SingleKeyLockCondition struct {
	RequiredAddressHash []byte
	Algorithm           string // "ECDSA_SECP256K1" | "ED25519"
}

func (l *SingleKeyLockCondition) Type() LockingConditionType {
	return LockingConditionTypeSingleKey
}

func (l *SingleKeyLockCondition) ToProto() (map[string]interface{}, error) {
	algorithm := l.Algorithm
	if algorithm == "" {
		algorithm = "ECDSA_SECP256K1"
	}
	return map[string]interface{}{
		"single_key_lock": map[string]interface{}{
			"required_address_hash": hex.EncodeToString(l.RequiredAddressHash),
			"required_algorithm":    algorithm,
			"sighash_type":          "SIGHASH_ALL",
		},
	}, nil
}

func (l *SingleKeyLockCondition) Validate() error {
	if len(l.RequiredAddressHash) != 20 {
		return fmt.Errorf("address hash must be 20 bytes")
	}
	return nil
}

// MultiKeyLockCondition 多密钥锁定条件
type MultiKeyLockCondition struct {
	RequiredSignatures      uint32
	AuthorizedKeys          [][]byte // 公钥列表
	RequireOrderedSignatures bool
}

func (l *MultiKeyLockCondition) Type() LockingConditionType {
	return LockingConditionTypeMultiKey
}

func (l *MultiKeyLockCondition) ToProto() (map[string]interface{}, error) {
	keys := make([]map[string]interface{}, len(l.AuthorizedKeys))
	for i, key := range l.AuthorizedKeys {
		keys[i] = map[string]interface{}{
			"value":     hex.EncodeToString(key),
			"algorithm": "ECDSA_SECP256K1",
		}
	}
	return map[string]interface{}{
		"multi_key_lock": map[string]interface{}{
			"required_signatures":       l.RequiredSignatures,
			"authorized_keys":           keys,
			"required_algorithm":        "ECDSA_SECP256K1",
			"require_ordered_signatures": l.RequireOrderedSignatures,
			"sighash_type":             "SIGHASH_ALL",
		},
	}, nil
}

func (l *MultiKeyLockCondition) Validate() error {
	if l.RequiredSignatures == 0 {
		return fmt.Errorf("required_signatures must be > 0")
	}
	if len(l.AuthorizedKeys) == 0 {
		return fmt.Errorf("authorized_keys cannot be empty")
	}
	if l.RequiredSignatures > uint32(len(l.AuthorizedKeys)) {
		return fmt.Errorf("required_signatures cannot exceed authorized_keys count")
	}
	return nil
}

// TimeLockCondition 时间锁定条件
type TimeLockCondition struct {
	UnlockTimestamp uint64
	BaseLock        LockingCondition
}

func (l *TimeLockCondition) Type() LockingConditionType {
	return LockingConditionTypeTimeLock
}

func (l *TimeLockCondition) ToProto() (map[string]interface{}, error) {
	baseProto, err := l.BaseLock.ToProto()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"time_lock": map[string]interface{}{
			"unlock_timestamp": l.UnlockTimestamp,
			"base_lock":        baseProto,
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
		},
	}, nil
}

func (l *TimeLockCondition) Validate() error {
	if l.BaseLock == nil {
		return fmt.Errorf("base_lock is required")
	}
	return l.BaseLock.Validate()
}

// HeightLockCondition 高度锁定条件
type HeightLockCondition struct {
	UnlockHeight       uint64
	BaseLock           LockingCondition
	ConfirmationBlocks uint32
}

func (l *HeightLockCondition) Type() LockingConditionType {
	return LockingConditionTypeHeightLock
}

func (l *HeightLockCondition) ToProto() (map[string]interface{}, error) {
	baseProto, err := l.BaseLock.ToProto()
	if err != nil {
		return nil, err
	}
	confirmationBlocks := l.ConfirmationBlocks
	if confirmationBlocks == 0 {
		confirmationBlocks = 6 // 默认值
	}
	return map[string]interface{}{
		"height_lock": map[string]interface{}{
			"unlock_height":       l.UnlockHeight,
			"base_lock":           baseProto,
			"confirmation_blocks": confirmationBlocks,
		},
	}, nil
}

func (l *HeightLockCondition) Validate() error {
	if l.BaseLock == nil {
		return fmt.Errorf("base_lock is required")
	}
	return l.BaseLock.Validate()
}

// DelegationLockCondition 委托锁定条件
type DelegationLockCondition struct {
	OriginalOwner        []byte
	AllowedDelegates     [][]byte
	AuthorizedOperations []string
	ExpiryDurationBlocks uint64
	MaxValuePerOperation uint64
}

func (l *DelegationLockCondition) Type() LockingConditionType {
	return LockingConditionTypeDelegation
}

func (l *DelegationLockCondition) ToProto() (map[string]interface{}, error) {
	delegates := make([]string, len(l.AllowedDelegates))
	for i, delegate := range l.AllowedDelegates {
		delegates[i] = hex.EncodeToString(delegate)
	}
	return map[string]interface{}{
		"delegation_lock": map[string]interface{}{
			"original_owner":         hex.EncodeToString(l.OriginalOwner),
			"allowed_delegates":     delegates,
			"authorized_operations": l.AuthorizedOperations,
			"expiry_duration_blocks": l.ExpiryDurationBlocks,
			"max_value_per_operation": l.MaxValuePerOperation,
		},
	}, nil
}

func (l *DelegationLockCondition) Validate() error {
	if len(l.OriginalOwner) != 20 {
		return fmt.Errorf("original_owner must be 20 bytes")
	}
	if len(l.AllowedDelegates) == 0 {
		return fmt.Errorf("allowed_delegates cannot be empty")
	}
	return nil
}

// ContractLockCondition 合约锁定条件
type ContractLockCondition struct {
	ContractAddress    []byte
	RequiredMethod     string
	ParameterSchema    string
	StateRequirements  []string
	MaxExecutionTimeMs uint64
}

func (l *ContractLockCondition) Type() LockingConditionType {
	return LockingConditionTypeContract
}

func (l *ContractLockCondition) ToProto() (map[string]interface{}, error) {
	maxExecTime := l.MaxExecutionTimeMs
	if maxExecTime == 0 {
		maxExecTime = 5000 // 默认5秒
	}
	return map[string]interface{}{
		"contract_lock": map[string]interface{}{
			"contract_address":     hex.EncodeToString(l.ContractAddress),
			"required_method":      l.RequiredMethod,
			"parameter_schema":     l.ParameterSchema,
			"state_requirements":   l.StateRequirements,
			"max_execution_time_ms": maxExecTime,
		},
	}, nil
}

func (l *ContractLockCondition) Validate() error {
	if len(l.ContractAddress) != 20 {
		return fmt.Errorf("contract_address must be 20 bytes")
	}
	if l.RequiredMethod == "" {
		return fmt.Errorf("required_method cannot be empty")
	}
	return nil
}

// ThresholdLockCondition 门限签名锁定条件
type ThresholdLockCondition struct {
	Threshold             uint32
	TotalParties          uint32
	PartyVerificationKeys [][]byte
	SignatureScheme       string
}

func (l *ThresholdLockCondition) Type() LockingConditionType {
	return LockingConditionTypeThreshold
}

func (l *ThresholdLockCondition) ToProto() (map[string]interface{}, error) {
	scheme := l.SignatureScheme
	if scheme == "" {
		scheme = "BLS_THRESHOLD"
	}
	keys := make([]string, len(l.PartyVerificationKeys))
	for i, key := range l.PartyVerificationKeys {
		keys[i] = hex.EncodeToString(key)
	}
	return map[string]interface{}{
		"threshold_lock": map[string]interface{}{
			"threshold":               l.Threshold,
			"total_parties":          l.TotalParties,
			"party_verification_keys": keys,
			"signature_scheme":       scheme,
			"security_level":         256,
		},
	}, nil
}

func (l *ThresholdLockCondition) Validate() error {
	if l.Threshold == 0 {
		return fmt.Errorf("threshold must be > 0")
	}
	if l.TotalParties == 0 {
		return fmt.Errorf("total_parties must be > 0")
	}
	if l.Threshold > l.TotalParties {
		return fmt.Errorf("threshold cannot exceed total_parties")
	}
	if len(l.PartyVerificationKeys) != int(l.TotalParties) {
		return fmt.Errorf("party_verification_keys count must match total_parties")
	}
	return nil
}

// convertLockingConditionsToProto 将 Host ABI 层的 LockingCondition 转换为 proto 格式
func convertLockingConditionsToProto(conditions []LockingCondition) ([]interface{}, error) {
	result := make([]interface{}, 0, len(conditions))
	for _, condition := range conditions {
		protoCondition, err := condition.ToProto()
		if err != nil {
			return nil, fmt.Errorf("failed to convert locking condition: %w", err)
		}
		result = append(result, protoCondition)
	}
	return result, nil
}

// createDefaultSingleKeyLock 创建默认单密钥锁
func createDefaultSingleKeyLock(address []byte) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"single_key_lock": map[string]interface{}{
				"required_address_hash": hex.EncodeToString(address),
				"required_algorithm":    "ECDSA_SECP256K1",
				"sighash_type":          "SIGHASH_ALL",
			},
		},
	}
}

// validateLockingConditions 验证锁定条件有效性
func validateLockingConditions(conditions []LockingCondition, allowContractLockCycles bool) error {
	// 检查 ContractLock 循环依赖
	contractAddresses := make(map[string]bool)
	for _, condition := range conditions {
		if contractLock, ok := condition.(*ContractLockCondition); ok {
			addrHex := hex.EncodeToString(contractLock.ContractAddress)
			if contractAddresses[addrHex] {
				return fmt.Errorf("duplicate contract lock address: %s", addrHex)
			}
			contractAddresses[addrHex] = true
			
			// TODO: 实现循环检测逻辑
			// 检查 contractAddress 是否形成循环引用
			if !allowContractLockCycles {
				// 可以查询链上状态，检查该合约的锁定条件是否又引用了当前合约
				// 这里简化处理，实际应该调用节点 API 检查
			}
		}
		
		// 验证每种锁定条件的参数有效性
		if err := condition.Validate(); err != nil {
			return fmt.Errorf("invalid locking condition: %w", err)
		}
	}
	return nil
}

