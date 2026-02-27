package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocationUsesEnvVar(t *testing.T) {
	t.Setenv("AMBIENT_CONFIG", "/tmp/test-config.json")
	loc, err := Location()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc != "/tmp/test-config.json" {
		t.Errorf("expected /tmp/test-config.json, got %s", loc)
	}
}

func TestLocationDefaultsToUserConfigDir(t *testing.T) {
	t.Setenv("AMBIENT_CONFIG", "")
	loc, err := Location()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(loc) != "config.json" {
		t.Errorf("expected config.json, got %s", filepath.Base(loc))
	}
}

func TestLoadNonExistentReturnsEmpty(t *testing.T) {
	t.Setenv("AMBIENT_CONFIG", "/tmp/nonexistent-acpctl-config-12345.json")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIUrl != "" || cfg.AccessToken != "" || cfg.Project != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("AMBIENT_CONFIG", path)

	cfg := &Config{
		APIUrl:      "https://api.example.com",
		AccessToken: "test-token-123",
		Project:     "my-project",
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.APIUrl != cfg.APIUrl {
		t.Errorf("APIUrl mismatch: got %s, want %s", loaded.APIUrl, cfg.APIUrl)
	}
	if loaded.AccessToken != cfg.AccessToken {
		t.Errorf("AccessToken mismatch: got %s, want %s", loaded.AccessToken, cfg.AccessToken)
	}
	if loaded.Project != cfg.Project {
		t.Errorf("Project mismatch: got %s, want %s", loaded.Project, cfg.Project)
	}
}

func TestGetAPIUrlDefaults(t *testing.T) {
	t.Setenv("AMBIENT_API_URL", "")
	cfg := &Config{}
	if got := cfg.GetAPIUrl(); got != "http://localhost:8000" {
		t.Errorf("expected default URL, got %s", got)
	}
}

func TestGetAPIUrlFromConfig(t *testing.T) {
	t.Setenv("AMBIENT_API_URL", "")
	cfg := &Config{APIUrl: "https://custom.example.com"}
	if got := cfg.GetAPIUrl(); got != "https://custom.example.com" {
		t.Errorf("expected custom URL, got %s", got)
	}
}

func TestGetAPIUrlFromEnv(t *testing.T) {
	t.Setenv("AMBIENT_API_URL", "https://env.example.com")
	cfg := &Config{}
	if got := cfg.GetAPIUrl(); got != "https://env.example.com" {
		t.Errorf("expected env URL, got %s", got)
	}
}

func TestGetAPIUrlEnvOverridesConfig(t *testing.T) {
	t.Setenv("AMBIENT_API_URL", "https://env.example.com")
	cfg := &Config{APIUrl: "https://config.example.com"}
	if got := cfg.GetAPIUrl(); got != "https://env.example.com" {
		t.Errorf("env should override config: got %s, want https://env.example.com", got)
	}
}

func TestGetTokenFromConfig(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "")
	cfg := &Config{AccessToken: "my-token"}
	if got := cfg.GetToken(); got != "my-token" {
		t.Errorf("expected my-token, got %s", got)
	}
}

func TestGetTokenFromEnv(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "env-token")
	cfg := &Config{}
	if got := cfg.GetToken(); got != "env-token" {
		t.Errorf("expected env-token, got %s", got)
	}
}

func TestGetProjectFromConfig(t *testing.T) {
	t.Setenv("AMBIENT_PROJECT", "")
	cfg := &Config{Project: "my-proj"}
	if got := cfg.GetProject(); got != "my-proj" {
		t.Errorf("expected my-proj, got %s", got)
	}
}

func TestGetProjectFromEnv(t *testing.T) {
	t.Setenv("AMBIENT_PROJECT", "env-proj")
	cfg := &Config{}
	if got := cfg.GetProject(); got != "env-proj" {
		t.Errorf("expected env-proj, got %s", got)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("AMBIENT_CONFIG", path)

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
