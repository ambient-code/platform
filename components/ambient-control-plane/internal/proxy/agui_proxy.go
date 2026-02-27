package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/rs/zerolog"
)

type AGUIProxy struct {
	listenAddr string
	corsOrigin string
	manager    *process.Manager
	logger     zerolog.Logger
	server     *http.Server
}

func NewAGUIProxy(listenAddr, corsOrigin string, manager *process.Manager, logger zerolog.Logger) *AGUIProxy {
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000"
	}
	return &AGUIProxy{
		listenAddr: listenAddr,
		corsOrigin: corsOrigin,
		manager:    manager,
		logger:     logger.With().Str("component", "agui-proxy").Logger(),
	}
}

func (p *AGUIProxy) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/sessions/", p.handleRequest)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	p.server = &http.Server{
		Addr:         p.listenAddr,
		Handler:      p.corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	p.logger.Info().Str("addr", p.listenAddr).Msg("starting AG-UI proxy")

	go func() {
		<-ctx.Done()
		p.server.Shutdown(context.Background())
	}()

	err := p.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (p *AGUIProxy) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", p.corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *AGUIProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	sessionID, suffix := p.parseSessionPath(r.URL.Path)
	if sessionID == "" {
		p.writeError(w, http.StatusBadRequest, "invalid path: missing session ID")
		return
	}

	rp, ok := p.manager.GetProcess(sessionID)
	if !ok {
		p.writeError(w, http.StatusNotFound, "session not found")
		return
	}

	targetURL := fmt.Sprintf("http://127.0.0.1:%d", rp.Port)

	switch {
	case suffix == "/agui/events" || suffix == "/agui/events/":
		p.proxySSE(w, r, rp.Port, targetURL)
	case suffix == "/agui/run" || suffix == "/agui/run/":
		p.proxyRequest(w, r, targetURL, "/")
	case suffix == "/agui/interrupt" || suffix == "/agui/interrupt/":
		p.proxyRequest(w, r, targetURL, "/interrupt")
	case suffix == "/health" || suffix == "/health/":
		p.proxyRequest(w, r, targetURL, "/health")
	case suffix == "/mcp/status" || suffix == "/mcp/status/":
		p.proxyRequest(w, r, targetURL, "/mcp/status")
	case suffix == "/repos/status" || suffix == "/repos/status/":
		p.proxyRequest(w, r, targetURL, "/repos/status")
	default:
		p.writeError(w, http.StatusNotFound, "unknown endpoint: "+suffix)
	}
}

func (p *AGUIProxy) proxySSE(w http.ResponseWriter, r *http.Request, port int, targetURL string) {
	target, _ := url.Parse(targetURL)

	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = "/"
		req.Header.Set("Accept", "text/event-stream")
		req.Host = target.Host
	}

	reverseProxy := &httputil.ReverseProxy{
		Director:      director,
		FlushInterval: -1,
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Set("Content-Type", "text/event-stream")
			resp.Header.Set("Cache-Control", "no-cache")
			resp.Header.Set("X-Accel-Buffering", "no")
			resp.Header.Set("Connection", "keep-alive")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Warn().Err(err).Int("port", port).Msg("SSE proxy error")
			p.writeError(w, http.StatusBadGateway, "runner not reachable")
		},
	}

	reverseProxy.ServeHTTP(w, r)
}

func (p *AGUIProxy) proxyRequest(w http.ResponseWriter, r *http.Request, targetURL, targetPath string) {
	target, _ := url.Parse(targetURL)

	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = targetPath
		req.Host = target.Host
	}

	reverseProxy := &httputil.ReverseProxy{
		Director: director,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.logger.Warn().Err(err).Str("target", targetURL+targetPath).Msg("proxy error")
			p.writeError(w, http.StatusBadGateway, "runner not reachable")
		},
	}

	reverseProxy.ServeHTTP(w, r)
}

func (p *AGUIProxy) parseSessionPath(path string) (sessionID, suffix string) {
	path = strings.TrimPrefix(path, "/sessions/")
	if path == "" {
		return "", ""
	}

	idx := strings.Index(path, "/")
	if idx < 0 {
		return path, ""
	}

	return path[:idx], path[idx:]
}

func (p *AGUIProxy) writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
