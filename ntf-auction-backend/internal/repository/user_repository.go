package repository

import (
	"context"

	"gorm.io/gorm"

	"ntf-auction-backend/internal/model"
)

type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (model.User, error)
	Create(ctx context.Context, user model.User) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) GetByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return user, err
}

func (r *GormUserRepository) Create(ctx context.Context, user model.User) error {
	return r.db.WithContext(ctx).Create(&user).Error
}
