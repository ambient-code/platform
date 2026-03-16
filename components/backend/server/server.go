// Package server provides HTTP server setup, middleware, and routing configuration.
package server

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		// Fallback: if userID is still empty, verify the Bearer token via
		// TokenReview to securely resolve the ServiceAccount identity, then
		// read the created-by-user-id annotation. This enables API key-
		// authenticated requests to inherit the creating user's identity
		// so that integration credentials (GitHub, Jira, GitLab) are accessible.
		if c.GetString("userID") == "" && K8sClient != nil {
			if ns, saName, ok := resolveServiceAccountFromToken(c); ok {
				sa, err := K8sClient.CoreV1().ServiceAccounts(ns).Get(c.Request.Context(), saName, v1.GetOptions{})
				if err == nil && sa.Annotations != nil {
					if uid := sa.Annotations["ambient-code.io/created-by-user-id"]; uid != "" {
						c.Set("userID", uid)
					}
				}
			}
		}

		c.Next()
	}
}

// resolveServiceAccountFromToken verifies the Bearer token via K8s TokenReview
// and extracts the ServiceAccount namespace and name from the authenticated identity.
// Returns (namespace, saName, true) when verified, otherwise ("","",false).
func resolveServiceAccountFromToken(c *gin.Context) (string, string, bool) {
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", "", false
	}

	tr := &authnv1.TokenReview{Spec: authnv1.TokenReviewSpec{Token: token}}
	rv, err := K8sClient.AuthenticationV1().TokenReviews().Create(c.Request.Context(), tr, v1.CreateOptions{})
	if err != nil || !rv.Status.Authenticated || rv.Status.Error != "" {
		return "", "", false
	}

	subj := strings.TrimSpace(rv.Status.User.Username)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(subj, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(subj, prefix)
	segs := strings.SplitN(rest, ":", 2)
	if len(segs) != 2 {
		return "", "", false
	}
	return segs[0], segs[1], true
}

// ExtractServiceAccountFromAuth extracts namespace and ServiceAccount name
// from the X-Remote-User header (OpenShift OAuth proxy format).
// Returns (namespace, saName, true) when a SA subject is present, otherwise ("","",false).
func ExtractServiceAccountFromAuth(c *gin.Context) (string, string, bool) {
	if remoteUser := c.GetHeader("X-Remote-User"); remoteUser != "" {
		const prefix = "system:serviceaccount:"
		if strings.HasPrefix(remoteUser, prefix) {
			parts := strings.SplitN(strings.TrimPrefix(remoteUser, prefix), ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], true
			}
		}
	}
	return "", "", false
}
