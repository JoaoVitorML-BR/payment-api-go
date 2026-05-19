package bootstrap

import (
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
	"github.com/gin-gonic/gin"
)

// NewRouter initializes the payment service and sets up the HTTP router with the appropriate handlers.
func NewRouter(cfg *config.Config) *gin.Engine {
	paymentRepository := handler.NewPaymentRepositoryDB(cfg.Pool)
	paymentService := handler.NewPaymentService(paymentRepository)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	return server.SetupRouter(paymentHandler)
}
