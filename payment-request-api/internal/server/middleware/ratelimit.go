// payment-request-api/internal/server/middleware/ratelimit.go
package middleware

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// clientLimiter tracks a rate.Limiter and the last access time for an IP/client key.
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages per-client rate limiters with automatic cleanup.
type IPRateLimiter struct {
	mu          sync.RWMutex
	clients     map[string]*clientLimiter
	limit       rate.Limit
	burst       int
	cleanupFreq time.Duration
	clientTTL   time.Duration
	stopChan    chan struct{}
}

// Config represents rate limit configuration options.
type Config struct {
	// Rate is the number of allowed events per duration (e.g. 30 requests per minute).
	RequestsPerMinute int
	// Burst is the maximum number of tokens that can be consumed simultaneously.
	Burst int
	// ClientTTL is how long an idle client remains in memory before being evicted.
	ClientTTL time.Duration
	// CleanupFreq is how often the cleanup routine scans for expired clients.
	CleanupFreq time.Duration
}

// NewIPRateLimiter creates a new IPRateLimiter instance and starts its cleanup worker.
func NewIPRateLimiter(cfg Config) *IPRateLimiter {
	if cfg.RequestsPerMinute <= 0 {
		cfg.RequestsPerMinute = 60
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 10
	}
	if cfg.ClientTTL <= 0 {
		cfg.ClientTTL = 5 * time.Minute
	}
	if cfg.CleanupFreq <= 0 {
		cfg.CleanupFreq = 1 * time.Minute
	}

	limit := rate.Every(time.Minute / time.Duration(cfg.RequestsPerMinute))

	limiter := &IPRateLimiter{
		clients:     make(map[string]*clientLimiter),
		limit:       limit,
		burst:       cfg.Burst,
		cleanupFreq: cfg.CleanupFreq,
		clientTTL:   cfg.ClientTTL,
		stopChan:    make(chan struct{}),
	}

	go limiter.startCleanupWorker()

	return limiter
}

// Stop terminates the background cleanup goroutine.
func (rl *IPRateLimiter) Stop() {
	select {
	case <-rl.stopChan:
		// already closed
	default:
		close(rl.stopChan)
	}
}

// getLimiter retrieves or creates a rate.Limiter for the given client key.
func (rl *IPRateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cl, exists := rl.clients[key]
	if !exists {
		l := rate.NewLimiter(rl.limit, rl.burst)
		rl.clients[key] = &clientLimiter{
			limiter:  l,
			lastSeen: now,
		}
		return l
	}

	cl.lastSeen = now
	return cl.limiter
}

// startCleanupWorker periodically cleans up client entries that have been idle.
func (rl *IPRateLimiter) startCleanupWorker() {
	ticker := time.NewTicker(rl.cleanupFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

// cleanup removes clients that haven't been seen for longer than clientTTL.
func (rl *IPRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, cl := range rl.clients {
		if now.Sub(cl.lastSeen) > rl.clientTTL {
			delete(rl.clients, key)
		}
	}
}

// Len returns the current number of tracked clients (useful for testing and monitoring).
func (rl *IPRateLimiter) Len() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.clients)
}

// RateLimitMiddleware returns a Gin middleware handler enforcing the rate limit policy.
func RateLimitMiddleware(rl *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}

		limiter := rl.getLimiter(clientIP)
		res := limiter.Reserve()

		if !res.OK() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               "rate_limit_exceeded",
				"message":             "Too many requests. Please try again later.",
				"retry_after_seconds": 60,
			})
			return
		}

		delay := res.Delay()
		if delay > 0 {
			// Limit exceeded for immediate execution
			res.Cancel()
			retryAfterSec := int(math.Ceil(delay.Seconds()))
			if retryAfterSec < 1 {
				retryAfterSec = 1
			}

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSec))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               "rate_limit_exceeded",
				"message":             "Too many requests. Please try again later.",
				"retry_after_seconds": retryAfterSec,
			})
			return
		}

		c.Next()
	}
}
