package http

import (
	"context"
	"strconv"
	"time"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/response"
	"ntf-auction-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type BlockHandler struct {
	// 注入 service.BlockService
	blkserv *service.BlockService
}

func NewBlockHandler(blkserv *service.BlockService) *BlockHandler {
	return &BlockHandler{blkserv: blkserv}
}

// ===== 请求 DTO（建议放 transport/http 层） =====
type GetBlocksRangeRequest struct {
	Start     uint64 `form:"start" binding:"required"`
	End       uint64 `form:"end" binding:"required"`
	RateLimit int    `form:"rate_limit"`
}

func (b *BlockHandler) getLatestBlock(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	out, err := b.blkserv.GetLatestBlock(ctx)
	if err != nil {
		response.Error(c, apperr.Internal("failed to get latest block"))
		return
	}
	response.Success(c, out)
}

func (b *BlockHandler) getBlockByNumber(c *gin.Context) {
	numberStr := c.Param("number")
	number, err := strconv.ParseUint(numberStr, 10, 64)
	if err != nil {
		response.Error(c, apperr.InvalidArgument("number must be uint64"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	out, err := b.blkserv.GetBlockByNumber(ctx, number)
	if err != nil {
		response.Error(c, apperr.Internal("failed to get block by number"))
		return
	}

	response.Success(c, out)
}

func (b *BlockHandler) getBlocksByRange(c *gin.Context) {
	var req GetBlocksRangeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperr.InvalidArgument("invalid start/end query parameters"))
		return
	}

	if req.RateLimit == 0 {
		req.RateLimit = 200
	}
	if req.RateLimit < 0 {
		response.Error(c, apperr.InvalidArgument("rate_limit must be greater than or equal to 0"))
		return
	}
	if req.Start > req.End {
		response.Error(c, apperr.InvalidArgument("start must be less than or equal to end"))
		return
	}
	// 限定最大范围，防止请求过大导致服务压力过大，测试暂定100 ，生产环境改成可配置
	if req.End-req.Start > 100 {
		response.Error(c, apperr.InvalidArgument("range too large, max 100 blocks"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	out, err := b.blkserv.GetBlocksByRange(ctx, req.Start, req.End, req.RateLimit)
	if err != nil {
		response.Error(c, apperr.Internal("failed to get blocks by range"))
		return
	}
	response.Success(c, out)
}
