package repository

import (
	"context"

	"gorm.io/gorm"

	"ntf-auction-backend/internal/model"
)

type TxRecordRepository interface {
	Create(ctx context.Context, record *model.TxRecord) error
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (model.TxRecord, error)
	UpdateBroadcastResult(ctx context.Context, id uint, txHash string, nonce uint64) error
	UpdateStatus(ctx context.Context, id uint, status model.TxStatus, errMsg string) error
}

type GormTxRecordRepository struct {
	db *gorm.DB
}

// NewGormTxRecordRepository 创建基于 Gorm 的交易记录仓储。
func NewGormTxRecordRepository(db *gorm.DB) *GormTxRecordRepository {
	return &GormTxRecordRepository{db: db}
}

// Create 创建一条新的交易记录。
func (r *GormTxRecordRepository) Create(ctx context.Context, record *model.TxRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByIdempotencyKey 按幂等键查询交易记录。
func (r *GormTxRecordRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (model.TxRecord, error) {
	var record model.TxRecord
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&record).Error
	return record, err
}

// UpdateBroadcastResult 回写广播结果，并更新交易哈希、nonce 和状态。
func (r *GormTxRecordRepository) UpdateBroadcastResult(ctx context.Context, id uint, txHash string, nonce uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.TxRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"tx_hash": txHash,
			"nonce":   nonce,
			"status":  model.TxStatusBroadcasted,
		}).Error
}

// UpdateStatus 更新交易状态与错误信息。
func (r *GormTxRecordRepository) UpdateStatus(ctx context.Context, id uint, status model.TxStatus, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&model.TxRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"error_message": errMsg,
		}).Error
}
