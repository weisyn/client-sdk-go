package resource

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/weisyn/client-sdk-go/wallet"
)

// deployStaticResource 部署静态资源实现
//
// ⚠️ **当前实现说明**：
// 当前节点提供了 `wes_deployContract` API，但可能没有专门的静态资源部署方法。
// 
// **理想流程**（待实现）：
// 1. 调用节点API部署静态资源（创建 ResourceOutput）
//    - 可以使用 `wes_deployContract` API（如果支持静态资源）
//    - 或需要节点提供 `wes_deployStaticResource` JSON-RPC 方法
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/resource/service.go` - 业务逻辑实现
// - `internal/api/jsonrpc/methods/tx.go` - `wes_deployContract` 实现（参考参数格式）
//
// **当前限制**：
// - 节点可能没有提供专门的静态资源部署 API
// - 需要确认 `wes_deployContract` 是否支持静态资源
func (s *resourceService) deployStaticResource(ctx context.Context, req *DeployStaticResourceRequest, wallets ...wallet.Wallet) (*DeployStaticResourceResult, error) {
	// 1. 参数验证
	if err := s.validateDeployStaticResourceRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.From) {
		return nil, fmt.Errorf("wallet address does not match from address")
	}

	// 4. 读取静态资源文件
	fileBytes, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// 5. Base64 编码文件内容
	fileContentBase64 := base64.StdEncoding.EncodeToString(fileBytes)

	// 6. 获取私钥（用于 API 调用）
	privateKey := w.PrivateKey()
	if privateKey == nil {
		return nil, fmt.Errorf("wallet private key not available")
	}
	// 从 ECDSA 私钥中提取 D 值（32字节）
	privateKeyBytes := privateKey.D.Bytes()
	// 确保是32字节（可能需要填充）
	if len(privateKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privateKeyBytes):], privateKeyBytes)
		privateKeyBytes = padded
	}
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// 7. 调用 `wes_deployContract` API（静态资源可以作为特殊类型的合约）
	// 注意：当前实现使用 wes_deployContract，如果未来有专门的 wes_deployStaticResource API，可以切换
	deployParams := map[string]interface{}{
		"private_key":  privateKeyHex,
		"wasm_content": fileContentBase64, // 使用文件内容作为 wasm_content
		"abi_version":  "v1",
		"name":         req.FilePath, // 使用文件路径作为名称
		"description":  fmt.Sprintf("Static resource: %s", req.MimeType),
	}

	result, err := s.client.Call(ctx, "wes_deployContract", []interface{}{deployParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_deployContract failed: %w", err)
	}

	// 8. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_deployContract")
	}

	success, _ := resultMap["success"].(bool)
	if !success {
		message, _ := resultMap["message"].(string)
		return nil, fmt.Errorf("deploy static resource failed: %s", message)
	}

	contentHashStr, _ := resultMap["content_hash"].(string)
	txHash, _ := resultMap["tx_hash"].(string)

	// 9. 解析 contentHash
	contentHash, err := hex.DecodeString(contentHashStr)
	if err != nil {
		return nil, fmt.Errorf("decode content hash failed: %w", err)
	}

	// 10. 返回结果
	return &DeployStaticResourceResult{
		ContentHash: contentHash,
		TxHash:      txHash,
		Success:     true,
	}, nil
}

// validateDeployStaticResourceRequest 验证部署静态资源请求
func (s *resourceService) validateDeployStaticResourceRequest(req *DeployStaticResourceRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证文件路径
	if req.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	// 3. 验证MIME类型
	if req.MimeType == "" {
		return fmt.Errorf("MIME type is required")
	}

	return nil
}

// deployContract 部署合约实现
//
// ⚠️ **当前实现说明**：
// 当前节点提供了 `wes_deployContract` JSON-RPC 方法，但需要确认参数格式。
// 
// **理想流程**（待实现）：
// 1. 调用节点 `wes_deployContract` API 构建部署合约交易（创建 ResourceOutput）
//    - 参考 `internal/api/jsonrpc/methods/tx.go` 中的 `DeployContract` 实现
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/resource/service.go` - 业务逻辑实现
// - `internal/api/jsonrpc/methods/tx.go` - `wes_deployContract` 实现（参考参数格式）
//
// **当前限制**：
// - 需要确认 `wes_deployContract` 的参数格式是否匹配
// - 需要确认是否需要先上传 WASM 文件到节点
func (s *resourceService) deployContract(ctx context.Context, req *DeployContractRequest, wallets ...wallet.Wallet) (*DeployContractResult, error) {
	// 1. 参数验证
	if err := s.validateDeployContractRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.From) {
		return nil, fmt.Errorf("wallet address does not match from address")
	}

	// 4. 读取 WASM 文件
	wasmBytes, err := os.ReadFile(req.WasmPath)
	if err != nil {
		return nil, fmt.Errorf("read WASM file failed: %w", err)
	}

	// 5. Base64 编码 WASM 内容
	wasmContentBase64 := base64.StdEncoding.EncodeToString(wasmBytes)

	// 6. 获取私钥（用于 API 调用）
	privateKey := w.PrivateKey()
	if privateKey == nil {
		return nil, fmt.Errorf("wallet private key not available")
	}
	// 从 ECDSA 私钥中提取 D 值（32字节）
	privateKeyBytes := privateKey.D.Bytes()
	// 确保是32字节（可能需要填充）
	if len(privateKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privateKeyBytes):], privateKeyBytes)
		privateKeyBytes = padded
	}
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// 7. 调用 `wes_deployContract` API
	// 注意：当前 API 需要 private_key，如果未来支持 return_unsigned_tx，可以改为使用 Wallet 签名
	deployParams := map[string]interface{}{
		"private_key":  privateKeyHex,
		"wasm_content": wasmContentBase64,
		"abi_version":  "v1", // 默认 ABI 版本
		"name":         req.ContractName,
		"description":  "", // 可选
	}

	// 如果有初始化参数，添加到 payload 中
	if len(req.InitArgs) > 0 {
		// InitArgs 是字节数组，需要 Base64 编码
		initArgsBase64 := base64.StdEncoding.EncodeToString(req.InitArgs)
		deployParams["init_args"] = initArgsBase64
	}

	result, err := s.client.Call(ctx, "wes_deployContract", []interface{}{deployParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_deployContract failed: %w", err)
	}

	// 8. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_deployContract")
	}

	success, _ := resultMap["success"].(bool)
	if !success {
		message, _ := resultMap["message"].(string)
		return nil, fmt.Errorf("deploy contract failed: %s", message)
	}

	contentHashStr, _ := resultMap["content_hash"].(string)
	txHash, _ := resultMap["tx_hash"].(string)

	// 9. 解析 contentHash
	contentHash, err := hex.DecodeString(contentHashStr)
	if err != nil {
		return nil, fmt.Errorf("decode content hash failed: %w", err)
	}

	// 10. 返回结果
	// 注意：合约地址通常是 contentHash（32字节）
	return &DeployContractResult{
		ContractAddress: contentHash, // 使用 contentHash 作为合约地址
		ContentHash:     contentHash,
		TxHash:          txHash,
		Success:         true,
	}, nil
}

// validateDeployContractRequest 验证部署合约请求
func (s *resourceService) validateDeployContractRequest(req *DeployContractRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证WASM路径
	if req.WasmPath == "" {
		return fmt.Errorf("WASM path is required")
	}

	// 3. 验证合约名称
	if req.ContractName == "" {
		return fmt.Errorf("contract name is required")
	}

	return nil
}

// deployAIModel 部署AI模型实现
//
// ⚠️ **当前实现说明**：
// 当前节点可能没有提供专门的 AI 模型部署 JSON-RPC 方法（如 `wes_deployAIModel`）。
// 
// **理想流程**（待实现）：
// 1. 调用节点API部署AI模型（创建 ResourceOutput，类型为 ONNX）
//    - 可以使用 `wes_deployContract` API（如果支持 ONNX 模型）
//    - 或需要节点提供 `wes_deployAIModel` JSON-RPC 方法
// 2. 使用钱包签名交易
// 3. 调用 `wes_sendRawTransaction` 提交已签名交易
//
// **参考实现**：
// - `contract-sdk-go/helpers/resource/service.go` - 业务逻辑实现
//
// **当前限制**：
// - 节点可能没有提供专门的 AI 模型部署 API
// - 需要确认 `wes_deployContract` 是否支持 ONNX 模型
func (s *resourceService) deployAIModel(ctx context.Context, req *DeployAIModelRequest, wallets ...wallet.Wallet) (*DeployAIModelResult, error) {
	// 1. 参数验证
	if err := s.validateDeployAIModelRequest(req); err != nil {
		return nil, err
	}

	// 2. 获取 Wallet
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	// 3. 验证地址匹配
	if !bytes.Equal(w.Address(), req.From) {
		return nil, fmt.Errorf("wallet address does not match from address")
	}

	// 4. 读取 ONNX 模型文件
	onnxBytes, err := os.ReadFile(req.ModelPath)
	if err != nil {
		return nil, fmt.Errorf("read ONNX model file failed: %w", err)
	}

	// 5. Base64 编码 ONNX 内容
	onnxContentBase64 := base64.StdEncoding.EncodeToString(onnxBytes)

	// 6. 获取私钥（用于 API 调用）
	privateKey := w.PrivateKey()
	if privateKey == nil {
		return nil, fmt.Errorf("wallet private key not available")
	}
	// 从 ECDSA 私钥中提取 D 值（32字节）
	privateKeyBytes := privateKey.D.Bytes()
	// 确保是32字节（可能需要填充）
	if len(privateKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privateKeyBytes):], privateKeyBytes)
		privateKeyBytes = padded
	}
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// 7. 调用 `wes_deployAIModel` API
	deployParams := map[string]interface{}{
		"private_key":  privateKeyHex,
		"onnx_content": onnxContentBase64,
		"name":         req.ModelName,
		"description":  "", // 可选
	}

	result, err := s.client.Call(ctx, "wes_deployAIModel", []interface{}{deployParams})
	if err != nil {
		return nil, fmt.Errorf("call wes_deployAIModel failed: %w", err)
	}

	// 8. 解析结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_deployAIModel")
	}

	success, _ := resultMap["success"].(bool)
	if !success {
		message, _ := resultMap["message"].(string)
		return nil, fmt.Errorf("deploy AI model failed: %s", message)
	}

	contentHashStr, _ := resultMap["content_hash"].(string)
	txHash, _ := resultMap["tx_hash"].(string)

	// 9. 解析 contentHash
	contentHash, err := hex.DecodeString(contentHashStr)
	if err != nil {
		return nil, fmt.Errorf("decode content hash failed: %w", err)
	}

	// 10. 返回结果
	return &DeployAIModelResult{
		ContentHash: contentHash,
		TxHash:      txHash,
		Success:     true,
	}, nil
}

// validateDeployAIModelRequest 验证部署AI模型请求
func (s *resourceService) validateDeployAIModelRequest(req *DeployAIModelRequest) error {
	// 1. 验证地址
	if len(req.From) != 20 {
		return fmt.Errorf("from address must be 20 bytes")
	}

	// 2. 验证模型路径
	if req.ModelPath == "" {
		return fmt.Errorf("model path is required")
	}

	// 3. 验证模型名称
	if req.ModelName == "" {
		return fmt.Errorf("model name is required")
	}

	return nil
}

