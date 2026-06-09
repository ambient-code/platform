package middleware

import (
	"context"
	"os"
	"strings"

	"github.com/golang/glog"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
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

	// Register pre-auth gRPC interceptors here (not in bearer_token.go)
	// to guarantee configuredServiceAccount is already set. Go runs init
	// functions in source-file alphabetical order within a package, so
	// bearer_token.go's init would execute before this file's init,
	// seeing an empty service account and skipping registration.
	token := os.Getenv("AMBIENT_API_TOKEN")
	if token == "" && configuredServiceAccount == "" {
		glog.Infof("Service token auth disabled: neither AMBIENT_API_TOKEN nor GRPC_SERVICE_ACCOUNT set")
		return
	}
	if token != "" {
		glog.Infof("Service token auth enabled via AMBIENT_API_TOKEN (gRPC only)")
	}
	if configuredServiceAccount != "" {
		glog.Infof("OIDC service account username: %s", configuredServiceAccount)
	}
	pkgserver.RegisterPreAuthGRPCUnaryInterceptor(bearerTokenGRPCUnaryInterceptor(token, configuredServiceAccount))
	pkgserver.RegisterPreAuthGRPCStreamInterceptor(bearerTokenGRPCStreamInterceptor(token, configuredServiceAccount))
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
