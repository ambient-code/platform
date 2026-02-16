package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type ControlPlaneConfig struct {
	APIServerURL string
	APIToken     string
	APIProject   string
	GRPCServerAddr string
	PollInterval time.Duration
	WorkerCount  int
	LogLevel     string
	Kubeconfig   string
	Namespace    string
}

func Load() (*ControlPlaneConfig, error) {
	cfg := &ControlPlaneConfig{
		APIServerURL:   envOrDefault("AMBIENT_API_SERVER_URL", "http://localhost:8000"),
		APIToken:       os.Getenv("AMBIENT_API_TOKEN"),
		APIProject:     envOrDefault("AMBIENT_API_PROJECT", "default"),
		GRPCServerAddr: envOrDefault("AMBIENT_GRPC_SERVER_ADDR", "localhost:8001"),
		LogLevel:       envOrDefault("LOG_LEVEL", "info"),
		Kubeconfig:     os.Getenv("KUBECONFIG"),
		Namespace:      envOrDefault("NAMESPACE", "ambient-code"),
	}

	pollSeconds, err := strconv.Atoi(envOrDefault("POLL_INTERVAL_SECONDS", "2"))
	if err != nil {
		return nil, fmt.Errorf("invalid POLL_INTERVAL_SECONDS: %w", err)
	}
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second

	workers, err := strconv.Atoi(envOrDefault("WORKER_COUNT", "2"))
	if err != nil {
		return nil, fmt.Errorf("invalid WORKER_COUNT: %w", err)
	}
	cfg.WorkerCount = workers

	return cfg, nil
}

type LocalConfig struct {
	WorkspaceRoot  string
	ProxyAddr      string
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
		RunnerCommand: envOrDefault("LOCAL_RUNNER_COMMAND", "python local_entry.py"),
		BossURL:       os.Getenv("BOSS_URL"),
		BossSpace:     envOrDefault("BOSS_SPACE", "default"),
	}

	cfg.PortRangeStart = 9100
	cfg.PortRangeEnd = 9199
	if portRange := os.Getenv("LOCAL_PORT_RANGE"); portRange != "" {
		fmt.Sscanf(portRange, "%d-%d", &cfg.PortRangeStart, &cfg.PortRangeEnd)
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
