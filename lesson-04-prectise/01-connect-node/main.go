package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 连接以太坊节点，打印链 ID 和最新区块高度。
func main() {
	log.Println("01-connect-node start.")
	rpcURL := os.Getenv("ETH_RPC_URL") // 从环境变量获取以太坊节点的 RPC URL, 方便调试
	fmt.Printf("ETH_RPC_URL: %s\n", rpcURL)
	if rpcURL == "" {
		println("请设置 ETH_RPC_URL 环境变量")
		return
	}

	//超时机制
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//连接以太坊节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}
	log.Printf("Connected to Ethereum network with chain ID: %s", chainID.String())

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get latest block header: %v", err)
	}
	log.Printf("Latest block number: %d", header.Number.Uint64())

	// 打印查询到的区块节点信息
	fmt.Println("=== Ethereum Node Info ===")
	fmt.Printf("RPC URL           : %s\n", rpcURL)
	fmt.Printf("Chain ID          : %s\n", chainID.String())
	fmt.Printf("Latest Block      : %d\n", header.Number.Uint64())
	fmt.Printf("Latest Block Hash : %s\n", header.Hash().Hex())
	fmt.Printf("Latest Block Time : %s\n", time.Unix(int64(header.Time), 0).Format(time.RFC3339))
	fmt.Println("==========================")
	/*
		⚠️  注意: 'Latest' 区块是节点当前认为的最新区块，可能尚未被所有节点确认。
		不同RPC节点可能返回不同的 'latest' 区块，导致与浏览器不匹配
		建议对比 'Safe' 或 'Finalized' 区块（已确认的区块）
	*/

	// 示例： 也可以获取任意指定高度的区块头
	if header.Number.Uint64() > 0 {
		num := new(big.Int).Sub(header.Number, big.NewInt(1)) // 获取前一个区块
		prevHeader, err := client.HeaderByNumber(ctx, num)
		if err == nil {
			log.Printf("Previous block  : %d (%s) \n", prevHeader.Number.Uint64(), prevHeader.Hash().Hex())
		}
	}

	// 查询 safe 区块 （浏览器通常显示这个）
	safeHeader, safeHash, err := getBlockByTag(ctx, client, "safe")
	if err != nil {
		log.Printf("Failed to get safe block header: %v", err)
	} else {
		log.Println("\n=== Safe Block (推荐对比) ===")
		log.Printf("Safe block number: %d \n", safeHeader.Number.Uint64())
		log.Printf("Safe block hash  : %s (RPC提供的hash,与浏览器一致)\n", safeHash.Hex())
		log.Printf("Calculated hash  : %s (Header计算的hash,可能不准确)\n", safeHeader.Hash().Hex())
		log.Printf("Safe block time  : %s \n", time.Unix(int64(safeHeader.Time), 0).Format(time.RFC3339))
		log.Printf("Confirmations     : %d \n", header.Number.Uint64()-safeHeader.Number.Uint64())
	}

	// 查询 finalized 区块
	finalizedHeader, finalizedHash, err := getBlockByTag(ctx, client, "finalized")
	if err != nil {
		log.Printf("Failed to get finalized block header: %v", err)
	} else {
		log.Println("\n=== Finalized Block ===")
		log.Printf("Finalized block number: %d \n", finalizedHeader.Number.Uint64())
		log.Printf("Finalized block hash  : %s (RPC提供的hash,与浏览器一致)\n", finalizedHash.Hex())
		log.Printf("Calculated hash  : %s (Header计算的hash,可能不准确)\n", finalizedHeader.Hash().Hex())
		log.Printf("Finalized block time  : %s \n", time.Unix(int64(finalizedHeader.Time), 0).Format(time.RFC3339))
		log.Printf("Confirmations         : %d \n", header.Number.Uint64()-finalizedHeader.Number.Uint64())
	}

}

// 封装方法： getBlockByTag 查询指定标签的区块头（safe, finalized, latest 等）
// 返回 Header、RPC 提供的 Hash 和错误
// 注意：需要使用底层 RPC 调用，因为 ethclient 的高级 API 不直接支持这些标签
func getBlockByTag(ctx context.Context, client *ethclient.Client, tag string) (*types.Header, common.Hash, error) {
	// 获取底层 RPC 客户端
	rpcClient := client.Client()

	// 获取区块头数据 （ 使用 false 只获取 header 不包含交易）
	var raw json.RawMessage
	err := rpcClient.CallContext(ctx, &raw, "eth_getBlockByNumber", tag, false)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("RPC call failed: %v", err)
	}

	if len(raw) == 0 || string(raw) == "null" {
		return nil, common.Hash{}, fmt.Errorf("No block found for tag: %s", tag)
	}

	// 解析完整的区块头字段
	var blockData struct {
		Number      string         `json:"number"`
		Hash        common.Hash    `json:"hash"`
		ParentHash  common.Hash    `json:"parentHash"`
		UncleHash   common.Hash    `json:"sha3Uncles"`
		Coinbase    common.Address `json:"miner"`
		Root        common.Hash    `json:"stateRoot"`
		TxHash      common.Hash    `json:"transactionsRoot"`
		ReceiptHash common.Hash    `json:"receiptsRoot"`
		Bloom       hexutil.Bytes  `json:"logsBloom"`
		Difficulty  *hexutil.Big   `json:"difficulty"`
		GasLimit    hexutil.Uint64 `json:"gasLimit"`
		GasUsed     hexutil.Uint64 `json:"gasUsed"`
		Time        hexutil.Uint64 `json:"timestamp"`
		Extra       hexutil.Bytes  `json:"extraData"`
		MixDigest   common.Hash    `json:"mixHash"`
		Nonce       hexutil.Bytes  `json:"nonce"`
		BaseFee     *hexutil.Big   `json:"baseFeePerGas"` // EIP-1559 之后的区块才有这个字段
	}
	if err := json.Unmarshal(raw, &blockData); err != nil {
		return nil, common.Hash{}, fmt.Errorf("Failed to unmarshal block header: %w", err)
	}
	// 解析区块号
	if len(blockData.Number) < 3 || blockData.Number[:2] != "0x" {
		return nil, common.Hash{}, fmt.Errorf("Invalid block number format: %s", blockData.Number)
	}
	num, ok := new(big.Int).SetString(blockData.Number[2:], 16) // 去掉 "0x" 前缀
	if !ok {
		return nil, common.Hash{}, fmt.Errorf("Invalid block number format: %s", blockData.Number)
	}

	// 构造完整的 Header 结构体
	header := &types.Header{
		ParentHash:  blockData.ParentHash,
		UncleHash:   blockData.UncleHash,
		Coinbase:    blockData.Coinbase,
		Root:        blockData.Root,
		TxHash:      blockData.TxHash,
		ReceiptHash: blockData.ReceiptHash,
		Bloom:       types.BytesToBloom(blockData.Bloom),
		Difficulty:  big.NewInt(0),
		Number:      num,
		GasLimit:    uint64(blockData.GasLimit),
		GasUsed:     uint64(blockData.GasUsed),
		Time:        uint64(blockData.Time),
		Extra:       blockData.Extra,
		MixDigest:   blockData.MixDigest,
		BaseFee:     nil,
	}
	// 设置Difficulty
	if blockData.Difficulty != nil {
		header.Difficulty = blockData.Difficulty.ToInt()
	}
	// 设置BaseFee (EIP-1559 之后的区块才有这个字段)
	if blockData.BaseFee != nil {
		header.BaseFee = blockData.BaseFee.ToInt()
	}
	// 设置 Nonce
	if len(blockData.Nonce) > 8 {
		var nonceBytes [8]byte
		copy(nonceBytes[:], blockData.Nonce[:8]) // 取前8字节
		header.Nonce = types.BlockNonce(nonceBytes)
	}
	// 返回解析后的 Header 和 RPC 提供的 Hash
	// AI提示-注意：RPC 返回的 Hash 可能与 ethclient 获取的 Header.Hash() 不同，因为 RPC 可能返回 'latest' 区块，而 ethclient 可能返回 'safe' 区块
	// 注意：手动构造的 Header 计算出的 hash 可能不准确，因为：
	// 1. RPC 返回的某些字段可能格式不完全匹配 go-ethereum 的内部格式
	// 2. Header 的内部缓存字段可能未正确初始化
	// 因此，我们应该直接使用 RPC 返回的 hash，它与浏览器显示的 hash 一致
	return header, blockData.Hash, nil
}
