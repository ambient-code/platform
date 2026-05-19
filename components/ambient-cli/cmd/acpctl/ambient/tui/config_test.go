package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupConfigFile writes content to a temp config file and sets AMBIENT_CONFIG
// to point to it. Returns a cleanup function.
func setupConfigFile(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AMBIENT_CONFIG", path)
}

// clearEnvOverrides ensures env var overrides are unset for a test.
func clearEnvOverrides(t *testing.T) {
	t.Helper()
	t.Setenv("AMBIENT_API_URL", "")
	t.Setenv("AMBIENT_TOKEN", "")
	t.Setenv("AMBIENT_PROJECT", "")
}

func TestLoadTUIConfig_NewFormat(t *testing.T) {
	clearEnvOverrides(t)
	setupConfigFile(t, `{
		"current_context": "staging",
		"contexts": {
			"local": {
				"server": "http://localhost:8000",
				"access_token": "tok-local",
				"project": "proj-local"
			},
			"staging": {
				"server": "https://api.staging.ambient.io",
				"access_token": "tok-staging",
				"project": "proj-staging"
			}
		}
	}`)

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	if cfg.CurrentContext != "staging" {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "staging")
	}

	if len(cfg.Contexts) != 2 {
		t.Fatalf("len(Contexts) = %d, want 2", len(cfg.Contexts))
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil")
	}
	if cur.Server != "https://api.staging.ambient.io" {
		t.Errorf("Current().Server = %q, want %q", cur.Server, "https://api.staging.ambient.io")
	}
	if cur.AccessToken != "tok-staging" {
		t.Errorf("Current().AccessToken = %q, want %q", cur.AccessToken, "tok-staging")
	}
	if cur.Project != "proj-staging" {
		t.Errorf("Current().Project = %q, want %q", cur.Project, "proj-staging")
	}

	local := cfg.Contexts["local"]
	if local == nil {
		t.Fatal("Contexts[\"local\"] is nil")
	}
	if local.Server != "http://localhost:8000" {
		t.Errorf("local.Server = %q, want %q", local.Server, "http://localhost:8000")
	}
}

func TestLoadTUIConfig_LegacyFormat(t *testing.T) {
	clearEnvOverrides(t)
	setupConfigFile(t, `{
		"api_url": "https://api.prod.ambient.io",
		"access_token": "tok-legacy",
		"refresh_token": "ref-legacy",
		"issuer_url": "https://sso.example.com",
		"client_id": "my-client",
		"project": "legacy-proj"
	}`)

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	// Should auto-migrate to a context named from the hostname.
	expectedName := "api.prod.ambient.io"
	if cfg.CurrentContext != expectedName {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, expectedName)
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil after migration")
	}

	if cur.Server != "https://api.prod.ambient.io" {
		t.Errorf("Server = %q, want %q", cur.Server, "https://api.prod.ambient.io")
	}
	if cur.AccessToken != "tok-legacy" {
		t.Errorf("AccessToken = %q, want %q", cur.AccessToken, "tok-legacy")
	}
	if cur.Project != "legacy-proj" {
		t.Errorf("Project = %q, want %q", cur.Project, "legacy-proj")
	}
	if cur.RefreshToken != "ref-legacy" {
		t.Errorf("RefreshToken = %q, want %q", cur.RefreshToken, "ref-legacy")
	}
	if cur.IssuerURL != "https://sso.example.com" {
		t.Errorf("IssuerURL = %q, want %q", cur.IssuerURL, "https://sso.example.com")
	}
	if cur.ClientID != "my-client" {
		t.Errorf("ClientID = %q, want %q", cur.ClientID, "my-client")
	}
}

func TestLoadTUIConfig_LegacyLocalhostMigration(t *testing.T) {
	clearEnvOverrides(t)
	setupConfigFile(t, `{
		"api_url": "http://localhost:8000",
		"access_token": "tok-local"
	}`)

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	if cfg.CurrentContext != "local" {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "local")
	}
}

func TestLoadTUIConfig_LegacyNoAPIURL(t *testing.T) {
	clearEnvOverrides(t)
	// Legacy config with no api_url defaults to localhost.
	setupConfigFile(t, `{
		"access_token": "tok-nourl"
	}`)

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	if cfg.CurrentContext != "local" {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "local")
	}
	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil")
	}
	if cur.Server != "http://localhost:8000" {
		t.Errorf("Server = %q, want %q", cur.Server, "http://localhost:8000")
	}
}

func TestLoadTUIConfig_FileNotFound(t *testing.T) {
	clearEnvOverrides(t)
	t.Setenv("AMBIENT_CONFIG", filepath.Join(t.TempDir(), "nonexistent.json"))

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	if len(cfg.Contexts) != 0 {
		t.Errorf("expected empty contexts, got %d", len(cfg.Contexts))
	}
	if cfg.Current() != nil {
		t.Error("Current() should return nil for empty config")
	}
}

func TestLoadTUIConfig_EnvVarOverrides(t *testing.T) {
	setupConfigFile(t, `{
		"current_context": "local",
		"contexts": {
			"local": {
				"server": "http://localhost:8000",
				"access_token": "file-token",
				"project": "file-proj"
			}
		}
	}`)

	t.Setenv("AMBIENT_API_URL", "https://env-server.io")
	t.Setenv("AMBIENT_TOKEN", "env-token")
	t.Setenv("AMBIENT_PROJECT", "env-proj")

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil")
	}

	if cur.Server != "https://env-server.io" {
		t.Errorf("Server = %q, want %q (env override)", cur.Server, "https://env-server.io")
	}
	if cur.AccessToken != "env-token" {
		t.Errorf("AccessToken = %q, want %q (env override)", cur.AccessToken, "env-token")
	}
	if cur.Project != "env-proj" {
		t.Errorf("Project = %q, want %q (env override)", cur.Project, "env-proj")
	}
}

func TestLoadTUIConfig_EnvVarCreatesContext(t *testing.T) {
	// Empty config file, env vars should create a context.
	setupConfigFile(t, `{}`)

	t.Setenv("AMBIENT_API_URL", "https://env-only.io")
	t.Setenv("AMBIENT_TOKEN", "env-only-token")
	t.Setenv("AMBIENT_PROJECT", "env-only-proj")

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil; expected env-created context")
	}

	if cur.Server != "https://env-only.io" {
		t.Errorf("Server = %q, want %q", cur.Server, "https://env-only.io")
	}
	if cur.AccessToken != "env-only-token" {
		t.Errorf("AccessToken = %q, want %q", cur.AccessToken, "env-only-token")
	}
}

func TestLoadTUIConfig_EnvPartialOverride(t *testing.T) {
	setupConfigFile(t, `{
		"current_context": "prod",
		"contexts": {
			"prod": {
				"server": "https://api.prod.io",
				"access_token": "prod-token",
				"project": "prod-proj"
			}
		}
	}`)

	// Only override the token, leave server and project from file.
	t.Setenv("AMBIENT_API_URL", "")
	t.Setenv("AMBIENT_TOKEN", "override-token")
	t.Setenv("AMBIENT_PROJECT", "")

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil")
	}

	if cur.Server != "https://api.prod.io" {
		t.Errorf("Server = %q, want %q (should not be overridden)", cur.Server, "https://api.prod.io")
	}
	if cur.AccessToken != "override-token" {
		t.Errorf("AccessToken = %q, want %q", cur.AccessToken, "override-token")
	}
	if cur.Project != "prod-proj" {
		t.Errorf("Project = %q, want %q (should not be overridden)", cur.Project, "prod-proj")
	}
}

func TestContextNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"http://localhost:8000", "local"},
		{"http://localhost:18000", "local"},
		{"http://localhost", "local"},
		{"https://localhost:443", "local"},
		{"http://127.0.0.1:8000", "local"},
		{"http://[::1]:8000", "local"},
		{"https://api.staging.ambient.io", "api.staging.ambient.io"},
		{"https://api.ambient.io", "api.ambient.io"},
		{"https://api.ambient.io:8443", "api.ambient.io"},
		{"https://my-server.example.com/v1", "my-server.example.com"},
		{"not-a-valid-url", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := ContextNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("ContextNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestTUIConfig_SwitchContext(t *testing.T) {
	cfg := &TUIConfig{
		CurrentContext: "local",
		Contexts: map[string]*Context{
			"local": {Server: "http://localhost:8000"},
			"prod":  {Server: "https://api.prod.io"},
		},
	}

	// Switch to valid context.
	if err := cfg.SwitchContext("prod"); err != nil {
		t.Fatalf("SwitchContext(\"prod\") error: %v", err)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "prod")
	}
	if cfg.Current().Server != "https://api.prod.io" {
		t.Errorf("Current().Server = %q, want %q", cfg.Current().Server, "https://api.prod.io")
	}

	// Switch to invalid context.
	err := cfg.SwitchContext("nonexistent")
	if err == nil {
		t.Fatal("SwitchContext(\"nonexistent\") should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "not found")
	}
	// CurrentContext should remain unchanged after failed switch.
	if cfg.CurrentContext != "prod" {
		t.Errorf("CurrentContext = %q after failed switch, want %q", cfg.CurrentContext, "prod")
	}
}

func TestTUIConfig_SwitchContext_NilContexts(t *testing.T) {
	cfg := &TUIConfig{}
	err := cfg.SwitchContext("anything")
	if err == nil {
		t.Fatal("SwitchContext on nil Contexts should return error")
	}
}

func TestTUIConfig_ContextNames(t *testing.T) {
	cfg := &TUIConfig{
		Contexts: map[string]*Context{
			"prod":    {Server: "https://api.prod.io"},
			"staging": {Server: "https://api.staging.io"},
			"local":   {Server: "http://localhost:8000"},
		},
	}

	names := cfg.ContextNames()
	expected := []string{"local", "prod", "staging"}

	if len(names) != len(expected) {
		t.Fatalf("ContextNames() len = %d, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("ContextNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestTUIConfig_ContextNames_Empty(t *testing.T) {
	cfg := &TUIConfig{Contexts: map[string]*Context{}}
	names := cfg.ContextNames()
	if len(names) != 0 {
		t.Errorf("ContextNames() on empty = %v, want empty", names)
	}
}

func TestTUIConfig_Current_NoContext(t *testing.T) {
	cfg := &TUIConfig{
		CurrentContext: "missing",
		Contexts: map[string]*Context{
			"local": {Server: "http://localhost:8000"},
		},
	}
	if cfg.Current() != nil {
		t.Error("Current() should return nil when CurrentContext does not match any entry")
	}
}

func TestContext_StringRedactsToken(t *testing.T) {
	ctx := &Context{
		Server:      "https://api.prod.io",
		AccessToken: "super-secret-token-value",
		Project:     "my-proj",
	}

	s := ctx.String()
	if strings.Contains(s, "super-secret-token-value") {
		t.Errorf("String() should not contain the raw token: %s", s)
	}
	if !strings.Contains(s, "<redacted:") {
		t.Errorf("String() should contain '<redacted:': %s", s)
	}
	if !strings.Contains(s, fmt.Sprintf("<redacted:%d>", len("super-secret-token-value"))) {
		t.Errorf("String() should show token length: %s", s)
	}
	if !strings.Contains(s, "api.prod.io") {
		t.Errorf("String() should contain server: %s", s)
	}
}

func TestContext_StringEmptyToken(t *testing.T) {
	ctx := &Context{
		Server:  "http://localhost:8000",
		Project: "proj",
	}

	s := ctx.String()
	if !strings.Contains(s, "<none>") {
		t.Errorf("String() with empty token should contain '<none>': %s", s)
	}
}

func TestContext_GoStringRedactsTokens(t *testing.T) {
	ctx := &Context{
		Server:       "https://api.prod.io",
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		IssuerURL:    "https://sso.example.com",
		ClientID:     "my-client",
	}

	s := fmt.Sprintf("%#v", ctx)
	if strings.Contains(s, "access-secret") {
		t.Errorf("GoString() should not contain the raw access token: %s", s)
	}
	if strings.Contains(s, "refresh-secret") {
		t.Errorf("GoString() should not contain the raw refresh token: %s", s)
	}
	if !strings.Contains(s, "<redacted:") {
		t.Errorf("GoString() should contain '<redacted:': %s", s)
	}
}

func TestContext_GoStringEmptyTokens(t *testing.T) {
	ctx := &Context{
		Server: "http://localhost:8000",
	}

	s := fmt.Sprintf("%#v", ctx)
	// Empty tokens should show as empty strings, not redacted.
	if strings.Contains(s, "<redacted") {
		t.Errorf("GoString() with empty tokens should not contain '<redacted': %s", s)
	}
}

func TestLoadTUIConfig_NewFormatWithOIDCFields(t *testing.T) {
	clearEnvOverrides(t)
	setupConfigFile(t, `{
		"current_context": "sso",
		"contexts": {
			"sso": {
				"server": "https://api.sso.io",
				"access_token": "tok",
				"refresh_token": "ref",
				"issuer_url": "https://sso.example.com",
				"client_id": "cli-id",
				"project": "proj"
			}
		}
	}`)

	cfg, err := LoadTUIConfig()
	if err != nil {
		t.Fatalf("LoadTUIConfig() error: %v", err)
	}

	cur := cfg.Current()
	if cur == nil {
		t.Fatal("Current() returned nil")
	}
	if cur.RefreshToken != "ref" {
		t.Errorf("RefreshToken = %q, want %q", cur.RefreshToken, "ref")
	}
	if cur.IssuerURL != "https://sso.example.com" {
		t.Errorf("IssuerURL = %q, want %q", cur.IssuerURL, "https://sso.example.com")
	}
	if cur.ClientID != "cli-id" {
		t.Errorf("ClientID = %q, want %q", cur.ClientID, "cli-id")
	}
}

func TestLoadTUIConfig_InvalidJSON(t *testing.T) {
	clearEnvOverrides(t)
	setupConfigFile(t, `{not valid json}`)

	_, err := LoadTUIConfig()
	if err == nil {
		t.Fatal("LoadTUIConfig() should return error for invalid JSON")
	}
}
