package connection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
)

func writeConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("AMBIENT_CONFIG", path)
	t.Setenv("AMBIENT_TOKEN", "")
	t.Setenv("AMBIENT_PROJECT", "")
	t.Setenv("AMBIENT_API_URL", "")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientFromConfigEmptyToken(t *testing.T) {
	writeConfig(t, &config.Config{
		APIUrl:  "https://api.example.com",
		Project: "myproject",
	})
	_, err := NewClientFromConfig()
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClientFromConfigEmptyProject(t *testing.T) {
	writeConfig(t, &config.Config{
		APIUrl:      "https://api.example.com",
		AccessToken: "test-token-at-least-20chars",
	})
	_, err := NewClientFromConfig()
	if err == nil {
		t.Fatal("expected error for empty project")
	}
	if !strings.Contains(err.Error(), "no project set") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClientFromConfigMalformedURL(t *testing.T) {
	writeConfig(t, &config.Config{
		APIUrl:      "not-a-url",
		AccessToken: "test-token-at-least-20chars",
		Project:     "myproject",
	})
	_, err := NewClientFromConfig()
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
	if !strings.Contains(err.Error(), "invalid API URL") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClientFromConfigValid(t *testing.T) {
	writeConfig(t, &config.Config{
		APIUrl:      "https://acpctl-test.localhost:8000",
		AccessToken: "test-token-at-least-20chars",
		Project:     "myproject",
	})
	client, err := NewClientFromConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
