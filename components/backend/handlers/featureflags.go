// Package handlers: feature flag helpers for use inside HTTP handlers.
// Backed by the optional Unleash integration (see featureflags package).
// When Unleash is not configured, all flags are disabled.

package handlers

import (
	"ambient-code-backend/featureflags"

	"github.com/gin-gonic/gin"
)

// FeatureEnabled returns true if the named feature flag is enabled (global check).
// When Unleash is not configured, returns false. Safe to call from any handler.
func FeatureEnabled(flagName string) bool {
	return featureflags.IsEnabled(flagName)
}

// FeatureEnabledForRequest returns true if the flag is enabled for the current request.
// Uses forwarded user ID, client IP, and optional session for Unleash strategies (e.g. userId, remoteAddress).
// When Unleash is not configured, returns false.
func FeatureEnabledForRequest(c *gin.Context, flagName string) bool {
	userID := c.GetString("userID")
	sessionID := c.GetHeader("X-Session-Id") // optional
	remoteAddr := c.ClientIP()
	return featureflags.IsEnabledWithContext(flagName, userID, sessionID, remoteAddr)
}
