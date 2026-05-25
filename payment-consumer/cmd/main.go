package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/bootstrap"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/worker"
)

func main() {
	log.Println("Starting payment consumer...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	deps, err := bootstrap.Bootstrap(cfg)
	if err != nil {
		log.Fatalf("Failed to bootstrap: %v", err)
	}

	defer deps.Pool.Close()

	log.Println("Consumer initialized successfully")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	worker.StartWorker(ctx, deps.Pool, deps.StripeClient, cfg)
}
