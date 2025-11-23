package token

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

// txBuilder 交易构建辅助工具（SDK 层实现，不依赖 WES 内部类型）
type txBuilder struct {
	client client.Client
}

// UTXO UTXO 信息（从 wes_getUTXO API 返回）
type UTXO struct {
	Outpoint string `json:"outpoint"`          // "txHash:outputIndex"
	Height   string `json:"height"`            // "0x..."
	Amount   string `json:"amount"`            // 金额（字符串）
	TokenID  string `json:"tokenID,omitempty"` // 代币ID（hex编码，可选）
}

// TransactionDraft 交易草稿（JSON 格式，用于构建交易）
type TransactionDraft struct {
	Version           uint32          `json:"version"`
	Inputs            []TxInputDraft  `json:"inputs"`
	Outputs           []TxOutputDraft `json:"outputs"`
	Nonce             uint64          `json:"nonce"`
	CreationTimestamp uint64          `json:"creation_timestamp"`
	ChainId           string          `json:"chain_id"`
}

// TxInputDraft 交易输入草稿
type TxInputDraft struct {
	TxHash          string `json:"tx_hash"` // 十六进制
	OutputIndex     uint32 `json:"output_index"`
	IsReferenceOnly bool   `json:"is_reference_only"`
	Sequence        uint32 `json:"sequence"`
}

// TxOutputDraft 交易输出草稿
type TxOutputDraft struct {
	Owner            string                 `json:"owner"`       // 十六进制地址
	OutputType       string                 `json:"output_type"` // "asset" | "resource" | "state"
	AssetContent     *AssetOutputDraft      `json:"asset_content,omitempty"`
	LockingCondition *LockingConditionDraft `json:"locking_condition"`
}

// AssetOutputDraft 资产输出草稿
type AssetOutputDraft struct {
	AssetType string `json:"asset_type"` // "native_coin" | "contract_token"
	Amount    string `json:"amount"`     // 金额（字符串）
	// 合约代币字段
	ContractAddress string `json:"contract_address,omitempty"` // 十六进制
	TokenID         string `json:"token_id,omitempty"`         // 十六进制
}

// LockingConditionDraft 锁定条件草稿
type LockingConditionDraft struct {
	Type            string `json:"type"`             // "single_key_lock"
	RequiredAddress string `json:"required_address"` // 十六进制地址
}

// buildTransferTransaction 构建单笔转账交易（SDK 层实现）
//
// **架构说明**：
// Transfer 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 过滤匹配 tokenID 的 UTXO
// 3. 选择足够的 UTXO
// 4. 计算手续费和找零
// 5. 构建交易草稿（JSON 格式）
// 6. 调用 `wes_buildTransaction` API 获取未签名交易
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildTransferDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildTransferDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
func buildTransferTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	toAddress []byte,
	amount uint64,
	tokenID []byte,
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

	// 4. 转换为 UTXO 结构
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
		// 尝试获取 tokenID（如果存在）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
			utxo.TokenID = tokenIDStr
		}
		utxos = append(utxos, utxo)
	}

	if len(utxos) == 0 {
		return nil, fmt.Errorf("no available UTXOs")
	}

	// 5. 过滤匹配 tokenID 的 UTXO
	var matchingUTXOs []UTXO
	tokenIDHex := ""
	if len(tokenID) > 0 {
		tokenIDHex = hex.EncodeToString(tokenID)
	}

	for _, utxo := range utxos {
		// 如果 tokenID 为空，匹配原生币（没有 tokenID 的 UTXO）
		if len(tokenID) == 0 {
			if utxo.TokenID == "" {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		} else {
			// 如果 tokenID 不为空，匹配相同 tokenID 的 UTXO
			if utxo.TokenID == tokenIDHex {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		}
	}

	if len(matchingUTXOs) == 0 {
		if len(tokenID) == 0 {
			return nil, fmt.Errorf("no matching UTXOs for native coin")
		}
		return nil, fmt.Errorf("no matching UTXOs for tokenID: %s", tokenIDHex)
	}

	// 6. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range matchingUTXOs {
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
		return nil, fmt.Errorf("insufficient balance: required %d, but no UTXO found with sufficient amount", amount)
	}

	// 7. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, fmt.Errorf("invalid output index: %w", err)
	}

	// 8. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 9. 构建交易草稿（符合 host_build_transaction 的 DraftJSON 格式）
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

	// 10. 添加转账输出
	outputs := draft["outputs"].([]map[string]interface{})
	transferOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(toAddress),
		"amount": fmt.Sprintf("%d", amount),
	}
	if len(tokenID) > 0 {
		transferOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, transferOutput)

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

// buildBurnTransaction 构建 Burn 交易（SDK 层实现）
//
// **架构说明**：
// Burn 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
//  1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
//  2. 过滤匹配 tokenID 的 UTXO
//  3. 选择足够的 UTXO
//  4. 计算手续费和找零
//  5. 构建交易草稿（JSON 格式）
//  6. 调用 `wes_buildTransaction` API 获取未签名交易（如果存在）
//     或者直接构建 protobuf 交易（需要 SDK 知道 protobuf 格式）
//
// **注意**：
// - SDK 层不应该依赖 WES 内部类型
// - 当前简化：假设节点提供 `wes_buildTransaction` API
// - 如果没有，需要 SDK 层实现 protobuf 序列化（但 SDK 不应该依赖 WES 类型）
// buildBurnDraft 构建销毁交易草稿（DraftJSON）
//
// **功能**：
// 构建销毁代币的交易草稿，返回 DraftJSON 字节数组和输入索引。
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 过滤匹配 tokenID 的 UTXO
// 3. 选择足够的 UTXO
// 4. 计算手续费和找零
// 5. 构建交易草稿（JSON 格式）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引（用于签名）
func buildBurnDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	amount uint64,
	tokenID []byte,
) ([]byte, uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, 0, fmt.Errorf("fromAddress cannot be empty")
	}
	if amount == 0 {
		return nil, 0, fmt.Errorf("amount must be greater than 0")
	}
	if client == nil {
		return nil, 0, fmt.Errorf("client cannot be nil")
	}

	// 1. 将地址转换为 Base58 格式（wes_getUTXO 需要 Base58 地址）
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

	// 4. 转换为 UTXO 结构
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
		utxos = append(utxos, utxo)
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available UTXOs")
	}

	// 5. 过滤匹配 tokenID 的 UTXO
	var matchingUTXOs []UTXO
	tokenIDHex := ""
	if len(tokenID) > 0 {
		tokenIDHex = hex.EncodeToString(tokenID)
	}

	for _, utxo := range utxos {
		// 如果 tokenID 为空，匹配原生币（没有 tokenID 的 UTXO）
		if len(tokenID) == 0 {
			if utxo.TokenID == "" {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		} else {
			// 如果 tokenID 不为空，匹配相同 tokenID 的 UTXO
			if utxo.TokenID == tokenIDHex {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		}
	}

	if len(matchingUTXOs) == 0 {
		if len(tokenID) == 0 {
			return nil, 0, fmt.Errorf("no matching UTXOs for native coin")
		}
		return nil, 0, fmt.Errorf("no matching UTXOs for tokenID: %s", tokenIDHex)
	}

	// 6. 选择足够的 UTXO
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range matchingUTXOs {
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

	// 7. 解析 outpoint
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

	// 8. 计算找零
	// 注意：手续费从接收者扣除，Burn 操作没有接收者，手续费由节点端从销毁金额中扣除
	// 发送者只需要支付销毁金额，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 10. 构建交易草稿（符合 host_build_transaction 的 DraftJSON 格式）
	// DraftJSON 格式：
	// - inputs: []InputSpec {tx_hash, output_index, is_reference_only}
	// - outputs: []OutputSpec {type, owner, amount, token_id}
	// - sign_mode: "defer_sign" (返回未签名交易)
	// - metadata: {caller_address}
	draft := map[string]interface{}{
		"sign_mode": "defer_sign", // 延迟签名模式，返回未签名交易
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

	// 11. 添加找零输出（如果有剩余）
	// OutputSpec 格式：{type: "asset", owner: hex, amount: string, token_id: hex (optional)}
	if changeBig.Sign() > 0 {
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		// 如果有 tokenID，添加到输出中
		if len(tokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(tokenID)
		}
		outputs := draft["outputs"].([]map[string]interface{})
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 12. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndex, nil
}

// buildBurnTransaction 构建销毁交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildBurnDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildBurnDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
func buildBurnTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	amount uint64,
	tokenID []byte,
) ([]byte, error) {
	// 调用新的 buildBurnDraft 函数
	draftJSON, _, err := buildBurnDraft(ctx, client, fromAddress, amount, tokenID)
	if err != nil {
		return nil, err
	}

	// 调用 wes_buildTransaction API（保持向后兼容）
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", buildTxParams)
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildBatchTransferTransaction 构建批量转账交易（SDK 层实现）
//
// ⚠️ 已废弃：此函数已不再使用，请使用 buildBatchTransferDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft 路径
// Deprecated: This function is deprecated. Use buildBatchTransferDraft + wes_computeSignatureHashFromDraft + wes_finalizeTransactionFromDraft instead.
//
// **架构说明**：
// BatchTransfer 业务语义在 SDK 层，通过查询 UTXO、选择 UTXO、构建交易实现。
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 按 tokenID 分组 UTXO
// 3. 为每个转账选择足够的 UTXO
// 4. 计算手续费和找零
// 5. 构建交易草稿（JSON 格式）
// 6. 调用 `wes_buildTransaction` API 获取未签名交易
func buildBatchTransferTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	transfers []TransferItem,
) ([]byte, error) {
	if len(transfers) == 0 {
		return nil, fmt.Errorf("transfers list cannot be empty")
	}

	// 1. 验证所有转账使用同一个 tokenID
	var commonTokenID []byte
	var commonTokenIDHex string
	for i, transfer := range transfers {
		if i == 0 {
			// 第一个转账的 tokenID 作为标准
			commonTokenID = transfer.TokenID
			if len(transfer.TokenID) > 0 {
				commonTokenIDHex = hex.EncodeToString(transfer.TokenID)
			} else {
				commonTokenIDHex = "native"
			}
		} else {
			// 验证后续转账的 tokenID 是否一致
			currentTokenIDHex := "native"
			if len(transfer.TokenID) > 0 {
				currentTokenIDHex = hex.EncodeToString(transfer.TokenID)
			}
			if currentTokenIDHex != commonTokenIDHex {
				return nil, fmt.Errorf("all transfers must use the same tokenID, found %s and %s", commonTokenIDHex, currentTokenIDHex)
			}
		}
	}

	// 2. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 3. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 4. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid UTXOs format")
	}

	// 5. 转换为 UTXO 结构并过滤匹配的 tokenID
	var matchingUTXOs []UTXO
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
		// 尝试获取 tokenID（如果存在）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
			utxo.TokenID = tokenIDStr
		}

		// 过滤匹配的 tokenID
		utxoTokenIDHex := "native"
		if utxo.TokenID != "" {
			utxoTokenIDHex = utxo.TokenID
		}
		if utxoTokenIDHex == commonTokenIDHex {
			matchingUTXOs = append(matchingUTXOs, utxo)
		}
	}

	if len(matchingUTXOs) == 0 {
		return nil, fmt.Errorf("no matching UTXOs for tokenID: %s", commonTokenIDHex)
	}

	// 6. 构建交易草稿（符合 host_build_transaction 的 DraftJSON 格式）
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs":    []map[string]interface{}{},
		"outputs":   []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 7. 为每个转账选择 UTXO 并添加输入和输出
	selectedUTXOs := make(map[string]UTXO) // 记录已选择的 UTXO（避免重复使用）
	totalInputAmount := big.NewInt(0)
	totalOutputAmount := big.NewInt(0)

	for _, transfer := range transfers {
		// 选择足够的 UTXO
		requiredAmount := big.NewInt(int64(transfer.Amount))
		var selectedUTXO UTXO
		var selectedAmount *big.Int
		for _, utxo := range matchingUTXOs {
			// 检查是否已使用
			if _, used := selectedUTXOs[utxo.Outpoint]; used {
				continue
			}

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
				selectedUTXOs[utxo.Outpoint] = utxo
				break
			}
		}

		if selectedUTXO.Outpoint == "" {
			return nil, fmt.Errorf("insufficient balance for transfer to %s, amount %d", hex.EncodeToString(transfer.To), transfer.Amount)
		}

		// 解析 outpoint
		outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
		if len(outpointParts) != 2 {
			return nil, fmt.Errorf("invalid outpoint format: %s", selectedUTXO.Outpoint)
		}
		txHash := outpointParts[0]
		var outputIndex uint32
		if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
			return nil, fmt.Errorf("invalid output index: %w", err)
		}

		// 添加输入
		inputs := draft["inputs"].([]map[string]interface{})
		draft["inputs"] = append(inputs, map[string]interface{}{
			"tx_hash":           txHash,
			"output_index":      outputIndex,
			"is_reference_only": false,
		})

		// 添加转账输出（所有转账使用同一个 tokenID）
		outputs := draft["outputs"].([]map[string]interface{})
		transferOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(transfer.To),
			"amount": fmt.Sprintf("%d", transfer.Amount),
		}
		if len(commonTokenID) > 0 {
			transferOutput["token_id"] = hex.EncodeToString(commonTokenID)
		}
		draft["outputs"] = append(outputs, transferOutput)

		// 累计金额
		totalInputAmount.Add(totalInputAmount, selectedAmount)
		totalOutputAmount.Add(totalOutputAmount, requiredAmount)
	}

	// 8. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = totalInputAmount - totalOutputAmount
	changeBig := new(big.Int).Sub(totalInputAmount, totalOutputAmount)

	// 10. 添加找零输出（如果有剩余，使用共同的 tokenID）
	if changeBig.Sign() > 0 {
		outputs := draft["outputs"].([]map[string]interface{})
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		// 找零使用共同的 tokenID（所有转账使用同一个 tokenID）
		if len(commonTokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(commonTokenID)
		}
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

// buildBatchTransferDraft 构建批量转账交易草稿（DraftJSON）
//
// **功能**：
// 构建批量转账的交易草稿，返回 DraftJSON 字节数组。
//
// **流程**：
// 1. 查询发送方的 UTXO（通过 `wes_getUTXO` API）
// 2. 按 tokenID 分组 UTXO
// 3. 为每个转账选择足够的 UTXO
// 4. 计算手续费和找零
// 5. 构建交易草稿（JSON 格式）
//
// **返回**：
// - DraftJSON 字节数组
// - 输入索引列表（每个输入对应的索引）
func buildBatchTransferDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	transfers []TransferItem,
) ([]byte, []uint32, error) {
	// 0. 参数验证
	if len(fromAddress) == 0 {
		return nil, nil, fmt.Errorf("fromAddress cannot be empty")
	}
	if len(transfers) == 0 {
		return nil, nil, fmt.Errorf("transfers list cannot be empty")
	}
	if client == nil {
		return nil, nil, fmt.Errorf("client cannot be nil")
	}
	// 验证每个转账项
	for i, transfer := range transfers {
		if len(transfer.To) == 0 {
			return nil, nil, fmt.Errorf("transfer[%d]: toAddress cannot be empty", i)
		}
		if transfer.Amount == 0 {
			return nil, nil, fmt.Errorf("transfer[%d]: amount must be greater than 0", i)
		}
	}

	// 1. 验证所有转账使用同一个 tokenID
	var commonTokenID []byte
	var commonTokenIDHex string
	for i, transfer := range transfers {
		if i == 0 {
			commonTokenID = transfer.TokenID
			if len(transfer.TokenID) > 0 {
				commonTokenIDHex = hex.EncodeToString(transfer.TokenID)
			} else {
				commonTokenIDHex = "native"
			}
		} else {
			currentTokenIDHex := "native"
			if len(transfer.TokenID) > 0 {
				currentTokenIDHex = hex.EncodeToString(transfer.TokenID)
			}
			if currentTokenIDHex != commonTokenIDHex {
				return nil, nil, fmt.Errorf("all transfers must use the same tokenID, found %s and %s", commonTokenIDHex, currentTokenIDHex)
			}
		}
	}

	// 2. 将地址转换为 Base58 格式
	fromAddressBase58, err := utils.AddressBytesToBase58(fromAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("convert address to Base58 failed: %w", err)
	}

	// 3. 查询 UTXO
	utxoParams := []interface{}{fromAddressBase58}
	utxoResult, err := client.Call(ctx, "wes_getUTXO", utxoParams)
	if err != nil {
		return nil, nil, fmt.Errorf("query UTXO failed: %w", err)
	}

	// 4. 解析 UTXO 列表
	utxoMap, ok := utxoResult.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid UTXO response format")
	}

	utxosArray, ok := utxoMap["utxos"].([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid UTXOs format")
	}

	// 5. 转换为 UTXO 结构并过滤匹配的 tokenID
	var matchingUTXOs []UTXO
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

		utxoTokenIDHex := "native"
		if utxo.TokenID != "" {
			utxoTokenIDHex = utxo.TokenID
		}
		if utxoTokenIDHex == commonTokenIDHex {
			matchingUTXOs = append(matchingUTXOs, utxo)
		}
	}

	if len(matchingUTXOs) == 0 {
		return nil, nil, fmt.Errorf("no matching UTXOs for tokenID: %s", commonTokenIDHex)
	}

	// 6. 构建交易草稿
	draft := map[string]interface{}{
		"sign_mode": "defer_sign",
		"inputs":    []map[string]interface{}{},
		"outputs":   []map[string]interface{}{},
		"metadata": map[string]interface{}{
			"caller_address": hex.EncodeToString(fromAddress),
		},
	}

	// 7. 计算所有转账的总需求
	totalOutputAmount := big.NewInt(0)
	for _, transfer := range transfers {
		totalOutputAmount.Add(totalOutputAmount, big.NewInt(int64(transfer.Amount)))
	}

	// 8. 选择足够的UTXO来满足所有转账需求
	var selectedUTXOList []UTXO
	totalInputAmount := big.NewInt(0)
	for _, utxo := range matchingUTXOs {
		if utxo.Amount == "" {
			continue
		}
		utxoAmount, ok := new(big.Int).SetString(utxo.Amount, 10)
		if !ok {
			continue
		}
		selectedUTXOList = append(selectedUTXOList, utxo)
		totalInputAmount.Add(totalInputAmount, utxoAmount)
		// 如果累计金额已经足够，就停止选择
		// 注意：手续费从接收者扣除，发送者不需要支付手续费，只需要满足总输出金额即可
		if totalInputAmount.Cmp(totalOutputAmount) >= 0 {
			break
		}
	}

	// 验证是否有足够的UTXO
	// 注意：手续费从接收者扣除，发送者只需要满足总输出金额即可
	if totalInputAmount.Cmp(totalOutputAmount) < 0 {
		return nil, nil, fmt.Errorf("insufficient balance: total required %s, available %s",
			totalOutputAmount.String(), totalInputAmount.String())
	}

	// 9. 为所有选中的UTXO添加输入
	var inputIndices []uint32
	for _, selectedUTXO := range selectedUTXOList {
		outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
		if len(outpointParts) != 2 {
			return nil, nil, fmt.Errorf("invalid outpoint format: %s", selectedUTXO.Outpoint)
		}
		txHash := outpointParts[0]
		var outputIndex uint32
		if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
			return nil, nil, fmt.Errorf("invalid output index: %w", err)
		}

		// 添加输入
		inputs := draft["inputs"].([]map[string]interface{})
		inputIndex := uint32(len(inputs))
		draft["inputs"] = append(inputs, map[string]interface{}{
			"tx_hash":           txHash,
			"output_index":      outputIndex,
			"is_reference_only": false,
		})
		inputIndices = append(inputIndices, inputIndex)
	}

	// 10. 为每个转账添加输出
	for _, transfer := range transfers {
		outputs := draft["outputs"].([]map[string]interface{})
		transferOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(transfer.To),
			"amount": fmt.Sprintf("%d", transfer.Amount),
		}
		if len(commonTokenID) > 0 {
			transferOutput["token_id"] = hex.EncodeToString(commonTokenID)
		}
		draft["outputs"] = append(outputs, transferOutput)
	}

	// 11. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = totalInputAmount - totalOutputAmount
	changeBig := new(big.Int).Sub(totalInputAmount, totalOutputAmount)

	// 12. 添加找零输出
	if changeBig.Sign() > 0 {
		outputs := draft["outputs"].([]map[string]interface{})
		changeOutput := map[string]interface{}{
			"type":   "asset",
			"owner":  hex.EncodeToString(fromAddress),
			"amount": changeBig.String(),
		}
		if len(commonTokenID) > 0 {
			changeOutput["token_id"] = hex.EncodeToString(commonTokenID)
		}
		draft["outputs"] = append(outputs, changeOutput)
	}

	// 13. 序列化交易草稿为 JSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	return draftJSON, inputIndices, nil
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

// buildTransferDraft 构建单笔转账交易的 DraftJSON（仅构建草稿，不直接构建交易）
//
// **用途**：
// - 由 SDK 在链外完成 UTXO 选择、金额计算等逻辑
// - 返回 DraftJSON 和主要消费输入索引（当前实现始终为 0）
// - 后续交由链侧通用交易 API（如 wes_buildTransaction / wes_computeSignatureHashFromDraft 等）完成交易构建和签名
func buildTransferDraft(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	toAddress []byte,
	amount uint64,
	tokenID []byte,
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

	// 4. 转换为 UTXO 结构
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
		// 尝试获取 tokenID（如果存在）
		if tokenIDStr := getString(utxoMap, "tokenID"); tokenIDStr != "" {
			utxo.TokenID = tokenIDStr
		}
		utxos = append(utxos, utxo)
	}

	if len(utxos) == 0 {
		return nil, 0, fmt.Errorf("no available UTXOs")
	}

	// 5. 过滤匹配 tokenID 的 UTXO
	var matchingUTXOs []UTXO
	tokenIDHex := ""
	if len(tokenID) > 0 {
		tokenIDHex = hex.EncodeToString(tokenID)
	}

	for _, utxo := range utxos {
		// 如果 tokenID 为空，匹配原生币（没有 tokenID 的 UTXO）
		if len(tokenID) == 0 {
			if utxo.TokenID == "" {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		} else {
			// 如果 tokenID 不为空，匹配相同 tokenID 的 UTXO
			if utxo.TokenID == tokenIDHex {
				matchingUTXOs = append(matchingUTXOs, utxo)
			}
		}
	}

	if len(matchingUTXOs) == 0 {
		if len(tokenID) == 0 {
			return nil, 0, fmt.Errorf("no matching UTXOs for native coin")
		}
		return nil, 0, fmt.Errorf("no matching UTXOs for tokenID: %s", tokenIDHex)
	}

	// 6. 选择足够的 UTXO（当前实现只选择一个即可满足需求）
	requiredAmount := big.NewInt(int64(amount))
	var selectedUTXO UTXO
	var selectedAmount *big.Int
	for _, utxo := range matchingUTXOs {
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

	// 7. 解析 outpoint
	outpointParts := strings.Split(selectedUTXO.Outpoint, ":")
	if len(outpointParts) != 2 {
		return nil, 0, fmt.Errorf("invalid outpoint format")
	}
	txHash := outpointParts[0]
	var outputIndex uint32
	if _, err := fmt.Sscanf(outpointParts[1], "%d", &outputIndex); err != nil {
		return nil, 0, fmt.Errorf("invalid output index: %w", err)
	}

	// 8. 计算找零
	// 注意：手续费从接收者扣除，发送者不需要支付手续费，找零 = selectedAmount - amount
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)

	// 9. 构建 DraftJSON（符合 host_build_transaction 的 DraftJSON 格式）
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

	// 11. 添加转账输出
	outputs := draft["outputs"].([]map[string]interface{})
	transferOutput := map[string]interface{}{
		"type":   "asset",
		"owner":  hex.EncodeToString(toAddress),
		"amount": fmt.Sprintf("%d", amount),
	}
	if len(tokenID) > 0 {
		transferOutput["token_id"] = hex.EncodeToString(tokenID)
	}
	draft["outputs"] = append(outputs, transferOutput)

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

	// 13. 序列化 DraftJSON
	draftJSON, err := json.Marshal(draft)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 当前实现只有 1 个消费型输入，其索引恒为 0
	return draftJSON, 0, nil
}
