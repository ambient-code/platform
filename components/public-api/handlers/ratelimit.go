package handlers

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter configuration
var (
	// RequestsPerSecond is the rate limit (configurable via RATE_LIMIT_RPS env var)
	RequestsPerSecond = getRateLimitFromEnv("RATE_LIMIT_RPS", 100)

	// BurstSize is the maximum burst size (configurable via RATE_LIMIT_BURST env var)
	BurstSize = getBurstFromEnv("RATE_LIMIT_BURST", 200)

	// Per-IP rate limiters
	limiters = sync.Map{}

	// Cleanup interval for stale limiters
	cleanupInterval = 5 * time.Minute
)

func init() {
	// Start background cleanup of stale limiters
	go cleanupStaleLimiters()
}

// limiterEntry holds a rate limiter and its last access time
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

func getRateLimitFromEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
			return f
		}
	}
	return defaultValue
}

func getBurstFromEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil && i > 0 {
			return i
		}
	}
	return defaultValue
}

// RateLimitMiddleware returns a middleware that rate limits requests per IP
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health/ready endpoints
		path := c.Request.URL.Path
		if path == "/health" || path == "/ready" || path == "/metrics" {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Get or create limiter for this IP
		limiter := getLimiter(clientIP)

		// Check if request is allowed
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": "1s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getLimiter returns the rate limiter for a given IP, creating one if needed
func getLimiter(ip string) *rate.Limiter {
	now := time.Now()

	// Try to get existing limiter
	if entry, ok := limiters.Load(ip); ok {
		e := entry.(*limiterEntry)
		e.lastAccess = now
		return e.limiter
	}

	// Create new limiter
	limiter := rate.NewLimiter(rate.Limit(RequestsPerSecond), BurstSize)
	entry := &limiterEntry{
		limiter:    limiter,
		lastAccess: now,
	}

	// Store it (may race with another goroutine, that's fine)
	actual, _ := limiters.LoadOrStore(ip, entry)
	return actual.(*limiterEntry).limiter
}

// cleanupStaleLimiters removes limiters that haven't been used in a while
func cleanupStaleLimiters() {
	for {
		time.Sleep(cleanupInterval)

		cutoff := time.Now().Add(-cleanupInterval)
		limiters.Range(func(key, value interface{}) bool {
			entry := value.(*limiterEntry)
			if entry.lastAccess.Before(cutoff) {
				limiters.Delete(key)
			}
			return true
		})
	}
}
