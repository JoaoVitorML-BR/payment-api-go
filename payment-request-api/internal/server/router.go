// payment-request-api\internal\server\router.go
package server

import (
	"net/http"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter(paymentHandler *handler.PaymentHandler, oauthHandler *OAuthHandler, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	var paymentLimiterMiddleware, statusLimiterMiddleware, refundLimiterMiddleware gin.HandlerFunc

	if cfg == nil || cfg.RateLimitEnabled {
		paymentRPM := 30
		paymentBurst := 5
		statusRPM := 120
		statusBurst := 15
		refundRPM := 10
		refundBurst := 3

		if cfg != nil {
			if cfg.RateLimitPaymentRPM > 0 {
				paymentRPM = cfg.RateLimitPaymentRPM
			}
			if cfg.RateLimitPaymentBurst > 0 {
				paymentBurst = cfg.RateLimitPaymentBurst
			}
			if cfg.RateLimitStatusRPM > 0 {
				statusRPM = cfg.RateLimitStatusRPM
			}
			if cfg.RateLimitStatusBurst > 0 {
				statusBurst = cfg.RateLimitStatusBurst
			}
			if cfg.RateLimitRefundRPM > 0 {
				refundRPM = cfg.RateLimitRefundRPM
			}
			if cfg.RateLimitRefundBurst > 0 {
				refundBurst = cfg.RateLimitRefundBurst
			}
		}

		paymentLimiter := middleware.NewIPRateLimiter(middleware.Config{
			RequestsPerMinute: paymentRPM,
			Burst:             paymentBurst,
		})
		statusLimiter := middleware.NewIPRateLimiter(middleware.Config{
			RequestsPerMinute: statusRPM,
			Burst:             statusBurst,
		})
		refundLimiter := middleware.NewIPRateLimiter(middleware.Config{
			RequestsPerMinute: refundRPM,
			Burst:             refundBurst,
		})

		paymentLimiterMiddleware = middleware.RateLimitMiddleware(paymentLimiter)
		statusLimiterMiddleware = middleware.RateLimitMiddleware(statusLimiter)
		refundLimiterMiddleware = middleware.RateLimitMiddleware(refundLimiter)
	}

	wrap := func(mw gin.HandlerFunc, h gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, h}
		}
		return []gin.HandlerFunc{h}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/payment/client-secret/:payment_id", wrap(statusLimiterMiddleware, paymentHandler.GetPaymentClientSecretHandler)...)
	router.GET("/payment/:payment_id/status", wrap(statusLimiterMiddleware, paymentHandler.GetPaymentStatusHandler)...)

	router.POST("/payment", wrap(paymentLimiterMiddleware, paymentHandler.CreatePaymentRequestHandler)...)

	// Webhooks are intentionally exempted from restrictive rate limits to prevent dropping MP notifications during traffic spikes
	router.POST("/webhook/mercadopago", paymentHandler.MercadoPagoWebhookHandler)
	router.POST("/webhooks/mercadopago", paymentHandler.MercadoPagoWebhookHandler)
	router.POST("/", paymentHandler.MercadoPagoWebhookHandler)

	router.POST("/payment/refund", wrap(refundLimiterMiddleware, paymentHandler.RefundHandler)...)

	if oauthHandler != nil {
		router.GET("/oauth/mercadopago/start", oauthHandler.Start)
		router.GET("/oauth/mercadopago/callback", oauthHandler.Callback)
	}

	return router
}
