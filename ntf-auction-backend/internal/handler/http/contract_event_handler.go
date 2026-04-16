package http

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/repository"
	"ntf-auction-backend/internal/response"
)

type ContractEventHandler struct {
	repo repository.ContractEventRepository
}

func NewContractEventHandler(repo repository.ContractEventRepository) *ContractEventHandler {
	return &ContractEventHandler{repo: repo}
}

// List 查询合约事件，支持 network_ws_url / contract_address / event_type 三个维度过滤。
// GET /api/v1/contract-events?network_ws_url=...&contract_address=...&event_type=...&page=1&page_size=20
func (h *ContractEventHandler) List(c *gin.Context) {
	filter := repository.ContractEventFilter{
		NetworkWSURL:    c.Query("network_ws_url"),
		ContractAddress: c.Query("contract_address"),
		EventType:       c.Query("event_type"),
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	events, total, err := h.repo.List(ctx, filter, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"events":   events,
	})
}
