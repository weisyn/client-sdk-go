package governance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// buildProposeDraft 构建提案交易草稿（DraftJSON）
//
// **功能**：
// 构建提案交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询提案者 UTXO（用于支付手续费）
// 2. 选择第一个原生币 UTXO
// 3. 构建交易草稿（包含 StateOutput + ThresholdLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildProposeDraft(
	ctx context.Context,
	client client.Client,
	proposerAddress []byte,
	title string,
	description string,
	votingPeriod uint64, // 投票期限（区块数）
	validatorAddresses [][]byte, // 验证者地址列表（用于 ThresholdLock）
	threshold uint32, // 门限值（需要多少个签名）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(proposerAddress) == 0 {
		return nil, 0, fmt.Errorf("proposerAddress cannot be empty")
	}
	if title == "" {
		return nil, 0, fmt.Errorf("title cannot be empty")
	}
	if votingPeriod == 0 {
		return nil, 0, fmt.Errorf("votingPeriod must be greater than 0")
	}
	if len(validatorAddresses) == 0 {
		return nil, 0, fmt.Errorf("validatorAddresses cannot be empty")
	}
	if threshold == 0 {
		return nil, 0, fmt.Errorf("threshold must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	proposerAddressBase58, err := utils.AddressBytesToBase58(proposerAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{proposerAddressBase58}
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

	// 4. 选择第一个可用 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		selectedUTXO = &UTXO{
			Outpoint: getString(utxoMap, "outpoint"),
			Height:   getString(utxoMap, "height"),
			Amount:   getString(utxoMap, "amount"),
			TokenID:  getString(utxoMap, "tokenID"),
		}
		break
	}

	if selectedUTXO == nil {
		return nil, 0, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
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

	// 6. 构建提案数据（存储在 StateOutput 中）
	proposalData := map[string]interface{}{
		"type":          "proposal",
		"title":         title,
		"description":   description,
		"voting_period": votingPeriod,
		"proposer":      hex.EncodeToString(proposerAddress),
	}
	proposalDataJSON, err := json.Marshal(proposalData)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal proposal data failed: %w", err)
	}

	// 8. 为 StateOutput 构建元数据（满足节点端 state 输出要求）
	// 根据提案数据生成一个 deterministic 的 state_id（仅用于测试与追踪）
	stateHash := sha256.Sum256(proposalDataJSON)
	stateIDHex := hex.EncodeToString(stateHash[:])

	stateMetadata := map[string]interface{}{
		"state_id":      stateIDHex,
		"state_version": uint64(1),
		// 其他字段（execution_result_hash / public_inputs 等）可以留空，由节点使用默认值
	}

	// 9. 构建交易草稿（DraftJSON）
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
			"caller_address": hex.EncodeToString(proposerAddress),
		},
	}

	// 10. 添加提案 StateOutput
	// 注意：
	// - OutputSpec 要求字段: type/owner/amount/token_id/metadata
	// - metadata 中的 state_id 是必填字段，否则节点端会返回“状态 state_id 不能为空”
	outputs := draft["outputs"].([]map[string]interface{})
	proposalOutput := map[string]interface{}{
		"type":     "state",
		"owner":    hex.EncodeToString(proposerAddress),
		"amount":   "0", // 状态输出本身不携带资产金额
		"token_id": "",  // 状态输出不关联 token
		"metadata": stateMetadata,
		// 提案内容仍然保留在 data 字段，便于后续扩展或调试（节点目前不强制要求）
		"data": string(proposalDataJSON),
	}
	draft["outputs"] = append(outputs, proposalOutput)

	// 11. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildProposeTransaction 构建提案交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildProposeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildProposeDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Propose 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 提案使用 StateOutput + MultiKeyLock/ThresholdLock 锁定条件。
//
// **流程**：
// 1. 查询提案者 UTXO（用于支付手续费）
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 StateOutput + MultiKeyLock/ThresholdLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildProposeTransaction(
	ctx context.Context,
	client client.Client,
	proposerAddress []byte,
	title string,
	description string,
	votingPeriod uint64, // 投票期限（区块数）
	validatorAddresses [][]byte, // 验证者地址列表（用于 ThresholdLock）
	threshold uint32, // 门限值（需要多少个签名）
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	proposerAddressBase58, err := utils.AddressBytesToBase58(proposerAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{proposerAddressBase58}
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

	// 4. 选择第一个原生币 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tokenIDStr := getString(utxoMap, "tokenID")
		if tokenIDStr == "" {
			selectedUTXO = &UTXO{
				Outpoint: getString(utxoMap, "outpoint"),
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if selectedUTXO == nil {
		return nil, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 6. 构建 ThresholdLock 锁定条件
	requiredKeys := make([]string, len(validatorAddresses))
	for i, addr := range validatorAddresses {
		requiredKeys[i] = hex.EncodeToString(addr)
	}

	thresholdLock := map[string]interface{}{
		"type":          "threshold_lock",
		"required_keys": requiredKeys,
		"threshold":     threshold,
	}

	// 7. 构建提案数据（存储在 StateOutput 中）
	proposalData := map[string]interface{}{
		"type":          "proposal",
		"title":         title,
		"description":   description,
		"voting_period": votingPeriod,
		"proposer":      hex.EncodeToString(proposerAddress),
	}
	proposalDataJSON, err := json.Marshal(proposalData)
	if err != nil {
		return nil, fmt.Errorf("marshal proposal data failed: %w", err)
	}

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
			"caller_address": hex.EncodeToString(proposerAddress),
		},
	}

	// 9. 添加提案 StateOutput（带 ThresholdLock）
	outputs := draft["outputs"].([]map[string]interface{})
	proposalOutput := map[string]interface{}{
		"type":              "state",
		"owner":             hex.EncodeToString(proposerAddress),
		"data":              string(proposalDataJSON),
		"locking_condition": thresholdLock,
	}
	draft["outputs"] = append(outputs, proposalOutput)

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 11. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 12. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 13. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildVoteDraft 构建投票交易草稿（DraftJSON）
//
// **功能**：
// 构建投票交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询投票者 UTXO（用于支付手续费）
// 2. 选择第一个原生币 UTXO
// 3. 构建交易草稿（包含 StateOutput + SingleKeyLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildVoteDraft(
	ctx context.Context,
	client client.Client,
	voterAddress []byte,
	proposalID []byte, // ProposalID（outpoint 格式：txHash:index）
	choice int, // 投票选择（1=支持, 0=反对, -1=弃权）
	voteWeight uint64, // 投票权重
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(voterAddress) == 0 {
		return nil, 0, fmt.Errorf("voterAddress cannot be empty")
	}
	if len(proposalID) == 0 {
		return nil, 0, fmt.Errorf("proposalID cannot be empty")
	}
	if voteWeight == 0 {
		return nil, 0, fmt.Errorf("voteWeight must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	voterAddressBase58, err := utils.AddressBytesToBase58(voterAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{voterAddressBase58}
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

	// 4. 选择第一个原生币 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tokenIDStr := getString(utxoMap, "tokenID")
		if tokenIDStr == "" {
			selectedUTXO = &UTXO{
				Outpoint: getString(utxoMap, "outpoint"),
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if selectedUTXO == nil {
		return nil, 0, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
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

	// 6. 构建投票数据（存储在 StateOutput 中）
	voteData := map[string]interface{}{
		"type":        "vote",
		"proposal_id": string(proposalID),
		"choice":      choice,
		"weight":      voteWeight,
		"voter":       hex.EncodeToString(voterAddress),
	}
	voteDataJSON, err := json.Marshal(voteData)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal vote data failed: %w", err)
	}

	// 7. 构建 SingleKeyLock 锁定条件
	singleKeyLock := map[string]interface{}{
		"type":             "single_key_lock",
		"required_address": hex.EncodeToString(voterAddress),
	}

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
			"caller_address": hex.EncodeToString(voterAddress),
		},
	}

	// 9. 添加投票 StateOutput（带 SingleKeyLock）
	outputs := draft["outputs"].([]map[string]interface{})
	voteOutput := map[string]interface{}{
		"type":              "state",
		"owner":             hex.EncodeToString(voterAddress),
		"data":              string(voteDataJSON),
		"locking_condition": singleKeyLock,
	}
	draft["outputs"] = append(outputs, voteOutput)

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildVoteTransaction 构建投票交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildVoteDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildVoteDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// Vote 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 投票使用 StateOutput + SingleKeyLock 锁定条件。
func buildVoteTransaction(
	ctx context.Context,
	client client.Client,
	voterAddress []byte,
	proposalID []byte, // ProposalID（outpoint 格式：txHash:index）
	choice int, // 投票选择（1=支持, 0=反对, -1=弃权）
	voteWeight uint64, // 投票权重
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	voterAddressBase58, err := utils.AddressBytesToBase58(voterAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{voterAddressBase58}
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

	// 4. 选择第一个原生币 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tokenIDStr := getString(utxoMap, "tokenID")
		if tokenIDStr == "" {
			selectedUTXO = &UTXO{
				Outpoint: getString(utxoMap, "outpoint"),
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if selectedUTXO == nil {
		return nil, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 6. 构建投票数据（存储在 StateOutput 中）
	voteData := map[string]interface{}{
		"type":        "vote",
		"proposal_id": string(proposalID),
		"choice":      choice,
		"weight":      voteWeight,
		"voter":       hex.EncodeToString(voterAddress),
	}
	voteDataJSON, err := json.Marshal(voteData)
	if err != nil {
		return nil, fmt.Errorf("marshal vote data failed: %w", err)
	}

	// 7. 构建 SingleKeyLock 锁定条件
	singleKeyLock := map[string]interface{}{
		"type":             "single_key_lock",
		"required_address": hex.EncodeToString(voterAddress),
	}

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
			"caller_address": hex.EncodeToString(voterAddress),
		},
	}

	// 9. 添加投票 StateOutput（带 SingleKeyLock）
	outputs := draft["outputs"].([]map[string]interface{})
	voteOutput := map[string]interface{}{
		"type":              "state",
		"owner":             hex.EncodeToString(voterAddress),
		"data":              string(voteDataJSON),
		"locking_condition": singleKeyLock,
	}
	draft["outputs"] = append(outputs, voteOutput)

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 11. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 12. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 13. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildUpdateParamDraft 构建更新参数交易草稿（DraftJSON）
//
// **功能**：
// 构建更新参数交易的草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询提案者 UTXO（用于支付手续费）
// 2. 选择第一个原生币 UTXO
// 3. 构建交易草稿（包含 StateOutput + ThresholdLock）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildUpdateParamDraft(
	ctx context.Context,
	client client.Client,
	proposerAddress []byte,
	paramKey string,
	paramValue string,
	validatorAddresses [][]byte, // 验证者地址列表（用于 ThresholdLock）
	threshold uint32, // 门限值（需要多少个签名）
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(proposerAddress) == 0 {
		return nil, 0, fmt.Errorf("proposerAddress cannot be empty")
	}
	if paramKey == "" {
		return nil, 0, fmt.Errorf("paramKey cannot be empty")
	}
	if len(validatorAddresses) == 0 {
		return nil, 0, fmt.Errorf("validatorAddresses cannot be empty")
	}
	if threshold == 0 {
		return nil, 0, fmt.Errorf("threshold must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式
	proposerAddressBase58, err := utils.AddressBytesToBase58(proposerAddress)
	if err != nil {
		return nil, 0, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{proposerAddressBase58}
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

	// 4. 选择第一个原生币 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tokenIDStr := getString(utxoMap, "tokenID")
		if tokenIDStr == "" {
			selectedUTXO = &UTXO{
				Outpoint: getString(utxoMap, "outpoint"),
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if selectedUTXO == nil {
		return nil, 0, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
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

	// 6. 构建 ThresholdLock 锁定条件
	requiredKeys := make([]string, len(validatorAddresses))
	for i, addr := range validatorAddresses {
		requiredKeys[i] = hex.EncodeToString(addr)
	}

	thresholdLock := map[string]interface{}{
		"type":          "threshold_lock",
		"required_keys": requiredKeys,
		"threshold":     threshold,
	}

	// 7. 构建参数更新数据（存储在 StateOutput 中）
	paramUpdateData := map[string]interface{}{
		"type":        "param_update",
		"param_key":   paramKey,
		"param_value": paramValue,
		"proposer":    hex.EncodeToString(proposerAddress),
	}
	paramUpdateDataJSON, err := json.Marshal(paramUpdateData)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal param update data failed: %w", err)
	}

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
			"caller_address": hex.EncodeToString(proposerAddress),
		},
	}

	// 9. 添加参数更新 StateOutput（带 ThresholdLock）
	outputs := draft["outputs"].([]map[string]interface{})
	paramUpdateOutput := map[string]interface{}{
		"type":              "state",
		"owner":             hex.EncodeToString(proposerAddress),
		"data":              string(paramUpdateDataJSON),
		"locking_condition": thresholdLock,
	}
	draft["outputs"] = append(outputs, paramUpdateOutput)

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildUpdateParamTransaction 构建更新参数交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildUpdateParamDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildUpdateParamDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// UpdateParam 业务语义在 SDK 层，通过查询 UTXO、构建交易实现。
// 更新参数使用 StateOutput + ThresholdLock 锁定条件（需要治理投票通过）。
//
// **流程**：
// 1. 查询提案者 UTXO（用于支付手续费）
// 2. 选择足够的 UTXO
// 3. 构建交易草稿（包含 StateOutput + ThresholdLock）
// 4. 调用 `wes_buildTransaction` API 获取未签名交易
func buildUpdateParamTransaction(
	ctx context.Context,
	client client.Client,
	proposerAddress []byte,
	paramKey string,
	paramValue string,
	validatorAddresses [][]byte, // 验证者地址列表（用于 ThresholdLock）
	threshold uint32, // 门限值（需要多少个签名）
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式
	proposerAddressBase58, err := utils.AddressBytesToBase58(proposerAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 2. 查询 UTXO（用于支付手续费）
	utxoParams := []interface{}{proposerAddressBase58}
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

	// 4. 选择第一个原生币 UTXO（用于支付手续费）
	var selectedUTXO *UTXO
	for _, item := range utxosArray {
		utxoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tokenIDStr := getString(utxoMap, "tokenID")
		if tokenIDStr == "" {
			selectedUTXO = &UTXO{
				Outpoint: getString(utxoMap, "outpoint"),
				Height:   getString(utxoMap, "height"),
				Amount:   getString(utxoMap, "amount"),
			}
			break
		}
	}

	if selectedUTXO == nil {
		return nil, fmt.Errorf("no available native coin UTXO for fee")
	}

	// 5. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 6. 构建 ThresholdLock 锁定条件
	requiredKeys := make([]string, len(validatorAddresses))
	for i, addr := range validatorAddresses {
		requiredKeys[i] = hex.EncodeToString(addr)
	}

	thresholdLock := map[string]interface{}{
		"type":          "threshold_lock",
		"required_keys": requiredKeys,
		"threshold":     threshold,
	}

	// 7. 构建参数更新数据（存储在 StateOutput 中）
	paramUpdateData := map[string]interface{}{
		"type":        "param_update",
		"param_key":   paramKey,
		"param_value": paramValue,
		"proposer":    hex.EncodeToString(proposerAddress),
	}
	paramUpdateDataJSON, err := json.Marshal(paramUpdateData)
	if err != nil {
		return nil, fmt.Errorf("marshal param update data failed: %w", err)
	}

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
			"caller_address": hex.EncodeToString(proposerAddress),
		},
	}

	// 9. 添加参数更新 StateOutput（带 ThresholdLock）
	outputs := draft["outputs"].([]map[string]interface{})
	paramUpdateOutput := map[string]interface{}{
		"type":              "state",
		"owner":             hex.EncodeToString(proposerAddress),
		"data":              string(paramUpdateDataJSON),
		"locking_condition": thresholdLock,
	}
	draft["outputs"] = append(outputs, paramUpdateOutput)

	// 10. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 11. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 12. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 13. 解码未签名交易
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
