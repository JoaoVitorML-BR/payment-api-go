// payment-request-api/internal/server/router_test.go
package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	handler "github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSetupRouter_RateLimitingIntegration(t *testing.T) {
	// Create mock config with small burst for testing
	cfg := &config.Config{
		RateLimitEnabled:      true,
		RateLimitPaymentRPM:   60,
		RateLimitPaymentBurst: 2,
		RateLimitStatusRPM:    60,
		RateLimitStatusBurst:  2,
		RateLimitRefundRPM:    60,
		RateLimitRefundBurst:  2,
	}

	// We can pass a minimal payment handler; we're testing the middleware interception on /payment
	paymentHandler := &handler.PaymentHandler{}
	router := SetupRouter(paymentHandler, nil, cfg)

	// 1st request to /payment -> passes through middleware (hits handler or bad request)
	req1, _ := http.NewRequest(http.MethodPost, "/payment", bytes.NewBufferString("{}"))
	req1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code == http.StatusTooManyRequests {
		t.Fatalf("expected request 1 not to be rate limited, got %d", w1.Code)
	}

	// 2nd request to /payment -> passes through middleware
	req2, _ := http.NewRequest(http.MethodPost, "/payment", bytes.NewBufferString("{}"))
	req2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code == http.StatusTooManyRequests {
		t.Fatalf("expected request 2 not to be rate limited, got %d", w2.Code)
	}

	// 3rd request to /payment -> should be blocked with 429 Too Many Requests
	req3, _ := http.NewRequest(http.MethodPost, "/payment", bytes.NewBufferString("{}"))
	req3.RemoteAddr = "10.0.0.1:1234"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected request 3 to be rate limited (429), got %d", w3.Code)
	}

	// Health endpoint should remain completely accessible (exempt)
	for i := 0; i < 5; i++ {
		reqHealth, _ := http.NewRequest(http.MethodGet, "/health", nil)
		reqHealth.RemoteAddr = "10.0.0.1:1234"
		wHealth := httptest.NewRecorder()
		router.ServeHTTP(wHealth, reqHealth)
		if wHealth.Code != http.StatusOK {
			t.Fatalf("expected /health to return 200, got %d", wHealth.Code)
		}
	}
}

func TestSetupRouter_RateLimitingDisabled(t *testing.T) {
	cfg := &config.Config{
		RateLimitEnabled: false,
	}

	paymentHandler := &handler.PaymentHandler{}
	router := SetupRouter(paymentHandler, nil, cfg)

	// Multiple requests should not get 429 when rate limiting is disabled
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("expected not to be rate limited when disabled, got 429 on request %d", i+1)
		}
	}
}
