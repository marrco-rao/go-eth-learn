package http

import (
	"context"
	"ntf-auction-backend/internal/response"
	"ntf-auction-backend/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type ETHBalanceHandler struct {
	txserv *service.EthBalanceService
}

func NewETHBalanceHandler(txserv *service.EthBalanceService) *ETHBalanceHandler {
	return &ETHBalanceHandler{txserv: txserv}
}

func (h *ETHBalanceHandler) GetETHBalance(c *gin.Context) {
	address := c.Param("address")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	inputParam := service.GetETHBalanceInput{
		AddressHex:  address,
		BlockNumber: -1,
	}
	balance, err := h.txserv.GetETHBalance(ctx, inputParam)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, balance)
}
