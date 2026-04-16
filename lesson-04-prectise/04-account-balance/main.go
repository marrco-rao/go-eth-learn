package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 查询账户 ETH 余额（Wei 与 ETH）。
func main() {
	println("04-account-balance")
	addrHex := flag.String("address", "", "account address to query balance for（required）")
	blockNumber := flag.Int64("block", -1, "block number to query balance at (-1 means latest)")
	flag.Parse()

	if *addrHex == "" {
		log.Fatal("address flag is required")
	}

	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set (example: http://localhost:8545)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	defer client.Close()

	address := common.HexToAddress(*addrHex)

	var blockNum *big.Int
	if *blockNumber >= 0 {
		blockNum = big.NewInt(*blockNumber)
		log.Printf("Querying balance for %s at block %d", address.Hex(), blockNum.Int64())
	} else {
		log.Printf("Querying balance for %s at latest block", address.Hex())
	}

	balanceWei, err := client.BalanceAt(ctx, address, blockNum)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	fmt.Println("=== Account Balance ===")
	fmt.Printf("Address     : %s\n", address.Hex())
	if blockNum == nil {
		fmt.Printf("Block       : latest\n")
	} else {
		fmt.Printf("Block       : %d\n", blockNum.Uint64())
	}
	fmt.Printf("Balance Wei : %s\n", balanceWei.String())

	// 将余额从 Wei 转换为 ETH
	balanceEth := weiToEth(balanceWei)
	log.Printf("Balance: %s ETH (%s Wei)", balanceEth.Text('f', 6), balanceWei.String())
}

func weiToEth(wei *big.Int) *big.Float {
	fwei := new(big.Float).SetInt(wei)
	ethValue := new(big.Float).Quo(fwei, big.NewFloat(1e18))
	return ethValue
}
