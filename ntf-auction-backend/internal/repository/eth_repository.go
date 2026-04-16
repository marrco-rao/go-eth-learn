package repository

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthChainRepository struct {
	client *ethclient.Client
}

// NewEthChainRepository 创建以太坊链访问仓储。
func NewEthChainRepository(client *ethclient.Client) *EthChainRepository {
	return &EthChainRepository{client: client}
}

// GetLatestBlockNumber 获取链上最新区块高度。
func (r *EthChainRepository) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	return r.client.BlockNumber(ctx)
}

// GetBlockByNumber 按区块号查询区块详情。
func (r *EthChainRepository) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return r.client.BlockByNumber(ctx, number)
}

// GetAccountBalance 查询指定账户在给定区块高度的余额。
func (r *EthChainRepository) GetAccountBalance(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return r.client.BalanceAt(ctx, account, blockNumber)
}

// GetTransactionByHash 按哈希查询交易，并在已上链时附带交易回执。
func (r *EthChainRepository) GetTransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, *types.Receipt, error) {
	tx, isPending, err := r.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, nil, err
	}
	if isPending {
		return tx, nil, nil // 交易在内存池中，尚未打包
	}

	receipt, err := r.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return tx, nil, fmt.Errorf("get tx receipt failed: %w", err)
	}
	return tx, receipt, nil
}

// GetChainID 获取当前连接链的 Chain ID。
func (r *EthChainRepository) GetChainID(ctx context.Context) (*big.Int, error) {
	return r.client.ChainID(ctx)
}

// GetPendingNonceAt 获取账户在 pending 状态下的下一个 nonce。
func (r *EthChainRepository) GetPendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return r.client.PendingNonceAt(ctx, account)
}

// SuggestGasTipCap 获取节点建议的优先费上限。
func (r *EthChainRepository) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return r.client.SuggestGasTipCap(ctx)
}

// GetLatestHeader 获取最新区块头，用于读取 base fee 等动态参数。
func (r *EthChainRepository) GetLatestHeader(ctx context.Context) (*types.Header, error) {
	return r.client.HeaderByNumber(ctx, nil)
}

// EstimateGas 估算一笔交易的大致 gas 消耗。
func (r *EthChainRepository) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return r.client.EstimateGas(ctx, msg)
}

// SendTransaction 将已签名交易广播到链上节点。
func (r *EthChainRepository) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return r.client.SendTransaction(ctx, tx)
}
