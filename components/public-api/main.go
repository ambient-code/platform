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
	"log"
	"os"

	"ambient-code-public-api/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin mode from environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Recovery middleware
	r.Use(gin.Recovery())

	// Health endpoint (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Readiness endpoint
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	// v1 API routes
	// IMPORTANT: AuthMiddleware must run BEFORE LoggingMiddleware
	// to ensure we only log authenticated requests with valid project context
	v1 := r.Group("/v1")
	v1.Use(handlers.AuthMiddleware())
	v1.Use(handlers.LoggingMiddleware())
	{
		// Sessions
		v1.GET("/sessions", handlers.ListSessions)
		v1.POST("/sessions", handlers.CreateSession)
		v1.GET("/sessions/:id", handlers.GetSession)
		v1.DELETE("/sessions/:id", handlers.DeleteSession)
	}

	// Get port from environment or default to 8081
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting Public API server on :%s", port)
	log.Printf("Backend URL: %s", handlers.BackendURL)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
