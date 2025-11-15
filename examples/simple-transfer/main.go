package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/services/token"
	"github.com/weisyn/client-sdk-go/wallet"
)

func main() {
	// 1. 创建客户端
	cfg := &client.Config{
		Endpoint: "http://localhost:8545",
		Protocol: client.ProtocolHTTP,
		Timeout:  30,
		Debug:    true,
	}
	
	httpClient, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer httpClient.Close()

	// 2. 创建钱包（示例：从私钥创建）
	// 注意：实际应用中应该从Keystore加载
	privateKeyHex := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	wallet, err := wallet.NewWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		log.Fatalf("创建钱包失败: %v", err)
	}
	
	fmt.Printf("钱包地址: %s\n", hex.EncodeToString(wallet.Address()))

	// 3. 创建Token服务
	tokenService := token.NewService(httpClient)

	// 4. 准备转账参数
	fromAddr := wallet.Address()
	toAddr := make([]byte, 20)
	// 示例：设置接收地址（实际应该从用户输入获取）
	copy(toAddr, []byte("recipient_address_here"))

	// 5. 执行转账
	ctx := context.Background()
	result, err := tokenService.Transfer(ctx, &token.TransferRequest{
		From:   fromAddr,
		To:     toAddr,
		Amount: 1000, // 转账金额
		TokenID: nil, // nil表示原生币
	})
	
	if err != nil {
		log.Fatalf("转账失败: %v", err)
	}

	fmt.Printf("转账成功！\n")
	fmt.Printf("交易哈希: %s\n", result.TxHash)
}

