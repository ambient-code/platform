package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"ambient-code-operator/internal/config"
)

var (
	meter metric.Meter

	// cachedClient uses the controller-runtime informer cache for reads,
	// avoiding direct API server hits every 30s for gauge callbacks.
	cachedClient client.Client

	// Session lifecycle metrics (histograms)
	sessionStartupDuration metric.Float64Histogram
	sessionTotalDuration   metric.Float64Histogram
	reconcileDuration      metric.Float64Histogram
	tokenProvisionDuration metric.Float64Histogram
	imagePullDuration      metric.Float64Histogram

	// Session lifecycle metrics (counters)
	sessionPhaseTransitions metric.Int64Counter
	sessionsCompleted       metric.Int64Counter
	sessionsByUser          metric.Int64Counter
	sessionsByProject       metric.Int64Counter

	// Error metrics (counters)
	reconcileRetries   metric.Int64Counter
	sessionTimeouts    metric.Int64Counter
	s3Errors           metric.Int64Counter
	tokenRefreshErrors metric.Int64Counter
	podRestarts        metric.Int64Counter
)

// InitMetrics initializes OpenTelemetry metrics.
// Set OTEL_EXPORTER_OTLP_ENDPOINT to configure the collector address.
// Leave unset or empty to disable metrics export (no-op).
// If c is non-nil, gauge callbacks will use the informer cache instead of
// hitting the API server directly (saves a cluster-wide LIST every 30s).
func InitMetrics(ctx context.Context, c ...client.Client) (func(), error) {
	if len(c) > 0 && c[0] != nil {
		cachedClient = c[0]
	}
	// Get OTLP endpoint from environment; skip if not configured
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		log.Println("OTEL_EXPORTER_OTLP_ENDPOINT not set, metrics export disabled")
		return func() {}, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("agentic-operator"),
			semconv.ServiceVersion(os.Getenv("VERSION")),
			attribute.String("deployment.environment", os.Getenv("DEPLOYMENT_ENV")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create meter provider with periodic reader
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(30*time.Second),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Get meter for this package
	meter = meterProvider.Meter("ambient-code-operator")

	// Initialize metrics
	if err := initInstruments(); err != nil {
		return nil, fmt.Errorf("failed to initialize instruments: %w", err)
	}

	log.Println("OpenTelemetry metrics initialized, exporting to:", endpoint)

	// Return shutdown function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}, nil
}

// initInstruments creates all metric instruments
func initInstruments() error {
	var err error

	// === HISTOGRAMS (Duration metrics) ===

	// Session startup duration
	sessionStartupDuration, err = meter.Float64Histogram(
		"ambient.session.startup.duration",
		metric.WithDescription("Time from session creation (Pending) to Running phase in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionStartupDuration: %w", err)
	}

	// Session total duration
	sessionTotalDuration, err = meter.Float64Histogram(
		"ambient.session.total.duration",
		metric.WithDescription("Total time session was running (startTime to completionTime) in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionTotalDuration: %w", err)
	}

	// Reconcile duration
	reconcileDuration, err = meter.Float64Histogram(
		"ambient.reconcile.duration",
		metric.WithDescription("Time spent in reconciliation in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create reconcileDuration: %w", err)
	}

	// Token provision duration
	tokenProvisionDuration, err = meter.Float64Histogram(
		"ambient.token.provision.duration",
		metric.WithDescription("Time to provision runner token secret in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tokenProvisionDuration: %w", err)
	}

	// Image pull duration
	imagePullDuration, err = meter.Float64Histogram(
		"ambient.image.pull.duration",
		metric.WithDescription("Time to pull container image in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create imagePullDuration: %w", err)
	}

	// === COUNTERS (Lifecycle and business metrics) ===

	// Phase transitions
	sessionPhaseTransitions, err = meter.Int64Counter(
		"ambient.session.phase.transitions",
		metric.WithDescription("Total number of session phase transitions"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionPhaseTransitions: %w", err)
	}

	// Sessions completed
	sessionsCompleted, err = meter.Int64Counter(
		"ambient.sessions.completed",
		metric.WithDescription("Total number of sessions that reached terminal states"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionsCompleted: %w", err)
	}

	// Sessions by user
	sessionsByUser, err = meter.Int64Counter(
		"ambient.sessions.by_user",
		metric.WithDescription("Total sessions created per user"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionsByUser: %w", err)
	}

	// Sessions by project
	sessionsByProject, err = meter.Int64Counter(
		"ambient.sessions.by_project",
		metric.WithDescription("Total sessions created per project/namespace"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionsByProject: %w", err)
	}

	// === COUNTERS (Error metrics) ===

	// Reconcile retries
	reconcileRetries, err = meter.Int64Counter(
		"ambient.reconcile.retries",
		metric.WithDescription("Number of reconciliation retries due to errors"),
	)
	if err != nil {
		return fmt.Errorf("failed to create reconcileRetries: %w", err)
	}

	// Session timeouts
	sessionTimeouts, err = meter.Int64Counter(
		"ambient.session.timeouts",
		metric.WithDescription("Number of sessions that timed out"),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessionTimeouts: %w", err)
	}

	// S3 errors
	s3Errors, err = meter.Int64Counter(
		"ambient.s3.errors",
		metric.WithDescription("Number of S3 operation failures"),
	)
	if err != nil {
		return fmt.Errorf("failed to create s3Errors: %w", err)
	}

	// Token refresh errors
	tokenRefreshErrors, err = meter.Int64Counter(
		"ambient.token.refresh.errors",
		metric.WithDescription("Number of token refresh failures"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tokenRefreshErrors: %w", err)
	}

	// Pod restarts
	podRestarts, err = meter.Int64Counter(
		"ambient.pod.restarts",
		metric.WithDescription("Number of pod restarts"),
	)
	if err != nil {
		return fmt.Errorf("failed to create podRestarts: %w", err)
	}

	// === GAUGES (Async callbacks) ===
	if err := initGauges(); err != nil {
		return fmt.Errorf("failed to initialize gauges: %w", err)
	}

	return nil
}

// initGauges initializes gauge metrics with async callbacks
func initGauges() error {
	var err error

	// Active sessions gauge
	_, err = meter.Int64ObservableGauge(
		"ambient.sessions.active",
		metric.WithDescription("Number of currently running sessions"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			counts := countSessionsByPhase(ctx, "Running")
			for ns, count := range counts {
				o.Observe(count, metric.WithAttributes(
					attribute.String("namespace", ns),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessions.active gauge: %w", err)
	}

	// Pending sessions gauge
	_, err = meter.Int64ObservableGauge(
		"ambient.sessions.pending",
		metric.WithDescription("Number of sessions waiting to start"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			counts := countSessionsByPhase(ctx, "Pending", "Creating")
			for ns, count := range counts {
				o.Observe(count, metric.WithAttributes(
					attribute.String("namespace", ns),
				))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create sessions.pending gauge: %w", err)
	}

	// S3 storage bytes gauge
	_, err = meter.Int64ObservableGauge(
		"ambient.s3.storage.bytes",
		metric.WithDescription("Total S3 storage used per namespace in bytes"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			// TODO: Implement S3 storage calculation when S3 metrics endpoint is available
			// For now, return 0 to prevent errors
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create s3.storage.bytes gauge: %w", err)
	}

	return nil
}

// countSessionsByPhase counts sessions in the given phases, grouped by namespace.
// When cachedClient is set (controller-runtime), reads from the informer cache
// instead of making a cluster-wide LIST against the API server every 30s.
func countSessionsByPhase(ctx context.Context, phases ...string) map[string]int64 {
	counts := make(map[string]int64)

	phaseSet := make(map[string]bool)
	for _, p := range phases {
		phaseSet[p] = true
	}

	sessionList := &unstructured.UnstructuredList{}
	sessionList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "AgenticSessionList",
	})

	if cachedClient != nil {
		// Use informer cache - no API server round-trip
		if err := cachedClient.List(ctx, sessionList); err != nil {
			log.Printf("Failed to list sessions for metrics (cached): %v", err)
			return counts
		}
	} else {
		// Fallback to direct API call if cached client not available
		if config.DynamicClient == nil {
			return counts
		}
		gvr := schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}
		list, err := config.DynamicClient.Resource(gvr).List(ctx, v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to list sessions for metrics: %v", err)
			return counts
		}
		sessionList.Items = list.Items
	}

	for _, item := range sessionList.Items {
		ns := item.GetNamespace()
		status, found, _ := unstructured.NestedMap(item.Object, "status")
		if !found {
			continue
		}

		phase, ok := status["phase"].(string)
		if !ok {
			phase = "Pending"
		}

		if phaseSet[phase] {
			counts[ns]++
		}
	}

	return counts
}

// metricsEnabled returns true when instruments have been initialised.
// All Record* functions check this to avoid nil-pointer panics when
// OTEL_EXPORTER_OTLP_ENDPOINT is unset.
func metricsEnabled() bool { return meter != nil }

// Record functions for metrics

// === Duration metrics (histograms) ===

func RecordSessionStartupDuration(namespace string, duration float64) {
	if !metricsEnabled() {
		return
	}
	sessionStartupDuration.Record(context.Background(), duration,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

func RecordSessionTotalDuration(namespace string, duration float64) {
	if !metricsEnabled() {
		return
	}
	sessionTotalDuration.Record(context.Background(), duration,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

func RecordReconcileDuration(phase string, duration float64, success bool) {
	if !metricsEnabled() {
		return
	}
	successStr := "true"
	if !success {
		successStr = "false"
	}
	reconcileDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("phase", phase),
			attribute.String("success", successStr),
		))
}

func RecordTokenProvisionDuration(namespace string, duration float64) {
	if !metricsEnabled() {
		return
	}
	tokenProvisionDuration.Record(context.Background(), duration,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

func RecordImagePullDuration(namespace, image string, duration float64) {
	if !metricsEnabled() {
		return
	}
	imagePullDuration.Record(context.Background(), duration,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("image", image),
		))
}

// === Lifecycle counters ===

func RecordPhaseTransition(namespace, fromPhase, toPhase string) {
	if !metricsEnabled() {
		return
	}
	sessionPhaseTransitions.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("from_phase", fromPhase),
			attribute.String("to_phase", toPhase),
		))
}

func RecordSessionCompleted(namespace, finalPhase string) {
	if !metricsEnabled() {
		return
	}
	sessionsCompleted.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("final_phase", finalPhase),
		))
}

func RecordSessionCreatedByUser(namespace, user string) {
	if !metricsEnabled() {
		return
	}
	sessionsByUser.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("user", user),
		))
}

func RecordSessionCreatedByProject(namespace string) {
	if !metricsEnabled() {
		return
	}
	sessionsByProject.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

// === Error counters ===

func RecordReconcileRetry(namespace, phase string) {
	if !metricsEnabled() {
		return
	}
	reconcileRetries.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("phase", phase),
		))
}

func RecordSessionTimeout(namespace string) {
	if !metricsEnabled() {
		return
	}
	sessionTimeouts.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

func RecordS3Error(namespace, operation string) {
	if !metricsEnabled() {
		return
	}
	s3Errors.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("operation", operation),
		))
}

func RecordTokenRefreshError(namespace string) {
	if !metricsEnabled() {
		return
	}
	tokenRefreshErrors.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("namespace", namespace)))
}

func RecordPodRestart(namespace, session string) {
	if !metricsEnabled() {
		return
	}
	podRestarts.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("namespace", namespace),
			attribute.String("session", session),
		))
}
