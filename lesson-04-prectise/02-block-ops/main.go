package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 02-block-ops.go
// 查询最新区块、指定区块以及批量查询区块范围的信息。
//
// 使用示例：
//
//	# 查询最新区块
//	go run main.go
//
//	# 查询指定区块
//	go run main.go -number 123456
//
//	# 批量查询区块范围 [100, 105]
//	go run main.go -range-start 100 -range-end 105
//
//	# 批量查询，自定义请求间隔（毫秒）
//	go run main.go -range-start 100 -range-end 105 -rate-limit 500
func main() {
	println("02-block-ops")
	blockNumberFlag := flag.Uint64("number", 0, "block number to query (0 means skip)")
	rangeStartFlag := flag.Uint64("range-start", 0, "start block number for range query")
	rangeEndFlag := flag.Uint64("range-end", 0, "end block number for range query")
	rateLimitFlag := flag.Int("rate-limit", 200, "rate limit in milliseconds between requests")
	flag.Parse()

	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set (example: http://127.0.0.1:8545)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	defer client.Close()

	// 查询最新区块
	latestBlock, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get latest block: %v", err)
	}
	printBlockInfo("Latest Block", latestBlock)

	// 指定区块
	if *blockNumberFlag > 0 {
		number := new(big.Int).SetUint64(*blockNumberFlag)
		block, err := fetchBlockWithRetry(ctx, client, number, 3)
		if err != nil {
			log.Fatalf("Failed to get block %d: %v", *blockNumberFlag, err)
		}
		printBlockInfo(fmt.Sprintf("Block %d", *blockNumberFlag), block)
	}

	// 批量查询区块范围
	if *rangeStartFlag > 0 || *rangeEndFlag > 0 {
		if *rangeStartFlag == 0 || *rangeEndFlag == 0 {
			log.Fatal("both -range-start and -range-end must be set for range query")
		}
		if *rangeStartFlag > *rangeEndFlag {
			log.Fatal("-range-start must be <= to -range-end")
		}

		rateLimit := time.Duration(*rateLimitFlag) * time.Millisecond
		fetchBlockWithRange(ctx, client, *rangeStartFlag, *rangeEndFlag, rateLimit)
	}
}

// fetchBlockWithRetry 带重试机制的区块查询
func fetchBlockWithRetry(ctx context.Context, client *ethclient.Client, BlockNumber *big.Int, MaxRetries int) (*types.Block, error) {
	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		// 每次重试使用新的超时上下文，避免上下文被取消
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		block, err := client.BlockByNumber(reqCtx, BlockNumber)
		cancel()
		if err == nil {
			return block, nil
		}

		lastErr = err
		if i < MaxRetries-1 {
			backoff := time.Duration(i+1) * 500 * time.Millisecond
			log.Printf("[WARN] failed to fetch block %s, retrying  %d/%d after %v: %v \n",
				BlockNumber.String(), i+1, MaxRetries, backoff, err)
			time.Sleep(backoff)
		}

	}
	return nil, fmt.Errorf("failed to fetch block %d after %d attempts: last error: %v", BlockNumber.Uint64(), MaxRetries, lastErr)
}

// fetchBlockWithRange 批量查询区块范围
func fetchBlockWithRange(ctx context.Context, client *ethclient.Client, start, end uint64, rateLimit time.Duration) {
	log.Printf("\n=== Fetching Block Range [%d, %d] ===\n", start, end)
	log.Printf("Rate Limit: %v per request\n\n", rateLimit)

	successCount := 0
	skipCount := 0
	ticker := time.NewTicker(rateLimit)
	defer ticker.Stop()

	for num := start; num <= end; num++ {
		<-ticker.C // 等待下一个请求时间窗口

		blockNumber := big.NewInt(0).SetUint64(num)
		block, err := fetchBlockWithRetry(ctx, client, blockNumber, 3)
		if err != nil {
			log.Printf("[ERROR] failed to fetch block %d: %v\n", num, err)
			skipCount++
			continue
		}
		successCount++
		printBlockInfo(fmt.Sprintf("Block %d", num), block)

		// 检查上下文是否已取消（例如超时），如果是则提前退出
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context cancelled, stopping at block: %d", num)
			return
		default:
		}

	}
	log.Printf("\n=== Block Range Fetch Summary ===\n")
	log.Printf("Total blocks requested: %d\n", end-start+1)
	log.Printf("Successfully fetched: %d\n", successCount)
	log.Printf("Failed/skipped: %d\n", skipCount)
	log.Printf("Success rate: %.2f%%\n", float64(successCount)/float64(end-start+1)*100)
}

// printBlockInfo 打印详细的区块信息
func printBlockInfo(title string, block *types.Block) {
	log.Println("=====================================")
	log.Println(title)
	log.Println("=====================================")
	log.Printf("Block: %+v\n", block)

	// 基本信息
	log.Printf("Number: %d\n", block.Number().Uint64())
	log.Printf("Hash: %s\n", block.Hash().Hex())
	log.Printf("Parent Hash: %s\n", block.ParentHash().Hex())
	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	log.Printf("Time: %s\n", blockTime.Format(time.RFC3339))
	log.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))
	// Gas 信息
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()
	gasUsagePercent := float64(gasUsed) / float64(gasLimit) * 100
	log.Printf("Gas Used: %d (%.2f%%)\n", gasUsed, gasUsagePercent)
	log.Printf("Gas Limit: %d\n", gasLimit)
	// 交易信息
	txCount := len(block.Transactions())
	log.Printf("Tx Count: %d\n", txCount)
	// 区块根信息（Merkle 树根）
	log.Printf("State Root: %s\n", block.Root().Hex())
	log.Printf("Tx Root    : %s\n", block.TxHash().Hex())
	log.Printf("Receipts Root : %s\n", block.ReceiptHash().Hex())
	// 区块大小估算（简化版，实际大小还包括其他字段）
	if txCount > 0 {
		log.Printf("\nFirst Tx Hash: %s\n", block.Transactions()[0].Hash().Hex())
		if txCount > 1 {
			log.Printf("Last Tx Hash: %s\n", block.Transactions()[txCount-1].Hash().Hex())
		}
	}
	// 难度信息（PoW 相关，PoS 后基本固定）
	log.Printf("Difficulty.   : %s\n", block.Difficulty().String())
	// 区块奖励相关
	coinbase := block.Coinbase()
	if coinbase != (common.Address{}) {
		log.Printf("Coinbase: %s\n", coinbase.Hex())
	}
	log.Println("=====================================")
}
