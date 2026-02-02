// Package server provides HTTP server setup, middleware, and routing configuration.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RouterFunc is a function that can register routes on a Gin router
type RouterFunc func(r *gin.Engine)

// Run starts the server with the provided route registration function
func Run(registerRoutes RouterFunc) error {
	// Setup Gin router with custom logger that redacts tokens
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Redact token from query string
		path := param.Path
		if strings.Contains(param.Request.URL.RawQuery, "token=") {
			path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
		}

		// Redact Authorization header from logs
		authHeader := "[none]"
		if auth := param.Request.Header.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				authHeader = "Bearer [REDACTED]"
			} else {
				authHeader = "[REDACTED]"
			}
		}

		return fmt.Sprintf("[GIN] %s | %3d | %s | %s | Auth: %s\n",
			param.Method,
			param.StatusCode,
			param.ClientIP,
			path,
			authHeader,
		)
	}))

	// Middleware to populate user context from forwarded headers
	r.Use(forwardedIdentityMiddleware())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Register routes
	registerRoutes(r)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Using namespace: %s", Namespace)

	if err := r.Run(":" + port); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

// sanitizeUserID converts userID to a valid Kubernetes Secret data key
// K8s Secret keys must match regex: [-._a-zA-Z0-9]+
// Follows cert-manager's sanitization pattern for consistent, secure key generation
//
// Handles common username formats:
// - OpenShift: kube:admin, system:serviceaccount:ns:sa → kube-admin, system-serviceaccount-ns-sa
// - Email: user@company.com → user-company.com
// - LDAP: CN=User,OU=dept,DC=com → CN-User-OU-dept-DC-com
// - SSO: domain\username → domain-username
// - Spaces: "First Last" → "First-Last"
//
// Security: Only replaces characters, never interprets them (no injection risk)
func sanitizeUserID(userID string) string {
	if userID == "" {
		return ""
	}

	// Limit length to K8s maximum (253 chars for Secret keys)
	if len(userID) > 253 {
		// Take first 253 chars after sanitization to preserve readability
		userID = userID[:253]
	}

	// Replace all invalid characters with hyphen
	// Valid: a-z, A-Z, 0-9, hyphen, underscore, period
	// This approach is secure because we're doing character replacement,
	// not parsing or interpreting the input (no injection vectors)
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		// Replace all invalid chars with hyphen (like cert-manager)
		// Common replacements:
		// : (kube:admin) → -
		// @ (email) → -
		// / \ (paths, Windows) → -
		// , (LDAP DN) → -
		// space → -
		return '-'
	}, userID)

	// Ensure doesn't start/end with hyphen (K8s label-like constraint for consistency)
	sanitized = strings.Trim(sanitized, "-")

	// Collapse multiple consecutive hyphens to single hyphen for readability
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	return sanitized
}

// forwardedIdentityMiddleware populates Gin context from common OAuth proxy headers
func forwardedIdentityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.GetHeader("X-Forwarded-User"); v != "" {
			// Sanitize userID to make it valid for K8s Secret keys
			// Example: "kube:admin" becomes "kube-admin"
			c.Set("userID", sanitizeUserID(v))
			// Keep original for display purposes
			c.Set("userIDOriginal", v)
		}
		// Prefer preferred username; fallback to user id
		name := c.GetHeader("X-Forwarded-Preferred-Username")
		if name == "" {
			name = c.GetHeader("X-Forwarded-User")
		}
		if name != "" {
			c.Set("userName", name)
		}
		if v := c.GetHeader("X-Forwarded-Email"); v != "" {
			c.Set("userEmail", v)
		}
		if v := c.GetHeader("X-Forwarded-Groups"); v != "" {
			c.Set("userGroups", strings.Split(v, ","))
		}
		// Also expose access token if present
		auth := c.GetHeader("Authorization")
		if auth != "" {
			c.Set("authorizationHeader", auth)
		}
		if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
			c.Set("forwardedAccessToken", v)
		}
		c.Next()
	}
}

// RunContentService starts the server in content service mode with graceful shutdown
func RunContentService(registerContentRoutes RouterFunc) error {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		path := param.Path
		if strings.Contains(param.Request.URL.RawQuery, "token=") {
			path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
		}
		return fmt.Sprintf("[GIN] %s | %3d | %s | %s\n",
			param.Method,
			param.StatusCode,
			param.ClientIP,
			path,
		)
	}))

	// Register content service routes
	registerContentRoutes(r)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server for graceful shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Channel to receive shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Content service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Content service listen error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-quit
	log.Printf("Content service received signal %v, shutting down gracefully...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Content service forced to shutdown: %v", err)
		return err
	}

	log.Println("Content service shutdown complete")
	return nil
}
