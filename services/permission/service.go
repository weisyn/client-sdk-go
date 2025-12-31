package permission

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/weisyn/client-sdk-go/client"
	"github.com/weisyn/client-sdk-go/wallet"
)

// Service 权限管理服务接口
type Service interface {
	// TransferOwnership 转移所有权
	TransferOwnership(ctx context.Context, intent TransferOwnershipIntent, wallets ...wallet.Wallet) (*TransactionResult, error)

	// UpdateCollaborators 更新协作者
	UpdateCollaborators(ctx context.Context, intent UpdateCollaboratorsIntent, wallets ...wallet.Wallet) (*TransactionResult, error)

	// GrantDelegation 授予委托授权
	GrantDelegation(ctx context.Context, intent GrantDelegationIntent, wallets ...wallet.Wallet) (*TransactionResult, error)

	// SetTimeOrHeightLock 设置时间/高度锁
	SetTimeOrHeightLock(ctx context.Context, intent SetTimeOrHeightLockIntent, wallets ...wallet.Wallet) (*TransactionResult, error)
}

// TransactionResult 交易结果
type TransactionResult struct {
	TxHash  string
	Success bool
}

// permissionService 权限管理服务实现
type permissionService struct {
	client client.Client
	wallet wallet.Wallet // 可选：默认 Wallet
}

// NewService 创建权限管理服务（不带 Wallet）
func NewService(client client.Client) Service {
	return &permissionService{
		client: client,
	}
}

// NewServiceWithWallet 创建带默认 Wallet 的权限管理服务
func NewServiceWithWallet(client client.Client, w wallet.Wallet) Service {
	return &permissionService{
		client: client,
		wallet: w,
	}
}

// getWallet 获取 Wallet（优先使用参数，其次使用默认 Wallet）
func (s *permissionService) getWallet(wallets ...wallet.Wallet) wallet.Wallet {
	if len(wallets) > 0 && wallets[0] != nil {
		return wallets[0]
	}
	return s.wallet
}

// signAndSubmitTransaction 签名并提交交易（通用流程）
func (s *permissionService) signAndSubmitTransaction(
	ctx context.Context,
	unsignedTx *UnsignedTransaction,
	w wallet.Wallet,
) (*TransactionResult, error) {
	// 1. 序列化 draft
	draftJSON, err := json.Marshal(unsignedTx.Draft)
	if err != nil {
		return nil, fmt.Errorf("marshal draft failed: %w", err)
	}

	// 2. 调用 wes_computeSignatureHashFromDraft 获取签名哈希
	hashParams := map[string]interface{}{
		"draft":        json.RawMessage(draftJSON),
		"input_index":  unsignedTx.InputIndex,
		"sighash_type": "SIGHASH_ALL",
	}
	hashResult, err := s.client.Call(ctx, "wes_computeSignatureHashFromDraft", hashParams)
	if err != nil {
		return nil, fmt.Errorf("compute signature hash failed: %w", err)
	}

	hashMap, ok := hashResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_computeSignatureHashFromDraft")
	}
	hashHex, ok := hashMap["hash"].(string)
	if !ok || hashHex == "" {
		return nil, fmt.Errorf("missing hash in wes_computeSignatureHashFromDraft response")
	}

	unsignedTxHex, _ := hashMap["unsignedTx"].(string)

	hashBytes, err := hex.DecodeString(strings.TrimPrefix(hashHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode signature hash failed: %w", err)
	}

	// 3. 使用 Wallet 对哈希进行签名
	sigBytes, err := w.SignHash(hashBytes)
	if err != nil {
		return nil, fmt.Errorf("sign hash failed: %w", err)
	}

	// 4. 获取压缩公钥
	priv := w.PrivateKey()
	if priv == nil {
		return nil, fmt.Errorf("wallet private key is nil")
	}
	pubCompressed := ethcrypto.CompressPubkey(&priv.PublicKey)

	// 5. 调用 wes_finalizeTransactionFromDraft 完成交易
	finalizeParams := map[string]interface{}{
		"draft":        json.RawMessage(draftJSON),
		"unsignedTx":   unsignedTxHex,
		"input_index":  unsignedTx.InputIndex,
		"sighash_type": "SIGHASH_ALL",
		"pubkey":       "0x" + hex.EncodeToString(pubCompressed),
		"signature":    "0x" + hex.EncodeToString(sigBytes),
	}
	finalResult, err := s.client.Call(ctx, "wes_finalizeTransactionFromDraft", finalizeParams)
	if err != nil {
		return nil, fmt.Errorf("finalize transaction from draft failed: %w", err)
	}

	finalMap, ok := finalResult.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format from wes_finalizeTransactionFromDraft")
	}

	txHex, ok := finalMap["tx"].(string)
	if !ok {
		if txHex, ok = finalMap["txHex"].(string); !ok {
			return nil, fmt.Errorf("missing tx in wes_finalizeTransactionFromDraft response")
		}
	}

	// 6. 提交交易
	sendResult, err := s.client.SendRawTransaction(ctx, txHex)
	if err != nil {
		return nil, fmt.Errorf("send raw transaction failed: %w", err)
	}

	if !sendResult.Accepted {
		return nil, fmt.Errorf("transaction rejected: %s", sendResult.Reason)
	}

	return &TransactionResult{
		TxHash:  sendResult.TxHash,
		Success: true,
	}, nil
}

// TransferOwnership 转移所有权
func (s *permissionService) TransferOwnership(ctx context.Context, intent TransferOwnershipIntent, wallets ...wallet.Wallet) (*TransactionResult, error) {
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	unsignedTx, err := BuildTransferOwnershipTx(ctx, s.client, intent)
	if err != nil {
		return nil, fmt.Errorf("build transfer ownership tx failed: %w", err)
	}

	return s.signAndSubmitTransaction(ctx, unsignedTx, w)
}

// UpdateCollaborators 更新协作者
func (s *permissionService) UpdateCollaborators(ctx context.Context, intent UpdateCollaboratorsIntent, wallets ...wallet.Wallet) (*TransactionResult, error) {
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	unsignedTx, err := BuildUpdateCollaboratorsTx(ctx, s.client, intent)
	if err != nil {
		return nil, fmt.Errorf("build update collaborators tx failed: %w", err)
	}

	return s.signAndSubmitTransaction(ctx, unsignedTx, w)
}

// GrantDelegation 授予委托授权
func (s *permissionService) GrantDelegation(ctx context.Context, intent GrantDelegationIntent, wallets ...wallet.Wallet) (*TransactionResult, error) {
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	unsignedTx, err := BuildGrantDelegationTx(ctx, s.client, intent)
	if err != nil {
		return nil, fmt.Errorf("build grant delegation tx failed: %w", err)
	}

	return s.signAndSubmitTransaction(ctx, unsignedTx, w)
}

// SetTimeOrHeightLock 设置时间/高度锁
func (s *permissionService) SetTimeOrHeightLock(ctx context.Context, intent SetTimeOrHeightLockIntent, wallets ...wallet.Wallet) (*TransactionResult, error) {
	w := s.getWallet(wallets...)
	if w == nil {
		return nil, fmt.Errorf("wallet is required")
	}

	unsignedTx, err := BuildSetLockTx(ctx, s.client, intent)
	if err != nil {
		return nil, fmt.Errorf("build set lock tx failed: %w", err)
	}

	return s.signAndSubmitTransaction(ctx, unsignedTx, w)
}
