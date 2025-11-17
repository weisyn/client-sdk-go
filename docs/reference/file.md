# 大文件处理参考

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

SDK 提供了大文件处理工具，支持分块处理、流式读取和进度监控，避免一次性加载大文件到内存。

---

## 🔗 关联文档

- **Resource 服务**：[Resource 指南](../guides/resource.md)
- **性能优化**：[性能优化指南](../archive/performance.md)

---

## 📦 导入

```go
import "github.com/weisyn/client-sdk-go/utils"
```

---

## 🔪 文件分块

### ChunkFile()

将文件内容分成多个块。

```go
func ChunkFile(data []byte, chunkSize int64) [][]byte
```

### 示例

```go
fileContent := make([]byte, 10*1024*1024) // 10MB
chunks := utils.ChunkFile(fileContent, 1024*1024) // 1MB 每块

fmt.Printf("分成 %d 块\n", len(chunks))
```

---

## ⚙️ 分块处理

### ProcessFileInChunks()

分块处理文件，支持并发和进度监控。

```go
func ProcessFileInChunks[T any](
    ctx context.Context,
    data []byte,
    processor func(chunk []byte, index int) (T, error),
    config *ChunkConfig,
) ([]T, error)
```

### 示例

```go
fileContent, err := os.ReadFile("large_file.bin")
if err != nil {
    log.Fatal(err)
}

results, err := utils.ProcessFileInChunks(ctx, fileContent, func(chunk []byte, index int) (string, error) {
    // 处理每个分块
    hash := sha256.Sum256(chunk)
    return hex.EncodeToString(hash[:]), nil
}, &utils.ChunkConfig{
    ChunkSize:   5 * 1024 * 1024, // 5MB
    Concurrency: 3,
    OnProgress: func(progress utils.FileProgress) {
        fmt.Printf("进度: %d%% (%d/%d 块)\n", 
            progress.Percentage, 
            progress.CurrentChunk, 
            progress.TotalChunks)
    },
})
```

---

## 📖 流式读取

### ReadFileAsStream()

流式读取文件，支持进度回调。

```go
func ReadFileAsStream(
    filePath string,
    onProgress func(FileProgress),
) ([]byte, error)
```

### 示例

```go
data, err := utils.ReadFileAsStream("large_file.bin", func(progress utils.FileProgress) {
    fmt.Printf("读取进度: %d%%\n", progress.Percentage)
})
if err != nil {
    log.Fatal(err)
}
```

---

## ⏱️ 时间估算

### EstimateProcessingTime()

估算文件处理时间。

```go
func EstimateProcessingTime(
    fileSize int64,
    chunkSize int64,
    processingSpeed int64,
) time.Duration
```

### 示例

```go
fileSize := int64(100 * 1024 * 1024) // 100MB
chunkSize := int64(5 * 1024 * 1024)  // 5MB
speed := int64(1024 * 1024)           // 1MB/s

estimatedTime := utils.EstimateProcessingTime(fileSize, chunkSize, speed)
fmt.Printf("估算处理时间: %v\n", estimatedTime)
```

---

## 🎯 使用建议

- ✅ 大文件（>10MB）建议使用分块处理
- ✅ 分块大小建议：1-5MB
- ✅ 并发数量建议：3-5 个
- ⚠️ 注意内存使用，避免一次性加载超大文件

---

## 🔗 相关文档

- **[Resource 指南](../guides/resource.md)** - 资源部署指南
- **[性能优化](../archive/performance.md)** - 性能优化建议

---

