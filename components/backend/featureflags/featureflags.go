// Package featureflags provides optional Unleash-backed feature flag checks for the backend.
// When UNLEASH_URL and UNLEASH_CLIENT_KEY are not set, all flags are disabled (IsEnabled returns false).
package featureflags

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Unleash/unleash-go-sdk/v5"
	unleashContext "github.com/Unleash/unleash-go-sdk/v5/context"
)

const appName = "ambient-code-backend"

var initialized bool

// Init initializes the Unleash client when UNLEASH_URL and UNLEASH_CLIENT_KEY are set.
// Safe to call multiple times; only initializes once when config is present.
// Call from main after loading env and before starting the server.
func Init() {
	url := strings.TrimSpace(os.Getenv("UNLEASH_URL"))
	clientKey := strings.TrimSpace(os.Getenv("UNLEASH_CLIENT_KEY"))
	if url == "" || clientKey == "" {
		return
	}
	// Ensure URL has a trailing slash for the SDK
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	unleash.Initialize(
		unleash.WithAppName(appName),
		unleash.WithUrl(url),
		unleash.WithCustomHeaders(http.Header{"Authorization": {clientKey}}),
	)
	initialized = true
	log.Printf("Unleash feature flags enabled (url=%s)", strings.TrimSuffix(url, "/"))
}

// IsEnabled returns true if the named feature flag is enabled.
// When Unleash is not configured, returns false. Safe to call from any handler.
func IsEnabled(flagName string) bool {
	if !initialized {
		return false
	}
	return unleash.IsEnabled(flagName)
}

// IsEnabledWithContext returns true if the flag is enabled for the given user context.
// Use for strategies that depend on userId, sessionId, or remoteAddress.
// When Unleash is not configured, returns false.
func IsEnabledWithContext(flagName string, userId, sessionId, remoteAddress string) bool {
	if !initialized {
		return false
	}
	ctx := unleashContext.Context{
		UserId:        userId,
		SessionId:     sessionId,
		RemoteAddress: remoteAddress,
	}
	return unleash.IsEnabled(flagName, unleash.WithContext(ctx))
}
