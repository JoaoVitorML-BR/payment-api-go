package server

import (
	"net/http"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Create repository using pgxpool from config
	paymentRepository := handler.NewPaymentRepositoryDB(cfg.Pool)
	paymentService := handler.NewPaymentService(paymentRepository)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/payment", func(c *gin.Context) {
		paymentHandler.CreatePaymentRequestHandler(c)
	})

	return router
}
