package main

import (
	"log"

	"ntf-auction-backend/internal/app"

	"ntf-auction-backend/internal/config"
)

func main() {
	cfg, err := config.Load("config/app.yaml")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app failed: %v", err)
	}

	log.Printf("server listening on :%d", cfg.Server.Port)
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
