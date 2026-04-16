package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ntf-auction-backend/internal/model"
)

// ContractEventFilter 三个维度的筛选条件，空字符串表示不过滤该维度。
type ContractEventFilter struct {
	NetworkWSURL    string // 对应 ETH_WS_URL
	ContractAddress string // 合约地址
	EventType       string // 事件名称
}

type ContractEventRepository interface {
	Create(ctx context.Context, event *model.ContractEvent) error
	List(ctx context.Context, filter ContractEventFilter, page, pageSize int) ([]model.ContractEvent, int64, error)
}

type GormContractEventRepository struct {
	db *gorm.DB
}

func NewGormContractEventRepository(db *gorm.DB) *GormContractEventRepository {
	return &GormContractEventRepository{db: db}
}

// Create 幂等写入：同一条链上日志（TxHash + LogIndex）重复到达时忽略。
func (r *GormContractEventRepository) Create(ctx context.Context, event *model.ContractEvent) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(event).Error
}

// List 支持按 NetworkWSURL / ContractAddress / EventType 三个维度过滤，结果按区块号和日志索引降序。
func (r *GormContractEventRepository) List(ctx context.Context, filter ContractEventFilter, page, pageSize int) ([]model.ContractEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).Model(&model.ContractEvent{})
	if filter.NetworkWSURL != "" {
		q = q.Where("network_ws_url = ?", filter.NetworkWSURL)
	}
	if filter.ContractAddress != "" {
		q = q.Where("contract_address = ?", filter.ContractAddress)
	}
	if filter.EventType != "" {
		q = q.Where("event_type = ?", filter.EventType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var events []model.ContractEvent
	err := q.Order("block_number desc, log_index desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&events).Error
	return events, total, err
}
