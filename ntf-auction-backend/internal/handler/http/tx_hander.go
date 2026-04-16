package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/response"
	"ntf-auction-backend/internal/service"
)

type TxHandler struct {
	txserv *service.TransactionService
}

type sendTransactionRequest struct {
	IdempotencyKey        string `json:"idempotencyKey"`
	BizID                 string `json:"bizId"`
	ToAddressHex          string `json:"toAddressHex"`
	ValueWei              string `json:"valueWei"`
	DataHex               string `json:"dataHex"`
	GasLimit              uint64 `json:"gasLimit"`
	MaxFeePerGasWei       string `json:"maxFeePerGasWei"`
	MaxPriorityFeeGasWei  string `json:"maxPriorityFeeGasWei"`
	UseSuggestedGasParams bool   `json:"useSuggestedGasParams"`
}

// NewTxHandler 创建交易相关的 HTTP 处理器。
func NewTxHandler(txserv *service.TransactionService) *TxHandler {
	return &TxHandler{txserv: txserv}
}

// GetTransaction 处理按交易哈希查询交易详情的请求。
func (h *TxHandler) GetTransaction(c *gin.Context) {
	txhash := c.Param("hash")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	transactionInput := service.GetTransactionInput{
		TxHashHex: txhash,
	}
	out, err := h.txserv.GetTransaction(ctx, transactionInput)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

// SendTransaction 处理发送交易请求，并返回广播后的交易哈希。
func (h *TxHandler) SendTransaction(c *gin.Context) {
	var req sendTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	input := service.SendTransactionInput{
		IdempotencyKey:        req.IdempotencyKey,
		BizID:                 req.BizID,
		ToAddressHex:          req.ToAddressHex,
		ValueWei:              req.ValueWei,
		DataHex:               req.DataHex,
		GasLimit:              req.GasLimit,
		MaxFeePerGasWei:       req.MaxFeePerGasWei,
		MaxPriorityFeeGasWei:  req.MaxPriorityFeeGasWei,
		UseSuggestedGasParams: req.UseSuggestedGasParams || (req.MaxFeePerGasWei == "" && req.MaxPriorityFeeGasWei == ""),
	}
	out, err := h.txserv.SendTransaction(ctx, input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}
