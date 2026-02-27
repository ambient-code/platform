// Package config manages CLI configuration persistence and environment variable overrides.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIUrl      string `json:"api_url,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Project     string `json:"project,omitempty"`
	Pager       string `json:"pager,omitempty"` // TODO: Wire pager support into output commands (e.g. pipe through less)
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
