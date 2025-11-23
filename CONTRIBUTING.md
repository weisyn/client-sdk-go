# 贡献指南

感谢您对 WES Client SDK for Go 的关注！

## 如何贡献

### 报告 Bug

请使用 [Issue 模板](.github/ISSUE_TEMPLATE/bug_report.md) 报告 bug。

### 提出新功能

请使用 [Feature Request 模板](.github/ISSUE_TEMPLATE/feature_request.md) 提出新功能建议。

### 提交代码

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 代码规范

- 遵循 Go 官方代码规范
- 运行 `go fmt` 格式化代码
- 运行 `go vet` 检查代码
- 添加必要的单元测试和集成测试

## 测试要求

- 新功能需要添加相应的测试用例
- 所有测试必须通过
- 集成测试需要 WES 节点运行

详见：[测试文档](test/integration/README.md)

## 提交信息规范

提交信息应清晰描述变更内容：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**：
```
feat(token): add batch transfer support

Add BatchTransfer method to token service for efficient multi-recipient transfers.

Closes #123
```

---

**最后更新**: 2025-11-17

