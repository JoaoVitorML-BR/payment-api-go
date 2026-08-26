// payment-request-api\internal\server\router.go
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

	router.GET("/payment/client-secret/:payment_id", func(c *gin.Context) {
		paymentHandler.GetPaymentClientSecretHandler(c)
	})

	router.POST("/payment", func(c *gin.Context) {
		paymentHandler.CreatePaymentRequestHandler(c)
	})

	router.POST("/webhook/mercadopago", func(c *gin.Context) {
		paymentHandler.MercadoPagoWebhookHandler(c)
	})

	router.POST("/payment/refund", func(c *gin.Context) {
		paymentHandler.RefundHandler(c)
	})

	return router
}
