package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/response"
)

func JWTAuth(secret string) gin.HandlerFunc {
	secretBytes := []byte(secret)

	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Error(c, apperr.Unauthorized("missing Authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, apperr.Unauthorized("invalid Authorization header"))
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return secretBytes, nil
		})
		if err != nil || !token.Valid {
			response.Error(c, apperr.Unauthorized("invalid token"))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, apperr.Unauthorized("invalid token claims"))
			c.Abort()
			return
		}

		c.Set("jwt_claims", claims)
		c.Next()
	}
}

func WriteUnauthorized(c *gin.Context) {
	response.Error(c, apperr.New(apperr.CodeUnauthorized, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized))
}
