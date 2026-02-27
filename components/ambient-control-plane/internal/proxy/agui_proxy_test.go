package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/rs/zerolog"
)

func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func findFreePorts(t *testing.T, count int) (int, int) {
	t.Helper()
	start := 19200
	for s := start; s < 19900; s++ {
		allFree := true
		for p := s; p < s+count; p++ {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
			if err != nil {
				allFree = false
				break
			}
			ln.Close()
		}
		if allFree {
			return s, s + count - 1
		}
	}
	t.Fatal("could not find enough free ports")
	return 0, 0
}

func testProxyWithManager(t *testing.T) (*AGUIProxy, *process.Manager, int) {
	t.Helper()
	portStart, portEnd := findFreePorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	proxyPort := findFreePort(t)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	p := NewAGUIProxy(proxyAddr, "", mgr, logger)

	return p, mgr, proxyPort
}

func TestParseSessionPath(t *testing.T) {
	p := &AGUIProxy{}

	tests := []struct {
		path      string
		sessionID string
		suffix    string
	}{
		{"/sessions/abc-123/agui/events", "abc-123", "/agui/events"},
		{"/sessions/abc-123/agui/run", "abc-123", "/agui/run"},
		{"/sessions/abc-123/agui/interrupt", "abc-123", "/agui/interrupt"},
		{"/sessions/abc-123/health", "abc-123", "/health"},
		{"/sessions/abc-123/mcp/status", "abc-123", "/mcp/status"},
		{"/sessions/abc-123/repos/status", "abc-123", "/repos/status"},
		{"/sessions/", "", ""},
		{"/sessions/abc-123", "abc-123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			sid, suf := p.parseSessionPath(tt.path)
			if sid != tt.sessionID {
				t.Errorf("sessionID: got %q, want %q", sid, tt.sessionID)
			}
			if suf != tt.suffix {
				t.Errorf("suffix: got %q, want %q", suf, tt.suffix)
			}
		})
	}
}

func TestProxyUnknownSession(t *testing.T) {
	p, _, proxyPort := testProxyWithManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/sessions/nonexistent/agui/events", proxyPort))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "session not found" {
		t.Errorf("expected 'session not found', got %q", body["error"])
	}
}

func TestProxyHealthEndpoint(t *testing.T) {
	p, _, proxyPort := testProxyWithManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", proxyPort))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestProxyCORSHeaders(t *testing.T) {
	p, _, proxyPort := testProxyWithManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("http://127.0.0.1:%d/sessions/test/agui/events", proxyPort), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", resp.StatusCode)
	}

	origin := resp.Header.Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected CORS origin 'http://localhost:3000', got %q", origin)
	}

	methods := resp.Header.Get("Access-Control-Allow-Methods")
	if !strings.Contains(methods, "GET") || !strings.Contains(methods, "POST") {
		t.Errorf("expected GET,POST in Allow-Methods, got %q", methods)
	}
}

func TestProxyForwardsToRunner(t *testing.T) {
	fakeRunner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		case "/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"run":"started"}`))
		case "/interrupt":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"interrupted":true}`))
		case "/mcp/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"mcp":"connected"}`))
		case "/repos/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"repos":"cloned"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fakeRunner.Close()

	fakePort := fakeRunner.Listener.Addr().(*net.TCPAddr).Port

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	portStart, portEnd := findFreePorts(t, 5)
	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	rp, err := mgr.Spawn(ctx, "test-session", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	originalPort := rp.Port
	rp.Port = fakePort

	proxyPort := findFreePort(t)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	p := NewAGUIProxy(proxyAddr, "", mgr, logger)

	pCtx, pCancel := context.WithCancel(context.Background())
	defer pCancel()
	go p.Start(pCtx)
	time.Sleep(200 * time.Millisecond)

	tests := []struct {
		name     string
		path     string
		method   string
		expected string
	}{
		{"health", "/sessions/test-session/health", "GET", "healthy"},
		{"run", "/sessions/test-session/agui/run", "POST", "started"},
		{"interrupt", "/sessions/test-session/agui/interrupt", "POST", "interrupted"},
		{"mcp-status", "/sessions/test-session/mcp/status", "GET", "connected"},
		{"repos-status", "/sessions/test-session/repos/status", "GET", "cloned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				req, _ = http.NewRequest(tt.method, fmt.Sprintf("http://127.0.0.1:%d%s", proxyPort, tt.path), strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, fmt.Sprintf("http://127.0.0.1:%d%s", proxyPort, tt.path), nil)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.expected) {
				t.Errorf("response body %q does not contain %q", string(body), tt.expected)
			}
		})
	}

	rp.Port = originalPort
	mgr.Shutdown(context.Background())
}

func TestProxySSEStreaming(t *testing.T) {
	fakeRunner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		events := []string{
			"data: {\"type\":\"RUN_STARTED\"}\n\n",
			"data: {\"type\":\"TEXT_MESSAGE_START\"}\n\n",
			"data: {\"type\":\"RUN_FINISHED\"}\n\n",
		}

		for _, evt := range events {
			fmt.Fprint(w, evt)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer fakeRunner.Close()

	fakePort := fakeRunner.Listener.Addr().(*net.TCPAddr).Port

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	portStart, portEnd := findFreePorts(t, 5)
	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	rp, err := mgr.Spawn(ctx, "sse-session", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	originalPort := rp.Port
	rp.Port = fakePort

	proxyPort := findFreePort(t)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	p := NewAGUIProxy(proxyAddr, "", mgr, logger)

	pCtx, pCancel := context.WithCancel(context.Background())
	defer pCancel()
	go p.Start(pCtx)
	time.Sleep(200 * time.Millisecond)

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/sessions/sse-session/agui/events", proxyPort), nil)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SSE request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream content type, got %q", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "RUN_STARTED") {
		t.Error("expected RUN_STARTED in SSE stream")
	}
	if !strings.Contains(bodyStr, "RUN_FINISHED") {
		t.Error("expected RUN_FINISHED in SSE stream")
	}

	rp.Port = originalPort
	mgr.Shutdown(context.Background())
}

func TestProxyBadRequest(t *testing.T) {
	p, _, proxyPort := testProxyWithManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/sessions/", proxyPort))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProxyUnknownEndpoint(t *testing.T) {
	portStart, portEnd := findFreePorts(t, 5)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	_, err := mgr.Spawn(ctx, "endpoint-test", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	proxyPort := findFreePort(t)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	p := NewAGUIProxy(proxyAddr, "", mgr, logger)

	pCtx, pCancel := context.WithCancel(context.Background())
	defer pCancel()
	go p.Start(pCtx)
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/sessions/endpoint-test/unknown/path", proxyPort))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if !strings.Contains(body["error"], "unknown endpoint") {
		t.Errorf("expected 'unknown endpoint' error, got %q", body["error"])
	}

	mgr.Shutdown(context.Background())
}

func TestProxyShutdown(t *testing.T) {
	p, _, proxyPort := testProxyWithManager(t)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- p.Start(ctx)
	}()
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", proxyPort))
	if err != nil {
		t.Fatalf("pre-shutdown request: %v", err)
	}
	resp.Body.Close()

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil error on shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timed out")
	}
}
