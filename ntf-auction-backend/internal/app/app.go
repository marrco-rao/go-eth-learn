package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ntf-auction-backend/internal/config"
	"ntf-auction-backend/internal/database"
	httpHandler "ntf-auction-backend/internal/handler/http"
	"ntf-auction-backend/internal/logger"
	"ntf-auction-backend/internal/middleware"
	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
	"ntf-auction-backend/internal/service"
)

type Application struct {
	cfg             config.App
	DB              *gorm.DB
	Engine          *gin.Engine
	auctionListener *service.AuctionEventSubscriber
}

func New(cfg config.App) (*Application, error) {
	db, err := database.Open(cfg.Database)
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.User{}, &model.Log{}, &model.TxRecord{}, &model.ContractEvent{}); err != nil {
		return nil, err
	}

	userRepo := repository.NewGormUserRepository(db)
	logRepo := repository.NewGormLogRepository(db)
	txRecordRepo := repository.NewGormTxRecordRepository(db)
	contractEventRepo := repository.NewGormContractEventRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	ensureCtx, ensureCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ensureCancel()
	if err := authService.EnsureUser(ensureCtx, cfg.DefaultUser.Username, cfg.DefaultUser.Password); err != nil {
		return nil, err
	}

	logService := service.NewLogService(logRepo)
	auctionListener, err := service.NewAuctionEventSubscriberFromEnv(contractEventRepo, cfg.Ethereum.WatchEvents)
	if err != nil {
		return nil, err
	}
	dbLogger := logger.NewDBLogger(logRepo)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger(dbLogger))

	authHandler := httpHandler.NewAuthHandler(authService)
	logHandler := httpHandler.NewLogHandler(logService)

	rpcURL := cfg.Ethereum.RPCURL
	if rpcURL == "" {
		return nil, fmt.Errorf("ETH_RPC_URL is required")
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dialCancel()
	ethCli, err := ethclient.DialContext(dialCtx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial ETH_RPC_URL failed: %w", err)
	}

	blockRepo := repository.NewEthChainRepository(ethCli)
	blockService := service.NewBlockService(blockRepo)
	balanceService := service.NewEthBalanceService(blockRepo)
	txService := service.NewTransactionService(blockRepo, txRecordRepo, cfg.Ethereum.TxSignerPrivateKeyHex)
	blockHandler := httpHandler.NewBlockHandler(blockService)
	balanceHandler := httpHandler.NewETHBalanceHandler(balanceService)
	txHandler := httpHandler.NewTxHandler(txService)
	contractEventHandler := httpHandler.NewContractEventHandler(contractEventRepo)

	httpHandler.RegisterRoutes(engine, authHandler, logHandler, blockHandler, balanceHandler, txHandler, contractEventHandler, cfg.JWT.Secret)

	return &Application{cfg: cfg, DB: db, Engine: engine, auctionListener: auctionListener}, nil
}

func (a *Application) Run() error {
	if a.auctionListener != nil {
		log.Printf("[auction-subscriber] enabled, starting background subscription")
		go a.auctionListener.Run(context.Background())
	} else {
		log.Printf("[auction-subscriber] disabled (need %s and %s)", "ETH_WS_URL", "MARKETPLACE_CONTRACT_ADDRESS")
	}

	addr := fmt.Sprintf(":%d", a.cfg.Server.Port)
	return a.Engine.Run(addr)
}
