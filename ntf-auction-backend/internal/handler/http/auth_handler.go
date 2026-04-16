package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/response"
	"ntf-auction-backend/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.InvalidArgument("invalid request body"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	out, err := h.auth.Login(ctx, service.LoginInput{Username: req.Username, Password: req.Password})
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Envelope{Code: apperr.CodeOK, Message: "success", Data: out})
}
