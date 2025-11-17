# Staking 服务指南

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

Staking Service 提供质押相关功能，包括质押、解质押、委托、取消委托和奖励领取。

---

## 🔗 关联文档

- **API 参考**：[Services API - Staking](../api/services.md#-staking-service)
- **WES 协议**：[WES 质押机制](https://github.com/weisyn/weisyn/blob/main/docs/system/platforms/staking/README.md)（待确认）

---

## 🚀 快速开始

### 创建服务

```go
import (
    "context"
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/staking"
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

stakingService := staking.NewService(cli)
```

---

## 💎 质押

### 基本质押

```go
ctx := context.Background()
validatorWallet, _ := wallet.NewWallet()

result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:         w.Address(),
    ValidatorAddr: validatorWallet.Address(),
    Amount:       1000000, // 1 WES
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("质押成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("质押 ID: %s\n", result.StakeID)
```

### 带锁定期的质押

```go
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:         w.Address(),
    ValidatorAddr: validatorWallet.Address(),
    Amount:       1000000,
    LockBlocks:   1000, // 锁定 1000 个区块
}, w)
```

### 实现原理

SDK 内部：
1. 构建交易草稿，使用 `ContractLock` + `HeightLock`（如果指定了 `lockBlocks`）
2. 调用 `wes_buildTransaction` 构建交易
3. 签名并提交交易
4. 从交易输出中提取 `stakeId`

---

## 🔓 解质押

### 解质押

```go
result, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
    From:    w.Address(),
    StakeID: stakeID, // 之前质押时获得的 stakeID
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("解质押成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("解质押金额: %d\n", result.UnstakeAmount)
fmt.Printf("奖励金额: %d\n", result.RewardAmount)
```

### 注意事项

- ⚠️ 需要满足锁定条件（如 `lockBlocks` 已过期）
- ✅ SDK 自动计算解质押金额和奖励金额
- ✅ 解质押后，资金会返回到钱包

---

## 👥 委托

### 基本委托

```go
result, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
    From:         w.Address(),
    ValidatorAddr: validatorWallet.Address(),
    Amount:       500000, // 0.5 WES
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("委托成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("委托 ID: %s\n", result.DelegateID)
```

### 永不过期委托

```go
// 不指定 lockBlocks，表示永不过期
result, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
    From:         w.Address(),
    ValidatorAddr: validatorWallet.Address(),
    Amount:       500000,
}, w)
```

### 实现原理

SDK 内部使用 `DelegationLock` 锁定条件，表示资金委托给验证者。

---

## ❌ 取消委托

### 取消委托

```go
result, err := stakingService.Undelegate(ctx, &staking.UndelegateRequest{
    From:       w.Address(),
    DelegateID: delegateID, // 之前委托时获得的 delegateID
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("取消委托成功！交易哈希: %s\n", result.TxHash)
```

### 部分取消委托

```go
// 如果有多笔委托，可以部分取消
result, err := stakingService.Undelegate(ctx, &staking.UndelegateRequest{
    From:       w.Address(),
    DelegateID: delegateID,
    Amount:     200000, // 只取消部分金额
}, w)
```

---

## 🎁 领取奖励

### 通过 StakeID 领取

```go
result, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
    From:    w.Address(),
    StakeID: stakeID, // 质押 ID
}, w)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("领取奖励成功！交易哈希: %s\n", result.TxHash)
fmt.Printf("奖励金额: %d\n", result.Reward)
```

### 通过 DelegateID 领取

```go
result, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
    From:       w.Address(),
    DelegateID: delegateID, // 委托 ID
}, w)
```

### 注意事项

- ⚠️ 如果没有奖励，方法可能会失败
- ✅ SDK 自动查询奖励金额
- ✅ 奖励会直接转入钱包

---

## 🎯 典型场景

### 场景 1：完整质押流程

```go
func completeStakingFlow(
    ctx context.Context,
    stakerWallet wallet.Wallet,
    validatorAddr []byte,
    stakingService staking.Service,
) error {
    // 1. 质押
    stakeResult, err := stakingService.Stake(ctx, &staking.StakeRequest{
        From:         stakerWallet.Address(),
        ValidatorAddr: validatorAddr,
        Amount:       1000000,
        LockBlocks:   1000,
    }, stakerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("质押 ID: %s\n", stakeResult.StakeID)
    
    // 2. 等待一段时间后领取奖励
    // ... 等待区块生成 ...
    
    claimResult, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
        From:    stakerWallet.Address(),
        StakeID: stakeResult.StakeID,
    }, stakerWallet)
    if err != nil {
        fmt.Println("暂无奖励")
    } else {
        fmt.Printf("奖励: %d\n", claimResult.Reward)
    }
    
    // 3. 解质押
    unstakeResult, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
        From:    stakerWallet.Address(),
        StakeID: stakeResult.StakeID,
    }, stakerWallet)
    if err != nil {
        return err
    }
    
    fmt.Printf("解质押金额: %d\n", unstakeResult.UnstakeAmount)
    return nil
}
```

### 场景 2：委托给多个验证者

```go
func delegateToMultipleValidators(
    ctx context.Context,
    delegatorWallet wallet.Wallet,
    validators [][]byte,
    stakingService staking.Service,
) ([]string, error) {
    var delegateIDs []string
    
    for _, validator := range validators {
        result, err := stakingService.Delegate(ctx, &staking.DelegateRequest{
            From:         delegatorWallet.Address(),
            ValidatorAddr: validator,
            Amount:       100000,
        }, delegatorWallet)
        if err != nil {
            return nil, err
        }
        
        delegateIDs = append(delegateIDs, result.DelegateID)
    }
    
    return delegateIDs, nil
}
```

---

## ⚠️ 常见错误

### 余额不足

```go
result, err := stakingService.Stake(ctx, &staking.StakeRequest{
    From:         w.Address(),
    ValidatorAddr: validatorAddr,
    Amount:       1000000000, // 非常大的金额
}, w)
if err != nil {
    if strings.Contains(err.Error(), "insufficient balance") {
        log.Fatal("余额不足")
    }
    log.Fatal(err)
}
```

### 锁定未到期

```go
result, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
    From:    w.Address(),
    StakeID: stakeID,
}, w)
if err != nil {
    if strings.Contains(err.Error(), "lock not expired") {
        log.Fatal("锁定未到期，无法解质押")
    }
    log.Fatal(err)
}
```

---

## 🔗 相关文档

- **[API 参考](../api/services.md#-staking-service)** - 完整 API 文档
- **[Market 指南](./market.md)** - 市场服务指南
- **[故障排查](../troubleshooting.md)** - 常见问题

---

