# 重试机制参考

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

SDK 内置了请求重试机制，可以在网络不稳定或节点暂时性故障时自动重试，提高应用程序的健壮性。

---

## 🔗 关联文档

- **Client API**：[Client API 参考](../api/client.md)
- **故障排查**：[故障排查指南](../troubleshooting.md)

---

## ⚙️ 配置重试

### 基本配置

```go
import (
    "time"
    "github.com/weisyn/client-sdk-go/client"
)

cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: &client.RetryConfig{
        MaxRetries:        3,              // 最大重试 3 次
        InitialDelay:      500,            // 首次重试延迟 500ms
        MaxDelay:          10000,          // 最大延迟 10 秒
        BackoffMultiplier: 2.0,            // 退避乘数（每次延迟翻倍）
    },
}
cli, err := client.NewClient(cfg)
```

### 自定义重试条件

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: &client.RetryConfig{
        MaxRetries:        5,
        InitialDelay:      500,
        MaxDelay:          10000,
        BackoffMultiplier: 2.0,
        Retryable: func(err error) bool {
            // 只重试网络错误或 5xx 错误
            if netErr, ok := err.(net.Error); ok {
                return netErr.Timeout() || netErr.Temporary()
            }
            return false
        },
    },
}
```

### 重试回调

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: &client.RetryConfig{
        MaxRetries:        3,
        InitialDelay:      500,
        OnRetry: func(attempt int, err error) {
            log.Printf("重试第 %d 次: %v", attempt, err)
            // 可以在这里记录日志、发送监控事件等
        },
    },
}
```

---

## 📊 重试策略

### 指数退避

SDK 使用指数退避策略，延迟时间按以下公式计算：

```
delay = InitialDelay * (BackoffMultiplier ^ attempt)
```

**示例**：
- 第 1 次重试：500ms
- 第 2 次重试：1000ms
- 第 3 次重试：2000ms
- 第 4 次重试：4000ms（不超过 MaxDelay）

### 可重试的错误

默认情况下，以下错误会自动重试：
- 网络错误（连接失败、超时等）
- DNS 错误
- HTTP 5xx 错误（服务器错误）
- HTTP 429 错误（请求过多）

---

## 🎯 使用示例

### 示例 1：基本重试

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: client.DefaultRetryConfig(),
}
cli, err := client.NewClient(cfg)
```

### 示例 2：自定义重试策略

```go
cfg := &client.Config{
    Endpoint: "http://localhost:8545",
    Protocol: client.ProtocolHTTP,
    Retry: &client.RetryConfig{
        MaxRetries:        5,
        InitialDelay:      1000,
        MaxDelay:          30000,
        BackoffMultiplier: 1.5,
        Retryable: func(err error) bool {
            // 自定义重试逻辑
            return strings.Contains(err.Error(), "timeout")
        },
        OnRetry: func(attempt int, err error) {
            log.Printf("重试第 %d 次: %v", attempt, err)
        },
    },
}
```

---

## ⚠️ 注意事项

- ⚠️ 重试会增加请求延迟，请根据业务需求调整重试次数
- ✅ 重试机制会自动处理临时性网络故障
- ✅ 可以通过 `Retryable` 函数自定义重试条件

---

## 🔗 相关文档

- **[Client API](../api/client.md)** - 完整 API 文档
- **[故障排查](../troubleshooting.md)** - 常见问题

---

