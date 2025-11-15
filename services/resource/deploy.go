package resource

import (
	"bytes"
	"context"
	"fmt"

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

	// 4. TODO: 调用节点API部署静态资源
	// 当前节点可能没有提供专门的静态资源部署 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_deployStaticResource`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者使用 `wes_deployContract` API（如果支持静态资源）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("deploy static resource not implemented yet: requires node API support (wes_deployStaticResource) or use wes_deployContract")
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

	// 4. TODO: 调用节点 `wes_deployContract` API
	// 当前节点提供了 `wes_deployContract` JSON-RPC 方法
	// 需要：
	//   a) 确认参数格式（参考 `internal/api/jsonrpc/methods/tx.go`）
	//   b) 可能需要先上传 WASM 文件到节点（或直接传递文件内容）
	//   c) 调用 `wes_deployContract` 获取未签名交易
	//   d) 使用 Wallet 签名未签名交易
	//   e) 调用 `wes_sendRawTransaction` 提交

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("deploy contract not implemented yet: requires integration with wes_deployContract API")
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

	// 4. TODO: 调用节点API部署AI模型
	// 当前节点可能没有提供专门的 AI 模型部署 JSON-RPC 方法
	// 需要：
	//   a) 节点提供业务服务API（如 `wes_deployAIModel`）- 推荐方案
	//   b) 使用 Wallet 签名未签名交易
	//   c) 调用 wes_sendRawTransaction 提交
	//   d) 或者使用 `wes_deployContract` API（如果支持 ONNX 模型）

	// 临时返回错误，提示需要实现
	return nil, fmt.Errorf("deploy AI model not implemented yet: requires node API support (wes_deployAIModel) or use wes_deployContract")
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

