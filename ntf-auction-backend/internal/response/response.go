package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/apperr"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: apperr.CodeOK, Message: "success", Data: data})
}

func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*apperr.AppError); ok {
		c.JSON(appErr.HTTPStatus, Envelope{Code: appErr.Code, Message: appErr.Message})
		return
	}

	c.JSON(http.StatusInternalServerError, Envelope{Code: apperr.CodeInternal, Message: "internal server error"})
}
