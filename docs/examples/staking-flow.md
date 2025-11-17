# 质押流程示例

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

本示例演示完整的质押流程：质押、领取奖励、解质押。

---

## 💻 完整代码

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/weisyn/client-sdk-go/client"
    "github.com/weisyn/client-sdk-go/services/staking"
    "github.com/weisyn/client-sdk-go/wallet"
)

func main() {
    // 1. 创建客户端和钱包
    cfg := &client.Config{
        Endpoint: "http://localhost:8545",
        Protocol: client.ProtocolHTTP,
    }
    c, err := client.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    w, err := wallet.NewWallet()
    if err != nil {
        log.Fatal(err)
    }
    
    validatorWallet, _ := wallet.NewWallet()
    
    stakingService := staking.NewService(c)
    ctx := context.Background()
    
    // 2. 质押
    stakeResult, err := stakingService.Stake(ctx, &staking.StakeRequest{
        From:         w.Address(),
        ValidatorAddr: validatorWallet.Address(),
        Amount:       1000000,
        LockBlocks:   1000,
    }, w)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("质押成功！交易哈希: %s\n", stakeResult.TxHash)
    fmt.Printf("质押 ID: %s\n", stakeResult.StakeID)
    
    // 3. 等待一段时间后领取奖励
    fmt.Println("等待区块生成...")
    time.Sleep(10 * time.Second)
    
    claimResult, err := stakingService.ClaimReward(ctx, &staking.ClaimRewardRequest{
        From:    w.Address(),
        StakeID: stakeResult.StakeID,
    }, w)
    if err != nil {
        fmt.Println("暂无奖励:", err)
    } else {
        fmt.Printf("领取奖励成功！奖励金额: %d\n", claimResult.Reward)
    }
    
    // 4. 解质押
    unstakeResult, err := stakingService.Unstake(ctx, &staking.UnstakeRequest{
        From:    w.Address(),
        StakeID: stakeResult.StakeID,
    }, w)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("解质押成功！交易哈希: %s\n", unstakeResult.TxHash)
    fmt.Printf("解质押金额: %d\n", unstakeResult.UnstakeAmount)
    fmt.Printf("奖励金额: %d\n", unstakeResult.RewardAmount)
}
```

---

## 🔗 相关文档

- **[Staking 指南](../guides/staking.md)** - 详细使用指南
- **[快速开始](../getting-started.md)** - 安装和配置

---

