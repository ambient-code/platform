// Package config manages CLI configuration persistence and environment variable overrides.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	APIUrl            string `json:"api_url,omitempty"`
	AccessToken       string `json:"access_token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty"`
	IssuerURL         string `json:"issuer_url,omitempty"`
	ClientID          string `json:"client_id,omitempty"`
	Project           string `json:"project,omitempty"`
	Pager             string `json:"pager,omitempty"`            // TODO: Wire pager support into output commands (e.g. pipe through less)
	RequestTimeout    int    `json:"request_timeout,omitempty"`  // Request timeout in seconds
	PollingInterval   int    `json:"polling_interval,omitempty"` // Watch polling interval in seconds
	InsecureTLSVerify bool   `json:"insecure_tls_verify,omitempty"`
}

func Location() (string, error) {
	if env := os.Getenv("AMBIENT_CONFIG"); env != "" {
		return env, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}

	return filepath.Join(configDir, "ambient", "config.json"), nil
}

func Load() (*Config, error) {
	location, err := Location()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(location)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config file %q: %w", location, err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", location, err)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	location, err := Location()
	if err != nil {
		return err
	}

	dir := filepath.Dir(location)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(location, data, 0600); err != nil {
		return fmt.Errorf("write config file %q: %w", location, err)
	}

	return nil
}

func (c *Config) ClearToken() {
	c.AccessToken = ""
}

func (c *Config) GetAPIUrl() string {
	if env := os.Getenv("AMBIENT_API_URL"); env != "" {
		return env
	}
	if c.APIUrl != "" {
		return c.APIUrl
	}
	return "http://localhost:8000"
}

func (c *Config) GetProject() string {
	if env := os.Getenv("AMBIENT_PROJECT"); env != "" {
		return env
	}
	if c.Project != "" {
		return c.Project
	}
	return ""
}

func (c *Config) GetToken() string {
	if env := os.Getenv("AMBIENT_TOKEN"); env != "" {
		return env
	}
	return c.AccessToken
}

func (c *Config) GetTokenWithRefresh() (string, error) {
	if env := os.Getenv("AMBIENT_TOKEN"); env != "" {
		return env, nil
	}

	if c.AccessToken == "" {
		return "", nil
	}

	expired, err := IsTokenExpired(c.AccessToken)
	if err != nil || !expired {
		return c.AccessToken, nil
	}

	if c.RefreshToken == "" || c.IssuerURL == "" || c.ClientID == "" {
		return c.AccessToken, nil
	}

	newAccess, newRefresh, refreshErr := RefreshAccessToken(c.IssuerURL, c.ClientID, c.RefreshToken)
	if refreshErr != nil {
		return c.AccessToken, nil
	}

	c.AccessToken = newAccess
	if newRefresh != "" {
		c.RefreshToken = newRefresh
	}

	if saveErr := Save(c); saveErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to persist refreshed token: %v\n", saveErr)
	}

	return c.AccessToken, nil
}

// GetRequestTimeout returns the request timeout duration with fallback to default
func (c *Config) GetRequestTimeout() time.Duration {
	if env := os.Getenv("AMBIENT_REQUEST_TIMEOUT"); env != "" {
		if seconds, err := strconv.Atoi(env); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	if c.RequestTimeout > 0 {
		return time.Duration(c.RequestTimeout) * time.Second
	}
	return 30 * time.Second // Default 30 seconds
}

// GetPollingInterval returns the watch polling interval with fallback to default
func (c *Config) GetPollingInterval() time.Duration {
	if env := os.Getenv("AMBIENT_POLLING_INTERVAL"); env != "" {
		if seconds, err := strconv.Atoi(env); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	if c.PollingInterval > 0 {
		return time.Duration(c.PollingInterval) * time.Second
	}
	return 2 * time.Second // Default 2 seconds
}
