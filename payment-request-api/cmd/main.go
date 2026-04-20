package main

import (
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
)

func main() {
	router := server.SetupRouter()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	if err := server.Run(cfg, router); err != nil {
		log.Fatal(err)
	}
}
