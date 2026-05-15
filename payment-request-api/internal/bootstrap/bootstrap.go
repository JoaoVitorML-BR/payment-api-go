package bootstrap

import (
	"fmt"
	"log"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
	"github.com/gin-gonic/gin"
)

// NewRouter initializes the payment service and sets up the HTTP router with the appropriate handlers.
func NewRouter(cfg *config.Config) (*gin.Engine, error) {
	log.Println("Initializing Payment Router")

	paymentRepository, err := handler.NewPaymentRepositoryDB(cfg.Pool)
	if err != nil {
		return nil, fmt.Errorf("init payment repository: %w", err)
	}

	paymentService, err := handler.NewPaymentService(paymentRepository)
	if err != nil {
		return nil, fmt.Errorf("init payment service: %w", err)
	}

	paymentHandler, err := handler.NewPaymentHandler(paymentService)
	if err != nil {
		return nil, fmt.Errorf("init payment handler: %w", err)
	}

	return server.SetupRouter(paymentHandler), nil
}