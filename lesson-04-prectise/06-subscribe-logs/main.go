package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 06-subscribe-logs.go
// 订阅指定合约的日志事件（如 ERC-20 Transfer），并解析事件参数。
// 本示例展示了如何从 logs 中解析出事件，包括 indexed 参数和普通参数。

// ERC-20 标准 ABI（包含 Transfer 事件定义）
const erc20ABIJSON = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "name": "from", "type": "address"},
      {"indexed": true, "name": "to", "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ],
    "name": "Transfer",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "name": "owner", "type": "address"},
      {"indexed": true, "name": "spender", "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ],
    "name": "Approval",
    "type": "event"
  }
]`

func main() {
	log.Println("06-subscribe-logs")
	contractAddr := flag.String("contract", "", "contract address to subscribe logs from (required)")
	flag.Parse()

	if *contractAddr == "" {
		log.Println("Please provide a contract address using --contract flag")
		return
	}

	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		fmt.Println("ETH_WS_URL is not set (example: ws://localhost:8546)")
		rpcURL = os.Getenv("ETH_RPC_URL")
	}
	if rpcURL == "" {
		log.Fatalf("ETH_WS_URL or ETH_RPC_URL must be set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 连接以太坊节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v \n	", err)
	}
	defer client.Close()

	// 解析 ABI，获取事件签名和参数信息
	// 这里可以使用 go-ethereum/accounts/abi 包来解析 ABI JSON，并获取事件 ID 和参数类型
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v \n", err)
	}

	contract := common.HexToAddress(*contractAddr)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contract},
	}

	logsCh := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		log.Fatalf("Failed to subscribe to logs: %v \n", err)
	}
	fmt.Printf("Subscribed to logs of contract %s via %s \n", contract.Hex(), rpcURL)
	fmt.Printf("Listening for events... \n\n")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case vlog := <-logsCh:
			// 解析日志事件
			parseLogEvent(&vlog, parsedABI)
		case err := <-sub.Err():
			log.Printf("Subscription error: %v \n", err)
			return
		case sig := <-sigCh:
			fmt.Printf("Received signal: %s, exiting...\n", sig.String())
			return
		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting...")
			return
		}
	}

}

// parseLogEvent 解析日志事件，根据 ABI 定义提取事件名称和参数值
// 展示如何从 logs 中解析出事件，包括 indexed 参数和普通参数。
func parseLogEvent(vlog *types.Log, parsedABI abi.ABI) {
	// 检查日志是否包含至少一个 Topic（事件 ID）：没有Topics 的日志不是事件日志，直接忽略
	if len(vlog.Topics) == 0 {
		return
	}

	// 步骤 1: 识别事件类型
	// Topic[0] 是事件签名的 keccak256 哈希值。
	// 例如: Transfer(address,address,uint256) 的事件 ID 是 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
	eventTopic := vlog.Topics[0]

	// 尝试识别是哪个事件（通过比较 Topics[0] 和事件签名的哈希）
	var eventName string
	var eventSig abi.Event
	// 遍历 ABI 中的所有事件，找到匹配的事件签名
	for name, event := range parsedABI.Events {
		// 计算事件签名的哈希值
		eventSigHash := crypto.Keccak256Hash([]byte(event.Sig))
		if eventSigHash == eventTopic {
			eventName = name
			eventSig = event
			break
		}
	}

	if eventName == "" {
		// 无法识别事件类型，打印原始信息
		fmt.Printf("[%s] Unknown event - Block: %d TxHash: %s, Topic[0]: %s \n",
			time.Now().Format(time.RFC3339),
			vlog.BlockNumber,
			vlog.TxHash.Hex(),
			eventTopic.Hex(),
		)
		return
	}

	// 步骤 2: 解析事件参数
	// indexed 参数在 Topics 中，非 indexed 参数在 Data 中
	fmt.Printf("========================================================\n\n")
	fmt.Printf("[%s] Event: %s\n", time.Now().Format(time.RFC3339), eventName)
	fmt.Printf("  Block Number: %d\n", vlog.BlockNumber)
	fmt.Printf("  Tx Hash     : %s\n", vlog.TxHash.Hex())
	fmt.Printf("  Log Index   : %d\n", vlog.Index)
	fmt.Printf("  Contract    : %s\n", vlog.Address.Hex())
	fmt.Printf("  Topics Count: %d\n", len(vlog.Topics))

	// 步骤3: 解析 indexed 参数（从 Topics 中提取）
	// Topics[0] 是事件签名哈希，Topics[1..N] 是 indexed 参数
	// 注意：只有前 3 个 indexed 参数会放在 Topics 中（Ethereum 限制）
	fmt.Printf("\n Indexed Parameters (from Topics):\n")

	// Topics[0] 是事件签名，所以 indexed 参数从 Topics[1] 开始
	// 注意：topicIndex 只针对 indexed 参数计数，不考虑非 indexed 参数
	indexedParamIndex := 0
	for i, input := range eventSig.Inputs {
		if !input.Indexed {
			continue
		}
		// indexed 参数在 Topics 中的位置 = 1 + indexed 参数的索引
		topicIndex := 1 + indexedParamIndex
		indexedParamIndex++

		if topicIndex >= len(vlog.Topics) {
			continue
		}

		topic := vlog.Topics[topicIndex]
		fmt.Printf("  [%d] %s (%s): \n", i+1, input.Name, input.Type)

		// 根据类型解析 indexed 参数
		switch input.Type.T {
		case abi.AddressTy:
			// address 类型：去除前 12 字节的 0 填充，后 20 字节是地址
			addr := common.BytesToAddress(topic.Bytes()[12:])
			fmt.Printf("      Address: %s\n", addr.Hex())
		case abi.UintTy, abi.IntTy:
			// 数字类型：直接解析为 big.Int
			value := new(big.Int).SetBytes(topic.Bytes())
			fmt.Printf("%s\n", value.String())
		case abi.BoolTy:
			// bool 类型：检查最后一个字节
			fmt.Printf("%t\n", topic[31] != 0)
		case abi.BytesTy, abi.FixedBytesTy:
			// bytes 类型：直接打印十六进制
			fmt.Printf("%s\n", topic.Hex())
		default:
			// 其他类型：显示原始十六进制
			fmt.Printf("%s （raw）\n", topic.Hex())
		}
	}

	// 步骤4: 解析非 indexed 参数（从 Data 中提取）
	// Data 字段包含所有非 indexed 参数的编码值
	if len(vlog.Data) > 0 {
		fmt.Printf("\n Non-Indexed Parameters (from Data):\n")
		// 创建一个结构体来接收解码后的参数
		// 注意：这里使用通用方法，实际应用中可能需要根据具体事件定义结构体
		nonIndexedInputs := make([]abi.Argument, 0)
		for _, input := range eventSig.Inputs {
			if !input.Indexed {
				nonIndexedInputs = append(nonIndexedInputs, input)
			}
		}

		if len(nonIndexedInputs) > 0 {
			// 使用 ABI 包解码 Data 字段
			// 方法 1: 使用 UnpackIntoInterface（需要预定义结构体）
			// 方法 2: 使用 Unpack（返回 []interface{}）
			values, err := parsedABI.Unpack(eventName, vlog.Data)
			if err != nil {
				fmt.Printf("Failed to decode data: %v\n", err)
			} else {
				// 只输出非 indexed 参数
				nonIndexedIdx := 0
				for i, input := range eventSig.Inputs {
					if !input.Indexed {
						value := values[nonIndexedIdx]
						fmt.Printf("  [%d] %s (%s): ", i+1, input.Name, input.Type)
						// 根据类型进一步格式化输出
						switch v := value.(type) {
						case *big.Int:
							fmt.Printf("%s\n", v.String())
						case common.Address:
							fmt.Printf("%s\n", v.Hex())
						case []byte:
							// - %x 格式化占位符用于将字节数组以十六进制格式输出，不带 0x 前缀
							fmt.Printf("0x%x\n", v)
						default:
							fmt.Printf("%v\n", v)
						}
						nonIndexedIdx++
					}
				}
			}
		}

	} else {
		fmt.Printf("\n No non-indexed parameters (Data is empty)\n")
	}
	fmt.Printf("========================================================\n\n")

}
