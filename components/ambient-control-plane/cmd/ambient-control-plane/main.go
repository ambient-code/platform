package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-control-plane/internal/config"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/ambient/platform/components/ambient-control-plane/internal/proxy"
	"github.com/ambient/platform/components/ambient-control-plane/internal/reconciler"
	"github.com/rs/zerolog"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := setupLogger(cfg.LogLevel)

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "kube"
	}

	logger.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("api_server", cfg.APIServerURL).
		Str("mode", mode).
		Dur("poll_interval", cfg.PollInterval).
		Int("workers", cfg.WorkerCount).
		Msg("starting ambient-control-plane")

	apiClient := buildAPIClient(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	inf := informer.New(apiClient, cfg.PollInterval, logger)

	if mode == "local" {
		localCfg := config.LoadLocalConfig()

		procManager := process.NewManager(process.ManagerConfig{
			WorkspaceRoot: localCfg.WorkspaceRoot,
			RunnerCommand: localCfg.RunnerCommand,
			PortStart:     localCfg.PortRangeStart,
			PortEnd:       localCfg.PortRangeEnd,
			MaxSessions:   localCfg.MaxSessions,
			BossURL:       localCfg.BossURL,
			BossSpace:     localCfg.BossSpace,
		}, logger)

		aguiProxy := proxy.NewAGUIProxy(localCfg.ProxyAddr, procManager, logger)
		localReconciler := reconciler.NewLocalSessionReconciler(apiClient, procManager, logger)

		registerReconciler(inf, localReconciler)

		go aguiProxy.Start(ctx)
		go localReconciler.ReapLoop(ctx)

		logger.Info().
			Str("mode", "local").
			Str("proxy_addr", localCfg.ProxyAddr).
			Str("workspace_root", localCfg.WorkspaceRoot).
			Int("port_range_start", localCfg.PortRangeStart).
			Int("port_range_end", localCfg.PortRangeEnd).
			Int("max_sessions", localCfg.MaxSessions).
			Msg("running in local mode (no Kubernetes)")

		go func() {
			<-ctx.Done()
			procManager.Shutdown(context.Background())
		}()
	} else {
		kube, err := kubeclient.New(cfg.Kubeconfig, cfg.Namespace, logger)
		if err != nil {
			return fmt.Errorf("initializing kubernetes client: %w", err)
		}

		sessionReconciler := reconciler.NewSessionReconciler(apiClient, kube, logger)
		workflowReconciler := reconciler.NewWorkflowReconciler(apiClient, kube, logger)
		taskReconciler := reconciler.NewTaskReconciler(apiClient, kube, logger)
		projectReconciler := reconciler.NewProjectReconciler(apiClient, kube, logger)
		projectSettingsReconciler := reconciler.NewProjectSettingsReconciler(apiClient, kube, logger)

		registerReconciler(inf, sessionReconciler)
		registerReconciler(inf, workflowReconciler)
		registerReconciler(inf, taskReconciler)
		registerReconciler(inf, projectReconciler)
		registerReconciler(inf, projectSettingsReconciler)

		logger.Info().
			Str("mode", "kube").
			Str("namespace", cfg.Namespace).
			Msg("running in Kubernetes mode")
	}

	logger.Info().Msg("all reconcilers registered, entering run loop")

	err = inf.Run(ctx)
	if err != nil && ctx.Err() != nil {
		logger.Info().Msg("shutdown complete")
		return nil
	}
	return err
}

func buildAPIClient(cfg *config.ControlPlaneConfig) *openapi.APIClient {
	apiCfg := openapi.NewConfiguration()
	apiCfg.Servers = openapi.ServerConfigurations{
		{URL: cfg.APIServerURL, Description: "ambient-api-server"},
	}
	apiCfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}

	if cfg.APIToken != "" {
		apiCfg.AddDefaultHeader("Authorization", "Bearer "+cfg.APIToken)
	}

	return openapi.NewAPIClient(apiCfg)
}

func registerReconciler(inf *informer.Informer, rec reconciler.Reconciler) {
	inf.RegisterHandler(rec.Resource(), rec.Reconcile)
}

func setupLogger(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", "ambient-control-plane").
		Logger().
		Level(lvl)
}
