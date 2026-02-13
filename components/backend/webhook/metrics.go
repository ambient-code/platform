package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebhookEventsReceived counts total webhook events received (FR-021)
	WebhookEventsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_received_total",
			Help: "Total number of GitHub webhook events received",
		},
		[]string{"event_type"}, // issue_comment, pull_request, workflow_run
	)

	// WebhookEventsAccepted counts webhook events that passed signature verification
	WebhookEventsAccepted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_accepted_total",
			Help: "Total number of webhook events accepted after signature verification",
		},
		[]string{"event_type"},
	)

	// WebhookEventsRejected counts webhook events rejected due to invalid signature
	WebhookEventsRejected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_rejected_total",
			Help: "Total number of webhook events rejected due to invalid signature or authorization",
		},
		[]string{"reason"}, // invalid_signature, not_authorized, payload_too_large, etc.
	)

	// WebhookEventsFailed counts webhook events that failed during processing
	WebhookEventsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_failed_total",
			Help: "Total number of webhook events that failed during processing after validation",
		},
		[]string{"event_type", "reason"}, // session_creation_failed, github_api_error, etc.
	)

	// WebhookSessionsCreated counts agentic sessions created from webhooks
	WebhookSessionsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_sessions_created_total",
			Help: "Total number of agentic sessions created from webhook events",
		},
		[]string{"event_type", "trigger_reason"}, // keyword, auto_review, ci_failure
	)

	// WebhookDuplicatesDetected counts duplicate delivery IDs rejected
	WebhookDuplicatesDetected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "webhook_duplicates_detected_total",
			Help: "Total number of duplicate webhook delivery IDs detected and rejected",
		},
	)

	// WebhookProcessingDuration measures webhook processing latency (added in Phase 8)
	// This will be implemented as a histogram for p50/p95/p99 tracking
	WebhookProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_processing_duration_seconds",
			Help:    "Duration of webhook event processing from receipt to completion",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0}, // 100ms, 500ms, 1s, 2s, 5s, 10s
		},
		[]string{"event_type"},
	)

	// WebhookPayloadSizeBytes tracks the size of webhook payloads (added in Phase 8)
	WebhookPayloadSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_payload_size_bytes",
			Help:    "Size of webhook payloads in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760}, // 1KB, 10KB, 100KB, 1MB, 10MB
		},
		[]string{"event_type"},
	)

	// WebhookCacheSize tracks the deduplication cache size (for monitoring)
	WebhookCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_deduplication_cache_size",
			Help: "Current number of entries in the webhook deduplication cache",
		},
	)

	// WebhookInstallationCacheSize tracks the installation verification cache size
	WebhookInstallationCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "webhook_installation_cache_size",
			Help: "Current number of entries in the installation verification cache",
		},
	)
)

// RecordWebhookReceived increments the received counter
func RecordWebhookReceived(eventType string) {
	WebhookEventsReceived.WithLabelValues(eventType).Inc()
}

// RecordWebhookAccepted increments the accepted counter
func RecordWebhookAccepted(eventType string) {
	WebhookEventsAccepted.WithLabelValues(eventType).Inc()
}

// RecordWebhookRejected increments the rejected counter with a reason
func RecordWebhookRejected(reason string) {
	WebhookEventsRejected.WithLabelValues(reason).Inc()
}

// RecordWebhookFailed increments the failed counter
func RecordWebhookFailed(eventType, reason string) {
	WebhookEventsFailed.WithLabelValues(eventType, reason).Inc()
}

// RecordSessionCreated increments the sessions created counter
func RecordSessionCreated(eventType, triggerReason string) {
	WebhookSessionsCreated.WithLabelValues(eventType, triggerReason).Inc()
}

// RecordDuplicateDetected increments the duplicate detection counter
func RecordDuplicateDetected() {
	WebhookDuplicatesDetected.Inc()
}

// RecordProcessingDuration records the duration of webhook processing
func RecordProcessingDuration(eventType string, durationSeconds float64) {
	WebhookProcessingDuration.WithLabelValues(eventType).Observe(durationSeconds)
}

// RecordPayloadSize records the size of the webhook payload
func RecordPayloadSize(eventType string, sizeBytes int) {
	WebhookPayloadSizeBytes.WithLabelValues(eventType).Observe(float64(sizeBytes))
}

// UpdateCacheSizes updates the cache size gauges (should be called periodically)
func UpdateCacheSizes(deduplicationCacheSize, installationCacheSize int) {
	WebhookCacheSize.Set(float64(deduplicationCacheSize))
	WebhookInstallationCacheSize.Set(float64(installationCacheSize))
}
