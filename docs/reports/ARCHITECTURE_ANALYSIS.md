# Market 服务 AMM 功能架构分析

---

## 📌 版本信息

- **版本**：1.0
- **状态**：stable
- **最后更新**：2025-11-17
- **最后审核**：2025-11-17
- **所有者**：WES SDK 团队
- **适用范围**：Market 服务 AMM 功能架构分析

---

## 🔍 WES 架构分析

### 1. WES 对 AMM 的支持 ✅

**结论**：WES 完全支持 AMM 功能，通过合约调用实现。

**证据**：
1. ✅ WES 有 AMM 合约示例代码（`docs/specs/ispc/examples/wasm-contracts/amm-swap.go`）
2. ✅ WES 支持合约调用（`wes_callContract` API）
3. ✅ WES 支持合约部署（`wes_deployContract` API）
4. ✅ WES 支持合约查询（`wes_getContract` API）
5. ✅ WES 支持 StateOutput（可用于存储 AMM 池状态）

### 2. AMM 合约地址确定方式

**WES 架构**：
- 合约通过 `contentHash`（32字节）唯一标识
- 合约地址是 `hash160(contentHash)`（20字节）
- 合约地址在部署时确定，终身不变

**AMM 合约地址查询方案**：

#### 方案1：在请求类型中添加 `AMMContractAddr` 字段 ✅ **推荐**

**优点**：
- ✅ 最简单直接
- ✅ 符合 WES 架构原则（业务语义在 SDK 层）
- ✅ 调用方明确指定合约地址，避免歧义
- ✅ 不需要额外的查询逻辑

**实现**：
- 在 `SwapRequest`、`AddLiquidityRequest`、`RemoveLiquidityRequest` 中添加 `AMMContractAddr []byte` 字段
- 调用方需要提供 AMM 合约的 `contentHash`（32字节）

**调用方如何获取 AMM 合约地址**：
- 从 AMM 合约部署时获得
- 从 AMM 合约的文档或配置中获取
- 通过 `wes_getContract` 查询（如果知道合约名称或其他标识）

#### 方案2：通过 StateOutput 查询 ⚠️ **不推荐**

**缺点**：
- ❌ 需要 AMM 合约在链上存储池地址映射
- ❌ 需要额外的查询逻辑
- ❌ 增加系统复杂度

#### 方案3：从配置中获取 ⚠️ **不推荐**

**缺点**：
- ❌ 需要 SDK 配置支持
- ❌ 配置管理复杂
- ❌ 不够灵活

---

## 📐 实现方案

### 推荐方案：在请求类型中添加 `AMMContractAddr` 字段

**理由**：
1. **符合 WES 架构原则**：业务语义在 SDK 层实现，调用方明确指定合约地址
2. **简单直接**：不需要额外的查询逻辑
3. **灵活性强**：支持多个 AMM 合约
4. **易于维护**：代码清晰，易于理解

**实现步骤**：
1. 在 `SwapRequest`、`AddLiquidityRequest`、`RemoveLiquidityRequest` 中添加 `AMMContractAddr []byte` 字段
2. 更新验证逻辑，确保 `AMMContractAddr` 为 32 字节（contentHash）
3. 实现真实的 `SwapAMM`、`AddLiquidity`、`RemoveLiquidity` 方法
4. 使用 `wes_callContract` API，设置 `return_unsigned_tx=true` 获取未签名交易
5. 使用 Wallet 签名未签名交易
6. 调用 `wes_sendRawTransaction` 提交已签名交易

---

## ✅ 结论

**WES 完全满足 AMM 功能的需求**，推荐使用**方案1**（在请求类型中添加 `AMMContractAddr` 字段）实现。

---

