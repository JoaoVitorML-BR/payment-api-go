package server

import (
	"net/http"

	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/gin-gonic/gin"
)

func SetupRouter(paymentHandler *handler.PaymentHandler) *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/payment", func(c *gin.Context) {
		paymentHandler.CreatePaymentRequestHandler(c)
	})

	return router
}
