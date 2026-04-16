package http

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/response"
	"ntf-auction-backend/internal/service"
)

type LogHandler struct {
	logs *service.LogService
}

func NewLogHandler(logs *service.LogService) *LogHandler {
	return &LogHandler{logs: logs}
}

func (h *LogHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		response.Error(c, apperr.InvalidArgument("page must be integer"))
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		response.Error(c, apperr.InvalidArgument("page_size must be integer"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	out, listErr := h.logs.List(ctx, page, pageSize)
	if listErr != nil {
		response.Error(c, listErr)
		return
	}

	response.Success(c, out)
}
