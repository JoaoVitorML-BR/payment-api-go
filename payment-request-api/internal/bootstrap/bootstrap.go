// payment-request-api\internal\bootstrap\bootstrap.go
package bootstrap

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/messaging/rabbitmq"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/paymentmercadopago"
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

	mpAccessToken := strings.TrimSpace(os.Getenv("MERCADO_PAGO_ACCESS_TOKEN"))
	if mpAccessToken == "" {
		log.Fatal("MERCADO_PAGO_ACCESS_TOKEN is required for webhook validation")
	}
	gatewayReader := paymentmercadopago.NewGatewayReader(mpAccessToken)

	paymentService, err := handler.NewPaymentService(paymentRepository, publisher, gatewayReader)
	if err != nil {
		panic("Failed to initialize payment service")
	}
	paymentHandler, err := handler.NewPaymentHandler(paymentService, cfg)
	if err != nil {
		panic("Failed to initialize payment handler")
	}

	intervalSec := 60
	if envInterval := os.Getenv("RECONCILIATION_INTERVAL_SECONDS"); envInterval != "" {
		if val, err := strconv.Atoi(envInterval); err == nil && val > 0 {
			intervalSec = val
		}
	}

	minAgeMin := 2
	if envMinAge := os.Getenv("RECONCILIATION_MIN_AGE_MINUTES"); envMinAge != "" {
		if val, err := strconv.Atoi(envMinAge); err == nil && val > 0 {
			minAgeMin = val
		}
	}

	reconciler := handler.NewReconcilerWorker(
		paymentService,
		time.Duration(intervalSec)*time.Second,
		time.Duration(minAgeMin)*time.Minute,
		50,
	)
	reconciler.Start(context.Background())

	return server.SetupRouter(paymentHandler)
}

