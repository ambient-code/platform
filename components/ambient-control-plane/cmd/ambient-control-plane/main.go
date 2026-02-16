package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient/platform/components/ambient-control-plane/internal/config"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/ambient/platform/components/ambient-control-plane/internal/proxy"
	"github.com/ambient/platform/components/ambient-control-plane/internal/reconciler"
	"github.com/ambient/platform/components/ambient-control-plane/internal/watcher"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		Str("grpc_server", cfg.GRPCServerAddr).
		Str("mode", mode).
		Dur("poll_interval", cfg.PollInterval).
		Int("workers", cfg.WorkerCount).
		Msg("starting ambient-control-plane")

	sdk, err := buildSDKClient(cfg)
	if err != nil {
		return fmt.Errorf("building SDK client: %w", err)
	}

	grpcConn, err := grpc.NewClient(
		cfg.GRPCServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connecting to gRPC server: %w", err)
	}
	defer grpcConn.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	watchMgr := watcher.NewWatchManager(grpcConn, logger)
	inf := informer.New(sdk, watchMgr, logger)

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
		localReconciler := reconciler.NewLocalSessionReconciler(sdk, procManager, logger)

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

		sessionReconciler := reconciler.NewSessionReconciler(sdk, kube, logger)
		projectReconciler := reconciler.NewProjectReconciler(sdk, kube, logger)
		projectSettingsReconciler := reconciler.NewProjectSettingsReconciler(sdk, kube, logger)

		registerReconciler(inf, sessionReconciler)
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

func buildSDKClient(cfg *config.ControlPlaneConfig) (*sdkclient.Client, error) {
	token := cfg.APIToken
	if token == "" {
		token = "control-plane-internal"
	}

	project := cfg.APIProject
	if project == "" {
		project = "default"
	}

	return sdkclient.NewClient(
		cfg.APIServerURL,
		token,
		project,
		sdkclient.WithTimeout(30*time.Second),
	)
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
