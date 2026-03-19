package config

import (
	"fmt"
	"os"
	"strings"
)

type ControlPlaneConfig struct {
	APIServerURL          string
	APIToken              string
	GRPCServerAddr        string
	GRPCUseTLS            bool
	LogLevel              string
	Kubeconfig            string
	Mode                  string
	Reconcilers           []string
	RunnerImage           string
	RunnerGRPCUseTLS      bool
	BackendURL            string
	Namespace             string
	AnthropicAPIKey       string
	VertexEnabled         bool
	VertexProjectID       string
	VertexRegion          string
	VertexCredentialsPath string
	VertexSecretName      string
	VertexSecretNamespace string
	RunnerImageNamespace  string
}

func Load() (*ControlPlaneConfig, error) {
	cfg := &ControlPlaneConfig{
		APIServerURL:          envOrDefault("AMBIENT_API_SERVER_URL", "http://localhost:8000"),
		APIToken:              os.Getenv("AMBIENT_API_TOKEN"),
		GRPCServerAddr:        envOrDefault("AMBIENT_GRPC_SERVER_ADDR", "localhost:8001"),
		GRPCUseTLS:            os.Getenv("AMBIENT_GRPC_USE_TLS") == "true",
		LogLevel:              envOrDefault("LOG_LEVEL", "info"),
		Kubeconfig:            os.Getenv("KUBECONFIG"),
		Mode:                  envOrDefault("MODE", "kube"),
		Reconcilers:           parseReconcilers(envOrDefault("RECONCILERS", "tally,kube")),
		RunnerImage:           envOrDefault("RUNNER_IMAGE", "quay.io/ambient_code/vteam_claude_runner:latest"),
		RunnerGRPCUseTLS:      os.Getenv("AMBIENT_GRPC_USE_TLS") == "true",
		BackendURL:            envOrDefault("BACKEND_API_URL", "http://backend-service.ambient-code.svc:8080/api"),
		Namespace:             envOrDefault("NAMESPACE", "ambient-code"),
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		VertexEnabled:         os.Getenv("USE_VERTEX") == "1" || os.Getenv("USE_VERTEX") == "true",
		VertexProjectID:       os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"),
		VertexRegion:          envOrDefault("CLOUD_ML_REGION", "us-east5"),
		VertexCredentialsPath: envOrDefault("GOOGLE_APPLICATION_CREDENTIALS", "/app/vertex/ambient-code-key.json"),
		VertexSecretName:      envOrDefault("VERTEX_SECRET_NAME", "ambient-vertex"),
		VertexSecretNamespace: envOrDefault("VERTEX_SECRET_NAMESPACE", "ambient-code"),
		RunnerImageNamespace:  os.Getenv("RUNNER_IMAGE_NAMESPACE"),
	}

	if cfg.APIToken == "" {
		return nil, fmt.Errorf("AMBIENT_API_TOKEN environment variable is required")
	}

	switch cfg.Mode {
	case "kube", "test":
	default:
		return nil, fmt.Errorf("unknown MODE %q: must be one of kube, test", cfg.Mode)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseReconcilers(reconcilersStr string) []string {
	if reconcilersStr == "" {
		return []string{"tally"}
	}

	reconcilers := strings.Split(reconcilersStr, ",")
	var result []string
	for _, r := range reconcilers {
		r = strings.TrimSpace(r)
		if r != "" {
			result = append(result, r)
		}
	}

	if len(result) == 0 {
		return []string{"tally"}
	}

	return result
}
