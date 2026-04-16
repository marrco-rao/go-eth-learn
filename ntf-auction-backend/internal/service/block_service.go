package service

import (
	"context"
	"math/big"
	"time"

	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

type BlockService struct {
	repo *repository.EthChainRepository
}

type BlocksRangeOutput struct {
	Success uint64                       `json:"success"`
	Skipped uint64                       `json:"skipped"`
	Data    []model.BlockSummaryResponse `json:"data"`
}

func NewBlockService(repo *repository.EthChainRepository) *BlockService {
	return &BlockService{repo: repo}
}

func (s *BlockService) GetLatestBlock(ctx context.Context) (model.BlockDetailResponse, error) {
	latest, err := s.repo.GetLatestBlockNumber(ctx)
	if err != nil {
		return model.BlockDetailResponse{}, err
	}

	block, err := s.repo.GetBlockByNumber(ctx, new(big.Int).SetUint64(latest))
	if err != nil {
		return model.BlockDetailResponse{}, err
	}

	return model.ToBlockDetailResponse(block), nil
}

func (s *BlockService) GetBlockByNumber(ctx context.Context, number uint64) (model.BlockDetailResponse, error) {
	block, err := s.repo.GetBlockByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return model.BlockDetailResponse{}, err
	}

	return model.ToBlockDetailResponse(block), nil
}

func (s *BlockService) GetBlocksByRange(ctx context.Context, start, end uint64, rateLimit int) (BlocksRangeOutput, error) {
	if rateLimit <= 0 {
		rateLimit = 200
	}

	interval := time.Second / time.Duration(rateLimit)
	items := make([]model.BlockSummaryResponse, 0, end-start+1)
	var success, skipped uint64

	for number := start; number <= end; number++ {
		block, err := s.repo.GetBlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			// Skip failed blocks and continue processing
			skipped++
			continue
		}

		items = append(items, model.ToBlockSummaryResponse(block))
		success++
		if number < end {
			select {
			case <-ctx.Done():
				return BlocksRangeOutput{}, ctx.Err()
			case <-time.After(interval):
			}
		}
	}

	return BlocksRangeOutput{
		Success: success,
		Skipped: skipped,
		Data:    items,
	}, nil
}
