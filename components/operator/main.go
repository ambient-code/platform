package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/controller"
	"ambient-code-operator/internal/handlers"
	"ambient-code-operator/internal/preflight"
)

// Build-time metadata (set via -ldflags -X during build)
var (
	GitCommit  = "unknown"
	GitBranch  = "unknown"
	GitVersion = "unknown"
	BuildDate  = "unknown"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	// Parse command line flags
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var maxConcurrentReconciles int

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 10,
		"Maximum number of concurrent Reconciles which can be run. Higher values allow more throughput but consume more resources.")
	flag.Parse()

	// Allow environment variable override for max concurrent reconciles
	if envVal := os.Getenv("MAX_CONCURRENT_RECONCILES"); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil && v > 0 {
			maxConcurrentReconciles = v
		}
	}

	// Set up logging
	opts := zap.Options{
		Development: os.Getenv("DEV_MODE") == "true",
	}
	ctrllog.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	logger := ctrllog.Log.WithName("setup")

	// Log build information
	logBuildInfo()
	logger.Info("Starting Agentic Session Operator",
		"maxConcurrentReconciles", maxConcurrentReconciles,
		"leaderElection", enableLeaderElection,
	)

	// Initialize Kubernetes clients (needed for namespace/projectsettings handlers and config)
	if err := config.InitK8sClients(); err != nil {
		logger.Error(err, "Failed to initialize Kubernetes clients")
		os.Exit(1)
	}

	// Load application configuration
	appConfig := config.LoadConfig()

	logger.Info("Configuration loaded",
		"namespace", appConfig.Namespace,
		"backendNamespace", appConfig.BackendNamespace,
		"runnerImage", appConfig.AmbientCodeRunnerImage,
	)

	// Initialize OpenTelemetry metrics
	shutdownMetrics, err := controller.InitMetrics(context.Background())
	if err != nil {
		logger.Error(err, "Failed to initialize OpenTelemetry metrics, continuing without metrics")
	} else {
		defer shutdownMetrics()
	}

	// Validate Vertex AI configuration at startup if enabled
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1" {
		if err := preflight.ValidateVertexConfig(appConfig.Namespace); err != nil {
			logger.Error(err, "Vertex AI validation failed")
			os.Exit(1)
		}
	}

	// Create controller-runtime manager with increased QPS/Burst to avoid client-side throttling
	// Default is QPS=5, Burst=10 which causes delays when handling multiple sessions
	restConfig := ctrl.GetConfigOrDie()
	restConfig.QPS = 100
	restConfig.Burst = 200

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ambient-code-operator.ambient-code.io",
	})
	if err != nil {
		logger.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	// Set up AgenticSession controller with concurrent reconcilers
	agenticSessionReconciler := controller.NewAgenticSessionReconciler(
		mgr.GetClient(),
		maxConcurrentReconciles,
	)
	if err := agenticSessionReconciler.SetupWithManager(mgr); err != nil {
		logger.Error(err, "Unable to create AgenticSession controller")
		os.Exit(1)
	}
	logger.Info("AgenticSession controller registered",
		"maxConcurrentReconciles", maxConcurrentReconciles,
	)

	// Add health check endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	// Start namespace and project settings watchers (these remain as watch loops for now)
	// Note: These could be migrated to controller-runtime controllers in the future
	go handlers.WatchNamespaces()
	go handlers.WatchProjectSettings()

	logger.Info("Starting manager with controller-runtime",
		"maxConcurrentReconciles", maxConcurrentReconciles,
	)

	// Start the manager (blocks until stopped)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func logBuildInfo() {
	log.Println("==============================================")
	log.Println("Agentic Session Operator - Build Information")
	log.Println("==============================================")
	log.Printf("Version:     %s", GitVersion)
	log.Printf("Commit:      %s", GitCommit)
	log.Printf("Branch:      %s", GitBranch)
	log.Printf("Repository:  %s", getEnvOrDefault("GIT_REPO", "unknown"))
	log.Printf("Built:       %s", BuildDate)
	log.Printf("Built by:    %s", getEnvOrDefault("BUILD_USER", "unknown"))
	log.Println("==============================================")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
