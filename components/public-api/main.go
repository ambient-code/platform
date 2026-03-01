// Public API Gateway Service
//
// ARCHITECTURE NOTE: This service is a stateless HTTP gateway that forwards
// authenticated requests to the backend. Unlike the backend service, we do NOT
// create K8s clients here - all K8s operations and RBAC validation happen in
// the backend service.
//
// Our role is to:
// 1. Extract and validate tokens (middleware.go)
// 2. Extract project context (from header or token)
// 3. Validate input parameters (prevent injection attacks)
// 4. Forward requests with proper authorization headers
//
// This is intentionally different from the backend pattern (GetK8sClientsForRequest)
// because this service should never access Kubernetes directly. The ServiceAccount
// for this service has NO RBAC permissions. All K8s operations are performed by
// the backend using the user's forwarded token.
package main

import (
	"fmt"
	"os"
	"time"

	"ambient-code-public-api/handlers"
	"ambient-code-public-api/observability"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	// Validate required environment variables
	if err := validateConfig(); err != nil {
		observability.Logger.Fatal().Err(err).Msg("Configuration validation failed")
	}

	// Initialize OpenTelemetry (if enabled)
	shutdown := observability.InitTracer()
	defer shutdown()

	// Set Gin mode from environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Recovery middleware
	r.Use(gin.Recovery())

	// CORS middleware - allow browser clients
	r.Use(cors.New(cors.Config{
		AllowOrigins:     getAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Ambient-Project"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// OpenTelemetry tracing middleware
	if observability.TracingEnabled() {
		r.Use(otelgin.Middleware("public-api"))
	}

	// Structured logging middleware (JSON format)
	r.Use(observability.StructuredLoggingMiddleware())

	// Rate limiting middleware
	r.Use(handlers.RateLimitMiddleware())

	// Health endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Readiness endpoint
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	// Metrics endpoint (Prometheus format)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// v1 API routes
	// IMPORTANT: AuthMiddleware must run BEFORE LoggingMiddleware
	// to ensure we only log authenticated requests with valid project context
	v1 := r.Group("/v1")
	v1.Use(handlers.AuthMiddleware())
	{
		// Sessions
		v1.GET("/sessions", handlers.ListSessions)
		v1.POST("/sessions", handlers.CreateSession)
		v1.GET("/sessions/:id", handlers.GetSession)
		v1.DELETE("/sessions/:id", handlers.DeleteSession)

		// Session runs (AG-UI prompt delivery)
		v1.POST("/sessions/:id/runs", handlers.CreateSessionRun)
	}

	// Get port from environment or default to 8081
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	observability.Logger.Info().
		Str("port", port).
		Str("backend_url", handlers.BackendURL).
		Bool("tracing_enabled", observability.TracingEnabled()).
		Msg("Starting Public API server")

	if err := r.Run(":" + port); err != nil {
		observability.Logger.Fatal().Err(err).Msg("Failed to start server")
	}
}

// validateConfig validates required configuration on startup
func validateConfig() error {
	// BACKEND_URL is optional (has default), but warn if not set
	if os.Getenv("BACKEND_URL") == "" {
		observability.Logger.Warn().
			Str("default", "http://backend-service:8080").
			Msg("BACKEND_URL not set, using default")
	}

	// Validate BACKEND_URL format if set
	backendURL := handlers.BackendURL
	if backendURL == "" {
		return fmt.Errorf("BACKEND_URL resolved to empty string")
	}

	return nil
}

// getAllowedOrigins returns the list of allowed CORS origins
func getAllowedOrigins() []string {
	// Check for explicit CORS origins
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		// Parse comma-separated list
		return splitAndTrim(origins)
	}

	// Default: allow common development origins
	return []string{
		"http://localhost:3000",      // Next.js dev server
		"http://localhost:8080",      // Frontend in kind
		"https://*.apps-crc.testing", // CRC routes
	}
}

func splitAndTrim(s string) []string {
	parts := []string{}
	for _, p := range splitString(s, ",") {
		if trimmed := trimString(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
