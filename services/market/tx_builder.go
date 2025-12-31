package market

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

// buildVestingDraft 构建归属计划交易草稿（DraftJSON）
//
// **功能**：
// 构建归属计划交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询用户 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 TimeLock + ContractLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildVestingDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	toAddress []byte,
	amount uint64,
	tokenID []byte,
	startTime uint64, // 开始时间（Unix时间戳）
	duration uint64, // 持续时间（秒）
	vestingContractAddr []byte, // Vesting 合约地址（可选）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(toAddress) == 0 {
		return nil, 0, fmt.Errorf("toAddress cannot be empty")
	}
	if amount == 0 {
		return nil, 0, fmt.Errorf("amount must be greater than 0")
	}
	if duration == 0 {
		return nil, 0, fmt.Errorf("duration must be greater than 0")
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

	// 4. 转换为 UTXO 结构（根据 tokenID 过滤）
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
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
			utxo.TokenID = tokenIDStr
		}
		// 如果提供了 tokenID，只选择匹配的 UTXO；否则选择原生币
		if len(tokenID) > 0 {
			if utxo.TokenID == hex.EncodeToString(tokenID) {
				utxos = append(utxos, utxo)
			}
		} else {
			if utxo.TokenID == "" {
				utxos = append(utxos, utxo)
			}
		}
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available UTXOs")
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
		return nil, 0, fmt.Errorf("insufficient balance: required %s", requiredAmount.String())
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

	// 8. 计算解锁时间戳
	unlockTimestamp := startTime + duration

	// 9. 构建 TimeLock 锁定条件
	var lockingCondition map[string]interface{}
	if len(vestingContractAddr) > 0 {
		// TimeLock + ContractLock 组合
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", unlockTimestamp),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "contract_lock",
				"contract_address": hex.EncodeToString(vestingContractAddr),
			},
		}
	} else {
		// TimeLock + SingleKeyLock
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", unlockTimestamp),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "single_key_lock",
				"required_address": hex.EncodeToString(toAddress),
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

	// 11. 添加归属计划输出（给受益人，带 TimeLock）
	outputs := draft["outputs"].([]map[string]interface{})
	vestingOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(toAddress),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	if len(tokenID) > 0 {
		vestingOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, vestingOutput)

	// 12. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		if len(tokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(tokenID)
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

// buildVestingTransaction 构建归属计划交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildVestingDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildVestingDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// CreateVesting 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 归属计划使用 TimeLock + ContractLock 锁定条件。
//
// **流程**：
// 1. 查询用户 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 TimeLock + ContractLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildVestingTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	toAddress []byte,
	amount uint64,
	tokenID []byte,
	startTime uint64, // 开始时间（Unix时间戳）
	duration uint64, // 持续时间（秒）
	vestingContractAddr []byte, // Vesting 合约地址（可选）
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

	// 4. 转换为 UTXO 结构（根据 tokenID 过滤）
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
		tokenIDStr := getString(utxoMap, "tokenID")
		// 如果提供了 tokenID，只选择匹配的 UTXO；否则选择原生币
		if len(tokenID) > 0 {
			if tokenIDStr == hex.EncodeToString(tokenID) {
				utxos = append(utxos, utxo)
			}
		} else {
			if tokenIDStr == "" {
				utxos = append(utxos, utxo)
			}
		}
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no available UTXOs")
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

	// 9. 计算解锁时间戳
	unlockTimestamp := startTime + duration

	// 10. 构建 TimeLock 锁定条件
	var lockingCondition map[string]interface{}
	if len(vestingContractAddr) > 0 {
		// TimeLock + ContractLock 组合
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", unlockTimestamp),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "contract_lock",
				"contract_address": hex.EncodeToString(vestingContractAddr),
			},
		}
	} else {
		// TimeLock + SingleKeyLock
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", unlockTimestamp),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "single_key_lock",
				"required_address": hex.EncodeToString(toAddress),
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

	// 12. 添加归属计划输出（给受益人，带 TimeLock）
	outputs := draft["outputs"].([]map[string]interface{})
	vestingOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(toAddress),
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	if len(tokenID) > 0 {
		vestingOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, vestingOutput)

	// 13. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		if len(tokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(tokenID)
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
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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

// buildEscrowDraft 构建托管交易草稿（DraftJSON）
//
// **功能**：
// 构建托管交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询买方 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 MultiKeyLock 或 ContractLock + TimeLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildEscrowDraft(
	ctx context.Context,
	client client.Client,
	buyerAddress []byte,
	sellerAddress []byte,
	amount uint64,
	tokenID []byte,
	expiryTime uint64, // 过期时间（Unix时间戳）
	escrowContractAddr []byte, // Escrow 合约地址（可选）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(buyerAddress) == 0 {
		return nil, 0, fmt.Errorf("buyerAddress cannot be empty")
	}
	if len(sellerAddress) == 0 {
		return nil, 0, fmt.Errorf("sellerAddress cannot be empty")
	}
	if amount == 0 {
		return nil, 0, fmt.Errorf("amount must be greater than 0")
	}
	if expiryTime == 0 {
		return nil, 0, fmt.Errorf("expiryTime must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	buyerAddressBase58, err := utils.AddressBytesToBase58(buyerAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{buyerAddressBase58}
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

	// 4. 转换为 UTXO 结构（根据 tokenID 过滤）
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
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
			utxo.TokenID = tokenIDStr
		}
		// 如果提供了 tokenID，只选择匹配的 UTXO；否则选择原生币
		if len(tokenID) > 0 {
			if utxo.TokenID == hex.EncodeToString(tokenID) {
				utxos = append(utxos, utxo)
			}
		} else {
			if utxo.TokenID == "" {
				utxos = append(utxos, utxo)
			}
		}
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available UTXOs")
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
		return nil, 0, fmt.Errorf("insufficient balance: required %s", requiredAmount.String())
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

	// 8. 构建锁定条件（MultiKeyLock 或 ContractLock + TimeLock）
	var lockingCondition map[string]interface{}
	if len(escrowContractAddr) > 0 {
		// ContractLock + TimeLock（过期后可以退款）
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", expiryTime),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "contract_lock",
				"contract_address": hex.EncodeToString(escrowContractAddr),
			},
		}
	} else {
		// MultiKeyLock（买方和卖方都需要签名）
		lockingCondition = map[string]interface{}{
			"type": "multi_key_lock",
			"required_keys": []string{
				hex.EncodeToString(buyerAddress),
				hex.EncodeToString(sellerAddress),
			},
			"threshold": 2, // 需要两个签名
		}
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
			"caller_address": hex.EncodeToString(buyerAddress),
		},
	}

	// 10. 添加托管输出（带 MultiKeyLock）
	outputs := draft["outputs"].([]map[string]interface{})
	escrowOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(buyerAddress), // 托管给买方（但需要双方签名才能解锁）
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	if len(tokenID) > 0 {
		escrowOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, escrowOutput)

	// 11. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(buyerAddress),
			"amount": changeBig.String(),
		}
		if len(tokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(tokenID)
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

// buildEscrowTransaction 构建托管交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// CreateEscrow 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 托管使用 MultiKeyLock 锁定条件（买方和卖方都需要签名才能解锁）。
//
// **流程**：
// 1. 查询买方 UTXO
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 MultiKeyLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildEscrowTransaction(
	ctx context.Context,
	client client.Client,
	buyerAddress []byte,
	sellerAddress []byte,
	amount uint64,
	tokenID []byte,
	expiryTime uint64, // 过期时间（Unix时间戳）
	escrowContractAddr []byte, // Escrow 合约地址（可选）
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	buyerAddressBase58, err := utils.AddressBytesToBase58(buyerAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO
	utxoParams := []interface{}{buyerAddressBase58}
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

	// 4. 转换为 UTXO 结构（根据 tokenID 过滤）
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
		tokenIDStr := getString(utxoMap, "tokenID")
		// 如果提供了 tokenID，只选择匹配的 UTXO；否则选择原生币
		if len(tokenID) > 0 {
			if tokenIDStr == hex.EncodeToString(tokenID) {
				utxos = append(utxos, utxo)
			}
		} else {
			if tokenIDStr == "" {
				utxos = append(utxos, utxo)
			}
		}
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no available UTXOs")
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

	// 9. 构建锁定条件（MultiKeyLock 或 ContractLock + TimeLock）
	var lockingCondition map[string]interface{}
	if len(escrowContractAddr) > 0 {
		// ContractLock + TimeLock（过期后可以退款）
		lockingCondition = map[string]interface{}{
			"type":             "time_lock",
			"unlock_timestamp": fmt.Sprintf("%d", expiryTime),
			"time_source":      "TIME_SOURCE_BLOCK_TIMESTAMP",
			"base_lock": map[string]interface{}{
				"type":             "contract_lock",
				"contract_address": hex.EncodeToString(escrowContractAddr),
			},
		}
	} else {
		// MultiKeyLock（买方和卖方都需要签名）
		lockingCondition = map[string]interface{}{
			"type": "multi_key_lock",
			"required_keys": []string{
				hex.EncodeToString(buyerAddress),
				hex.EncodeToString(sellerAddress),
			},
			"threshold": 2, // 需要两个签名
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
			"caller_address": hex.EncodeToString(buyerAddress),
		},
	}

	// 11. 添加托管输出（带 MultiKeyLock）
	outputs := draft["outputs"].([]map[string]interface{})
	escrowOutput := map[string]interface{}{
		"type":              "asset",
		"owner":             hex.EncodeToString(buyerAddress), // 托管给买方（但需要双方签名才能解锁）
		"amount":            fmt.Sprintf("%d", amount),
		"locking_condition": lockingCondition,
	}
	if len(tokenID) > 0 {
		escrowOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, escrowOutput)

	// 12. 添加找零输出（如果有剩余）
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(buyerAddress),
			"amount": changeBig.String(),
		}
		if len(tokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(tokenID)
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
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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

// buildClaimVestingDraft 构建领取归属代币交易草稿（DraftJSON）
//
// **功能**：
// 构建领取归属代币交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 解析 VestingID（outpoint 格式：txHash:index）
// 2. 查询归属 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费归属 UTXO，返回给受益人）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildClaimVestingDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte, // 领取者地址（受益人）
	vestingID []byte, // VestingID（outpoint 格式：txHash:index）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(vestingID) == 0 {
		return nil, 0, fmt.Errorf("vestingID cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 解析 VestingID（outpoint 格式：txHash:index）
	vestingIDStr := string(vestingID)
	outpointParts := strings.Split(vestingIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid vesting ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 2. 查询归属 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的归属 UTXO
	var vestingUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == vestingIDStr {
			vestingUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
				vestingUTXO.TokenID = tokenIDStr
			}
			break
		}
	}

	if vestingUTXO == nil {
		return nil, 0, fmt.Errorf("vesting UTXO not found: %s", vestingIDStr)
	}

	// 4. 解析归属金额
	vestingAmount, ok := new(big.Int).SetString(vestingUTXO.Amount, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid vesting amount: %s", vestingUTXO.Amount)
	}

	// 5. 计算领取金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，领取金额 = vestingAmount
	claimAmount := vestingAmount
	if claimAmount.Sign() <= 0 {
		return nil, 0, fmt.Errorf("vesting amount too small")
	}

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

	// 7. 添加领取归属代币输出（返回给受益人）
	outputs := draft["outputs"].([]map[string]interface{})
	claimOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(fromAddress),
		"amount": claimAmount.String(),
	}
	if vestingUTXO.TokenID != "" {
		claimOutput["token_id"] = vestingUTXO.TokenID
	}
	draft["outputs"] = append(outputs, claimOutput)

	// 8. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildClaimVestingTransaction 构建领取归属代币交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildClaimVestingDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildClaimVestingDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// ClaimVesting 业务语义在 SDK 层，通过查询归属 UTXO、构建交易实现。
// 领取归属代币需要消费带有 TimeLock 的归属 UTXO（需要满足时间条件）。
//
// **流程**：
// 1. 解析 VestingID（outpoint 格式：txHash:index）
// 2. 查询归属 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费归属 UTXO，返回给受益人）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildClaimVestingTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	vestingID []byte, // VestingID（outpoint 格式：txHash:index）
) ([]byte, error) {
	// 1. 解析 VestingID（outpoint 格式：txHash:index）
	vestingIDStr := string(vestingID)
	outpointParts := strings.Split(vestingIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid vesting ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 2. 查询归属 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的归属 UTXO
	var vestingUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == vestingIDStr {
			vestingUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if vestingUTXO == nil {
		return nil, fmt.Errorf("vesting UTXO not found: %s", vestingIDStr)
	}

	// 4. 解析归属金额
	vestingAmount, ok := new(big.Int).SetString(vestingUTXO.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid vesting amount: %s", vestingUTXO.Amount)
	}

	// 5. 计算领取金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，领取金额 = vestingAmount
	claimAmount := vestingAmount
	if claimAmount.Sign() <= 0 {
		return nil, fmt.Errorf("vesting amount too small to cover fee")
	}

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

	// 8. 添加领取归属代币输出（返回给受益人）
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
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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

// buildReleaseEscrowDraft 构建释放托管交易草稿（DraftJSON）
//
// **功能**：
// 构建释放托管交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 解析 EscrowID（outpoint 格式：txHash:index）
// 2. 查询托管 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费托管 UTXO，返回给卖方）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildReleaseEscrowDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte, // 释放者地址（通常是买方）
	sellerAddress []byte, // 卖方地址
	escrowID []byte, // EscrowID（outpoint 格式：txHash:index）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(sellerAddress) == 0 {
		return nil, 0, fmt.Errorf("sellerAddress cannot be empty")
	}
	if len(escrowID) == 0 {
		return nil, 0, fmt.Errorf("escrowID cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 解析 EscrowID（outpoint 格式：txHash:index）
	escrowIDStr := string(escrowID)
	outpointParts := strings.Split(escrowIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid escrow ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 2. 查询托管 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的托管 UTXO
	var escrowUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == escrowIDStr {
			escrowUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
				escrowUTXO.TokenID = tokenIDStr
			}
			break
		}
	}

	if escrowUTXO == nil {
		return nil, 0, fmt.Errorf("escrow UTXO not found: %s", escrowIDStr)
	}

	// 4. 解析托管金额
	escrowAmount, ok := new(big.Int).SetString(escrowUTXO.Amount, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid escrow amount: %s", escrowUTXO.Amount)
	}

	// 5. 计算释放金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，释放金额 = escrowAmount
	releaseAmount := escrowAmount
	if releaseAmount.Sign() <= 0 {
		return nil, 0, fmt.Errorf("escrow amount too small")
	}

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

	// 7. 添加释放托管输出（返回给卖方）
	outputs := draft["outputs"].([]map[string]interface{})
	releaseOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(sellerAddress),
		"amount": releaseAmount.String(),
	}
	if escrowUTXO.TokenID != "" {
		releaseOutput["token_id"] = escrowUTXO.TokenID
	}
	draft["outputs"] = append(outputs, releaseOutput)

	// 8. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildReleaseEscrowTransaction 构建释放托管交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildReleaseEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildReleaseEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// ReleaseEscrow 业务语义在 SDK 层，通过查询托管 UTXO、构建交易实现。
// 释放托管需要消费带有 MultiKeyLock 的托管 UTXO（需要买方和卖方签名）。
//
// **流程**：
// 1. 解析 EscrowID（outpoint 格式：txHash:index）
// 2. 查询托管 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费托管 UTXO，返回给卖方）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildReleaseEscrowTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte, // 释放者地址（通常是买方）
	sellerAddress []byte, // 卖方地址
	escrowID []byte, // EscrowID（outpoint 格式：txHash:index）
) ([]byte, error) {
	// 1. 解析 EscrowID（outpoint 格式：txHash:index）
	escrowIDStr := string(escrowID)
	outpointParts := strings.Split(escrowIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid escrow ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 2. 查询托管 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的托管 UTXO
	var escrowUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == escrowIDStr {
			escrowUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if escrowUTXO == nil {
		return nil, fmt.Errorf("escrow UTXO not found: %s", escrowIDStr)
	}

	// 4. 解析托管金额
	escrowAmount, ok := new(big.Int).SetString(escrowUTXO.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid escrow amount: %s", escrowUTXO.Amount)
	}

	// 5. 计算释放金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，释放金额 = escrowAmount
	releaseAmount := escrowAmount
	if releaseAmount.Sign() <= 0 {
		return nil, fmt.Errorf("escrow amount too small to cover fee")
	}

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

	// 8. 添加释放托管输出（返回给卖方）
	outputs := draft["outputs"].([]map[string]interface{})
	releaseOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(sellerAddress),
		"amount": releaseAmount.String(),
	}
	draft["outputs"] = append(outputs, releaseOutput)

	// 9. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 10. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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

// buildRefundEscrowDraft 构建退款托管交易草稿（DraftJSON）
//
// **功能**：
// 构建退款托管交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 解析 EscrowID（outpoint 格式：txHash:index）
// 2. 查询托管 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费托管 UTXO，返回给买方）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildRefundEscrowDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte, // 退款者地址（通常是买方或卖方）
	buyerAddress []byte, // 买方地址
	escrowID []byte, // EscrowID（outpoint 格式：txHash:index）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(buyerAddress) == 0 {
		return nil, 0, fmt.Errorf("buyerAddress cannot be empty")
	}
	if len(escrowID) == 0 {
		return nil, 0, fmt.Errorf("escrowID cannot be empty")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 解析 EscrowID（outpoint 格式：txHash:index）
	escrowIDStr := string(escrowID)
	outpointParts := strings.Split(escrowIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid escrow ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	inputIndex := uint32(0) // 只有一个输入，索引为0

	// 2. 查询托管 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的托管 UTXO
	var escrowUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == escrowIDStr {
			escrowUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
				escrowUTXO.TokenID = tokenIDStr
			}
			break
		}
	}

	if escrowUTXO == nil {
		return nil, 0, fmt.Errorf("escrow UTXO not found: %s", escrowIDStr)
	}

	// 4. 解析托管金额
	escrowAmount, ok := new(big.Int).SetString(escrowUTXO.Amount, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid escrow amount: %s", escrowUTXO.Amount)
	}

	// 5. 计算退款金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，退款金额 = escrowAmount
	refundAmount := escrowAmount
	if refundAmount.Sign() <= 0 {
		return nil, 0, fmt.Errorf("escrow amount too small")
	}

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

	// 7. 添加退款托管输出（返回给买方）
	outputs := draft["outputs"].([]map[string]interface{})
	refundOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(buyerAddress),
		"amount": refundAmount.String(),
	}
	if escrowUTXO.TokenID != "" {
		refundOutput["token_id"] = escrowUTXO.TokenID
	}
	draft["outputs"] = append(outputs, refundOutput)

	// 8. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildRefundEscrowTransaction 构建退款托管交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildRefundEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildRefundEscrowDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// RefundEscrow 业务语义在 SDK 层，通过查询托管 UTXO、构建交易实现。
// 退款托管需要消费带有 MultiKeyLock 的托管 UTXO（过期后可以退款给买方）。
//
// **流程**：
// 1. 解析 EscrowID（outpoint 格式：txHash:index）
// 2. 查询托管 UTXO（通过 `wes_getUTXO`）
// 3. 构建交易草稿（消费托管 UTXO，返回给买方）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildRefundEscrowTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte, // 退款者地址（通常是买方或卖方）
	buyerAddress []byte, // 买方地址
	escrowID []byte, // EscrowID（outpoint 格式：txHash:index）
) ([]byte, error) {
	// 1. 解析 EscrowID（outpoint 格式：txHash:index）
	escrowIDStr := string(escrowID)
	outpointParts := strings.Split(escrowIDStr, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid escrow ID format, expected txHash:index")
	}

	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 2. 查询托管 UTXO（通过查询用户的 UTXO 列表，找到对应的 UTXO）
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

	// 3. 查找对应的托管 UTXO
	var escrowUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		outpoint := getString(utxoMap, "outpoint")
		if outpoint == escrowIDStr {
			escrowUTXO = &UTXO{
				Outpoint: outpoint,
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if escrowUTXO == nil {
		return nil, fmt.Errorf("escrow UTXO not found: %s", escrowIDStr)
	}

	// 4. 解析托管金额
	escrowAmount, ok := new(big.Int).SetString(escrowUTXO.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid escrow amount: %s", escrowUTXO.Amount)
	}

	// 5. 计算退款金额
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，退款金额 = escrowAmount
	refundAmount := escrowAmount
	if refundAmount.Sign() <= 0 {
		return nil, fmt.Errorf("escrow amount too small to cover fee")
	}

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

	// 8. 添加退款托管输出（返回给买方）
	outputs := draft["outputs"].([]map[string]interface{})
	refundOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(buyerAddress),
		"amount": refundAmount.String(),
	}
	draft["outputs"] = append(outputs, refundOutput)

	// 9. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 10. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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
