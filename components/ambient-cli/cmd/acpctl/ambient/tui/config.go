package tui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
)

// TUIConfig holds the multi-context configuration for the TUI.
// It supports both the new multi-context format and the legacy flat format
// used by the existing acpctl CLI.
type TUIConfig struct {
	CurrentContext string              `json:"current_context,omitempty"`
	Contexts       map[string]*Context `json:"contexts,omitempty"`
}

// String implements fmt.Stringer. All context tokens are redacted.
func (c *TUIConfig) String() string {
	names := c.ContextNames()
	return fmt.Sprintf("TUIConfig{CurrentContext:%q, Contexts:[%s]}", c.CurrentContext, strings.Join(names, ", "))
}

// GoString implements fmt.GoStringer. All context tokens are redacted.
func (c *TUIConfig) GoString() string {
	return c.String()
}

// Context represents a single server connection with its credentials and project scope.
type Context struct {
	Server       string `json:"server"`
	AccessToken  string `json:"access_token"`
	Project      string `json:"project,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IssuerURL    string `json:"issuer_url,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
}

// Username extracts the username from the JWT access token claims.
// Checks preferred_username, sub, email in order. Returns "unknown" on failure.
func (c *Context) Username() string {
	if c.AccessToken == "" {
		return "unknown"
	}
	parts := strings.SplitN(c.AccessToken, ".", 3)
	if len(parts) < 2 {
		return "unknown"
	}
	// Decode the payload (base64url, no padding).
	payload := parts[1]
	if rem := len(payload) % 4; rem != 0 {
		payload += strings.Repeat("=", 4-rem)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.NewReplacer("-", "+", "_", "/").Replace(payload))
	if err != nil {
		return "unknown"
	}
	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "unknown"
	}
	for _, key := range []string{"preferred_username", "sub", "email"} {
		if v, ok := claims[key].(string); ok && v != "" {
			return v
		}
	}
	return "unknown"
}

// String implements fmt.Stringer. The access token is redacted for security.
func (c *Context) String() string {
	token := "<none>"
	if c.AccessToken != "" {
		token = fmt.Sprintf("<redacted:%d>", len(c.AccessToken))
	}
	return fmt.Sprintf("Context{Server:%q, AccessToken:%s, Project:%q}", c.Server, token, c.Project)
}

// GoString implements fmt.GoStringer. The access token is redacted for security.
func (c *Context) GoString() string {
	token := `""`
	if c.AccessToken != "" {
		token = fmt.Sprintf(`"<redacted:%d>"`, len(c.AccessToken))
	}
	refresh := `""`
	if c.RefreshToken != "" {
		refresh = fmt.Sprintf(`"<redacted:%d>"`, len(c.RefreshToken))
	}
	return fmt.Sprintf(
		"tui.Context{Server:%q, AccessToken:%s, Project:%q, RefreshToken:%s, IssuerURL:%q, ClientID:%q}",
		c.Server, token, c.Project, refresh, c.IssuerURL, c.ClientID,
	)
}

// legacyConfig mirrors the flat config format from pkg/config for deserialization
// during migration detection. Fields match config.Config JSON tags.
type legacyConfig struct {
	APIUrl       string `json:"api_url,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IssuerURL    string `json:"issuer_url,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	Project      string `json:"project,omitempty"`
}

// LoadTUIConfig reads the shared config file and returns a multi-context TUIConfig.
//
// Format detection:
//   - If the file contains a "contexts" key, it is parsed as the new multi-context format.
//   - If the file contains "api_url" but no "contexts", it is the legacy flat format.
//     A single context entry is created, auto-named from the server hostname, and set as current.
//   - If the file does not exist, an empty TUIConfig is returned.
//
// Environment variable overrides (AMBIENT_API_URL, AMBIENT_TOKEN, AMBIENT_PROJECT)
// are applied to the current context's values after loading.
func LoadTUIConfig() (*TUIConfig, error) {
	location, err := config.Location()
	if err != nil {
		return nil, fmt.Errorf("determine config location: %w", err)
	}

	data, err := os.ReadFile(location)
	if err != nil {
		if os.IsNotExist(err) {
			return &TUIConfig{
				Contexts: make(map[string]*Context),
			}, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", location, err)
	}

	// Probe the raw JSON to determine format.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", location, err)
	}

	var cfg *TUIConfig

	if _, hasContexts := probe["contexts"]; hasContexts {
		// New multi-context format.
		cfg = &TUIConfig{}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse multi-context config %q: %w", location, err)
		}
		if cfg.Contexts == nil {
			cfg.Contexts = make(map[string]*Context)
		}
	} else {
		// Legacy flat format — migrate in memory.
		var legacy legacyConfig
		if err := json.Unmarshal(data, &legacy); err != nil {
			return nil, fmt.Errorf("parse legacy config %q: %w", location, err)
		}
		cfg = migrateFromLegacy(&legacy)
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

// migrateFromLegacy converts a flat legacy config into a single-context TUIConfig.
// The context name is derived from the server URL hostname.
func migrateFromLegacy(legacy *legacyConfig) *TUIConfig {
	server := legacy.APIUrl
	if server == "" {
		server = "http://localhost:8000"
	}

	name := ContextNameFromURL(server)

	ctx := &Context{
		Server:       server,
		AccessToken:  legacy.AccessToken,
		Project:      legacy.Project,
		RefreshToken: legacy.RefreshToken,
		IssuerURL:    legacy.IssuerURL,
		ClientID:     legacy.ClientID,
	}

	return &TUIConfig{
		CurrentContext: name,
		Contexts: map[string]*Context{
			name: ctx,
		},
	}
}

// applyEnvOverrides applies AMBIENT_API_URL, AMBIENT_TOKEN, and AMBIENT_PROJECT
// environment variable overrides to the current context. If no current context exists
// and an override is present, a context is created.
func applyEnvOverrides(cfg *TUIConfig) {
	envURL := os.Getenv("AMBIENT_API_URL")
	envToken := os.Getenv("AMBIENT_TOKEN")
	envProject := os.Getenv("AMBIENT_PROJECT")

	if envURL == "" && envToken == "" && envProject == "" {
		return
	}

	cur := cfg.Current()
	if cur == nil {
		// No current context — create one from env vars.
		server := envURL
		if server == "" {
			server = "http://localhost:8000"
		}
		name := ContextNameFromURL(server)
		cur = &Context{Server: server}
		cfg.Contexts[name] = cur
		cfg.CurrentContext = name
	}

	if envURL != "" {
		cur.Server = envURL
	}
	if envToken != "" {
		cur.AccessToken = envToken
	}
	if envProject != "" {
		cur.Project = envProject
	}
}

// Current returns the active context, or nil if no context is set.
func (c *TUIConfig) Current() *Context {
	if c.CurrentContext == "" || c.Contexts == nil {
		return nil
	}
	return c.Contexts[c.CurrentContext]
}

// ContextNames returns a sorted list of all context names.
func (c *TUIConfig) ContextNames() []string {
	names := make([]string, 0, len(c.Contexts))
	for name := range c.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SwitchContext changes the current context to the named context.
// Returns an error if the context name does not exist.
func (c *TUIConfig) SwitchContext(name string) error {
	if c.Contexts == nil {
		return fmt.Errorf("context %q not found", name)
	}
	if _, ok := c.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	c.CurrentContext = name
	return nil
}

// ContextNameFromURL derives a context name from a server URL.
//
// Rules (from the TUI spec):
//   - localhost (any port) → "local"
//   - All other servers → hostname portion of the URL
func ContextNameFromURL(serverURL string) string {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return "default"
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return "default"
	}

	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return "local"
	}

	// Strip port via Hostname() already done; return the hostname.
	return strings.TrimPrefix(hostname, "www.")
}
