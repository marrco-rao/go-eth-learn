package repository

import (
	"context"

	"gorm.io/gorm"

	"ntf-auction-backend/internal/model"
)

type LogRepository interface {
	Create(ctx context.Context, log model.Log) error
	List(ctx context.Context, page, pageSize int) ([]model.Log, int64, error)
}

type GormLogRepository struct {
	db *gorm.DB
}

func NewGormLogRepository(db *gorm.DB) *GormLogRepository {
	return &GormLogRepository{db: db}
}

func (r *GormLogRepository) Create(ctx context.Context, entry model.Log) error {
	return r.db.WithContext(ctx).Create(&entry).Error
}

func (r *GormLogRepository) List(ctx context.Context, page, pageSize int) ([]model.Log, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Log{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.Log
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("id desc").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
