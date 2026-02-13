package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimitMiddleware_SkipsHealthEndpoints(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"/health"},
		{"/ready"},
		{"/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := gin.New()
			r.Use(RateLimitMiddleware())
			r.GET(tt.path, func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			// Make many requests - should not be rate limited
			for i := 0; i < 300; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)
				r.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Request %d to %s got status %d, want 200", i, tt.path, w.Code)
					break
				}
			}
		})
	}
}

func TestRateLimitMiddleware_LimitsAPIEndpoints(t *testing.T) {
	// Use a very low limit for testing
	originalRPS := RequestsPerSecond
	originalBurst := BurstSize
	RequestsPerSecond = 1
	BurstSize = 2
	defer func() {
		RequestsPerSecond = originalRPS
		BurstSize = originalBurst
	}()

	// Clear limiters
	limiters.Range(func(key, value interface{}) bool {
		limiters.Delete(key)
		return true
	})

	r := gin.New()
	r.Use(RateLimitMiddleware())
	r.GET("/v1/sessions", func(c *gin.Context) {
		c.JSON(200, gin.H{"items": []interface{}{}})
	})

	// First few requests should succeed (burst)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i, w.Code)
		}
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit (429), got %d", w.Code)
	}
}

func TestRateLimitMiddleware_PerIPLimiting(t *testing.T) {
	// Use a very low limit for testing
	originalRPS := RequestsPerSecond
	originalBurst := BurstSize
	RequestsPerSecond = 1
	BurstSize = 1
	defer func() {
		RequestsPerSecond = originalRPS
		BurstSize = originalBurst
	}()

	// Clear limiters
	limiters.Range(func(key, value interface{}) bool {
		limiters.Delete(key)
		return true
	})

	r := gin.New()
	r.Use(RateLimitMiddleware())
	r.GET("/v1/sessions", func(c *gin.Context) {
		c.JSON(200, gin.H{"items": []interface{}{}})
	})

	// Request from IP 1
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("First request from IP1 should succeed, got %d", w1.Code)
	}

	// Second request from IP 1 should be rate limited
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request from IP1 should be rate limited, got %d", w2.Code)
	}

	// Request from IP 2 should succeed (different IP has its own limit)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req3.RemoteAddr = "192.168.1.2:12345"
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("First request from IP2 should succeed, got %d", w3.Code)
	}
}

func TestGetLimiter(t *testing.T) {
	// Clear limiters
	limiters.Range(func(key, value interface{}) bool {
		limiters.Delete(key)
		return true
	})

	// Get limiter for new IP
	limiter1 := getLimiter("10.0.0.1")
	if limiter1 == nil {
		t.Error("getLimiter should return a limiter")
	}

	// Get limiter for same IP should return same limiter
	limiter2 := getLimiter("10.0.0.1")
	if limiter1 != limiter2 {
		t.Error("getLimiter should return same limiter for same IP")
	}

	// Get limiter for different IP should return different limiter
	limiter3 := getLimiter("10.0.0.2")
	if limiter1 == limiter3 {
		t.Error("getLimiter should return different limiter for different IP")
	}
}
