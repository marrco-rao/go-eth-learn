package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 本示例演示一个“简单连接池与多节点策略”：
// - 多个 ethclient.Client 连接不同节点
// - 读操作做简单负载均衡（轮询）
// - 写操作固定主节点（主节点挂了再切换）
// - 节点不可用时自动标记失效并输出告警日志
//
// 使用方式：
//   export ETH_RPC_URLS="http://127.0.0.1:8545,https://sepolia.infura.io/v3/<project-id>"
//   go run main.go

// NodeStatus 代表单个节点的状态
type NodeStatus struct {
	URL    string
	Client *ethclient.Client
	Alive  bool
}

// EthClientPool 简单连接池
type EthClientPool struct {
	mu sync.RWMutex

	nodes []*NodeStatus

	// 写主节点索引（默认0），写操作固定主节点，主节点挂了再切换）
	primaryIdx int
	// 读操作轮询索引
	readIdx int
}

// NewEthClientPool 创建连接池，连接所有节点并初始化状态
func NewEthClientPool(ctx context.Context, urls []string) (*EthClientPool, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no RPC URLs provided")
	}

	nodes := make([]*NodeStatus, 0, len(urls))
	for _, url := range urls {
		u := strings.TrimSpace(url)
		if u == "" {
			continue
		}
		client, err := ethclient.DialContext(ctx, u)
		if err != nil {
			log.Printf("[WARN] connect rpc node failed. url=%s: err=%v\n", u, err)
			nodes = append(nodes, &NodeStatus{URL: u, Client: nil, Alive: false})
			continue
		}

		log.Printf("[INFO] Connected to rpc node %s\n", u)
		nodes = append(nodes, &NodeStatus{URL: u, Client: client, Alive: true})
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no rpc node connected successfully")
	}

	p := &EthClientPool{nodes: nodes, primaryIdx: 0, readIdx: 0}
	return p, nil
}

// pickReadClient 获取一个可用的读节点客户端，轮询方式
func (p *EthClientPool) pickReadClient() *NodeStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.nodes)
	for i := 0; i < n; i++ {
		idx := (p.readIdx + i) % n
		node := p.nodes[idx]
		if node.Alive && node.Client != nil {
			p.readIdx = (idx + 1) % n // 更新轮询索引
			return node
		}
	}
	return nil
}

// pickPrimaryClient 获取主节点客户端，主节点挂了尝试切换到下一个可用节点
func (p *EthClientPool) pickPrimaryClient() *NodeStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.nodes)

	// 先看当前primaryIdx指向的节点是否可用
	if n > 0 && p.primaryIdx < n {
		node := p.nodes[p.primaryIdx]

		if node.Alive && node.Client != nil {
			return node
		}
	}

	// 否则从头找一个可用的，顺便更新 primaryIdx
	for i := 0; i < n; i++ {
		node := p.nodes[i]
		if node.Alive && node.Client != nil {
			p.primaryIdx = i // 更新主节点索引
			return node
		}
	}
	return nil
}

// markNodeDead 标记节点不可用
func (p *EthClientPool) markNodeDead(url string, cause error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, node := range p.nodes {
		if node.URL == url {
			if node.Alive {
				log.Printf("[ERROR] Node is marked as dead, %s, err=%v\n", url, cause)
			}
			node.Alive = false
			return
		}
	}
}

// GetLatestBlockNumber 读操作：获取最新区块号（简单读负载均衡）
func (p *EthClientPool) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	node := p.pickReadClient()
	if node == nil {
		return nil, fmt.Errorf("no available read node")
	}

	blockNumber, err := node.Client.BlockNumber(ctx)
	if err != nil {
		p.markNodeDead(node.URL, err)
		return nil, err
	}
	return big.NewInt(int64(blockNumber)), nil
}

// GetBalance 读操作：获取账户余额
func (p *EthClientPool) GetBalance(ctx context.Context, addr common.Address) (*big.Int, error) {
	node := p.pickReadClient()
	if node == nil {
		return nil, fmt.Errorf("no available read node")
	}

	bal, err := node.Client.BalanceAt(ctx, addr, nil)
	if err != nil {
		p.markNodeDead(node.URL, err)
		return nil, err
	}
	return bal, nil
}

// SendTransaction 写操作：发送交易示例
// 这里不真实发送交易，模拟调用主节点接口
func (p *EthClientPool) SendTransaction(ctx context.Context, txData []byte) (common.Hash, error) {

	node := p.pickPrimaryClient()
	if node == nil {
		return common.Hash{}, fmt.Errorf("no available primary node")
	}
	log.Printf("[INFO] perform write operation via primary node: %s", node.URL)

	// 真实场景中，入校操作调用发起交易
	// node.Client.SendTransaction(ctx, nil)

	// 其他写操作

	// 模拟发送交易，返回一个随机哈希
	hash := common.BytesToHash(txData)
	return hash, nil
}

func main() {
	rpcURLsEnv := os.Getenv("ETH_RPC_URLS")
	if rpcURLsEnv == "" {
		log.Fatal("ETH_RPC_URLS is not set (example: http://127.0.0.1:8545,https://sepolia.infura.io/v3/<project-id>)")
	}
	// 从环境变量获取多个 RPC URL，逗号分隔
	rpcURLs := strings.Split(strings.TrimSpace(rpcURLsEnv), ",")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewEthClientPool(ctx, rpcURLs)
	if err != nil {
		log.Fatalf("Failed to create EthClientPool: %v", err)
	}

	fmt.Println("=== Multi Node Pool Demo ===")
	fmt.Printf("Configured RPC URLs:\n")
	for _, u := range rpcURLs {
		fmt.Printf("  - %s\n", strings.TrimSpace(u))
	}
	fmt.Println("============================")

	//示例 1：多次获取最新区块号，演示读负载均衡（轮询不同节点）
	for i := 0; i < 3; i++ {
		num, err := pool.GetLatestBlockNumber(ctx)
		if err != nil {
			log.Printf("[READ] get latest block failed: %v", err)
			continue
		}
		log.Printf("[READ] Latest block number: %d", num.Uint64())
		// time.Sleep(1 * time.Second)
	}

	// 示例 2：查询一个地址余额（这里使用 0 地址，仅做演示）
	addr := common.HexToAddress("0x0000000000000000000000000000000000000000")
	bal, err := pool.GetBalance(ctx, addr)
	if err != nil {
		log.Printf("[READ] get balance failed: %v", err)
	} else {
		log.Printf("[READ] Balance of %s: %s wei", addr.Hex(), bal.String())
	}

	// 示例 3：模拟发送交易（写操作），演示主节点选择
	txData := []byte("fake transaction data")
	hash, err := pool.SendTransaction(ctx, txData)
	if err != nil {
		log.Printf("[WRITE] send transaction failed: %v", err)
	} else {
		log.Printf("[WRITE] Transaction sent, hash: %s", hash.Hex())
	}
}
