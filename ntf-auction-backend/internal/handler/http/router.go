package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ntf-auction-backend/internal/middleware"
)

func RegisterRoutes(engine *gin.Engine, authHandler *AuthHandler, logHandler *LogHandler, blockHandler *BlockHandler, balanceHandler *ETHBalanceHandler, txHandler *TxHandler, contractEventHandler *ContractEventHandler, jwtSecret string) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)

		authGroup := api.Group("")
		authGroup.Use(middleware.JWTAuth(jwtSecret))
		authGroup.GET("/logs", logHandler.List)

		blockGroup := authGroup.Group("/blocks")
		{
			blockGroup.GET("/latest", blockHandler.getLatestBlock)
			blockGroup.GET("/:number", blockHandler.getBlockByNumber)
			blockGroup.GET("", blockHandler.getBlocksByRange)
		}
		// eth balance
		authGroup.GET("/balance/:address", balanceHandler.GetETHBalance)

		// transactions（有多个路由，用 group）
		txGroup := authGroup.Group("/transactions")
		{
			txGroup.GET("/:hash", txHandler.GetTransaction)
			txGroup.POST("", txHandler.SendTransaction)
		}

		// contract events（链上订阅事件查询）
		authGroup.GET("/contract-events", contractEventHandler.List)

	}
}
