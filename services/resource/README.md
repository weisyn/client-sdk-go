# Resource Service - 资源服务

**版本**: 1.0.0-alpha  
**状态**: ✅ 基础结构完成  
**最后更新**: 2025-01-23

---

## 📋 概述

Resource Service 提供资源管理相关的业务操作，包括部署静态资源、智能合约、AI 模型和查询资源信息等功能。所有操作都使用 Wallet 接口进行签名，符合 SDK 架构原则。

---

## 🔧 核心功能

### 1. DeployStaticResource - 部署静态资源 ✅

**功能**: 部署静态资源（如图片、文档等）

**使用示例**:
```go
resourceService := resource.NewService(client)

result, err := resourceService.DeployStaticResource(ctx, &resource.DeployStaticResourceRequest{
    From:     deployerAddr,
    Name:     "my-image.png",
    MimeType: "image/png",
    Data:     imageData,
}, wallet)
```

### 2. DeployContract - 部署智能合约 ✅

**功能**: 部署 WASM 智能合约

**使用示例**:
```go
result, err := resourceService.DeployContract(ctx, &resource.DeployContractRequest{
    From:     deployerAddr,
    Name:     "MyContract",
    WasmCode: wasmBytes,
    ABI:      abiJSON,
}, wallet)
```

### 3. DeployAIModel - 部署 AI 模型 ✅

**功能**: 部署 AI 模型资源

**使用示例**:
```go
result, err := resourceService.DeployAIModel(ctx, &resource.DeployAIModelRequest{
    From:     deployerAddr,
    Name:     "my-model",
    ModelType: "tensorflow",
    ModelData: modelBytes,
}, wallet)
```

### 4. GetResource - 查询资源信息 ✅

**功能**: 查询已部署资源的信息（不需要 Wallet）

**使用示例**:
```go
resourceInfo, err := resourceService.GetResource(ctx, contentHash)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("资源名称: %s\n", resourceInfo.Name)
fmt.Printf("资源类型: %s\n", resourceInfo.Category)
fmt.Printf("大小: %d 字节\n", resourceInfo.Size)
```

---

## 🏗️ 服务架构

```
┌─────────────────────────────────────────┐
│        Resource Service 架构             │
└─────────────────────────────────────────┘

Resource Service
    │
    ├─> DeployStaticResource: 部署静态资源
    ├─> DeployContract: 部署智能合约
    ├─> DeployAIModel: 部署 AI 模型
    └─> GetResource: 查询资源信息
```

---

## 📚 API 参考

### Service 接口

```go
type Service interface {
    DeployStaticResource(ctx context.Context, req *DeployStaticResourceRequest, wallets ...wallet.Wallet) (*DeployStaticResourceResult, error)
    DeployContract(ctx context.Context, req *DeployContractRequest, wallets ...wallet.Wallet) (*DeployContractResult, error)
    DeployAIModel(ctx context.Context, req *DeployAIModelRequest, wallets ...wallet.Wallet) (*DeployAIModelResult, error)
    GetResource(ctx context.Context, contentHash []byte) (*ResourceInfo, error)
}
```

---

## 🔗 相关文档

- [Services 总览](../README.md) - 业务服务层文档
- [主 README](../../README.md) - SDK 总体文档

---

**最后更新**: 2025-01-23  
**维护者**: WES Core Team

