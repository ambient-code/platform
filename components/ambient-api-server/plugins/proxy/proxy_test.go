package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/proxy"
)

// buildHandler wraps a mock backend with the proxy middleware and a sentinel native handler.
func buildHandler(backendURL string) http.Handler {
	nativeHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("native"))
	})
	mw := proxy.NewBackendProxyMiddleware(backendURL)
	return mw(nativeHandler)
}

// ---------------------------------------------------------------------------
// Native paths pass through (not forwarded to backend)
// ---------------------------------------------------------------------------

func TestNativePath_ApiAmbient_PassesThrough(t *testing.T) {
	handler := buildHandler("http://does-not-exist.invalid")
	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || rr.Body.String() != "native" {
		t.Errorf("expected native handler, got %d: %s", rr.Code, rr.Body)
	}
}

func TestNativePath_ApiAmbientExact_PassesThrough(t *testing.T) {
	handler := buildHandler("http://does-not-exist.invalid")
	req := httptest.NewRequest(http.MethodGet, "/api/ambient", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || rr.Body.String() != "native" {
		t.Errorf("expected native handler, got %d: %s", rr.Code, rr.Body)
	}
}

func TestNativePath_Metrics_PassesThrough(t *testing.T) {
	handler := buildHandler("http://does-not-exist.invalid")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || rr.Body.String() != "native" {
		t.Errorf("expected native handler for /metrics, got %d: %s", rr.Code, rr.Body)
	}
}

// ---------------------------------------------------------------------------
// Non-native paths are forwarded to the backend
// ---------------------------------------------------------------------------

func TestProxyPath_ForwardsToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("from-backend"))
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj-1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != "from-backend" {
		t.Errorf("expected backend body, got %s", rr.Body)
	}
}

func TestProxyPath_PreservesMethod(t *testing.T) {
	var capturedMethod string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-1/sessions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, backend saw %s", capturedMethod)
	}
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestProxyPath_ForwardsHeaders(t *testing.T) {
	var capturedAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedAuth != "Bearer test-token" {
		t.Errorf("expected auth header forwarded, got %q", capturedAuth)
	}
}

func TestProxyPath_PreservesQueryString(t *testing.T) {
	var capturedQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj-1/sessions?page=2&size=10", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedQuery != "page=2&size=10" {
		t.Errorf("expected query preserved, got %q", capturedQuery)
	}
}

func TestProxyPath_ForwardsBody(t *testing.T) {
	var capturedBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"test"}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedBody != `{"user":"test"}` {
		t.Errorf("expected body forwarded, got %q", capturedBody)
	}
}

func TestProxyPath_BackendDown_Returns502(t *testing.T) {
	handler := buildHandler("http://127.0.0.1:1") // nothing listening on port 1
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rr.Code)
	}
}

func TestProxyPath_ResponseHeadersCopied(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "proxy-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := buildHandler(backend.URL)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Custom-Header") != "proxy-value" {
		t.Errorf("expected response header copied, got %q", rr.Header().Get("X-Custom-Header"))
	}
}
