package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

const ambientAPITokenEnv = "AMBIENT_API_TOKEN"

var httpBypassPaths = map[string]bool{
	"/healthcheck": true,
	"/health":      true,
	"/metrics":     true,
}

func init() {
	token := os.Getenv(ambientAPITokenEnv)
	if token == "" {
		glog.Infof("Bearer token auth disabled: %s not set", ambientAPITokenEnv)
		return
	}
	glog.Infof("Bearer token auth enabled via %s", ambientAPITokenEnv)
	pkgserver.RegisterPreAuthMiddleware(BearerTokenAuth(token))
	pkgserver.RegisterPreAuthGRPCUnaryInterceptor(bearerTokenGRPCUnaryInterceptor(token))
	pkgserver.RegisterPreAuthGRPCStreamInterceptor(bearerTokenGRPCStreamInterceptor(token))
}

func extractBearerToken(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}

func BearerTokenAuth(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if httpBypassPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				glog.Warningf("HTTP auth failure: missing authorization header for %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			token, err := extractBearerToken(authHeader)
			if err != nil {
				glog.Warningf("HTTP auth failure: %v for %s %s from %s", err, r.Method, r.URL.Path, r.RemoteAddr)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				glog.Warningf("HTTP auth failure: invalid bearer token for %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
				http.Error(w, "invalid bearer token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
