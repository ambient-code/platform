// Package controller implements Kubernetes controllers using controller-runtime.
// This provides work queues, concurrent reconciliation, rate limiting, and proper
// event handling that scales much better than raw watch loops.
package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/handlers"
	optypes "ambient-code-operator/internal/types"
)

// AgenticSessionReconciler reconciles AgenticSession resources.
// It uses controller-runtime's work queue and concurrent reconcilers
// for better performance under load.
type AgenticSessionReconciler struct {
	client.Client

	// MaxConcurrentReconciles controls how many sessions can be reconciled in parallel.
	// Higher values allow more throughput but consume more resources.
	MaxConcurrentReconciles int

	// appConfig holds operator configuration (images, namespaces, etc.)
	appConfig *config.Config
}

// NewAgenticSessionReconciler creates a new reconciler with the given configuration.
func NewAgenticSessionReconciler(c client.Client, maxConcurrent int) *AgenticSessionReconciler {
	return &AgenticSessionReconciler{
		Client:                  c,
		MaxConcurrentReconciles: maxConcurrent,
		appConfig:               config.LoadConfig(),
	}
}

// Reconcile handles a single AgenticSession reconciliation.
// This is called from the work queue, NOT directly from watch events.
// The work queue provides:
// - Event deduplication (multiple rapid events = single reconcile)
// - Rate limiting (prevents API server overload)
// - Automatic retries with exponential backoff
func (r *AgenticSessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	reconcileStart := time.Now()

	// Fetch the AgenticSession
	session := &unstructured.Unstructured{}
	session.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "AgenticSession",
	})

	if err := r.Get(ctx, req.NamespacedName, session); err != nil {
		if errors.IsNotFound(err) {
			// Object deleted - cleanup is handled by OwnerReferences
			logger.V(1).Info("AgenticSession deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get AgenticSession: %w", err)
	}

	// Check if namespace is managed
	if !r.isNamespaceManaged(ctx, session.GetNamespace()) {
		logger.V(2).Info("Skipping unmanaged namespace", "namespace", session.GetNamespace())
		return ctrl.Result{}, nil
	}

	// Get current phase
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	phase := ""
	if status != nil {
		if p, ok := status["phase"].(string); ok {
			phase = p
		}
	}

	logger.V(1).Info("Reconciling AgenticSession",
		"name", session.GetName(),
		"namespace", session.GetNamespace(),
		"phase", phase,
	)

	// Delegate to the appropriate phase handler
	// Each handler returns a Result indicating whether to requeue
	var result ctrl.Result
	var err error

	switch phase {
	case "", "Pending":
		result, err = r.reconcilePending(ctx, session)
	case "Creating":
		result, err = r.reconcileCreating(ctx, session)
	case "Running":
		result, err = r.reconcileRunning(ctx, session)
	case "Stopping":
		result, err = r.reconcileStopping(ctx, session)
	case "Stopped", "Completed", "Failed":
		// Check if user wants to restart (desired-phase=Running)
		annotations := session.GetAnnotations()
		if annotations != nil && annotations["ambient-code.io/desired-phase"] == "Running" {
			logger.Info("Restarting session from terminal phase",
				"name", session.GetName(),
				"currentPhase", phase,
			)
			// Reset to Pending to restart the session
			if err := handlers.ResetToPending(ctx, session); err != nil {
				logger.Error(err, "Failed to reset session to Pending for restart")
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
			// Requeue to process the Pending phase
			return ctrl.Result{Requeue: true}, nil
		}
		// No restart requested - terminal phases, no action needed
		result, err = ctrl.Result{}, nil
	default:
		logger.Info("Unknown phase", "phase", phase)
		result, err = ctrl.Result{}, nil
	}

	// Record reconcile duration metric
	reconcileDuration := time.Since(reconcileStart).Seconds()
	success := err == nil
	if phase == "" {
		phase = "Pending" // Normalize empty phase
	}
	RecordReconcileDuration(phase, reconcileDuration, success)

	if err != nil {
		logger.Error(err, "Reconciliation failed",
			"name", session.GetName(),
			"phase", phase,
		)
		// Requeue with backoff on error
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	return result, nil
}

// isNamespaceManaged checks if the namespace has the managed label
func (r *AgenticSessionReconciler) isNamespaceManaged(ctx context.Context, namespace string) bool {
	ns := &unstructured.Unstructured{}
	ns.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	})

	if err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		return false
	}

	labels := ns.GetLabels()
	return labels != nil && labels["ambient-code.io/managed"] == "true"
}

// SetupWithManager sets up the controller with the Manager.
// This configures:
// - The work queue with rate limiting
// - Concurrent reconcilers for parallel processing
// - Watch predicates to filter events
func (r *AgenticSessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Get max concurrent reconciles from env or use default
	maxConcurrent := r.MaxConcurrentReconciles
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default to 10 concurrent reconcilers
	}

	// Create the controller with concurrency settings
	c, err := controller.New("agenticsession-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: maxConcurrent,
		// RateLimiter uses the default workqueue.DefaultControllerRateLimiter()
		// which provides exponential backoff: 5ms, 10ms, 20ms... up to 1000s max
	})
	if err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	// Watch AgenticSessions with predicates to filter unnecessary events
	agenticSessionGVK := schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "AgenticSession",
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(agenticSessionGVK)

	// Create typed predicates for *unstructured.Unstructured
	// This reduces work queue pressure by skipping events we don't care about
	typedPredicates := predicate.TypedFuncs[*unstructured.Unstructured]{
		CreateFunc: func(e event.TypedCreateEvent[*unstructured.Unstructured]) bool {
			return true // Always process Create events
		},
		UpdateFunc: func(e event.TypedUpdateEvent[*unstructured.Unstructured]) bool {
			// Process if generation changed (spec update)
			if e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration() {
				return true
			}
			// Process if annotations changed (desired-phase, etc.)
			oldAnns := e.ObjectOld.GetAnnotations()
			newAnns := e.ObjectNew.GetAnnotations()
			if !mapsEqual(oldAnns, newAnns) {
				return true
			}
			// Process if status changed (phase transitions)
			oldStatus, _, _ := unstructured.NestedMap(e.ObjectOld.Object, "status")
			newStatus, _, _ := unstructured.NestedMap(e.ObjectNew.Object, "status")
			oldPhase, _ := oldStatus["phase"].(string)
			newPhase, _ := newStatus["phase"].(string)
			return oldPhase != newPhase
		},
		DeleteFunc: func(e event.TypedDeleteEvent[*unstructured.Unstructured]) bool {
			return true // Always process Delete events
		},
		GenericFunc: func(e event.TypedGenericEvent[*unstructured.Unstructured]) bool {
			return true
		},
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), u, &handler.TypedEnqueueRequestForObject[*unstructured.Unstructured]{}, typedPredicates),
	); err != nil {
		return fmt.Errorf("unable to watch AgenticSessions: %w", err)
	}

	// Watch Pods and trigger reconcile on parent AgenticSession
	// This eliminates polling delays - we react immediately when pod status changes
	podHandler := handler.TypedEnqueueRequestForOwner[*corev1.Pod](
		mgr.GetScheme(),
		mgr.GetRESTMapper(),
		u, // Owner type: AgenticSession
		handler.OnlyControllerOwner(),
	)
	podPredicate := predicate.TypedFuncs[*corev1.Pod]{
		CreateFunc: func(e event.TypedCreateEvent[*corev1.Pod]) bool {
			return strings.HasSuffix(e.Object.Name, "-runner")
		},
		UpdateFunc: func(e event.TypedUpdateEvent[*corev1.Pod]) bool {
			// Only trigger on status changes for runner pods
			if !strings.HasSuffix(e.ObjectNew.Name, "-runner") {
				return false
			}
			// Trigger if phase changed
			return e.ObjectOld.Status.Phase != e.ObjectNew.Status.Phase
		},
		DeleteFunc: func(e event.TypedDeleteEvent[*corev1.Pod]) bool {
			return strings.HasSuffix(e.Object.Name, "-runner")
		},
		GenericFunc: func(e event.TypedGenericEvent[*corev1.Pod]) bool {
			return false
		},
	}
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &corev1.Pod{}, podHandler, podPredicate),
	); err != nil {
		// Non-fatal: fall back to polling if pod watch fails
		log.Log.Info("Warning: unable to watch Pods, falling back to polling", "error", err)
	}

	return nil
}

// GetGVR returns the GroupVersionResource for AgenticSession
func GetGVR() schema.GroupVersionResource {
	return optypes.GetAgenticSessionResource()
}

// mapsEqual compares two string maps for equality
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
