package middleware

import (
	"net/http"

	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

const forwardedAccessTokenHeader = "X-Forwarded-Access-Token"

func init() {
	pkgserver.RegisterPreAuthMiddleware(ForwardedAccessToken)
}

func ForwardedAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if token := r.Header.Get(forwardedAccessTokenHeader); token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
		}
		next.ServeHTTP(w, r)
	})
}
