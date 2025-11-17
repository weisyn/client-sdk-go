package staking

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/utils"
)

// UTXO UTXO 信息（从 wes_getUTXO API 返回）
type UTXO struct {
	Outpoint string `json:"outpoint"`          // "txHash:outputIndex"
	Height   string `json:"height"`            // "0x..."
	Amount   string `json:"amount"`            // 金额（字符串）
	TokenID  string `json:"tokenID,omitempty"` // 代币ID（hex编码，可选）
}

// buildStakeDraft 构建质押交易草稿（DraftJSON）
//
// **功能**：
// 构建质押交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 HeightLock + ContractLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildStakeDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	validatorAddr []byte,
	amount uint64,
	lockBlocks uint64,
	stakingContractAddr []byte, // Staking 合约地址（可选，如果为空则只使用 HeightLock）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(validatorAddr) == 0 {
		return nil, 0, fmt.Errorf("validatorAddr cannot be empty")
	}
	if amount == 0 {
		return nil, 0, fmt.Errorf("amount must be greater than 0")
	}
	if lockBlocks == 0 {
		return nil, 0, fmt.Errorf("lockBlocks must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, 0, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 3. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXOs format")
	}

	// 4. 转换为 UTXO 结构（原生币质押）
	var utxos []UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		utxo := UTXO{
			Outpoint: getString(utxoMap, "outpoint"),
			Height:   getString(utxoMap, "height"),
			Amount:   getString(utxoMap, "amount"),
		}
		// 只选择原生币（没有 tokenID 的 UTXO）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
			utxos = append(utxos, utxo)
		}
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available native coin UTXOs")
	}

	// 5. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range utxos {
		if utxo.Amount == "" {
			continue
		}
		utxoAmount, ok := new(big.Int).SetString(utxo.Amount, 10)
		if !ok {
			continue
		}
		if utxoAmount.Cmp(requiredAmount) >= 0 {
			selectedUTXO = utxo
			selectedAmount = utxoAmount
			break
		}
	}

	if selectedUTXO.Outpoint == "" {
		return nil, 0, fmt.Errorf("insufficient balance: required %d, but no UTXO found with sufficient amount", amount)
	}

	// 6. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 7. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 8. 获取当前区块高度（用于计算解锁高度）
	// 注意：如果无法获取当前高度，可以使用相对高度（lockBlocks），
	// 节点在构建交易时会自动处理相对高度转换为绝对高度
	currentHeight := uint64(0)
	if heightResult, err := client.Call(ctx, "wes_blockNumber", []interface{}{}); err == nil {
		// wes_blockNumber 返回十六进制字符串（如 "0x288"）
		if heightStr, ok := heightResult.(string); ok {
			// 尝试解析十六进制或十进制
			heightStr = strings.TrimPrefix(heightStr, "0x")
			if height, ok := new(big.Int).SetString(heightStr, 16); ok {
				currentHeight = height.Uint64()
			} else if height, ok := new(big.Int).SetString(heightStr, 10); ok {
				currentHeight = height.Uint64()
			}
		}
	}

	// 计算解锁高度
	// 如果获取到了当前高度，使用绝对高度；否则使用相对高度（节点会处理）
	var unlockHeightStr string
	if currentHeight > 0 {
		unlockHeight := currentHeight + lockBlocks
		unlockHeightStr = fmt.Sprintf("%d", unlockHeight)
	} else {
		// 使用相对高度（节点会在构建交易时转换为绝对高度）
		unlockHeightStr = fmt.Sprintf("%d", lockBlocks)
	}

	// 9. 构建锁定条件（HeightLock + ContractLock）
	// 如果提供了 Staking 合约地址，使用 HeightLock + ContractLock
	// 否则只使用 HeightLock + SingleKeyLock
	var lockingCondition map[string]interface{}

	if len(stakingContractAddr) > 0 {
		// HeightLock + ContractLock 组合
		lockingCondition = map[string]interface{}{
			"type":          "height_lock",
			"unlock_height": unlockHeightStr,
			"base_lock": map[string]interface{}{
				"type":            "contract_lock",
				"contract_address": hex.EncodeToString(stakingContractAddr),
			},
		}
	} else {
		// 只使用 HeightLock + SingleKeyLock
		lockingCondition = map[string]interface{}{
			"type":          "height_lock",
			"unlock_height": unlockHeightStr,
			"base_lock": map[string]interface{}{
				"type":            "single_key_lock",
				"required_address": hex.EncodeToString(validatorAddr),
			},
		}
	}

	// 10. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 11. 添加质押输出（给验证者，带锁定条件）
	outputs := draft["outputs"].([]map[string]interface{})
	stakeOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(validatorAddr),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	draft["outputs"] = append(outputs, stakeOutput)

	// 12. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 13. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildStakeTransaction 构建质押交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildStakeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildStakeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Stake 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
// 质押通过组合 HeightLock + ContractLock 实现：
// - HeightLock：锁定指定区块数
// - ContractLock：由 Staking 合约控制解锁逻辑
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 HeightLock + ContractLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildStakeTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	validatorAddr []byte,
	amount uint64,
	lockBlocks uint64,
	stakingContractAddr []byte, // Staking 合约地址（可选，如果为空则只使用 HeightLock）
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 3. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 4. 转换为 UTXO 结构（原生币质押）
	var utxos []UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		utxo := UTXO{
			Outpoint: getString(utxoMap, "outpoint"),
			Height:   getString(utxoMap, "height"),
			Amount:   getString(utxoMap, "amount"),
		}
		// 只选择原生币（没有 tokenID 的 UTXO）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
			utxos = append(utxos, utxo)
		}
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no available native coin UTXOs")
	}

	// 5. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range utxos {
		if utxo.Amount == "" {
			continue
		}
		utxoAmount, ok := new(big.Int).SetString(utxo.Amount, 10)
		if !ok {
			continue
		}
		if utxoAmount.Cmp(requiredAmount) >= 0 {
			selectedUTXO = utxo
			selectedAmount = utxoAmount
			break
		}
	}

	if selectedUTXO.Outpoint == "" {
		return nil, fmt.Errorf("insufficient balance")
	}

	// 6. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 7. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 9. 获取当前区块高度（用于计算解锁高度）
	// 注意：如果无法获取当前高度，可以使用相对高度（lockBlocks），
	// 节点在构建交易时会自动处理相对高度转换为绝对高度
	currentHeight := uint64(0)
	if heightResult, err := client.Call(ctx, "wes_blockNumber", []interface{}{}); err == nil {
		// wes_blockNumber 返回十六进制字符串（如 "0x288"）
		if heightStr, ok := heightResult.(string); ok {
				// 尝试解析十六进制或十进制
				heightStr = strings.TrimPrefix(heightStr, "0x")
				if height, ok := new(big.Int).SetString(heightStr, 16); ok {
					currentHeight = height.Uint64()
				} else if height, ok := new(big.Int).SetString(heightStr, 10); ok {
					currentHeight = height.Uint64()
			}
		}
	}

	// 计算解锁高度
	// 如果获取到了当前高度，使用绝对高度；否则使用相对高度（节点会处理）
	var unlockHeightStr string
	if currentHeight > 0 {
		unlockHeight := currentHeight + lockBlocks
		unlockHeightStr = fmt.Sprintf("%d", unlockHeight)
	} else {
		// 使用相对高度（节点会在构建交易时转换为绝对高度）
		unlockHeightStr = fmt.Sprintf("%d", lockBlocks)
	}

	// 10. 构建锁定条件（HeightLock + ContractLock）
	// 如果提供了 Staking 合约地址，使用 HeightLock + ContractLock
	// 否则只使用 HeightLock + SingleKeyLock
	var lockingCondition map[string]interface{}

	if len(stakingContractAddr) > 0 {
		// HeightLock + ContractLock 组合
		lockingCondition = map[string]interface{}{
			"type":          "height_lock",
			"unlock_height": unlockHeightStr,
			"base_lock": map[string]interface{}{
				"type":            "contract_lock",
				"contract_address": hex.EncodeToString(stakingContractAddr),
			},
		}
	} else {
		// 只使用 HeightLock + SingleKeyLock
		lockingCondition = map[string]interface{}{
			"type":          "height_lock",
			"unlock_height": unlockHeightStr,
			"base_lock": map[string]interface{}{
				"type":            "single_key_lock",
				"required_address": hex.EncodeToString(validatorAddr),
			},
		}
	}

	// 11. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 12. 添加质押输出（给验证者，带锁定条件）
	outputs := draft["outputs"].([]map[string]interface{})
	stakeOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(validatorAddr),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	draft["outputs"] = append(outputs, stakeOutput)

	// 13. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 14. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 15. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	// 注意：这里直接传递对象格式 {draft: {...}}，以匹配服务器端 BuildTransaction 的参数解析逻辑
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 16. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 17. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildUnstakeDraft 构建解质押交易草稿（DraftJSON）
//
// **功能**：
// 构建解质押交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 解析 StakeID（outpoint 格式：txHash:index）
// 2. 查询质押 UTXO
// 3. 构建交易草稿（消费质押 UTXO，返回给用户）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildUnstakeDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	stakeID []byte, // StakeID（outpoint 格式：txHash:index）
	amount uint64,  // 解质押金额（0表示全部）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(stakeID) == 0 {
		return nil, 0, fmt.Errorf("stakeID cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 解析 StakeID（假设是 outpoint 格式：txHash:index）
	stakeIDStr := string(stakeID)
	outpointParts := strings.Split(stakeIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid stake ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 2. 查询质押 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, 0, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找对应的质押 UTXO
	var stakeUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == stakeIDStr {
			stakeUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if stakeUTXO == nil {
		return nil, 0, fmt.Errorf("stake UTXO not found: %s", stakeIDStr)
	}

	// 4. 解析质押金额
	stakeAmount, ok := new(big.Int).SetString(stakeUTXO.Amount, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid stake amount: %s", stakeUTXO.Amount)
	}

	// 5. 计算解质押金额
	unstakeAmount := big.NewInt(int64(amount))
	if amount == 0 || unstakeAmount.Cmp(stakeAmount) > 0 {
		unstakeAmount = stakeAmount // 全部解质押
	}

	// 6. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = stakeAmount - unstakeAmount
	changeBig := new(big.Int).Sub(stakeAmount, unstakeAmount)

	// 7. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 8. 添加解质押输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	unstakeOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": unstakeAmount.String(),
	}
	draft["outputs"] = append(outputs, unstakeOutput)

	// 9. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildUnstakeTransaction 构建解质押交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildUnstakeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildUnstakeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Unstake 业务语义在 SDK 层，通过查询质押 UTXO、构建交易实现。
// 解质押需要消费带有 HeightLock 的质押 UTXO。
//
// **流程**：
// 1. 解析 StakeID（可能是 outpoint 或交易哈希）
// 2. 查询质押 UTXO
// 3. 构建交易草稿（消费质押 UTXO，返回给用户）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildUnstakeTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	stakeID []byte, // StakeID（可能是 outpoint 字符串或交易哈希）
	amount uint64,  // 解质押金额（0表示全部）
) ([]byte, error) {
	// 1. 解析 StakeID（假设是 outpoint 格式：txHash:index）
	stakeIDStr := string(stakeID)
	outpointParts := strings.Split(stakeIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid stake ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 2. 查询质押 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找对应的质押 UTXO
	var stakeUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == stakeIDStr {
			stakeUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if stakeUTXO == nil {
		return nil, fmt.Errorf("stake UTXO not found: %s", stakeIDStr)
	}

	// 4. 解析质押金额
	stakeAmount, ok := new(big.Int).SetString(stakeUTXO.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid stake amount: %s", stakeUTXO.Amount)
	}

	// 5. 计算解质押金额
	unstakeAmount := big.NewInt(int64(amount))
	if amount == 0 || unstakeAmount.Cmp(stakeAmount) > 0 {
		unstakeAmount = stakeAmount // 全部解质押
	}

	// 6. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = stakeAmount - unstakeAmount
	changeBig := new(big.Int).Sub(stakeAmount, unstakeAmount)

	// 8. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 9. 添加解质押输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	unstakeOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": unstakeAmount.String(),
	}
	draft["outputs"] = append(outputs, unstakeOutput)

	// 10. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 11. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 12. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	// 注意：这里直接传递对象格式 {draft: {...}}，以匹配服务器端 BuildTransaction 的参数解析逻辑
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 13. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 14. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildDelegateDraft 构建委托交易草稿（DraftJSON）
//
// **功能**：
// 构建委托交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询用户 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 DelegationLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildDelegateDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	validatorAddr []byte,
	amount uint64,
	expiryDurationBlocks uint64, // 委托有效期（区块数，0=永不过期）
	maxValuePerOperation uint64, // 单次操作最大价值
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(validatorAddr) == 0 {
		return nil, 0, fmt.Errorf("validatorAddr cannot be empty")
	}
	if amount == 0 {
		return nil, 0, fmt.Errorf("amount must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, 0, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 3. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXOs format")
	}

	// 4. 转换为 UTXO 结构（原生币委托）
	var utxos []UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		utxo := UTXO{
			Outpoint: getString(utxoMap, "outpoint"),
			Height:   getString(utxoMap, "height"),
			Amount:   getString(utxoMap, "amount"),
		}
		// 只选择原生币（没有 tokenID 的 UTXO）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
			utxos = append(utxos, utxo)
		}
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available native coin UTXOs")
	}

	// 5. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range utxos {
		if utxo.Amount == "" {
			continue
		}
		utxoAmount, ok := new(big.Int).SetString(utxo.Amount, 10)
		if !ok {
			continue
		}
		if utxoAmount.Cmp(requiredAmount) >= 0 {
			selectedUTXO = utxo
			selectedAmount = utxoAmount
			break
		}
	}

	if selectedUTXO.Outpoint == "" {
		return nil, 0, fmt.Errorf("insufficient balance: required %d, but no UTXO found with sufficient amount", amount)
	}

	// 6. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 7. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 8. 构建 DelegationLock 锁定条件
	delegationLock := map[string]interface{}{
		"type":                  "delegation_lock",
		"original_owner":        hex.EncodeToString(fromAddress),
		"allowed_delegates":    []string{hex.EncodeToString(validatorAddr)},
		"authorized_operations": []string{"stake", "consume"},
		"max_value_per_operation": fmt.Sprintf("%d", maxValuePerOperation),
	}
	if expiryDurationBlocks > 0 {
		delegationLock["expiry_duration_blocks"] = fmt.Sprintf("%d", expiryDurationBlocks)
	}

	// 9. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 10. 添加委托输出（给验证者，带 DelegationLock）
	outputs := draft["outputs"].([]map[string]interface{})
	delegateOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(validatorAddr),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": delegationLock,
	}
	draft["outputs"] = append(outputs, delegateOutput)

	// 11. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 12. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildDelegateTransaction 构建委托交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildDelegateDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildDelegateDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Delegate 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 委托使用 DelegationLock 锁定条件。
//
// **流程**：
// 1. 查询用户 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 DelegationLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildDelegateTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	validatorAddr []byte,
	amount uint64,
	expiryDurationBlocks uint64, // 委托有效期（区块数，0=永不过期）
	maxValuePerOperation uint64, // 单次操作最大价值
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 3. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 4. 转换为 UTXO 结构（原生币委托）
	var utxos []UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		utxo := UTXO{
			Outpoint: getString(utxoMap, "outpoint"),
			Height:   getString(utxoMap, "height"),
			Amount:   getString(utxoMap, "amount"),
		}
		// 只选择原生币（没有 tokenID 的 UTXO）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
			utxos = append(utxos, utxo)
		}
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no available native coin UTXOs")
	}

	// 5. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range utxos {
		if utxo.Amount == "" {
			continue
		}
		utxoAmount, ok := new(big.Int).SetString(utxo.Amount, 10)
		if !ok {
			continue
		}
		if utxoAmount.Cmp(requiredAmount) >= 0 {
			selectedUTXO = utxo
			selectedAmount = utxoAmount
			break
		}
	}

	if selectedUTXO.Outpoint == "" {
		return nil, fmt.Errorf("insufficient balance")
	}

	// 6. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 7. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 9. 构建 DelegationLock 锁定条件
	delegationLock := map[string]interface{}{
		"type":                  "delegation_lock",
		"original_owner":        hex.EncodeToString(fromAddress),
		"allowed_delegates":    []string{hex.EncodeToString(validatorAddr)},
		"authorized_operations": []string{"stake", "consume"},
		"max_value_per_operation": fmt.Sprintf("%d", maxValuePerOperation),
	}
	if expiryDurationBlocks > 0 {
		delegationLock["expiry_duration_blocks"] = fmt.Sprintf("%d", expiryDurationBlocks)
	}

	// 10. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 11. 添加委托输出（给验证者，带 DelegationLock）
	outputs := draft["outputs"].([]map[string]interface{})
	delegateOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(validatorAddr),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": delegationLock,
	}
	draft["outputs"] = append(outputs, delegateOutput)

	// 12. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 13. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 14. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	// 注意：这里直接传递对象格式 {draft: {...}}，以匹配服务器端 BuildTransaction 的参数解析逻辑
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 15. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 16. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildUndelegateDraft 构建取消委托交易草稿（DraftJSON）
//
// **功能**：
// 构建取消委托交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 解析 DelegateID（outpoint 格式：txHash:index）
// 2. 查询委托 UTXO
// 3. 构建交易草稿（消费委托 UTXO，返回给用户）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildUndelegateDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	delegateID []byte, // DelegateID（outpoint 格式：txHash:index）
	amount uint64,      // 取消委托金额（0表示全部）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(delegateID) == 0 {
		return nil, 0, fmt.Errorf("delegateID cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 解析 DelegateID（outpoint 格式：txHash:index）
	delegateIDStr := string(delegateID)
	outpointParts := strings.Split(delegateIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid delegate ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 2. 查询委托 UTXO（通过查询用户的 UTXO 列表）
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, 0, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找对应的委托 UTXO
	var delegateUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == delegateIDStr {
			delegateUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if delegateUTXO == nil {
		return nil, 0, fmt.Errorf("delegate UTXO not found: %s", delegateIDStr)
	}

	// 4. 解析委托金额
	delegateAmount, ok := new(big.Int).SetString(delegateUTXO.Amount, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid delegate amount: %s", delegateUTXO.Amount)
	}

	// 5. 计算取消委托金额
	undelegateAmount := big.NewInt(int64(amount))
	if amount == 0 || undelegateAmount.Cmp(delegateAmount) > 0 {
		undelegateAmount = delegateAmount // 全部取消委托
	}

	// 6. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = delegateAmount - undelegateAmount
	changeBig := new(big.Int).Sub(delegateAmount, undelegateAmount)

	// 7. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 8. 添加取消委托输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	undelegateOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": undelegateAmount.String(),
	}
	draft["outputs"] = append(outputs, undelegateOutput)

	// 9. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildUndelegateTransaction 构建取消委托交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildUndelegateDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildUndelegateDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Undelegate 业务语义在 SDK 层，通过查询委托 UTXO、构建交易实现。
// 取消委托需要消费带有 DelegationLock 的委托 UTXO。
func buildUndelegateTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	delegateID []byte, // DelegateID（outpoint 格式：txHash:index）
	amount uint64,      // 取消委托金额（0表示全部）
) ([]byte, error) {
	// 1. 解析 DelegateID（outpoint 格式：txHash:index）
	delegateIDStr := string(delegateID)
	outpointParts := strings.Split(delegateIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid delegate ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 2. 查询委托 UTXO（通过查询用户的 UTXO 列表）
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找对应的委托 UTXO
	var delegateUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == delegateIDStr {
			delegateUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if delegateUTXO == nil {
		return nil, fmt.Errorf("delegate UTXO not found: %s", delegateIDStr)
	}

	// 4. 解析委托金额
	delegateAmount, ok := new(big.Int).SetString(delegateUTXO.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid delegate amount: %s", delegateUTXO.Amount)
	}

	// 5. 计算取消委托金额
	undelegateAmount := big.NewInt(int64(amount))
	if amount == 0 || undelegateAmount.Cmp(delegateAmount) > 0 {
		undelegateAmount = delegateAmount // 全部取消委托
	}

	// 6. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = delegateAmount - undelegateAmount
	changeBig := new(big.Int).Sub(delegateAmount, undelegateAmount)

	// 8. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 9. 添加取消委托输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	undelegateOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": undelegateAmount.String(),
	}
	draft["outputs"] = append(outputs, undelegateOutput)

	// 10. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		outputs = draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 11. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 12. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	// 注意：这里直接传递对象格式 {draft: {...}}，以匹配服务器端 BuildTransaction 的参数解析逻辑
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 13. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 14. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildClaimRewardDraft 构建领取奖励交易草稿（DraftJSON）
//
// **功能**：
// 构建领取奖励交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询奖励 UTXO（通过 StakeID 或 DelegateID，或查询用户的 UTXO 列表）
// 2. 构建交易草稿（消费奖励 UTXO，返回给用户）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildClaimRewardDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	stakeID []byte,    // StakeID（可选，outpoint 格式）
	delegateID []byte, // DelegateID（可选，outpoint 格式）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询用户的 UTXO 列表
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, 0, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找奖励 UTXO
	// 奖励 UTXO 的特征：
	// - owner = 用户地址
	// - 可能是由 Staking 合约产生的（可以通过 metadata 或其他方式识别）
	// - 或者通过 StakeID/DelegateID 关联查询
	var rewardUTXO *UTXO
	var rewardAmount *big.Int

	// 优先通过 StakeID 或 DelegateID 查找
	var targetOutpoint string
	if len(stakeID) > 0 {
		targetOutpoint = string(stakeID)
	} else if len(delegateID) > 0 {
		targetOutpoint = string(delegateID)
	}

	// 查找对应的奖励 UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")

		// 如果提供了 StakeID 或 DelegateID，尝试通过关联查找
		// 这里简化处理：假设奖励 UTXO 的 outpoint 与质押/委托 UTXO 有关联
		// 实际实现可能需要通过合约调用或状态查询获取奖励 UTXO
		if targetOutpoint != "" {
			// 简化：查找金额大于 0 的 UTXO（可能是奖励）
			amountStr := getString(utxoMap, "amount")
			if amountStr != "" {
				if amount, ok := new(big.Int).SetString(amountStr, 10); ok && amount.Sign() > 0 {
					// 检查是否是原生币（奖励通常是原生币）
					if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
						rewardUTXO = &UTXO{
							Outpoint: outpoint,
							Height:   getString(utxoMap, "height"),
							Amount:   amountStr,
						}
						rewardAmount = amount
						break
					}
				}
			}
		} else {
			// 如果没有提供 ID，查找所有可能的奖励 UTXO（简化：选择第一个金额大于 0 的原生币 UTXO）
			amountStr := getString(utxoMap, "amount")
			if amountStr != "" {
				if amount, ok := new(big.Int).SetString(amountStr, 10); ok && amount.Sign() > 0 {
					if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
						if rewardUTXO == nil {
							rewardUTXO = &UTXO{
								Outpoint: outpoint,
								Height:   getString(utxoMap, "height"),
								Amount:   amountStr,
							}
							rewardAmount = amount
						}
					}
				}
			}
		}
	}

	if rewardUTXO == nil {
		return nil, 0, fmt.Errorf("reward UTXO not found (may need contract call or state query)")
	}

	// 4. 解析 outpoint
	outpointParts := strings.Split(rewardUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 5. 计算领取金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，领取金额 = rewardAmount
	claimAmount := rewardAmount

	// 6. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 7. 添加领取奖励输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	claimOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": claimAmount.String(),
	}
	draft["outputs"] = append(outputs, claimOutput)

	// 8. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildClaimRewardTransaction 构建领取奖励交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildClaimRewardDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildClaimRewardDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// ClaimReward 业务语义在 SDK 层，通过查询奖励 UTXO、构建交易实现。
// 奖励可能由合约产生，需要通过查询用户的 UTXO 列表或合约调用获取。
//
// **流程**：
// 1. 查询奖励 UTXO（通过 StakeID 或 DelegateID，或查询用户的 UTXO 列表）
// 2. 构建交易草稿（消费奖励 UTXO，返回给用户）
// 3. 调用 `wes_buildTransaction` API 获取未签名交易
//
// **注意**：
// - 奖励 UTXO 可能由合约产生，需要通过合约调用或查询链上状态获取
// - 当前实现假设奖励 UTXO 可以通过查询用户的 UTXO 列表找到（owner=用户地址）
func buildClaimRewardTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	stakeID []byte,    // StakeID（可选，outpoint 格式）
	delegateID []byte, // DelegateID（可选，outpoint 格式）
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询用户的 UTXO 列表
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 3. 查找奖励 UTXO
	// 奖励 UTXO 的特征：
	// - owner = 用户地址
	// - 可能是由 Staking 合约产生的（可以通过 metadata 或其他方式识别）
	// - 或者通过 StakeID/DelegateID 关联查询
	var rewardUTXO *UTXO
	var rewardAmount *big.Int

	// 优先通过 StakeID 或 DelegateID 查找
	var targetOutpoint string
	if len(stakeID) > 0 {
		targetOutpoint = string(stakeID)
	} else if len(delegateID) > 0 {
		targetOutpoint = string(delegateID)
	}

	// 查找对应的奖励 UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")

		// 如果提供了 StakeID 或 DelegateID，尝试通过关联查找
		// 这里简化处理：假设奖励 UTXO 的 outpoint 与质押/委托 UTXO 有关联
		// 实际实现可能需要通过合约调用或状态查询获取奖励 UTXO
		if targetOutpoint != "" {
			// 简化：查找金额大于 0 的 UTXO（可能是奖励）
			amountStr := getString(utxoMap, "amount")
			if amountStr != "" {
				if amount, ok := new(big.Int).SetString(amountStr, 10); ok && amount.Sign() > 0 {
					// 检查是否是原生币（奖励通常是原生币）
					if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
						rewardUTXO = &UTXO{
							Outpoint: outpoint,
							Height:   getString(utxoMap, "height"),
							Amount:   amountStr,
						}
						rewardAmount = amount
						break
					}
				}
			}
		} else {
			// 如果没有提供 ID，查找所有可能的奖励 UTXO（简化：选择第一个金额大于 0 的原生币 UTXO）
			amountStr := getString(utxoMap, "amount")
			if amountStr != "" {
				if amount, ok := new(big.Int).SetString(amountStr, 10); ok && amount.Sign() > 0 {
					if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr == "" {
						if rewardUTXO == nil {
							rewardUTXO = &UTXO{
								Outpoint: outpoint,
								Height:   getString(utxoMap, "height"),
								Amount:   amountStr,
							}
							rewardAmount = amount
						}
					}
				}
			}
		}
	}

	if rewardUTXO == nil {
		return nil, fmt.Errorf("reward UTXO not found (may need contract call or state query)")
	}

	// 4. 解析 outpoint
	outpointParts := strings.Split(rewardUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 5. 计算领取金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，领取金额 = rewardAmount
	claimAmount := rewardAmount

	// 7. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs": []map[string]interface{}{
			{
				"tx_hash":           txHash,
				"output_index":      outputIndex,
				"is_reference_only": false,
			},
		},
		"outputs": []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 8. 添加领取奖励输出（返回给用户）
	outputs := draft["outputs"].([]map[string]interface{})
	claimOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": claimAmount.String(),
	}
	draft["outputs"] = append(outputs, claimOutput)

	// 9. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 10. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	// 注意：这里直接传递对象格式 {draft: {...}}，以匹配服务器端 BuildTransaction 的参数解析逻辑
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 11. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 12. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// getString 从 map 中获取字符串值
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

