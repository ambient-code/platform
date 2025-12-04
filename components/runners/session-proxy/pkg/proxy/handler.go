// Package proxy implements the streaming exec API for workspace containers.
//
// The proxy runs as a sidecar in the runner pod and handles kubectl exec
// operations to workspace pods, providing streaming command execution over HTTP.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Rate limiting configuration (can be overridden via environment variables)
var (
	// MaxConcurrentExecs limits the number of concurrent exec operations
	MaxConcurrentExecs = getEnvInt("MAX_CONCURRENT_EXECS", 10)
	// ExecTimeout is the default timeout for exec operations
	ExecTimeout = time.Duration(getEnvInt("EXEC_TIMEOUT_SECONDS", 600)) * time.Second
)

// getEnvInt reads an integer from environment variable with a default fallback
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			return i
		}
	}
	return defaultVal
}

// Config holds proxy configuration
type Config struct {
	SessionName string
	Namespace   string
	ListenAddr  string
}

// ExecRequest is the request body for /exec
type ExecRequest struct {
	Command []string `json:"command"`
	Cwd     string   `json:"cwd,omitempty"`
}

// ExecResponse is sent as the final message after command completion
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

// SessionProxy implements the streaming exec API
type SessionProxy struct {
	config      Config
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config
	activeExecs int32 // Atomic counter for concurrent exec operations
}

// New creates a new SessionProxy
func New(config Config) (*SessionProxy, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &SessionProxy{
		config:     config,
		clientset:  clientset,
		restConfig: restConfig,
	}, nil
}

// Start starts the HTTP server
func (p *SessionProxy) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/exec", p.handleExec)
	mux.HandleFunc("/health", p.handleHealth)

	server := &http.Server{
		Addr:         p.config.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // No timeout for streaming responses
	}

	return server.ListenAndServe()
}

// handleHealth returns 200 OK for health checks
func (p *SessionProxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleExec handles streaming exec requests
func (p *SessionProxy) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting: check concurrent exec count
	current := atomic.AddInt32(&p.activeExecs, 1)
	defer atomic.AddInt32(&p.activeExecs, -1)

	if int(current) > MaxConcurrentExecs {
		http.Error(w, fmt.Sprintf("too many concurrent exec requests (max: %d)", MaxConcurrentExecs), http.StatusTooManyRequests)
		return
	}

	// Parse request
	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Command) == 0 {
		http.Error(w, "command is required", http.StatusBadRequest)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), ExecTimeout)
	defer cancel()

	// Find workspace pod by label
	workspacePod, err := p.findWorkspacePod(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find workspace pod: %v", err), http.StatusServiceUnavailable)
		return
	}

	log.Printf("Executing command in pod %s: %v", workspacePod, req.Command)

	// Build the command with optional cwd
	cmd := req.Command
	if req.Cwd != "" {
		// Wrap command to run in specified directory
		cmd = []string{"sh", "-c", fmt.Sprintf("cd %q && exec \"$@\"", req.Cwd), "--"}
		cmd = append(cmd, req.Command...)
	}

	// Set up streaming response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	// Flush headers immediately
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Execute command with streaming output (using context with timeout)
	exitCode, execErr := p.execInPod(ctx, workspacePod, cmd, w)

	// Send final response as JSON on a new line
	resp := ExecResponse{ExitCode: exitCode}
	if execErr != nil {
		resp.Error = execErr.Error()
	}

	// Write a delimiter and the final JSON response
	w.Write([]byte("\n---EXIT---\n"))
	json.NewEncoder(w).Encode(resp)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// findWorkspacePod finds the workspace pod for this session by label selector
func (p *SessionProxy) findWorkspacePod(ctx context.Context) (string, error) {
	labelSelector := fmt.Sprintf("session=%s,type=workspace", p.config.SessionName)

	pods, err := p.clientset.CoreV1().Pods(p.config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	// Find a running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no workspace pod found with selector %q", labelSelector)
	}

	return "", fmt.Errorf("workspace pod exists but is not running (phase: %s)", pods.Items[0].Status.Phase)
}

// execInPod executes a command in the workspace pod and streams output
func (p *SessionProxy) execInPod(ctx context.Context, podName string, cmd []string, output io.Writer) (int, error) {
	req := p.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(p.config.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "workspace",
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(p.restConfig, "POST", req.URL())
	if err != nil {
		return -1, fmt.Errorf("failed to create executor: %w", err)
	}

	// Create a streaming writer that flushes after each write
	streamWriter := &flushWriter{w: output}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: streamWriter,
		Stderr: streamWriter,
	})

	if err != nil {
		// Try to extract exit code from error
		if exitErr, ok := err.(interface{ ExitStatus() int }); ok {
			return exitErr.ExitStatus(), nil
		}
		return -1, err
	}

	return 0, nil
}

// flushWriter wraps an io.Writer and flushes after each write if possible
type flushWriter struct {
	w io.Writer
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if err != nil {
		return n, err
	}

	// Flush if the underlying writer supports it
	if f, ok := fw.w.(http.Flusher); ok {
		f.Flush()
	}

	return n, nil
}
