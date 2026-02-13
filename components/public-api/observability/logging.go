package observability

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Logger is the global structured logger (JSON format in production)
var Logger zerolog.Logger

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339

	// Use JSON format in production, pretty format in development
	if os.Getenv("GIN_MODE") == "debug" {
		Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		Logger = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Str("service", "public-api").
			Logger()
	}
}

// StructuredLoggingMiddleware returns a Gin middleware that logs requests in JSON format
func StructuredLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactSensitiveParams(c.Request.URL.RawQuery)

		// Process request
		c.Next()

		// Log after request completes
		latency := time.Since(start)
		status := c.Writer.Status()

		// Get project from context (set by auth middleware)
		project, _ := c.Get("project")
		projectStr, _ := project.(string)

		// Build log event
		event := Logger.Info()
		if status >= 400 {
			event = Logger.Warn()
		}
		if status >= 500 {
			event = Logger.Error()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Str("project", projectStr).
			Int("body_size", c.Writer.Size()).
			Msg("HTTP request")
	}
}

// redactSensitiveParams redacts sensitive query parameters
func redactSensitiveParams(query string) string {
	if query == "" {
		return ""
	}

	sensitiveParams := []string{"token", "access_token", "api_key", "apikey", "key", "secret"}
	result := query

	for _, param := range sensitiveParams {
		result = redactQueryParam(result, param)
	}

	return result
}

// redactQueryParam redacts a specific query parameter value
func redactQueryParam(query, param string) string {
	// Simple implementation - look for param= and redact until next & or end
	paramPrefix := param + "="
	idx := 0
	for {
		start := indexOf(query[idx:], paramPrefix)
		if start == -1 {
			break
		}
		start += idx
		valueStart := start + len(paramPrefix)
		valueEnd := indexOf(query[valueStart:], "&")
		if valueEnd == -1 {
			valueEnd = len(query)
		} else {
			valueEnd += valueStart
		}
		query = query[:valueStart] + "[REDACTED]" + query[valueEnd:]
		idx = valueStart + len("[REDACTED]")
	}
	return query
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
