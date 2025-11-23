package utils

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/weisyn/client-sdk-go/client"
)

// ParsedTx 解析后的交易信息
type ParsedTx struct {
	Hash        string
	Status      string // "pending" | "confirmed"
	BlockHeight uint64
	BlockHash   string
	TxIndex     uint32
	Inputs      []ParsedInput
	Outputs     []ParsedOutput
}

// ParsedInput 解析后的输入
type ParsedInput struct {
	TxHash      string
	OutputIndex uint32
	IsReference bool
}

// ParsedOutput 解析后的输出
type ParsedOutput struct {
	Index     uint32
	Type      string // "asset" | "state" | "resource" | "contract"
	Owner     []byte // 20字节地址
	Amount    *big.Int
	TokenID   []byte // 代币ID（nil表示原生币）
	StateID   []byte // StateOutput 的 state_id
	StateData []byte // StateOutput 的 execution_result_hash 或 metadata
	Outpoint  string // 格式: "txHash:index"
}

// parseOwnerAddress 解析 owner 地址，仅支持 Base64 格式（WES 标准格式）
func parseOwnerAddress(ownerStr string) []byte {
	if ownerStr == "" {
		return nil
	}
	// 仅支持 Base64 解码（WES API 返回的标准格式）
	if ownerBytes, err := base64.StdEncoding.DecodeString(ownerStr); err == nil && len(ownerBytes) == 20 {
		return ownerBytes
	}
	return nil
}

// FetchAndParseTx 获取并解析交易
//
// **功能**：
// 从节点获取交易详情，解析出所有输入输出信息，便于提取业务数据。
//
// **流程**：
// 1. 调用 `wes_getTransactionByHash` 获取交易详情
// 2. 解析交易结构，提取 inputs 和 outputs
// 3. 计算每个输出的 outpoint（txHash:index）
func FetchAndParseTx(ctx context.Context, client client.Client, txHash string) (*ParsedTx, error) {
	// 1. 使用交易哈希（WES 使用标准十六进制格式，不带 0x 前缀）
	txHashClean := txHash

	// 2. 调用 wes_getTransactionByHash
	result, err := client.Call(ctx, "wes_getTransactionByHash", []interface{}{txHashClean})
	if err != nil {
		return nil, fmt.Errorf("call wes_getTransactionByHash failed: %w", err)
	}

	// 3. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_getTransactionByHash")
	}

	// 4. 提取基本信息
	hash, _ := resultMap["hash"].(string)
	status, _ := resultMap["status"].(string)
	if status == "" {
		status = "confirmed" // 默认已确认
	}

	blockHeight := uint64(0)
	if bh, ok := resultMap["blockHeight"].(string); ok && bh != "" {
		// 解析为十六进制（支持 0x 前缀）
		bhClean := bh
		if len(bhClean) > 2 && bhClean[:2] == "0x" {
			bhClean = bhClean[2:]
		}
		if parsed, err := strconv.ParseUint(bhClean, 16, 64); err == nil {
			blockHeight = parsed
		}
	}

	blockHash, _ := resultMap["blockHash"].(string)
	txIndex := uint32(0)
	if ti, ok := resultMap["transactionIndex"].(string); ok && ti != "" {
		// 解析为十六进制（WES 标准格式，不带 0x 前缀）
		if parsed, err := strconv.ParseUint(ti, 16, 32); err == nil {
			txIndex = uint32(parsed)
		}
	}

	// 5. 解析 inputs
	var inputs []ParsedInput
	if inputsArray, ok := resultMap["inputs"].([]interface{}); ok {
		for _, inputItem := range inputsArray {
			inputMap, ok := inputItem.(map[string]interface{})
			if !ok {
				continue
			}

			prevOut, ok := inputMap["previous_output"].(map[string]interface{})
			if !ok {
				continue
			}

			txID, _ := prevOut["tx_id"].(string)
			outputIndex := uint32(0)
			if oi, ok := prevOut["output_index"].(float64); ok {
				outputIndex = uint32(oi)
			}

			isRef := false
			if ir, ok := inputMap["is_reference_only"].(bool); ok {
				isRef = ir
			}

			inputs = append(inputs, ParsedInput{
				TxHash:      txID,
				OutputIndex: outputIndex,
				IsReference: isRef,
			})
		}
	}

	// 6. 解析 outputs
	var outputs []ParsedOutput
	if outputsArray, ok := resultMap["outputs"].([]interface{}); ok {
		for idx, outputItem := range outputsArray {
			outputMap, ok := outputItem.(map[string]interface{})
			if !ok {
				continue
			}

			output := ParsedOutput{
				Index: uint32(idx),
			}

			// 先从顶层读取 owner（WES API 返回格式）
			ownerStr, _ := outputMap["owner"].(string)
			output.Owner = parseOwnerAddress(ownerStr)

			// 解析输出类型
			if asset, ok := outputMap["asset"].(map[string]interface{}); ok {
				output.Type = "asset"

				// 解析 native_coin.amount
				if nativeCoin, ok := asset["native_coin"].(map[string]interface{}); ok {
					amountStr, _ := nativeCoin["amount"].(string)
					if amountStr != "" {
						amount, ok := new(big.Int).SetString(amountStr, 10)
						if ok {
							output.Amount = amount
						}
					}
				}

				// 检查是否是合约代币
				if contractToken, ok := asset["contract_token"].(map[string]interface{}); ok {
					tokenIDStr, _ := contractToken["fungible_class_id"].(string)
					// WES 标准格式，不带 0x 前缀
					if tokenIDBytes, err := hex.DecodeString(tokenIDStr); err == nil {
						output.TokenID = tokenIDBytes
					}
				}
			} else if state, ok := outputMap["state"].(map[string]interface{}); ok {
				output.Type = "state"

				stateIDStr, _ := state["state_id"].(string)
				// WES 标准格式，不带 0x 前缀
				if stateIDBytes, err := hex.DecodeString(stateIDStr); err == nil {
					output.StateID = stateIDBytes
				}

				execResultHashStr, _ := state["execution_result_hash"].(string)
				// WES 标准格式，不带 0x 前缀
				if execResultHashBytes, err := hex.DecodeString(execResultHashStr); err == nil {
					output.StateData = execResultHashBytes
				}
			} else if _, ok := outputMap["resource"].(map[string]interface{}); ok {
				output.Type = "resource"
			} else if _, ok := outputMap["contract"].(map[string]interface{}); ok {
				output.Type = "contract"
			}

			// 计算 outpoint
			output.Outpoint = fmt.Sprintf("%s:%d", txHashClean, idx)

			outputs = append(outputs, output)
		}
	}

	return &ParsedTx{
		Hash:        hash,
		Status:      status,
		BlockHeight: blockHeight,
		BlockHash:   blockHash,
		TxIndex:     txIndex,
		Inputs:      inputs,
		Outputs:     outputs,
	}, nil
}

// FindOutputsByOwner 查找指定地址拥有的输出
func FindOutputsByOwner(outputs []ParsedOutput, owner []byte) []ParsedOutput {
	var result []ParsedOutput
	for _, output := range outputs {
		if len(output.Owner) == 20 && len(owner) == 20 {
			if string(output.Owner) == string(owner) {
				result = append(result, output)
			}
		}
	}
	return result
}

// FindOutputsByType 查找指定类型的输出
func FindOutputsByType(outputs []ParsedOutput, outputType string) []ParsedOutput {
	var result []ParsedOutput
	for _, output := range outputs {
		if output.Type == outputType {
			result = append(result, output)
		}
	}
	return result
}

// SumAmountsByToken 按代币类型汇总金额
func SumAmountsByToken(outputs []ParsedOutput, tokenID []byte) *big.Int {
	total := big.NewInt(0)
	for _, output := range outputs {
		if output.Amount != nil {
			// 如果 tokenID 为 nil，只统计原生币（TokenID 也为 nil）
			// 如果 tokenID 不为 nil，只统计匹配的代币
			if (tokenID == nil && output.TokenID == nil) ||
				(tokenID != nil && len(output.TokenID) > 0 && string(output.TokenID) == string(tokenID)) {
				total.Add(total, output.Amount)
			}
		}
	}
	return total
}

// FindStateOutputs 查找 StateOutput
func FindStateOutputs(outputs []ParsedOutput) []ParsedOutput {
	return FindOutputsByType(outputs, "state")
}

// GetOutpoint 生成 outpoint 字符串
func GetOutpoint(txHash string, index uint32) string {
	// WES 标准格式，不带 0x 前缀
	return fmt.Sprintf("%s:%d", txHash, index)
}
