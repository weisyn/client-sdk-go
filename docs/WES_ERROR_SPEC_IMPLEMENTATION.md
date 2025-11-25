# Client SDK Go - WES Error Spec 实施

**版本**: v1.0.0  


---

## 📋 文档定位

> 📌 **重要说明**：本文档说明 **Go SDK 如何对接错误规范**。  
> 如需了解 WES Error Specification，请参考主仓库文档。

**本文档目标**：
- 说明错误码映射
- 说明错误处理模式
- 提供错误处理最佳实践

---

## 🔧 错误码映射

### 错误类型

```go
// 网络错误
ErrNetworkError

// 参数错误
ErrInvalidParams

// 交易错误
ErrTransactionFailed

// 其他错误
ErrUnknown
```

---

## 📖 错误处理模式

### 基本错误处理

```go
result, err := tokenService.Transfer(ctx, req, wallet)
if err != nil {
    // 检查错误类型
    if errors.Is(err, client.ErrNetworkError) {
        // 网络错误处理
    } else if errors.Is(err, client.ErrInvalidParams) {
        // 参数错误处理
    } else {
        // 其他错误处理
    }
}
```

---

## 🔗 相关文档

- [开发者指南](./DEVELOPER_GUIDE.md) - 开发基础
- [WES Error Specification](../../../weisyn.git/docs/error-spec/) - 错误规范（主仓库）

---

  
**维护者**: WES Core Team

