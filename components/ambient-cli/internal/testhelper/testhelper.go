// Package testhelper provides shared test utilities for CLI integration tests.
package testhelper

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const TestToken = "sha256~test-token-for-cli-unit-tests-only"
const TestProject = "test-project"

type Server struct {
	*httptest.Server
	Requests []*http.Request
	Mux      *http.ServeMux
}

func NewServer(t *testing.T) *Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Server{Server: srv, Mux: mux}
}

func (s *Server) Handle(pattern string, fn func(w http.ResponseWriter, r *http.Request)) {
	s.Mux.HandleFunc(pattern, fn)
}

func (s *Server) RespondJSON(t *testing.T, w http.ResponseWriter, status int, body interface{}) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func Configure(t *testing.T, serverURL string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("AMBIENT_CONFIG", path)
	t.Setenv("AMBIENT_API_URL", serverURL)
	t.Setenv("AMBIENT_TOKEN", TestToken)
	t.Setenv("AMBIENT_PROJECT", TestProject)

	cfg := &config.Config{
		APIUrl:      serverURL,
		AccessToken: TestToken,
		Project:     TestProject,
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save test config: %v", err)
	}
}

type Result struct {
	Stdout string
	Stderr string
	Err    error
}

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

func Run(t *testing.T, cmd *cobra.Command, args ...string) Result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	resetFlags(cmd)
	err := cmd.Execute()
	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}
