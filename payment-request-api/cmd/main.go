// payment-request-api\cmd\main.go
package main

import (
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/bootstrap"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	defer cfg.Pool.Close()

	router := bootstrap.NewRouter(cfg)

	if err := server.Run(cfg, router); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}

