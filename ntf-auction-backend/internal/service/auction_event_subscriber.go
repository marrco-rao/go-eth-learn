package service

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	goabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	marketplaceabi "ntf-auction-backend/abi"
	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

const (
	marketplaceWSURLEnv      = "ETH_WS_URL"
	marketplaceAddressEnv    = "MARKETPLACE_CONTRACT_ADDRESS"
	defaultReconnectInterval = 5 * time.Second
)

// 各事件非 indexed 字段的解包结构
type evtAuctionListed struct {
	StartingBid   *big.Int
	EndTime       *big.Int
	PaymentMethod uint8
}

type evtAuctionBidPlaced struct {
	BidAmount *big.Int
}

type evtAuctionEnded struct {
	FinalBidAmount *big.Int
}

type evtNFTListed struct {
	Price         *big.Int
	PaymentMethod uint8
}

type evtNFTPurchased struct {
	Price         *big.Int
	PaymentMethod uint8
}

type evtNFTListingPriceUpdated struct {
	NewPrice *big.Int
}

type AuctionEventSubscriber struct {
	events            repository.ContractEventRepository
	wsURL             string
	marketplace       common.Address
	watchEvents       []string
	reconnectInterval time.Duration
}

func NewAuctionEventSubscriberFromEnv(events repository.ContractEventRepository, watchEvents []string) (*AuctionEventSubscriber, error) {
	wsURL := strings.TrimSpace(os.Getenv(marketplaceWSURLEnv))
	addr := strings.TrimSpace(os.Getenv(marketplaceAddressEnv))

	if wsURL == "" || addr == "" {
		return nil, nil
	}

	if !common.IsHexAddress(addr) {
		return nil, fmt.Errorf("invalid %s: %s", marketplaceAddressEnv, addr)
	}

	if len(watchEvents) == 0 {
		return nil, fmt.Errorf("watch_events is empty, no events to subscribe")
	}

	return &AuctionEventSubscriber{
		events:            events,
		wsURL:             wsURL,
		marketplace:       common.HexToAddress(addr),
		watchEvents:       watchEvents,
		reconnectInterval: defaultReconnectInterval,
	}, nil
}

func (s *AuctionEventSubscriber) Run(ctx context.Context) {
	for {
		err := s.subscribeAndListen(ctx)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[auction-subscriber] subscription stopped, retry in %s: %v", s.reconnectInterval, err)
		t := time.NewTimer(s.reconnectInterval)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

func (s *AuctionEventSubscriber) subscribeAndListen(ctx context.Context) error {
	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	client, err := ethclient.DialContext(dialCtx, s.wsURL)
	dialCancel()
	if err != nil {
		return fmt.Errorf("dial ethereum node: %w", err)
	}
	defer client.Close()

	contractABI, err := goabi.JSON(strings.NewReader(marketplaceabi.MarketplaceMetaData.ABI))
	if err != nil {
		return fmt.Errorf("parse marketplace abi: %w", err)
	}

	// 构建 topic → 事件名 映射，并收集所有 topic 签名用于订阅过滤
	topicToEvent := make(map[common.Hash]string, len(s.watchEvents))
	sigs := make([]common.Hash, 0, len(s.watchEvents))
	for _, name := range s.watchEvents {
		evt, ok := contractABI.Events[name]
		if !ok {
			return fmt.Errorf("event %s not found in ABI", name)
		}
		topicToEvent[evt.ID] = name
		sigs = append(sigs, evt.ID)
	}

	// Topics[0] 填多个值，go-ethereum 按 OR 逻辑过滤，一次订阅覆盖所有事件
	logsCh := make(chan types.Log, 256)
	sub, err := client.SubscribeFilterLogs(ctx, ethereum.FilterQuery{
		Addresses: []common.Address{s.marketplace},
		Topics:    [][]common.Hash{sigs},
	}, logsCh)
	if err != nil {
		return fmt.Errorf("subscribe filter logs: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("[auction-subscriber] subscribed to %d events on contract=%s", len(s.watchEvents), s.marketplace.Hex())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return err
		case evtLog := <-logsCh:
			if len(evtLog.Topics) == 0 {
				continue
			}
			eventName, ok := topicToEvent[evtLog.Topics[0]]
			if !ok {
				continue
			}
			if err := s.handleLog(ctx, &contractABI, eventName, evtLog); err != nil {
				log.Printf("[auction-subscriber] handle %s tx=%s failed: %v", eventName, evtLog.TxHash.Hex(), err)
			}
		}
	}
}

func (s *AuctionEventSubscriber) handleLog(ctx context.Context, contractABI *goabi.ABI, eventName string, evtLog types.Log) error {
	record := model.ContractEvent{
		NetworkWSURL:    s.wsURL,
		ContractAddress: s.marketplace.Hex(),
		EventType:       eventName,
		TxHash:          evtLog.TxHash.Hex(),
		BlockNumber:     evtLog.BlockNumber,
		LogIndex:        uint(evtLog.Index),
	}

	// Topics[1] → 主账户地址（seller / bidder / buyer / winner）
	if len(evtLog.Topics) > 1 {
		record.AccountAddress = common.BytesToAddress(evtLog.Topics[1].Bytes()[12:]).Hex()
	}
	// Topics[2] → tokenId
	if len(evtLog.Topics) > 2 {
		record.TokenID = new(big.Int).SetBytes(evtLog.Topics[2].Bytes()).String()
	}

	// 解包各事件的非 indexed 字段
	switch eventName {
	case "NFTAuctionListed":
		var d evtAuctionListed
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.StartingBid.String()
		record.EndTime = d.EndTime.Uint64()
		record.PaymentMethod = d.PaymentMethod

	case "NFTAuctionBidPlaced":
		var d evtAuctionBidPlaced
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.BidAmount.String()

	case "NFTAuctionEnded":
		var d evtAuctionEnded
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.FinalBidAmount.String()

	case "NFTAuctionCancelled", "NFTAuctionDelisted", "NFTDelisted":
		// 无非 indexed 字段，无需解包

	case "NFTListed":
		var d evtNFTListed
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.Price.String()
		record.PaymentMethod = d.PaymentMethod

	case "NFTPurchased":
		var d evtNFTPurchased
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.Price.String()
		record.PaymentMethod = d.PaymentMethod

	case "NFTListingPriceUpdated":
		var d evtNFTListingPriceUpdated
		if err := contractABI.UnpackIntoInterface(&d, eventName, evtLog.Data); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
		record.AmountWei = d.NewPrice.String()
	}

	return s.events.Create(ctx, &record)
}
