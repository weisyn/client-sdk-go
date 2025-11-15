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
		return nil, fmt.Errorf("insufficient balance")
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

	// 8. 计算手续费（万分之三，按金额内扣）
	feeBig := new(big.Int).Mul(requiredAmount, big.NewInt(3))
	feeBig.Div(feeBig, big.NewInt(10000))

	// 9. 计算找零
	changeBig := new(big.Int).Sub(selectedAmount, requiredAmount)
	changeBig.Sub(changeBig, feeBig)

	// 10. 构建交易草稿（符合 host_build_transaction 的 DraftJSON 格式）
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
func buildBurnTransaction(
	ctx context.Context,
	client client.Client,
	fromAddress []byte,
	amount uint64,
	tokenID []byte,
) ([]byte, error) {
	// 1. 将地址转换为 Base58 格式（wes_getUTXO 需要 Base58 地址）
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
		return nil, fmt.Errorf("insufficient balance")
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

	// 8. 计算手续费（万分之三，按金额内扣）
	feeBig := new(big.Int).Mul(requiredAmount, big.NewInt(3))
	feeBig.Div(feeBig, big.NewInt(10000))

	// 9. 计算找零
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
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 13. 调用 wes_buildTransaction API
	buildTxParams := map[string]interface{}{
		"draft": json.RawMessage(draftJSON),
	}
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_buildTransaction failed: %w", err)
	}

	// 14. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_buildTransaction")
	}

	unsignedTxHex, ok := resultMap["unsignedTx"].(string)
	if !ok || unsignedTxHex == "" {
		return nil, fmt.Errorf("missing unsignedTx in wes_buildTransaction response")
	}

	// 15. 解码未签名交易
	unsignedTxBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedTxHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode unsignedTx failed: %w", err)
	}

	return unsignedTxBytes, nil
}

// buildBatchTransferTransaction 构建批量转账交易（SDK 层实现）
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

	// 8. 计算手续费（万分之三，按总输出金额计算）
	feeBig := new(big.Int).Mul(totalOutputAmount, big.NewInt(3))
	feeBig.Div(feeBig, big.NewInt(10000))

	// 9. 计算找零
	changeBig := new(big.Int).Sub(totalInputAmount, totalOutputAmount)
	changeBig.Sub(changeBig, feeBig)

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
	result, err := client.Call(ctx, "wes_buildTransaction", []interface{}{buildTxParams})
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

// getString 从 map 中获取字符串值
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
