package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 04-reconnect-strategy.go
// 展示订阅断线后的简单重连策略（示意实现）。
func main() {
	println("07-reconnect-strategy")
	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		println("ETH_WS_URL or ETH_RPC_URL must be set")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal %s, shutting down...\n", sig.String())
		cancel()
	}()

	runWithReconnect(ctx, rpcURL)
}

func runWithReconnect(ctx context.Context, rpcURL string) {
	var attempt int

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting reconnect loop.")
			return
		default:
		}

		attempt++
		fmt.Printf("Attempting to connect (attempt #%d) to %s\n", attempt, rpcURL)

		client, err := ethclient.DialContext(ctx, rpcURL)
		if err != nil {
			fmt.Printf("Failed to connect: %v\n", err)
			sleepWithBackoff(ctx, attempt)
			continue
		}

		headers := make(chan *types.Header)
		sub, err := client.SubscribeNewHead(ctx, headers)
		if err != nil {
			fmt.Printf("Failed to subscribe new heads: %v\n", err)
			client.Close()
			sleepWithBackoff(ctx, attempt)
			continue
		}

		log.Println("subscription established")

		// 订阅循环：如果 sub.Err() 返回错误，则跳出重新连接
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
				fmt.Printf("Subscription error: %v\n", err)
				client.Close()
				sleepWithBackoff(ctx, attempt)
				goto RECONNECT
			case <-ctx.Done():
				fmt.Println("Context cancelled, exiting subscription loop.")
				sub.Unsubscribe()
				client.Close()
				return
			}
		}
	RECONNECT:
		// 下一轮 for 循环会尝试重新连接
	}
}

func sleepWithBackoff(ctx context.Context, attempt int) {
	// 简单的指数退避策略，最大等待时间限制在 1 分钟
	sec := int(math.Min(60, math.Pow(2, float64(attempt))))
	d := time.Duration(sec) * time.Second
	fmt.Printf("Waiting for %s before next attempt...\n", d)

	t := time.NewTicker(d)
	defer t.Stop()

	select {
	case <-t.C:
	case <-ctx.Done():
	}
}
