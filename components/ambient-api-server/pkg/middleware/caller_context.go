package middleware

import (
	"context"
	"os"
	"strings"
)

type callerTypeKey struct{}

const (
	CallerTypeService = "service"
	CallerTypeUser    = "user"
)

// configuredServiceAccount is the OIDC username of the platform's service
// account, read once from the GRPC_SERVICE_ACCOUNT env var at init time.
// Both the gRPC interceptor and the HTTP RBAC middleware use this value
// to detect service callers.
var configuredServiceAccount string

const keycloakServiceAccountPrefix = "service-account-"

func init() {
	configuredServiceAccount = strings.TrimSpace(os.Getenv("GRPC_SERVICE_ACCOUNT"))
}

// WithCallerType sets the caller type (service or user) on the context.
func WithCallerType(ctx context.Context, callerType string) context.Context {
	return context.WithValue(ctx, callerTypeKey{}, callerType)
}

// IsServiceCaller returns true if the context was tagged as a service caller.
func IsServiceCaller(ctx context.Context) bool {
	v, _ := ctx.Value(callerTypeKey{}).(string)
	return v == CallerTypeService
}

// IsConfiguredServiceAccount reports whether jwtUsername matches the
// platform's configured service account (exact or Keycloak-prefixed).
func IsConfiguredServiceAccount(jwtUsername string) bool {
	return isServiceAccount(jwtUsername, configuredServiceAccount)
}

// ConfiguredServiceAccountUsername returns the configured service account
// username (from GRPC_SERVICE_ACCOUNT env var). Empty if not configured.
func ConfiguredServiceAccountUsername() string {
	return configuredServiceAccount
}

func isServiceAccount(jwtUsername, configured string) bool {
	if configured == "" {
		return false
	}
	return jwtUsername == configured ||
		jwtUsername == keycloakServiceAccountPrefix+configured
}
