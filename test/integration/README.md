# 集成测试

集成测试需要真实的 WES 节点运行。

## 🚀 快速开始

```bash
# 1. 启动 WES 节点
cd /path/to/weisyn.git
bash scripts/testing/common/test_init.sh

# 2. 运行测试
cd /path/to/client-sdk-go.git
go test ./test/integration/... -v
```

## 📁 目录结构

```
test/integration/
├── README.md              # 本文档
├── setup.go               # 测试环境设置
├── helpers.go             # 测试辅助函数
└── services/              # 各服务测试
    ├── token/
    ├── staking/
    ├── market/
    ├── governance/
    └── resource/
```

## 📚 完整文档

👉 **测试规划与详细说明请见：[`docs/testing/plan.md`](../../docs/testing/plan.md)**

---

**最后更新**: 2025-11-17
