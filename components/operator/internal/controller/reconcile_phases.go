package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"ambient-code-operator/internal/handlers"
)

// recordPhaseTransition records a phase transition
func recordPhaseTransition(namespace, fromPhase, toPhase string) {
	if fromPhase == "" {
		fromPhase = "None"
	}
	if toPhase == "" {
		toPhase = "Unknown"
	}
	RecordPhaseTransition(namespace, fromPhase, toPhase)
}

// recordSessionCreated records a new session was created
func recordSessionCreated(namespace string, session *unstructured.Unstructured) {
	// Extract user from annotations
	user := "unknown"
	annotations := session.GetAnnotations()
	if annotations != nil {
		if createdBy := annotations["ambient-code.io/created-by"]; createdBy != "" {
			user = createdBy
		}
	}
	RecordSessionCreatedByUser(namespace, user)
	RecordSessionCreatedByProject(namespace)
}

// recordSessionCompleted records a session reached a terminal state
func recordSessionCompleted(namespace, finalPhase string, session *unstructured.Unstructured) {
	RecordSessionCompleted(namespace, finalPhase)
	recordSessionDuration(namespace, session)
}

// recordSessionDuration calculates and records the total session duration
func recordSessionDuration(namespace string, session *unstructured.Unstructured) {
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	if status == nil {
		return
	}

	startTimeStr, ok := status["startTime"].(string)
	if !ok || startTimeStr == "" {
		return
	}

	completionTimeStr, ok := status["completionTime"].(string)
	if !ok || completionTimeStr == "" {
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return
	}

	completionTime, err := time.Parse(time.RFC3339, completionTimeStr)
	if err != nil {
		return
	}

	duration := completionTime.Sub(startTime).Seconds()
	RecordSessionTotalDuration(namespace, duration)

	// Log the total session duration
	log.Log.Info("Session completed",
		"namespace", namespace,
		"session", session.GetName(),
		"total_duration_seconds", fmt.Sprintf("%.2f", duration),
	)
}

// recordImagePullDuration calculates and records image pull duration from pod status
func recordImagePullDuration(namespace string, pod *corev1.Pod) {
	podCreated := pod.CreationTimestamp.Time

	// Check all containers for image pull timing
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running != nil && cs.State.Running.StartedAt.After(podCreated) {
			// Approximate image pull duration as time from pod creation to container start
			// This includes scheduling + image pull + container creation
			duration := cs.State.Running.StartedAt.Sub(podCreated).Seconds()

			// Extract image name (remove tag/digest for cleaner metrics)
			image := cs.Image
			if idx := strings.Index(image, "@"); idx != -1 {
				image = image[:idx]
			} else if idx := strings.LastIndex(image, ":"); idx != -1 {
				image = image[:idx]
			}

			RecordImagePullDuration(namespace, image, duration)

			// Log for first container only (usually the runner)
			log.Log.Info("Image pull completed",
				"namespace", namespace,
				"pod", pod.Name,
				"image", image,
				"duration_seconds", fmt.Sprintf("%.2f", duration),
			)
			break // Only record for first container
		}
	}
}

// recordStartupTime calculates and records the startup duration from startTime to now
func recordStartupTime(namespace, sessionName string, session *unstructured.Unstructured) {
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	if status == nil {
		return
	}

	// Get startTime from status
	startTimeStr, ok := status["startTime"].(string)
	if !ok || startTimeStr == "" {
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		return
	}

	duration := time.Since(startTime).Seconds()
	RecordSessionStartupDuration(namespace, duration)

	// Log the startup time for visibility
	log.Log.Info("Session started",
		"namespace", namespace,
		"session", sessionName,
		"startup_duration_seconds", fmt.Sprintf("%.2f", duration),
	)
}

// reconcilePending handles sessions in Pending phase.
// This creates the runner pod and transitions to Creating phase.
func (r *AgenticSessionReconciler) reconcilePending(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	name := session.GetName()
	namespace := session.GetNamespace()

	logger.Info("Processing Pending session", "name", name, "namespace", namespace)

	// Record that a new session is being processed
	recordSessionCreated(namespace, session)

	// Check for desired-phase annotation (user-requested state transitions)
	annotations := session.GetAnnotations()
	desiredPhase := ""
	if annotations != nil {
		desiredPhase = strings.TrimSpace(annotations["ambient-code.io/desired-phase"])
	}

	// If user wants to stop, don't create pod
	if desiredPhase == "Stopped" {
		logger.Info("Session has desired-phase=Stopped, skipping pod creation", "name", name)
		recordPhaseTransition(namespace, "Pending", "Stopped")
		recordSessionCompleted(namespace, "Stopped", session)
		if err := handlers.TransitionToStopped(ctx, session); err != nil {
			logger.Error(err, "Failed to transition to Stopped", "name", name)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		return ctrl.Result{}, nil
	}

	// Delegate to existing handler logic (refactored to be called from here)
	// This preserves all the existing pod creation, secret handling, etc.
	if err := handlers.ReconcilePendingSession(ctx, session, r.appConfig); err != nil {
		logger.Error(err, "Failed to reconcile pending session", "name", name)
		RecordReconcileRetry(namespace, "Pending")
		// Requeue with backoff
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// Pod created - record transition to Creating
	recordPhaseTransition(namespace, "Pending", "Creating")

	// Pod created - requeue quickly to monitor startup
	// Pod typically ready in ~1s, so check after 2s
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

// reconcileCreating handles sessions in Creating phase.
// This monitors pod startup and transitions to Running when ready.
func (r *AgenticSessionReconciler) reconcileCreating(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	name := session.GetName()
	namespace := session.GetNamespace()
	podName := fmt.Sprintf("%s-runner", name)

	// Check if pod exists
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod doesn't exist - check if stop was requested
			annotations := session.GetAnnotations()
			if annotations != nil && annotations["ambient-code.io/desired-phase"] == "Stopped" {
				logger.Info("Pod gone and stop requested, transitioning to Stopped", "name", name)
				recordPhaseTransition(namespace, "Creating", "Stopped")
				recordSessionCompleted(namespace, "Stopped", session)
				if err := handlers.TransitionToStopped(ctx, session); err != nil {
					return ctrl.Result{RequeueAfter: 5 * time.Second}, err
				}
				return ctrl.Result{}, nil
			}

			// Pod missing unexpectedly - reset to Pending
			logger.Info("Pod missing in Creating phase, resetting to Pending", "name", name)
			RecordReconcileRetry(namespace, "Creating")
			if err := handlers.ResetToPending(ctx, session); err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get pod: %w", err)
	}

	// Check pod status and update session accordingly
	if err := handlers.UpdateSessionFromPodStatus(ctx, session, pod); err != nil {
		logger.Error(err, "Failed to update session from pod status", "name", name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// Re-fetch session to get updated status
	updatedSession := &unstructured.Unstructured{}
	updatedSession.SetGroupVersionKind(session.GroupVersionKind())
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, updatedSession); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Session deleted during reconciliation", "name", name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if phase changed
	newStatus, found, _ := unstructured.NestedMap(updatedSession.Object, "status")
	if !found {
		logger.V(1).Info("Session has no status yet", "name", name)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	newPhase, _, _ := unstructured.NestedString(newStatus, "phase")

	if newPhase == "Running" {
		// Record transition and startup time
		recordPhaseTransition(namespace, "Creating", "Running")
		recordStartupTime(namespace, name, updatedSession)
		// Record image pull duration from pod
		recordImagePullDuration(namespace, pod)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if newPhase == "Failed" {
		recordPhaseTransition(namespace, "Creating", "Failed")
		recordSessionCompleted(namespace, "Failed", updatedSession)
		return ctrl.Result{}, nil
	}

	// Still creating - requeue to continue monitoring
	// Use short interval since pods typically start quickly
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}

// reconcileRunning handles sessions in Running phase.
// This monitors for stop requests and token refresh.
func (r *AgenticSessionReconciler) reconcileRunning(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	name := session.GetName()
	namespace := session.GetNamespace()
	podName := fmt.Sprintf("%s-runner", name)

	// Check if pod still exists
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod deleted unexpectedly while Running - reset to Pending to recreate
			logger.Info("Pod missing during Running phase, resetting to Pending", "name", name)
			RecordReconcileRetry(namespace, "Running")
			if err := handlers.ResetToPending(ctx, session); err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get pod: %w", err)
	}

	// Check for desired-phase annotation (user-requested stop)
	annotations := session.GetAnnotations()
	desiredPhase := ""
	if annotations != nil {
		desiredPhase = strings.TrimSpace(annotations["ambient-code.io/desired-phase"])
	}

	if desiredPhase == "Stopped" {
		logger.Info("Stop requested for running session", "name", name)
		recordPhaseTransition(namespace, "Running", "Stopping")
		if err := handlers.InitiateStop(ctx, session); err != nil {
			logger.Error(err, "Failed to initiate stop", "name", name)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	// Check for generation drift (spec changed)
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	observedGen, _, _ := unstructured.NestedInt64(status, "observedGeneration")
	currentGen := session.GetGeneration()

	if currentGen != observedGen && observedGen != 0 {
		logger.Info("Generation drift detected, reconciling spec",
			"name", name,
			"current", currentGen,
			"observed", observedGen,
		)
		// Handle spec updates while running
		if err := handlers.ReconcileSpecChanges(ctx, session); err != nil {
			logger.Error(err, "Failed to reconcile running session spec", "name", name)
		}
	}

	// Refresh runner token if needed
	if err := handlers.EnsureFreshRunnerToken(ctx, session); err != nil {
		logger.Error(err, "Failed to refresh runner token", "name", name)
		// Non-fatal, continue
	}

	// Requeue to continue monitoring
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// reconcileStopping handles sessions in Stopping phase.
// This waits for pod deletion and transitions to Stopped.
func (r *AgenticSessionReconciler) reconcileStopping(ctx context.Context, session *unstructured.Unstructured) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	name := session.GetName()
	namespace := session.GetNamespace()
	podName := fmt.Sprintf("%s-runner", name)

	// Check if pod still exists
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod is gone - transition to Stopped
			logger.Info("Pod deleted, transitioning to Stopped", "name", name)
			recordPhaseTransition(namespace, "Stopping", "Stopped")
			recordSessionCompleted(namespace, "Stopped", session)
			if err := handlers.TransitionToStopped(ctx, session); err != nil {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get pod: %w", err)
	}

	// Pod still exists - try to delete it
	logger.Info("Pod still exists in Stopping phase, deleting", "name", name, "pod", podName)
	if err := handlers.DeletePodAndServices(ctx, namespace, podName, name); err != nil {
		logger.Error(err, "Failed to delete pod", "name", name)
	}

	// Requeue to check again
	return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
}
