// Package proxy implements a generic reverse-proxy for backend routes not natively
// served by the ambient-api-server. Any request whose path does NOT start with
// "/api/ambient/" is forwarded verbatim to BACKEND_URL (default http://localhost:8080).
//
// This satisfies the "Generic Proxy Surface" in the ambient-model spec: SDK/CLI
// clients reach the full backend surface through a single authenticated endpoint
// without requiring every backend route to be natively implemented.
package proxy

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

// backendHTTPClient is used for all proxy requests to the backend.
// Separate from the session runner client so timeouts can be tuned independently.
var backendHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	},
	Timeout: 60 * time.Second,
}

func init() {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	pkgserver.RegisterPreAuthMiddleware(newBackendProxy(backendURL))
}

// newBackendProxy returns a middleware that forwards non-ambient requests to backendURL.
// Exported so tests can call it directly without going through the init() global.
func newBackendProxy(backendURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isNativePath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			proxyRequest(w, r, backendURL)
		})
	}
}

// isNativePath returns true for paths handled natively by ambient-api-server.
func isNativePath(p string) bool {
	return strings.HasPrefix(p, "/api/ambient/") ||
		p == "/api/ambient" ||
		p == "/metrics" ||
		p == "/favicon.ico"
}

// proxyRequest forwards r verbatim to backendURL+r.URL.Path preserving all
// headers, query string, and body. The response is written back unchanged.
func proxyRequest(w http.ResponseWriter, r *http.Request, backendURL string) {
	target, err := url.Parse(backendURL)
	if err != nil {
		glog.Errorf("proxy: invalid backend URL %q: %v", backendURL, err)
		http.Error(w, "proxy configuration error", http.StatusInternalServerError)
		return
	}

	// Build the upstream URL: backend scheme+host + original path + query.
	upstreamURL := *target
	upstreamURL.Path = r.URL.Path
	upstreamURL.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		glog.Errorf("proxy: build request for %s: %v", upstreamURL.String(), err)
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}

	// Copy all headers from the original request (including Authorization).
	for k, vals := range r.Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	resp, err := backendHTTPClient.Do(req)
	if err != nil {
		glog.Warningf("proxy: backend %s unreachable for %s %s: %v",
			backendURL, r.Method, r.URL.Path, err)
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy all response headers.
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// NewBackendProxyMiddleware is the exported constructor used in tests.
// Tests call this directly instead of relying on init() and env vars.
func NewBackendProxyMiddleware(backendURL string) func(http.Handler) http.Handler {
	return newBackendProxy(backendURL)
}
