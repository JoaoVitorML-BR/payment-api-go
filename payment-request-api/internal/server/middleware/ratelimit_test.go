// payment-request-api/internal/server/middleware/ratelimit_test.go
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimitMiddleware_AllowsWithinLimit(t *testing.T) {
	limiter := NewIPRateLimiter(Config{
		RequestsPerMinute: 60,
		Burst:             5,
		ClientTTL:         1 * time.Minute,
		CleanupFreq:       1 * time.Minute,
	})
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 on request %d, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksExceedingBurst(t *testing.T) {
	limiter := NewIPRateLimiter(Config{
		RequestsPerMinute: 60,
		Burst:             2,
		ClientTTL:         1 * time.Minute,
		CleanupFreq:       1 * time.Minute,
	})
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.POST("/payment", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "created"})
	})

	// 1st request -> 200
	req1, _ := http.NewRequest(http.MethodPost, "/payment", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}

	// 2nd request -> 200
	req2, _ := http.NewRequest(http.MethodPost, "/payment", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}

	// 3rd request -> 429 Too Many Requests
	req3, _ := http.NewRequest(http.MethodPost, "/payment", nil)
	req3.RemoteAddr = "10.0.0.1:1234"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w3.Code)
	}

	if w3.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header to be set")
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w3.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp["error"] != "rate_limit_exceeded" {
		t.Fatalf("expected error rate_limit_exceeded, got %v", resp["error"])
	}
}

func TestRateLimitMiddleware_DifferentIPsAreIsolated(t *testing.T) {
	limiter := NewIPRateLimiter(Config{
		RequestsPerMinute: 60,
		Burst:             1,
		ClientTTL:         1 * time.Minute,
		CleanupFreq:       1 * time.Minute,
	})
	defer limiter.Stop()

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// IP 1 uses its burst
	req1, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	req1.RemoteAddr = "10.0.0.1:1000"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("IP 1 request 1 expected 200, got %d", w1.Code)
	}

	// IP 1 gets throttled
	req2, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	req2.RemoteAddr = "10.0.0.1:1000"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("IP 1 request 2 expected 429, got %d", w2.Code)
	}

	// IP 2 is unaffected
	req3, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	req3.RemoteAddr = "10.0.0.2:2000"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("IP 2 request 1 expected 200, got %d", w3.Code)
	}
}

func TestIPRateLimiter_CleanupExpiredClients(t *testing.T) {
	limiter := NewIPRateLimiter(Config{
		RequestsPerMinute: 60,
		Burst:             5,
		ClientTTL:         10 * time.Millisecond,
		CleanupFreq:       20 * time.Millisecond,
	})
	defer limiter.Stop()

	// Register 2 clients
	limiter.getLimiter("client-1")
	limiter.getLimiter("client-2")

	if count := limiter.Len(); count != 2 {
		t.Fatalf("expected 2 tracked clients, got %d", count)
	}

	// Wait for TTL and cleanup trigger
	time.Sleep(50 * time.Millisecond)

	if count := limiter.Len(); count != 0 {
		t.Fatalf("expected 0 tracked clients after cleanup, got %d", count)
	}
}
