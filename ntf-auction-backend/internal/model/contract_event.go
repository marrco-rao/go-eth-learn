package model

import "time"

// ContractEvent 存储从链上订阅到的合约事件记录。
// 可按 NetworkWSURL / ContractAddress / EventType 三个维度筛选查询。
type ContractEvent struct {
	ID uint `gorm:"primaryKey"`

	// 订阅维度字段
	NetworkWSURL    string `gorm:"size:255;not null;index"` // 订阅使用的 WebSocket 节点 URL（对应 ETH_WS_URL）
	ContractAddress string `gorm:"size:64;not null;index"`  // 合约地址（小写十六进制）
	EventType       string `gorm:"size:64;not null;index"`  // 事件名称（如 NFTAuctionListed）

	// 链上定位字段（TxHash + LogIndex 联合唯一，防止重复入库）
	TxHash      string `gorm:"size:80;not null;uniqueIndex:idx_contract_event_dedup"` // 交易哈希
	BlockNumber uint64 `gorm:"not null;index"`                                        // 区块号
	LogIndex    uint   `gorm:"not null;uniqueIndex:idx_contract_event_dedup"`         // 日志在区块中的索引

	// 通用参与方字段（对应 indexed 字段）
	AccountAddress string `gorm:"size:64;index"` // 主账户地址（seller / bidder / buyer / winner）
	TokenID        string `gorm:"size:78;index"` // NFT Token ID（十进制字符串）

	// 事件数值字段（对应非 indexed 字段，按事件类型语义不同）
	// NFTAuctionListed   → startingBid
	// NFTAuctionBidPlaced → bidAmount
	// NFTAuctionEnded    → finalBidAmount
	// NFTListed          → price
	// NFTPurchased       → price
	// NFTListingPriceUpdated → newPrice
	AmountWei     string `gorm:"size:78"` // 金额（Wei 单位）
	EndTime       uint64 // 拍卖结束时间戳（仅 NFTAuctionListed）
	PaymentMethod uint8  // 支付方式枚举（含支付方式的事件）

	CreatedAt time.Time
}
