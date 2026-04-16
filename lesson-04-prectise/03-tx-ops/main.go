package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 03-tx-ops.go
// 支持两种操作模式：
// 1. 查询交易：--tx <hash> - 按哈希查询交易与回执，解析关键字段
// 2. 发送交易：--send --to <address> --amount <eth> - 发起 ETH 转账交易
func main() {
	println("03-tx-ops")
	txHashHex := flag.String("tx", "", "transaction hash (for query mode)")
	sendMode := flag.Bool("send", false, "enable send transaction mode")
	toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")
	flag.Parse()

	// 判断操作模式
	if *sendMode {
		// 发送交易模式
		if *toAddrHex == "" || *amountEth <= 0 {
			log.Fatal("send mode requires --to and --amount flags")
		}
		sendTransaction(*toAddrHex, *amountEth)
	} else {
		// 查询交易模式
		if *txHashHex == "" {
			log.Fatal("query mode requires --tx flag, or use --send for send mode")
		}
		queryTransaction(*txHashHex)
	}
}

// 查询交易
func queryTransaction(txHashHex string) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 连接以太坊节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	// 查询交易
	txHash := common.HexToHash(txHashHex)
	tx, isPending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		log.Fatalf("Failed to get transaction: %v", err)
	}
	fmt.Println("=== Transaction ===")
	printTxBasicInfo(tx, isPending)

	// 查询交易回执:回执可能尚不可用（pending 交易）
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		log.Printf("Failed to get transaction receipt (may be pending): %v", err)
		return
	}
	fmt.Println("=== Receipt ===")
	printReceiptInfo(receipt)

}

// 发送交易
func sendTransaction(toAddrHex string, amountEth float64) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}
	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for send mode)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// 连接以太坊节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	// 解析私钥
	privkey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// 获取发送方地址
	publickey := privkey.Public()
	publicKeyECDSA, ok := publickey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Failed to cast public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddrHex)

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	// 获取 Nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("Failed to get suggested gas tip cap: %v", err)
	}
	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to get latest block header: %v", err)
	}
	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("Failed to get suggested gas price: %v", err)
		}
		baseFee = gasPrice
	}
	// fee cap = base fee * 2 + tip cap (简单策略，实际可根据情况调整)
	// gasFeeCap 是 EIP-1559 交易中的最大费用上限，计算方式为：
	// baseFee * 2 + gasTipCap
	// 其中 baseFee 是网络基础费用，gasTipCap 是给矿工的小费上限
	// gasFeeCap 表示用户愿意为每单位 gas 支付的最大总费用（包括基础费和优先费）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)
	// 估算 Gas Limit （普通转账固定为 21000）
	gasLimit := uint64(21000)

	// 转换 ETH 金额为 Wei
	amountWei := new(big.Float).Mul(
		big.NewFloat(amountEth),
		big.NewFloat(1e18),
	)
	valueWei, _ := amountWei.Int(nil)

	// 检查余额是否足够（包括手续费）
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}
	// 计算总费用： value + gasFeeCap * gasLimit
	totalCost := new(big.Int).Add(
		valueWei,
		new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit))),
	)
	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("Insufficient balance: have %s Wei, need at least %s Wei (including max fees)", balance.String(), totalCost.String())
	}

	// 构建交易
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      nil, // 普通转账没有数据
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privkey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// 发送交易
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())
}

func printTxBasicInfo(tx *types.Transaction, isPending bool) {
	fmt.Printf("Hash        : %s\n", tx.Hash().Hex())
	fmt.Printf("Nonce       : %d\n", tx.Nonce())
	fmt.Printf("Gas         : %d\n", tx.Gas())
	fmt.Printf("Gas Price   : %s\n", tx.GasPrice().String())
	fmt.Printf("To          : %v\n", tx.To())
	fmt.Printf("Value (Wei) : %s\n", tx.Value().String())
	fmt.Printf("Data Len    : %d bytes\n", len(tx.Data()))
	fmt.Printf("Pending     : %v\n", isPending)
}

func printReceiptInfo(r *types.Receipt) {
	fmt.Printf("Status      : %d\n", r.Status)
	fmt.Printf("BlockNumber : %d\n", r.BlockNumber.Uint64())
	fmt.Printf("BlockHash   : %s\n", r.BlockHash.Hex())
	fmt.Printf("TxIndex     : %d\n", r.TransactionIndex)
	fmt.Printf("Gas Used    : %d\n", r.GasUsed)
	fmt.Printf("Logs        : %d\n", len(r.Logs))
	if len(r.Logs) > 0 {
		fmt.Printf("First Log Address : %s\n", r.Logs[0].Address.Hex())
	}
}

// trim0x 移除十六进制字符串前缀 "0x"
func trim0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}
