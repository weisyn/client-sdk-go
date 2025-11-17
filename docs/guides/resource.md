# Resource 服务指南

---

## 📌 版本信息

- **版本**：0.1.0-alpha
- **状态**：draft
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：SDK 团队
- **适用范围**：Go 客户端 SDK

---

## 📖 概述

Resource Service 提供资源部署和查询功能，支持智能合约、AI 模型和静态资源的部署。

---

## 🔗 关联文档

- **API 参考**：[Services API - Resource](../api/services.md#-resource-service)
- **WES 协议**：[WES 资源模型](https://github.com/weisyn/weisyn/blob/main/docs/system/components/resource/README.md)（待确认）

---

## 🚀 快速开始

### 创建服务

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/resource"
    "github.com/weisyn/client-sdk-go/wallet"
)

cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
}
cli, err := client.NewClient(cfg)
if err != nil {
    log.Fatal(err)
}

w, err := wallet.NewWallet()
if err != nil {
    log.Fatal(err)
}

resourceService := resource.NewService(cli)
```

---

## 📦 部署智能合约

### 基本部署

```go
ctx := context.Background()

// 从文件读取 WASM 字节码
wasmBytes, err := os.ReadFile("contract.wasm")
if err != nil {
    log.Fatal(err)
}

result, err := resourceService.DeployContract(ctx, &resource.DeployContractRequest{
    WASMBytes:   wasmBytes,
    Name:        "MyContract",
    Description: "A simple smart contract",
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("合约部署成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("合约 ID: %s\n", result.ContractID)
```

---

## 🤖 部署 AI 模型

### 部署 ONNX 模型

```go
// 从文件读取 ONNX 模型
modelBytes, err := os.ReadFile("model.onnx")
if err != nil {
    log.Fatal(err)
}

result, err := resourceService.DeployAIModel(ctx, &resource.DeployAIModelRequest{
    ModelBytes: modelBytes,
    Name:       "ImageClassifier",
    Framework:  "ONNX",
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("AI 模型部署成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("模型 ID: %s\n", result.ModelID)
```

---

## 📄 部署静态资源

### 部署文件

```go
// 读取文件内容
fileContent, err := os.ReadFile("image.png")
if err != nil {
    log.Fatal(err)
}

result, err := resourceService.DeployStaticResource(ctx, &resource.DeployStaticResourceRequest{
    FileContent: fileContent,
    MimeType:   "image/png",
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("静态资源部署成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("资源 ID: %s\n", result.ResourceID)
```

---

## 🔍 查询资源

### 查询资源信息

```go
// 注意：查询资源不需要 Wallet
resourceInfo, err := resourceService.GetResource(ctx, resourceID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("资源类型: %s\n", resourceInfo.Type)
fmt.Printf("资源大小: %d 字节\n", resourceInfo.Size)
fmt.Printf("MIME 类型: %s\n", resourceInfo.MimeType)
```

---

## 🎯 典型场景

### 场景 1：部署并调用合约

```go
func deployAndCallContract(
    ctx context.Context,
    deployerWallet wallet.Wallet,
    wasmBytes []byte,
    resourceService resource.Service,
) error {
    // 1. 部署合约
    deployResult, err := resourceService.DeployContract(ctx, &resource.DeployContractRequest{
        WASMBytes: wasmBytes,
        Name:      "MyContract",
    }, deployerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("合约 ID: %s\n", deployResult.ContractID)
    
    // 2. 调用合约（通过 TokenService 或其他服务）
    // 例如：调用合约的 mint 方法
    // ...
    
    return nil
}
```

### 场景 2：部署大文件资源

```go
func deployLargeFile(
    ctx context.Context,
    filePath string,
    mimeType string,
    wallet wallet.Wallet,
    resourceService resource.Service,
) (string, error) {
    // 读取文件
    fileContent, err := os.ReadFile(filePath)
    if err != nil {
        return "", err
    }
    
    // 如果文件很大，可以显示进度
    if len(fileContent) > 10*1024*1024 {
        fmt.Printf("文件大小: %d 字节\n", len(fileContent))
        // 可以使用 utils/file 工具进行分块处理
    }
    
    // 部署资源
    result, err := resourceService.DeployStaticResource(ctx, &resource.DeployStaticResourceRequest{
        FileContent: fileContent,
        MimeType:    mimeType,
    }, wallet)
    if err != nil {
        return "", err
    }
    
    return result.ResourceID, nil
}
```

---

## ⚠️ 常见错误

### 文件太大

```go
largeFile := make([]byte, 200*1024*1024) // 200MB
result, err := resourceService.DeployStaticResource(ctx, &resource.DeployStaticResourceRequest{
    FileContent: largeFile,
    MimeType:    "application/octet-stream",
}, w)
if err != nil {
    if strings.Contains(err.Error(), "file too large") {
        log.Fatal("文件太大，请使用分块上传")
    }
    log.Fatal(err)
}
```

### 资源不存在

```go
invalidResourceID := make([]byte, 32)
resourceInfo, err := resourceService.GetResource(ctx, invalidResourceID)
if err != nil {
    if strings.Contains(err.Error(), "resource not found") {
        log.Fatal("资源不存在")
    }
    log.Fatal(err)
}
```

---

## 🔗 相关文档

- **[API 参考](../api/services.md#-resource-service)** - 完整 API 文档
- **[大文件处理](../reference/file.md)** - 大文件处理指南
- **[故障排查](../troubleshooting.md)** - 常见问题

---

