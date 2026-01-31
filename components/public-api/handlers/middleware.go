package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyProject is the context key for the project name
	ContextKeyProject = "project"
	// ContextKeyToken is the context key for the bearer token
	ContextKeyToken = "token"
)

// AuthMiddleware validates the token and extracts project information
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header or X-Forwarded-Access-Token
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid authorization"})
			c.Abort()
			return
		}

		// Store token in context for proxying to backend
		c.Set(ContextKeyToken, token)

		// Extract project from header or token
		project := extractProject(c, token)
		if project == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project required. Set X-Ambient-Project header or use a project-scoped access key."})
			c.Abort()
			return
		}

		c.Set(ContextKeyProject, project)
		c.Next()
	}
}

// extractToken extracts the bearer token from the request
func extractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try X-Forwarded-Access-Token (from OAuth proxy)
	if token := c.GetHeader("X-Forwarded-Access-Token"); token != "" {
		return token
	}

	return ""
}

// extractProject extracts the project from header or JWT token.
// SECURITY: If both header and token specify a project, they must match.
// This prevents routing attacks where an attacker forges a token with a different namespace.
func extractProject(c *gin.Context, token string) string {
	headerProject := c.GetHeader("X-Ambient-Project")
	tokenProject := extractProjectFromToken(token)

	// SECURITY: If both header and token specify a project, they must match
	// This prevents an attacker from using a forged token to route to a different project
	if headerProject != "" && tokenProject != "" && headerProject != tokenProject {
		log.Printf("SECURITY: Project mismatch - header=%s token=%s (possible routing attack)", headerProject, tokenProject)
		return "" // Force authentication failure
	}

	// Prefer explicit header
	if headerProject != "" {
		return headerProject
	}

	if tokenProject != "" {
		log.Printf("Extracted project %s from ServiceAccount token for routing (backend will validate)", tokenProject)
	}
	return tokenProject
}

// extractProjectFromToken extracts the project namespace from a ServiceAccount JWT token.
// ServiceAccount tokens have subject like: system:serviceaccount:<namespace>:<sa-name>
func extractProjectFromToken(token string) string {
	subject := extractJWTSubject(token)
	if strings.HasPrefix(subject, "system:serviceaccount:") {
		parts := strings.Split(subject, ":")
		if len(parts) >= 3 {
			return parts[2] // namespace is the project
		}
	}
	return ""
}

// extractJWTSubject extracts the 'sub' claim from a JWT WITHOUT validating signature.
//
// ========================================
// SECURITY WARNING - READ BEFORE MODIFYING
// ========================================
//
// This function does NOT validate the JWT signature. This is ONLY safe because:
//
// 1. PURPOSE: Used exclusively for routing (extracting project namespace from ServiceAccount tokens)
// 2. NO TRUST: The extracted value is NEVER used for authorization decisions
// 3. BACKEND VALIDATES: The Go backend performs FULL token validation including:
//    - Signature verification against K8s API server public keys
//    - Expiration checking
//    - RBAC enforcement via SelfSubjectAccessReview
// 4. FAIL-SAFE: If the token is invalid/forged, the backend rejects it with 401/403
//
// DO NOT use this function's output for:
// - Access control decisions
// - User identity verification
// - Any security-sensitive operations
//
// The only safe use is: routing requests to the correct project namespace,
// where the backend will validate the token before returning any data.
// ========================================
func extractJWTSubject(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try with standard base64
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	if sub, ok := claims["sub"].(string); ok {
		return sub
	}

	return ""
}

// GetProject returns the project from context
func GetProject(c *gin.Context) string {
	if project, exists := c.Get(ContextKeyProject); exists {
		if proj, ok := project.(string); ok {
			return proj
		}
	}
	return ""
}

// GetToken returns the token from context
func GetToken(c *gin.Context) string {
	if token, exists := c.Get(ContextKeyToken); exists {
		if tok, ok := token.(string); ok {
			return tok
		}
	}
	return ""
}

// LoggingMiddleware logs requests with redacted tokens.
// Follows the same redaction pattern as backend server/server.go:22-34
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Redact sensitive information from path
		path := redactSensitiveParams(c.Request.URL.Path, c.Request.URL.RawQuery)

		// Log request
		log.Printf("%s %s project=%s", c.Request.Method, path, GetProject(c))
		c.Next()
		// Log response status
		log.Printf("%s %s status=%d", c.Request.Method, path, c.Writer.Status())
	}
}

// redactSensitiveParams removes tokens and other sensitive data from URLs before logging.
// SECURITY: This prevents accidental token exposure in logs.
func redactSensitiveParams(path, rawQuery string) string {
	if rawQuery == "" {
		return path
	}

	// Redact known sensitive query parameters
	redactedQuery := rawQuery

	// Redact token parameter
	if strings.Contains(rawQuery, "token=") {
		redactedQuery = redactQueryParam(redactedQuery, "token")
	}

	// Redact access_token parameter (OAuth)
	if strings.Contains(rawQuery, "access_token=") {
		redactedQuery = redactQueryParam(redactedQuery, "access_token")
	}

	// Redact api_key parameter
	if strings.Contains(rawQuery, "api_key=") {
		redactedQuery = redactQueryParam(redactedQuery, "api_key")
	}

	if redactedQuery != rawQuery {
		return path + "?" + redactedQuery
	}
	return path + "?" + rawQuery
}

// redactQueryParam replaces the value of a query parameter with [REDACTED]
func redactQueryParam(query, param string) string {
	// Find the parameter
	prefix := param + "="
	idx := strings.Index(query, prefix)
	if idx == -1 {
		return query
	}

	// Find the end of the value (next & or end of string)
	valueStart := idx + len(prefix)
	valueEnd := strings.Index(query[valueStart:], "&")
	if valueEnd == -1 {
		// Parameter is at the end
		return query[:valueStart] + "[REDACTED]"
	}
	// Parameter is in the middle
	return query[:valueStart] + "[REDACTED]" + query[valueStart+valueEnd:]
}
