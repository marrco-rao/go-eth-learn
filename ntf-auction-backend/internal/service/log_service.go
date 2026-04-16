package service

import (
	"context"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

type LogService struct {
	logs repository.LogRepository
}

type LogListOutput struct {
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int64       `json:"total"`
	Items    []model.Log `json:"items"`
}

func NewLogService(logs repository.LogRepository) *LogService {
	return &LogService{logs: logs}
}

func (s *LogService) List(ctx context.Context, page, pageSize int) (LogListOutput, error) {
	if page < 1 {
		return LogListOutput{}, apperr.InvalidArgument("page must be >= 1")
	}
	if pageSize < 1 || pageSize > 100 {
		return LogListOutput{}, apperr.InvalidArgument("page_size must be in [1,100]")
	}

	items, total, err := s.logs.List(ctx, page, pageSize)
	if err != nil {
		return LogListOutput{}, apperr.Internal("query logs failed")
	}

	return LogListOutput{Page: page, PageSize: pageSize, Total: total, Items: items}, nil
}
