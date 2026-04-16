package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 05-subscribe-blocks.go
// 通过 SubscribeNewHead 订阅新区块头。
// 注意：大多数节点要求使用 WebSocket RPC，例如：ws://127.0.0.1:8546 或 wss://...
func main() {
	println("05-subscribe-blocks")
	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		// 回退到 ETH_RPC_URL，便于在只配置了 HTTP 的环境中看到错误提示
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		println("ETH_WS_URL or ETH_RPC_URL must be set")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		println("Failed to connect to the Ethereum client:", err.Error())
		return
	}
	defer client.Close()

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		println("Failed to subscribe to new head:", err.Error())
	}

	fmt.Printf("Subscribed to new block headers via %s. Waiting for new blocks...\n", rpcURL)

	// 捕获 Ctrl+C 信号以优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case h := <-headers:
			if h == nil {
				continue
			}
			fmt.Printf("[%s] New block received: number=%d hash=%s\n",
				time.Now().Format(time.RFC3339),
				h.Number.Uint64(),
				h.Hash().Hex(),
			)
		case err := <-sub.Err():
			fmt.Printf("[%s] Subscription error: %v\n", time.Now().Format(time.RFC3339), err)
			return
		case sig := <-sigCh:
			fmt.Printf("[%s] Received signal: %s. Shutting down...\n", time.Now().Format(time.RFC3339), sig.String())
			return
		case <-ctx.Done():
			fmt.Printf("[%s] Context cancelled. Exiting...\n", time.Now().Format(time.RFC3339))
			return

		}
	}

}
