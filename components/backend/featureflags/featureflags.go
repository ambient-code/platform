// Package featureflags provides optional Unleash-backed feature flag checks for the backend.
// When UNLEASH_URL and UNLEASH_CLIENT_KEY are not set, all flags are disabled (IsEnabled returns false).
package featureflags

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/Unleash/unleash-go-sdk/v5"
	unleashContext "github.com/Unleash/unleash-go-sdk/v5/context"
)

const appName = "ambient-code-backend"

var initialized atomic.Bool

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
	initialized.Store(true)
	log.Printf("Unleash feature flags enabled (url=%s)", strings.TrimSuffix(url, "/"))
}

// IsEnabled returns true if the named feature flag is enabled.
// When Unleash is not configured, returns false. Safe to call from any handler.
func IsEnabled(flagName string) bool {
	if !initialized.Load() {
		return false
	}
	return unleash.IsEnabled(flagName)
}

// IsEnabledWithContext returns true if the flag is enabled for the given user context.
// Use for strategies that depend on userId, sessionId, or remoteAddress.
// When Unleash is not configured, returns false.
func IsEnabledWithContext(flagName string, userID, sessionID, remoteAddress string) bool {
	if !initialized.Load() {
		return false
	}
	ctx := unleashContext.Context{
		UserId:        userID,
		SessionId:     sessionID,
		RemoteAddress: remoteAddress,
	}
	return unleash.IsEnabled(flagName, unleash.WithContext(ctx))
}

// IsModelEnabled returns true if a model feature flag is enabled.
// Unlike IsEnabled, this returns true when Unleash is not configured,
// because models should be enabled by default (flags only disable).
func IsModelEnabled(flagName string) bool {
	if !initialized.Load() {
		return true
	}
	return unleash.IsEnabled(flagName, unleash.WithFallback(true))
}

// IsModelEnabledWithContext returns true if a model feature flag is enabled
// for the given user context. Returns true when Unleash is not configured.
func IsModelEnabledWithContext(flagName string, userID, sessionID, remoteAddress string) bool {
	if !initialized.Load() {
		return true
	}
	ctx := unleashContext.Context{
		UserId:        userID,
		SessionId:     sessionID,
		RemoteAddress: remoteAddress,
	}
	return unleash.IsEnabled(flagName, unleash.WithContext(ctx), unleash.WithFallback(true))
}
