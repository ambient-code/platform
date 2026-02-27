package config

import (
	"fmt"
	"os"
	"strconv"
)

type ControlPlaneConfig struct {
	APIServerURL   string
	APIToken       string
	APIProject     string
	GRPCServerAddr string
	GRPCUseTLS     bool
	LogLevel       string
	Kubeconfig     string
	Namespace      string
	Mode           string
}

func Load() (*ControlPlaneConfig, error) {
	cfg := &ControlPlaneConfig{
		APIServerURL:   envOrDefault("AMBIENT_API_SERVER_URL", "http://localhost:8000"),
		APIToken:       os.Getenv("AMBIENT_API_TOKEN"),
		APIProject:     envOrDefault("AMBIENT_API_PROJECT", "default"),
		GRPCServerAddr: envOrDefault("AMBIENT_GRPC_SERVER_ADDR", "localhost:8001"),
		GRPCUseTLS:     os.Getenv("AMBIENT_GRPC_USE_TLS") == "true",
		LogLevel:       envOrDefault("LOG_LEVEL", "info"),
		Kubeconfig:     os.Getenv("KUBECONFIG"),
		Namespace:      envOrDefault("NAMESPACE", "ambient-code"),
		Mode:           envOrDefault("MODE", "kube"),
	}

	if cfg.APIToken == "" {
		return nil, fmt.Errorf("AMBIENT_API_TOKEN environment variable is required")
	}

	return cfg, nil
}

type LocalConfig struct {
	WorkspaceRoot  string
	ProxyAddr      string
	CORSOrigin     string
	PortRangeStart int
	PortRangeEnd   int
	RunnerCommand  string
	MaxSessions    int
	BossURL        string
	BossSpace      string
}

func LoadLocalConfig() *LocalConfig {
	cfg := &LocalConfig{
		WorkspaceRoot: envOrDefault("LOCAL_WORKSPACE_ROOT", defaultWorkspaceRoot()),
		ProxyAddr:     envOrDefault("LOCAL_PROXY_ADDR", "127.0.0.1:9080"),
		CORSOrigin:    os.Getenv("CORS_ALLOWED_ORIGIN"),
		RunnerCommand: envOrDefault("LOCAL_RUNNER_COMMAND", "python local_entry.py"),
		BossURL:       os.Getenv("BOSS_URL"),
		BossSpace:     envOrDefault("BOSS_SPACE", "default"),
	}

	cfg.PortRangeStart = 9100
	cfg.PortRangeEnd = 9199
	if portRange := os.Getenv("LOCAL_PORT_RANGE"); portRange != "" {
		_, _ = fmt.Sscanf(portRange, "%d-%d", &cfg.PortRangeStart, &cfg.PortRangeEnd)
	}

	maxSessions, err := strconv.Atoi(envOrDefault("LOCAL_MAX_SESSIONS", "10"))
	if err != nil {
		maxSessions = 10
	}
	cfg.MaxSessions = maxSessions

	return cfg
}

func defaultWorkspaceRoot() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/tmp"
	}
	return home + "/.ambient/workspaces"
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
