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
	PlatformMode          string
	MPPConfigNamespace    string
	CPRuntimeNamespace    string
	OIDCTokenURL          string
	OIDCClientID          string
	OIDCClientSecret      string
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
	MCPImage              string
	MCPAPIServerURL       string
	RunnerLogLevel        string
	ProjectKubeTokenFile  string
	CPTokenListenAddr     string
	CPTokenURL            string
	HTTPProxy             string
	HTTPSProxy            string
	NoProxy               string
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
		PlatformMode:          envOrDefault("PLATFORM_MODE", "standard"),
		MPPConfigNamespace:    envOrDefault("MPP_CONFIG_NAMESPACE", "ambient-code--config"),
		CPRuntimeNamespace:    envOrDefault("CP_RUNTIME_NAMESPACE", "ambient-code--runtime-int"),
		OIDCTokenURL:          envOrDefault("OIDC_TOKEN_URL", "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"),
		OIDCClientID:          os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret:      os.Getenv("OIDC_CLIENT_SECRET"),
		Reconcilers:           parseReconcilers(envOrDefault("RECONCILERS", "tally,kube")),
		RunnerImage:           envOrDefault("RUNNER_IMAGE", "quay.io/ambient_code/vteam_claude_runner:latest"),
		RunnerGRPCUseTLS:      os.Getenv("AMBIENT_GRPC_USE_TLS") == "true",
		BackendURL:            envOrDefault("BACKEND_API_URL", envOrDefault("AMBIENT_API_SERVER_URL", "http://localhost:8000")),
		Namespace:             envOrDefault("NAMESPACE", "ambient-code"),
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		VertexEnabled:         os.Getenv("USE_VERTEX") == "1" || os.Getenv("USE_VERTEX") == "true",
		VertexProjectID:       os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"),
		VertexRegion:          envOrDefault("CLOUD_ML_REGION", "us-east5"),
		VertexCredentialsPath: envOrDefault("GOOGLE_APPLICATION_CREDENTIALS", "/app/vertex/ambient-code-key.json"),
		VertexSecretName:      envOrDefault("VERTEX_SECRET_NAME", "ambient-vertex"),
		VertexSecretNamespace: envOrDefault("VERTEX_SECRET_NAMESPACE", "ambient-code"),
		RunnerImageNamespace:  os.Getenv("RUNNER_IMAGE_NAMESPACE"),
		MCPImage:              os.Getenv("MCP_IMAGE"),
		MCPAPIServerURL:       envOrDefault("MCP_API_SERVER_URL", ""),
		RunnerLogLevel:        envOrDefault("RUNNER_LOG_LEVEL", "info"),
		ProjectKubeTokenFile:  os.Getenv("PROJECT_KUBE_TOKEN_FILE"),
		CPTokenListenAddr:     envOrDefault("CP_TOKEN_LISTEN_ADDR", ":8080"),
		CPTokenURL:            os.Getenv("CP_TOKEN_URL"),
		HTTPProxy:             os.Getenv("HTTP_PROXY"),
		HTTPSProxy:            os.Getenv("HTTPS_PROXY"),
		NoProxy:               os.Getenv("NO_PROXY"),
	}

	if cfg.MCPAPIServerURL == "" {
		cfg.MCPAPIServerURL = cfg.APIServerURL
	}

	if cfg.APIToken == "" && (cfg.OIDCClientID == "" || cfg.OIDCClientSecret == "") {
		return nil, fmt.Errorf("either AMBIENT_API_TOKEN or both OIDC_CLIENT_ID and OIDC_CLIENT_SECRET must be set; set AMBIENT_API_TOKEN for k8s SA token auth or OIDC_CLIENT_ID+OIDC_CLIENT_SECRET for OIDC")
	}

	switch cfg.Mode {
	case "kube", "test":
	default:
		return nil, fmt.Errorf("unknown MODE %q: must be one of kube, test", cfg.Mode)
	}

	switch cfg.PlatformMode {
	case "standard", "mpp":
	default:
		return nil, fmt.Errorf("unknown PLATFORM_MODE %q: must be one of standard, mpp", cfg.PlatformMode)
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
