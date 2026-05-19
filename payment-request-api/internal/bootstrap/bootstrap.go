package bootstrap

import (
	"os"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/messaging/rabbitmq"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server"
	"github.com/gin-gonic/gin"
)

// NewRouter initializes the payment service and sets up the HTTP router with the appropriate handlers.
func NewRouter(cfg *config.Config) *gin.Engine {

	rabbitmqURI := os.Getenv("RABBITMQ_URL")
	if rabbitmqURI == "" {
		rabbitmqURI = "amqp://guest:guest@localhost:5672/"
	}

	rabbitmqQueue := os.Getenv("RABBITMQ_QUEUE")
	if rabbitmqQueue == "" {
		rabbitmqQueue = "payment_requests"
	}

	publisher := rabbitmq.NewRabbitMQPaymentRequestedEventPublisher(
		rabbitmqURI,
		"payment.events",
		rabbitmqQueue,
		"payment.requested.v1",
	)

	paymentRepository, err := handler.NewPaymentRepositoryDB(cfg.Pool)
	if err != nil {
		panic("Failed to initialize payment repository")
	}
	paymentService, err := handler.NewPaymentService(paymentRepository, publisher)
	if err != nil {
		panic("Failed to initialize payment service")
	}
	paymentHandler, err := handler.NewPaymentHandler(paymentService)
	if err != nil {
		panic("Failed to initialize payment handler")
	}

	return server.SetupRouter(paymentHandler)
}
