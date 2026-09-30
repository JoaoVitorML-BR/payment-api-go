// payment-request-api\cmd\main.go
package main

import (
	"fmt"
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/bootstrap"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	defer cfg.Pool.Close()

	router := bootstrap.NewRouter(cfg)

	if err := server.Run(cfg, router); err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	return nil
}


