package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/auth"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/config"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/keypair"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/reconciler"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/tokenserver"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/watcher"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	version   string
	buildTime string
)

func main() {
	installServiceCAIntoDefaultTransport(loadServiceCAPool())

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("log_level", level.String()).
		Str("mode", cfg.Mode).
		Str("api_server_url", cfg.APIServerURL).
		Str("grpc_server_addr", cfg.GRPCServerAddr).
		Bool("grpc_use_tls", cfg.GRPCUseTLS).
		Msg("ambient-control-plane starting")

	switch cfg.Mode {
	case "kube":
		if err := runKubeMode(ctx, cfg); err != nil {
			log.Fatal().Err(err).Msg("kube mode failed")
		}
	case "test":
		if err := runTestMode(ctx, cfg); err != nil {
			log.Fatal().Err(err).Msg("test mode failed")
		}
	default:
		log.Fatal().Str("mode", cfg.Mode).Msg("unknown mode")
	}
}

func buildTokenProvider(cfg *config.ControlPlaneConfig, logger zerolog.Logger) auth.TokenProvider {
	if cfg.OIDCClientID != "" && cfg.OIDCClientSecret != "" {
		logger.Info().
			Str("token_url", cfg.OIDCTokenURL).
			Str("client_id", cfg.OIDCClientID).
			Msg("using OIDC client credentials token provider")
		return auth.NewOIDCTokenProvider(cfg.OIDCTokenURL, cfg.OIDCClientID, cfg.OIDCClientSecret, logger)
	}
	logger.Info().Msg("using static token provider")
	return auth.NewStaticTokenProvider(cfg.APIToken)
}

func buildNamespaceProvisioner(cfg *config.ControlPlaneConfig, kube *kubeclient.KubeClient) kubeclient.NamespaceProvisioner {
	switch cfg.PlatformMode {
	case "mpp":
		log.Info().Str("config_namespace", cfg.MPPConfigNamespace).Msg("using MPP TenantNamespace provisioner")
		return kubeclient.NewMPPNamespaceProvisioner(kube, cfg.MPPConfigNamespace, log.Logger)
	default:
		log.Info().Msg("using standard Kubernetes namespace provisioner")
		return kubeclient.NewStandardNamespaceProvisioner(kube, log.Logger)
	}
}

func runKubeMode(ctx context.Context, cfg *config.ControlPlaneConfig) error {
	log.Info().Msg("starting in Kubernetes mode")

	kube, err := kubeclient.New(cfg.Kubeconfig, log.Logger)
	if err != nil {
		return fmt.Errorf("creating Kubernetes client: %w", err)
	}

	var projectKube *kubeclient.KubeClient
	if cfg.ProjectKubeTokenFile != "" {
		pk, err := kubeclient.NewFromTokenFile(cfg.ProjectKubeTokenFile, log.Logger)
		if err != nil {
			return fmt.Errorf("creating project Kubernetes client from token file: %w", err)
		}
		projectKube = pk
		log.Info().Str("token_file", cfg.ProjectKubeTokenFile).Msg("using separate project kube client")
	}

	provisionerKube := kube
	if projectKube != nil {
		provisionerKube = projectKube
	}
	provisioner := buildNamespaceProvisioner(cfg, provisionerKube)
	tokenProvider := buildTokenProvider(cfg, log.Logger)

	kp, err := keypair.EnsureKeypairSecret(ctx, provisionerKube, cfg.CPRuntimeNamespace, log.Logger)
	if err != nil {
		return fmt.Errorf("bootstrapping CP token keypair: %w", err)
	}
	log.Info().Str("namespace", cfg.CPRuntimeNamespace).Msg("CP token keypair ready")

	factory := reconciler.NewSDKClientFactory(cfg.APIServerURL, tokenProvider, log.Logger)
	kubeReconcilerCfg := reconciler.KubeReconcilerConfig{
		RunnerImage:           cfg.RunnerImage,
		BackendURL:            cfg.BackendURL,
		RunnerGRPCURL:         cfg.GRPCServerAddr,
		RunnerGRPCUseTLS:      cfg.RunnerGRPCUseTLS,
		AnthropicAPIKey:       cfg.AnthropicAPIKey,
		VertexEnabled:         cfg.VertexEnabled,
		VertexProjectID:       cfg.VertexProjectID,
		VertexRegion:          cfg.VertexRegion,
		VertexCredentialsPath: cfg.VertexCredentialsPath,
		VertexSecretName:      cfg.VertexSecretName,
		VertexSecretNamespace: cfg.VertexSecretNamespace,
		RunnerImageNamespace:  cfg.RunnerImageNamespace,
		MCPImage:              cfg.MCPImage,
		MCPAPIServerURL:       cfg.MCPAPIServerURL,
		RunnerLogLevel:        cfg.RunnerLogLevel,
		CPRuntimeNamespace:    cfg.CPRuntimeNamespace,
		CPTokenURL:            cfg.CPTokenURL,
		CPTokenPublicKey:      string(kp.PublicKeyPEM),
		HTTPProxy:             cfg.HTTPProxy,
		HTTPSProxy:            cfg.HTTPSProxy,
		NoProxy:               cfg.NoProxy,
	}

	conn, err := grpc.NewClient(cfg.GRPCServerAddr, grpc.WithTransportCredentials(grpcCredentials(cfg.GRPCUseTLS)))
	if err != nil {
		return fmt.Errorf("connecting to gRPC server: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Warn().Err(closeErr).
				Str("grpc_server_addr", cfg.GRPCServerAddr).
				Bool("grpc_use_tls", cfg.GRPCUseTLS).
				Msg("failed to close gRPC connection")
		}
	}()

	watchManager := watcher.NewWatchManager(conn, tokenProvider, log.Logger)

	initToken, err := tokenProvider.Token(ctx)
	if err != nil {
		return fmt.Errorf("resolving initial API token: %w", err)
	}

	sdk, err := sdkclient.NewClient(cfg.APIServerURL, initToken, "default")
	if err != nil {
		return fmt.Errorf("creating SDK client: %w", err)
	}

	inf := informer.New(sdk, watchManager, log.Logger)

	projectReconciler := reconciler.NewProjectReconciler(factory, kube, projectKube, provisioner, cfg.CPRuntimeNamespace, log.Logger)
	projectSettingsReconciler := reconciler.NewProjectSettingsReconciler(factory, kube, log.Logger)

	inf.RegisterHandler("projects", projectReconciler.Reconcile)
	inf.RegisterHandler("project_settings", projectSettingsReconciler.Reconcile)

	sessionReconcilers := createSessionReconcilers(cfg.Reconcilers, factory, kube, projectKube, provisioner, kubeReconcilerCfg, log.Logger)
	for _, sessionRec := range sessionReconcilers {
		inf.RegisterHandler("sessions", sessionRec.Reconcile)
	}

	tsErrCh := make(chan error, 1)
	go func() {
		tsErrCh <- startTokenServer(ctx, cfg, tokenProvider, kp)
	}()

	infErrCh := make(chan error, 1)
	go func() {
		infErrCh <- inf.Run(ctx)
	}()

	select {
	case tsErr := <-tsErrCh:
		if tsErr != nil {
			return fmt.Errorf("token server: %w", tsErr)
		}
		return <-infErrCh
	case infErr := <-infErrCh:
		return infErr
	}
}

func startTokenServer(ctx context.Context, cfg *config.ControlPlaneConfig, tokenProvider auth.TokenProvider, kp *keypair.KeyPair) error {
	privKey, err := keypair.ParsePrivateKey(kp.PrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("parsing CP token private key: %w", err)
	}
	ts, err := tokenserver.New(cfg.CPTokenListenAddr, tokenProvider, privKey, log.Logger)
	if err != nil {
		return fmt.Errorf("creating token server: %w", err)
	}
	return ts.Start(ctx)
}

func createSessionReconcilers(reconcilerTypes []string, factory *reconciler.SDKClientFactory, kube *kubeclient.KubeClient, projectKube *kubeclient.KubeClient, provisioner kubeclient.NamespaceProvisioner, cfg reconciler.KubeReconcilerConfig, logger zerolog.Logger) []reconciler.Reconciler {
	var reconcilers []reconciler.Reconciler

	for _, reconcilerType := range reconcilerTypes {
		switch reconcilerType {
		case "kube":
			kubeReconciler := reconciler.NewKubeReconciler(factory, kube, projectKube, provisioner, cfg, logger)
			reconcilers = append(reconcilers, kubeReconciler)
			log.Info().Str("type", "kube").Msg("enabled direct Kubernetes session reconciler")
		case "tally":
			tallyReconciler := reconciler.NewSessionTallyReconciler(logger)
			reconcilers = append(reconcilers, tallyReconciler)
			log.Info().Str("type", "tally").Msg("enabled session tally reconciler")
		default:
			log.Warn().Str("type", reconcilerType).Msg("unknown reconciler type, skipping")
		}
	}

	if len(reconcilers) == 0 {
		log.Warn().Msg("no valid reconcilers configured, falling back to tally reconciler")
		tallyReconciler := reconciler.NewSessionTallyReconciler(logger)
		reconcilers = append(reconcilers, tallyReconciler)
	}

	log.Info().Int("count", len(reconcilers)).Strs("types", reconcilerTypes).Msg("configured session reconcilers")
	return reconcilers
}

func loadServiceCAPool() *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if ca, readErr := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"); readErr == nil {
		pool.AppendCertsFromPEM(ca)
	}
	return pool
}

func installServiceCAIntoDefaultTransport(pool *x509.CertPool) {
	http.DefaultTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    pool,
		},
	}
}

func grpcCredentials(useTLS bool) credentials.TransportCredentials {
	if !useTLS {
		return insecure.NewCredentials()
	}
	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    loadServiceCAPool(),
	})
}

func runTestMode(ctx context.Context, cfg *config.ControlPlaneConfig) error {
	log.Info().Msg("starting in test mode")

	tokenProvider := buildTokenProvider(cfg, log.Logger)
	initToken, err := tokenProvider.Token(ctx)
	if err != nil {
		return fmt.Errorf("resolving API token: %w", err)
	}

	sdk, err := sdkclient.NewClient(cfg.APIServerURL, initToken, "default")
	if err != nil {
		return fmt.Errorf("creating SDK client: %w", err)
	}

	conn, err := grpc.NewClient(cfg.GRPCServerAddr, grpc.WithTransportCredentials(grpcCredentials(cfg.GRPCUseTLS)))
	if err != nil {
		return fmt.Errorf("connecting to gRPC server: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Warn().Err(closeErr).
				Str("grpc_server_addr", cfg.GRPCServerAddr).
				Bool("grpc_use_tls", cfg.GRPCUseTLS).
				Msg("failed to close gRPC connection")
		}
	}()

	watchManager := watcher.NewWatchManager(conn, tokenProvider, log.Logger)
	inf := informer.New(sdk, watchManager, log.Logger)

	tallyReconciler := reconciler.NewTallyReconciler("sessions", sdk, log.Logger)
	inf.RegisterHandler("sessions", tallyReconciler.Reconcile)

	return inf.Run(ctx)
}
