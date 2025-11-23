package permission

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/utils"
)

// BuildTransferOwnershipTx 构建所有权转移交易
//
// **流程**：
// 1. 查询当前资源 UTXO
// 2. 解析资源 ID（txId:outputIndex）
// 3. 获取当前资源的锁定条件和内容
// 4. 构建新的锁定条件（SingleKeyLock 指向新所有者）
// 5. 构建交易草稿（Draft）
//
// **注意**：
// - 资源内容不变，只改变锁定条件
// - 需要当前所有者签名才能消费旧 UTXO
func BuildTransferOwnershipTx(
	ctx context.Context,
	client client.Client,
	intent TransferOwnershipIntent,
) (*UnsignedTransaction, error) {
	// 1. 解析资源 ID
	parts := strings.Split(intent.ResourceID, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resourceId format: %s. Expected format: txId:outputIndex", intent.ResourceID)
	}
	txId := parts[0]
	outputIndexStr := parts[1]
	outputIndex, err := strconv.ParseUint(outputIndexStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid outputIndex: %w", err)
	}

	// 2. 查询当前资源 UTXO
	utxoParams := map[string]interface{}{
		"txId":        txId,
		"outputIndex": outputIndex,
	}
	var utxoResult interface{}
	utxoResult, err = client.Call(ctx, "wes_getUTXO", []interface{}{utxoParams})
	if err != nil {
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "NOT_FOUND") {
			return nil, fmt.Errorf("resource UTXO not found or already spent: %s. The resource may have been transferred or consumed", intent.ResourceID)
		}
		return nil, fmt.Errorf("failed to query UTXO: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		// 尝试数组格式
		if utxoArray, ok := utxoResult.([]interface{}); ok && len(utxoArray) > 0 {
			if utxoMap, ok = utxoArray[0].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid UTXO response format")
			}
		} else {
			return nil, fmt.Errorf("invalid UTXO response format")
		}
	}

	// 解析 UTXO 数据
	var utxo map[string]interface{}
	if utxos, ok := utxoMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
		utxo, ok = utxos[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid UTXO format")
		}
	} else {
		utxo = utxoMap
	}

	output, ok := utxo["output"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource UTXO not found: %s. The UTXO may have been spent or the resource ID is incorrect", intent.ResourceID)
	}

	resourceOutput, ok := output["resource_output"].(map[string]interface{})
	if !ok {
		// 尝试 resource 字段
		if resourceOutput, ok = output["resource"].(map[string]interface{}); !ok {
			return nil, fmt.Errorf("resource output not found in UTXO: %s", intent.ResourceID)
		}
	}

	// 3. 转换新所有者地址为 hex
	newOwnerAddressHex, err := utils.AddressBase58ToHex(intent.NewOwnerAddress)
	if err != nil {
		// 尝试 hex 格式
		if strings.HasPrefix(intent.NewOwnerAddress, "0x") {
			newOwnerAddressHex = intent.NewOwnerAddress
		} else if len(intent.NewOwnerAddress) == 40 {
			newOwnerAddressHex = "0x" + intent.NewOwnerAddress
		} else {
			return nil, fmt.Errorf("invalid address format: %s: %w", intent.NewOwnerAddress, err)
		}
	}

	// 4. 构建新的锁定条件（SingleKeyLock）
	newLockingConditions := []map[string]interface{}{
		{
			"single_key_lock": map[string]interface{}{
				"required_address_hash": strings.TrimPrefix(newOwnerAddressHex, "0x"),
				"required_algorithm":    "ECDSA_SECP256K1",
				"sighash_type":          "SIGHASH_ALL",
			},
		},
	}

	// 5. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txId,
				"output_index":      outputIndex,
				"is_reference_only": false, // 消费原资源 UTXO
			},
		},
		"outputs": []map[string]interface{}{
			{
				"owner": strings.TrimPrefix(newOwnerAddressHex, "0x"),
				"output_type": "resource",
				"resource_output": map[string]interface{}{
					"resource":            resourceOutput["resource"],
					"creation_timestamp":  resourceOutput["creation_timestamp"],
					"storage_strategy":    resourceOutput["storage_strategy"],
					"is_immutable":        resourceOutput["is_immutable"],
				},
				"locking_conditions": newLockingConditions,
			},
		},
		"metadata": map[string]interface{}{
			"operation": "transfer_ownership",
			"memo":      intent.Memo,
		},
	}

	return &UnsignedTransaction{
		Draft:      draft,
		InputIndex: 0, // 第一个输入需要签名
	}, nil
}

// BuildUpdateCollaboratorsTx 构建协作者管理交易
//
// **流程**：
// 1. 查询当前资源 UTXO
// 2. 解析当前锁定条件（可能是 SingleKey 或 MultiKey）
// 3. 合并现有协作者和新协作者，构建新的 MultiKeyLock
// 4. 构建交易草稿
func BuildUpdateCollaboratorsTx(
	ctx context.Context,
	client client.Client,
	intent UpdateCollaboratorsIntent,
) (*UnsignedTransaction, error) {
	// 1. 解析资源 ID
	parts := strings.Split(intent.ResourceID, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resourceId format: %s. Expected format: txId:outputIndex", intent.ResourceID)
	}
	txId := parts[0]
	outputIndexStr := parts[1]
	outputIndex, err := strconv.ParseUint(outputIndexStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid outputIndex: %w", err)
	}

	// 2. 查询当前资源 UTXO
	utxoParams := map[string]interface{}{
		"txId":        txId,
		"outputIndex": outputIndex,
	}
	var utxoResult interface{}
	utxoResult, err = client.Call(ctx, "wes_getUTXO", []interface{}{utxoParams})
	if err != nil {
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "NOT_FOUND") {
			return nil, fmt.Errorf("resource UTXO not found or already spent: %s. The resource may have been transferred or consumed", intent.ResourceID)
		}
		return nil, fmt.Errorf("failed to query UTXO: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		if utxoArray, ok := utxoResult.([]interface{}); ok && len(utxoArray) > 0 {
			if utxoMap, ok = utxoArray[0].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid UTXO response format")
			}
		} else {
			return nil, fmt.Errorf("invalid UTXO response format")
		}
	}

	var utxo map[string]interface{}
	if utxos, ok := utxoMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
		utxo, ok = utxos[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid UTXO format")
		}
	} else {
		utxo = utxoMap
	}

	output, ok := utxo["output"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource UTXO not found: %s. The UTXO may have been spent or the resource ID is incorrect", intent.ResourceID)
	}

	resourceOutput, ok := output["resource_output"].(map[string]interface{})
	if !ok {
		if resourceOutput, ok = output["resource"].(map[string]interface{}); !ok {
			return nil, fmt.Errorf("resource output not found in UTXO: %s", intent.ResourceID)
		}
	}

	// 3. 解析当前锁定条件
	currentLockingConditions, _ := output["locking_conditions"].([]interface{})
	var existingAuthorizedKeys []map[string]interface{}

	for _, condRaw := range currentLockingConditions {
		condition, ok := condRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if singleKey, ok := condition["single_key_lock"].(map[string]interface{}); ok {
			addrHash, _ := singleKey["required_address_hash"].(string)
			existingAuthorizedKeys = append(existingAuthorizedKeys, map[string]interface{}{
				"value":     strings.TrimPrefix(addrHash, "0x"),
				"algorithm": "ECDSA_SECP256K1",
			})
		} else if multiKey, ok := condition["multi_key_lock"].(map[string]interface{}); ok {
			if keys, ok := multiKey["authorized_keys"].([]interface{}); ok {
				for _, keyRaw := range keys {
					key, ok := keyRaw.(map[string]interface{})
					if !ok {
						if keyStr, ok := keyRaw.(string); ok {
							existingAuthorizedKeys = append(existingAuthorizedKeys, map[string]interface{}{
								"value":     strings.TrimPrefix(keyStr, "0x"),
								"algorithm": "ECDSA_SECP256K1",
							})
						}
						continue
					}
					keyValue, _ := key["value"].(string)
					if keyValue == "" {
						continue
					}
					existingAuthorizedKeys = append(existingAuthorizedKeys, map[string]interface{}{
						"value":     strings.TrimPrefix(keyValue, "0x"),
						"algorithm": "ECDSA_SECP256K1",
					})
				}
			}
			break
		}
	}

	// 4. 处理新协作者地址/公钥
	var newAuthorizedKeys []map[string]interface{}
	for _, collaborator := range intent.Collaborators {
		var keyValue string
		if strings.HasPrefix(collaborator, "0x") {
			keyValue = strings.TrimPrefix(collaborator, "0x")
		} else {
			// 尝试 Base58 转 hex
			hexAddr, err := utils.AddressBase58ToHex(collaborator)
			if err == nil {
				keyValue = strings.TrimPrefix(hexAddr, "0x")
			} else {
				keyValue = collaborator
			}
		}
		newAuthorizedKeys = append(newAuthorizedKeys, map[string]interface{}{
			"value":     keyValue,
			"algorithm": "ECDSA_SECP256K1",
		})
	}

	// 5. 合并现有和新协作者（去重）
	allKeys := make([]map[string]interface{}, 0)
	keyMap := make(map[string]bool)
	for _, key := range existingAuthorizedKeys {
		value := strings.ToLower(key["value"].(string))
		if !keyMap[value] {
			allKeys = append(allKeys, key)
			keyMap[value] = true
		}
	}
	for _, key := range newAuthorizedKeys {
		value := strings.ToLower(key["value"].(string))
		if !keyMap[value] {
			allKeys = append(allKeys, key)
			keyMap[value] = true
		}
	}

	// 验证 requiredSignatures
	if intent.RequiredSignatures > uint32(len(allKeys)) {
		return nil, fmt.Errorf("requiredSignatures (%d) cannot exceed number of authorized keys (%d)", intent.RequiredSignatures, len(allKeys))
	}
	if intent.RequiredSignatures < 1 {
		return nil, fmt.Errorf("requiredSignatures must be at least 1")
	}

	// 6. 构建新的 MultiKeyLock
	newLockingConditions := []map[string]interface{}{
		{
			"multi_key_lock": map[string]interface{}{
				"required_signatures":       intent.RequiredSignatures,
				"authorized_keys":          allKeys,
				"required_algorithm":       "ECDSA_SECP256K1",
				"require_ordered_signatures": false,
				"sighash_type":             "SIGHASH_ALL",
			},
		},
	}

	// 7. 构建交易草稿
	owner := ""
	if len(existingAuthorizedKeys) > 0 {
		owner = existingAuthorizedKeys[0]["value"].(string)
	} else if outputOwner, ok := output["owner"].(string); ok {
		owner = strings.TrimPrefix(outputOwner, "0x")
	}

	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txId,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{
			{
				"owner": owner,
				"output_type": "resource",
				"resource_output": map[string]interface{}{
					"resource":            resourceOutput["resource"],
					"creation_timestamp":  resourceOutput["creation_timestamp"],
					"storage_strategy":    resourceOutput["storage_strategy"],
					"is_immutable":        resourceOutput["is_immutable"],
				},
				"locking_conditions": newLockingConditions,
			},
		},
		"metadata": map[string]interface{}{
			"operation":          "update_collaborators",
			"required_signatures": intent.RequiredSignatures,
			"collaborators_count": len(allKeys),
		},
	}

	return &UnsignedTransaction{
		Draft:      draft,
		InputIndex: 0,
	}, nil
}

// BuildGrantDelegationTx 构建委托授权交易
func BuildGrantDelegationTx(
	ctx context.Context,
	client client.Client,
	intent GrantDelegationIntent,
) (*UnsignedTransaction, error) {
	// 1. 解析资源 ID
	parts := strings.Split(intent.ResourceID, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resourceId format: %s. Expected format: txId:outputIndex", intent.ResourceID)
	}
	txId := parts[0]
	outputIndexStr := parts[1]
	outputIndex, err := strconv.ParseUint(outputIndexStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid outputIndex: %w", err)
	}

	// 2. 查询当前资源 UTXO
	utxoParams := map[string]interface{}{
		"txId":        txId,
		"outputIndex": outputIndex,
	}
	var utxoResult interface{}
	utxoResult, err = client.Call(ctx, "wes_getUTXO", []interface{}{utxoParams})
	if err != nil {
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "NOT_FOUND") {
			return nil, fmt.Errorf("resource UTXO not found or already spent: %s. The resource may have been transferred or consumed", intent.ResourceID)
		}
		return nil, fmt.Errorf("failed to query UTXO: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		if utxoArray, ok := utxoResult.([]interface{}); ok && len(utxoArray) > 0 {
			if utxoMap, ok = utxoArray[0].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid UTXO response format")
			}
		} else {
			return nil, fmt.Errorf("invalid UTXO response format")
		}
	}

	var utxo map[string]interface{}
	if utxos, ok := utxoMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
		utxo, ok = utxos[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid UTXO format")
		}
	} else {
		utxo = utxoMap
	}

	output, ok := utxo["output"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource UTXO not found: %s. The UTXO may have been spent or the resource ID is incorrect", intent.ResourceID)
	}

	resourceOutput, ok := output["resource_output"].(map[string]interface{})
	if !ok {
		if resourceOutput, ok = output["resource"].(map[string]interface{}); !ok {
			return nil, fmt.Errorf("resource output not found in UTXO: %s", intent.ResourceID)
		}
	}

	// 3. 转换原始所有者地址
	currentLockingConditions, _ := output["locking_conditions"].([]interface{})
	if len(currentLockingConditions) == 0 {
		return nil, fmt.Errorf("no locking conditions found for resource %s. Cannot grant delegation", intent.ResourceID)
	}

	var originalOwnerHex string
	for _, condRaw := range currentLockingConditions {
		condition, ok := condRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if singleKey, ok := condition["single_key_lock"].(map[string]interface{}); ok {
			originalOwnerHex = singleKey["required_address_hash"].(string)
			if !strings.HasPrefix(originalOwnerHex, "0x") {
				originalOwnerHex = "0x" + originalOwnerHex
			}
			break
		} else if multiKey, ok := condition["multi_key_lock"].(map[string]interface{}); ok {
			if keys, ok := multiKey["authorized_keys"].([]interface{}); ok && len(keys) > 0 {
				firstKey := keys[0]
				if keyMap, ok := firstKey.(map[string]interface{}); ok {
					if value, ok := keyMap["value"].(string); ok {
						originalOwnerHex = value
						if !strings.HasPrefix(originalOwnerHex, "0x") {
							originalOwnerHex = "0x" + originalOwnerHex
						}
						break
					}
				} else if keyStr, ok := firstKey.(string); ok {
					originalOwnerHex = keyStr
					if !strings.HasPrefix(originalOwnerHex, "0x") {
						originalOwnerHex = "0x" + originalOwnerHex
					}
					break
				}
			}
		}
	}

	if originalOwnerHex == "" {
		return nil, fmt.Errorf("cannot determine original owner from current locking conditions for resource %s. The resource may have an unsupported locking condition type", intent.ResourceID)
	}

	// 4. 转换被委托者地址
	delegateAddressHex, err := utils.AddressBase58ToHex(intent.DelegateAddress)
	if err != nil {
		if strings.HasPrefix(intent.DelegateAddress, "0x") {
			delegateAddressHex = intent.DelegateAddress
		} else if len(intent.DelegateAddress) == 40 {
			delegateAddressHex = "0x" + intent.DelegateAddress
		} else {
			return nil, fmt.Errorf("invalid delegate address format: %s: %w", intent.DelegateAddress, err)
		}
	}

	// 5. 验证授权操作类型
	validOperations := map[string]bool{
		"reference": true,
		"execute":   true,
		"query":     true,
		"consume":   true,
		"transfer":  true,
		"stake":     true,
		"vote":      true,
	}
	for _, op := range intent.Operations {
		if !validOperations[op] {
			return nil, fmt.Errorf("invalid authorized operation: %s. Valid operations: reference, execute, query, consume, transfer, stake, vote", op)
		}
	}

	// 6. 构建 DelegationLock
	delegationLock := map[string]interface{}{
		"delegation_lock": map[string]interface{}{
			"original_owner":         strings.TrimPrefix(originalOwnerHex, "0x"),
			"allowed_delegates":       []string{strings.TrimPrefix(delegateAddressHex, "0x")},
			"authorized_operations":   intent.Operations,
			"max_value_per_operation": "0",
		},
	}
	if intent.ExpiryBlocks > 0 {
		delegationLock["delegation_lock"].(map[string]interface{})["expiry_duration_blocks"] = intent.ExpiryBlocks
	}
	if intent.MaxValuePerOperation != nil {
		delegationLock["delegation_lock"].(map[string]interface{})["max_value_per_operation"] = fmt.Sprintf("%d", *intent.MaxValuePerOperation)
	}

	// 7. 合并原有锁定条件和新的 DelegationLock
	newLockingConditions := make([]interface{}, len(currentLockingConditions))
	copy(newLockingConditions, currentLockingConditions)
	newLockingConditions = append(newLockingConditions, delegationLock)

	// 8. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txId,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{
			{
				"owner": strings.TrimPrefix(originalOwnerHex, "0x"),
				"output_type": "resource",
				"resource_output": map[string]interface{}{
					"resource":            resourceOutput["resource"],
					"creation_timestamp":  resourceOutput["creation_timestamp"],
					"storage_strategy":    resourceOutput["storage_strategy"],
					"is_immutable":        resourceOutput["is_immutable"],
				},
				"locking_conditions": newLockingConditions,
			},
		},
		"metadata": map[string]interface{}{
			"operation":           "grant_delegation",
			"delegate_address":    delegateAddressHex,
			"authorized_operations": strings.Join(intent.Operations, ","),
			"expiry_blocks":       intent.ExpiryBlocks,
		},
	}

	return &UnsignedTransaction{
		Draft:      draft,
		InputIndex: 0,
	}, nil
}

// BuildSetLockTx 构建时间/高度锁交易
func BuildSetLockTx(
	ctx context.Context,
	client client.Client,
	intent SetTimeOrHeightLockIntent,
) (*UnsignedTransaction, error) {
	// 1. 解析资源 ID
	parts := strings.Split(intent.ResourceID, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resourceId format: %s. Expected format: txId:outputIndex", intent.ResourceID)
	}
	txId := parts[0]
	outputIndexStr := parts[1]
	outputIndex, err := strconv.ParseUint(outputIndexStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid outputIndex: %w", err)
	}

	// 2. 验证参数
	if intent.UnlockTimestamp == nil && intent.UnlockHeight == nil {
		return nil, fmt.Errorf("either unlockTimestamp or unlockHeight must be provided")
	}
	if intent.UnlockTimestamp != nil && intent.UnlockHeight != nil {
		return nil, fmt.Errorf("cannot set both unlockTimestamp and unlockHeight")
	}

	// 3. 查询当前资源 UTXO
	utxoParams := map[string]interface{}{
		"txId":        txId,
		"outputIndex": outputIndex,
	}
	var utxoResult interface{}
	utxoResult, err = client.Call(ctx, "wes_getUTXO", []interface{}{utxoParams})
	if err != nil {
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "not found") || strings.Contains(errorMsg, "NOT_FOUND") {
			return nil, fmt.Errorf("resource UTXO not found or already spent: %s. The resource may have been transferred or consumed", intent.ResourceID)
		}
		return nil, fmt.Errorf("failed to query UTXO: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		if utxoArray, ok := utxoResult.([]interface{}); ok && len(utxoArray) > 0 {
			if utxoMap, ok = utxoArray[0].(map[string]interface{}); !ok {
				return nil, fmt.Errorf("invalid UTXO response format")
			}
		} else {
			return nil, fmt.Errorf("invalid UTXO response format")
		}
	}

	var utxo map[string]interface{}
	if utxos, ok := utxoMap["utxos"].([]interface{}); ok && len(utxos) > 0 {
		utxo, ok = utxos[0].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid UTXO format")
		}
	} else {
		utxo = utxoMap
	}

	output, ok := utxo["output"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource UTXO not found: %s. The UTXO may have been spent or the resource ID is incorrect", intent.ResourceID)
	}

	resourceOutput, ok := output["resource_output"].(map[string]interface{})
	if !ok {
		if resourceOutput, ok = output["resource"].(map[string]interface{}); !ok {
			return nil, fmt.Errorf("resource output not found in UTXO: %s", intent.ResourceID)
		}
	}

	// 4. 解析当前锁定条件（作为 base_lock）
	currentLockingConditions, _ := output["locking_conditions"].([]interface{})
	if len(currentLockingConditions) == 0 {
		return nil, fmt.Errorf("no locking conditions found for resource %s. Cannot set time/height lock", intent.ResourceID)
	}

	var baseLock interface{}
	for _, condRaw := range currentLockingConditions {
		condition, ok := condRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if condition["time_lock"] == nil && condition["height_lock"] == nil {
			baseLock = condition
			break
		} else if timeLock, ok := condition["time_lock"].(map[string]interface{}); ok {
			baseLock = timeLock["base_lock"]
			if baseLock != nil {
				break
			}
		} else if heightLock, ok := condition["height_lock"].(map[string]interface{}); ok {
			baseLock = heightLock["base_lock"]
			if baseLock != nil {
				break
			}
		}
	}

	// 如果没有找到 base_lock，创建一个默认的 SingleKeyLock
	if baseLock == nil {
		owner, ok := output["owner"].(string)
		if !ok || owner == "" {
			return nil, fmt.Errorf("cannot determine base lock for resource %s. No owner found in UTXO output", intent.ResourceID)
		}
		baseLock = map[string]interface{}{
			"single_key_lock": map[string]interface{}{
				"required_address_hash": strings.TrimPrefix(owner, "0x"),
				"required_algorithm":    "ECDSA_SECP256K1",
				"sighash_type":          "SIGHASH_ALL",
			},
		}
	}

	// 5. 构建新的锁定条件
	var newLockingCondition map[string]interface{}
	if intent.UnlockTimestamp != nil {
		// TimeLock
		newLockingCondition = map[string]interface{}{
			"time_lock": map[string]interface{}{
				"unlock_timestamp": *intent.UnlockTimestamp,
				"base_lock":        baseLock,
				"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			},
		}
	} else {
		// HeightLock
		unlockHeight := *intent.UnlockHeight
		// 如果 unlockHeight 看起来是相对高度，尝试查询当前高度
		if unlockHeight < 1000000 {
			blockNumberResult, err := client.Call(ctx, "wes_blockNumber", []interface{}{})
			if err == nil {
				if blockNumberStr, ok := blockNumberResult.(string); ok {
					if blockNum, err := strconv.ParseUint(strings.TrimPrefix(blockNumberStr, "0x"), 16, 64); err == nil {
						unlockHeight = blockNum + unlockHeight
					}
				}
			}
		}

		newLockingCondition = map[string]interface{}{
			"height_lock": map[string]interface{}{
				"unlock_height":       unlockHeight,
				"base_lock":           baseLock,
				"confirmation_blocks": uint32(6),
			},
		}
	}

	// 6. 构建交易草稿
	owner, _ := output["owner"].(string)
	if owner == "" {
		owner = "0x"
	}

	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txId,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{
			{
				"owner": strings.TrimPrefix(owner, "0x"),
				"output_type": "resource",
				"resource_output": map[string]interface{}{
					"resource":            resourceOutput["resource"],
					"creation_timestamp":  resourceOutput["creation_timestamp"],
					"storage_strategy":    resourceOutput["storage_strategy"],
					"is_immutable":        resourceOutput["is_immutable"],
				},
				"locking_conditions": []interface{}{newLockingCondition},
			},
		},
		"metadata": map[string]interface{}{
			"operation": "set_time_lock",
		},
	}
	if intent.UnlockTimestamp != nil {
		draft["metadata"].(map[string]interface{})["unlock_timestamp"] = *intent.UnlockTimestamp
		draft["metadata"].(map[string]interface{})["operation"] = "set_time_lock"
	} else {
		draft["metadata"].(map[string]interface{})["unlock_height"] = *intent.UnlockHeight
		draft["metadata"].(map[string]interface{})["operation"] = "set_height_lock"
	}

	return &UnsignedTransaction{
		Draft:      draft,
		InputIndex: 0,
	}, nil
}

