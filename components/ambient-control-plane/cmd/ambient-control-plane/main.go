package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/config"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/reconciler"
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

func runKubeMode(ctx context.Context, cfg *config.ControlPlaneConfig) error {
	log.Info().Msg("starting in Kubernetes mode")

	kube, err := kubeclient.New(cfg.Kubeconfig, log.Logger)
	if err != nil {
		return fmt.Errorf("creating Kubernetes client: %w", err)
	}

	factory := reconciler.NewSDKClientFactory(cfg.APIServerURL, cfg.APIToken, log.Logger)
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

	watchManager := watcher.NewWatchManager(conn, cfg.APIToken, log.Logger)

	sdk, err := sdkclient.NewClient(cfg.APIServerURL, cfg.APIToken, "default")
	if err != nil {
		return fmt.Errorf("creating SDK client: %w", err)
	}

	inf := informer.New(sdk, watchManager, log.Logger)

	projectReconciler := reconciler.NewProjectReconciler(factory, kube, log.Logger)
	projectSettingsReconciler := reconciler.NewProjectSettingsReconciler(factory, kube, log.Logger)

	inf.RegisterHandler("projects", projectReconciler.Reconcile)
	inf.RegisterHandler("project_settings", projectSettingsReconciler.Reconcile)

	sessionReconcilers := createSessionReconcilers(cfg.Reconcilers, factory, kube, kubeReconcilerCfg, log.Logger)
	for _, sessionRec := range sessionReconcilers {
		inf.RegisterHandler("sessions", sessionRec.Reconcile)
	}

	return inf.Run(ctx)
}

func createSessionReconcilers(reconcilerTypes []string, factory *reconciler.SDKClientFactory, kube *kubeclient.KubeClient, cfg reconciler.KubeReconcilerConfig, logger zerolog.Logger) []reconciler.Reconciler {
	var reconcilers []reconciler.Reconciler

	for _, reconcilerType := range reconcilerTypes {
		switch reconcilerType {
		case "kube":
			kubeReconciler := reconciler.NewKubeReconciler(factory, kube, cfg, logger)
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

	sdk, err := sdkclient.NewClient(cfg.APIServerURL, cfg.APIToken, "default")
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

	watchManager := watcher.NewWatchManager(conn, cfg.APIToken, log.Logger)
	inf := informer.New(sdk, watchManager, log.Logger)

	tallyReconciler := reconciler.NewTallyReconciler("sessions", sdk, log.Logger)
	inf.RegisterHandler("sessions", tallyReconciler.Reconcile)

	return inf.Run(ctx)
}
