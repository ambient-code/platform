// Package handlers implements Kubernetes watch handlers for AgenticSession, ProjectSettings, and Namespace resources.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	authnv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

// Track which pods are currently being monitored to prevent duplicate goroutines
var (
	monitoredPods   = make(map[string]bool)
	monitoredPodsMu sync.Mutex
)

// handleAgenticSessionEvent is the legacy reconciliation function containing all session
// lifecycle logic (~2,300 lines). It's called by ReconcilePendingSession() wrapper.
//
// TODO(controller-runtime-migration): This function should be refactored into smaller,
// phase-specific reconcilers that use controller-runtime patterns. Current architecture:
// - ✅ Controller-runtime framework adopted (work queue, leader election, metrics)
// - ⚠️ Business logic still uses legacy patterns (direct API calls, manual status updates)
// - 🔜 Future: Break into ReconcilePending, ReconcileRunning, ReconcileStopped functions
//
// This transitional approach allows framework adoption without rewriting 2,300 lines at once.
func handleAgenticSessionEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()
	sessionNamespace := obj.GetNamespace()

	// Verify the resource still exists before processing (in its own namespace)
	gvr := types.GetAgenticSessionResource()
	currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping processing", name)
			return nil
		}
		return fmt.Errorf("failed to verify AgenticSession %s exists: %v", name, err)
	}

	// Create status accumulator - all status changes will be batched into a single API call
	statusPatch := NewStatusPatch(sessionNamespace, name)

	// Get the current status from the fresh object (status may be empty right after creation
	// because the API server drops .status on create when the status subresource is enabled)
	stMap, found, _ := unstructured.NestedMap(currentObj.Object, "status")
	phase := ""
	if found {
		if p, ok := stMap["phase"].(string); ok {
			phase = p
		}
	}
	// If status.phase is missing, treat as Pending and initialize it
	if phase == "" {
		statusPatch.SetField("phase", "Pending")
		statusPatch.SetField("startTime", time.Now().UTC().Format(time.RFC3339))
		if err := statusPatch.ApplyAndReset(); err != nil {
			log.Printf("Warning: failed to initialize phase: %v", err)
		}
		phase = "Pending"
	}

	// Check for desired-phase annotation (user-requested state transitions)
	annotations := currentObj.GetAnnotations()
	desiredPhase := ""
	if annotations != nil {
		desiredPhase = strings.TrimSpace(annotations["ambient-code.io/desired-phase"])
	}

	log.Printf("Processing AgenticSession %s with phase %s (desired: %s)", name, phase, desiredPhase)

	// === DESIRED PHASE RECONCILIATION ===
	// Handle user-requested state transitions via annotations

	// Handle desired-phase=Running (user wants to start/restart)
	if desiredPhase == "Running" && phase != "Running" && phase != "Creating" && phase != "Pending" {
		log.Printf("[DesiredPhase] Session %s/%s: user requested start/restart (current=%s → desired=Running)", sessionNamespace, name, phase)

		// Delete old pod if it exists (from previous run)
		podName := fmt.Sprintf("%s-runner", name)
		_, err = config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err == nil {
			log.Printf("[DesiredPhase] Cleaning up old pod %s before restart", podName)
			if err := deletePodAndPerPodService(sessionNamespace, podName, name); err != nil {
				log.Printf("[DesiredPhase] Warning: failed to cleanup old pod: %v", err)
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("[DesiredPhase] Error checking for old job: %v", err)
		}

		// Regenerate runner token if this is a continuation
		// Check if parent-session-id annotation is set
		if parentSessionID := strings.TrimSpace(annotations["vteam.ambient-code/parent-session-id"]); parentSessionID != "" {
			log.Printf("[DesiredPhase] Continuation detected (parent=%s), ensuring fresh runner token", parentSessionID)
			if err := regenerateRunnerToken(sessionNamespace, name, currentObj); err != nil {
				log.Printf("[DesiredPhase] Warning: failed to regenerate token: %v", err)
				// Non-fatal - backend may have already done it
			}
		}

		// Set phase=Pending to trigger job creation (using StatusPatch)
		// Set phase explicitly and clear completion time for restart
		statusPatch.SetField("phase", "Pending")
		statusPatch.SetField("startTime", time.Now().UTC().Format(time.RFC3339))
		statusPatch.DeleteField("completionTime")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "Restarting",
			Message: "Preparing to start session",
		})
		// Apply immediately since we need to proceed with job creation
		if err := statusPatch.ApplyAndReset(); err != nil {
			log.Printf("[DesiredPhase] Warning: failed to update status: %v", err)
		}

		// DON'T clear desired-phase annotation yet!
		// The watch may still have queued events with the old phase=Failed.
		// We'll clear it after the job is successfully created (below).
		// Only clear start-requested-at timestamp
		_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/start-requested-at")

		log.Printf("[DesiredPhase] Session %s/%s: set phase=Pending, will create job on next reconciliation", sessionNamespace, name)
		// Continue to reconciliation logic below instead of returning
		// This ensures we proceed even if the status update hasn't propagated yet
		phase = "Pending"
		// Note: Don't return early - let the code fall through to the Pending handler below
	}

	// Handle desired-phase=Stopped (user wants to stop)
	if desiredPhase == "Stopped" && (phase == "Running" || phase == "Creating") {
		log.Printf("[DesiredPhase] Session %s/%s: user requested stop (current=%s → desired=Stopped)", sessionNamespace, name, phase)

		// Delete running pod
		podName := fmt.Sprintf("%s-runner", name)
		if err := deletePodAndPerPodService(sessionNamespace, podName, name); err != nil {
			log.Printf("[DesiredPhase] Warning: failed to delete pod: %v", err)
		}

		// Set phase=Stopping explicitly (transitional state)
		// The Stopping phase handler will verify cleanup and transition to Stopped
		statusPatch.SetField("phase", "Stopping")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "Stopping",
			Message: "Session is stopping",
		})
		if err := statusPatch.Apply(); err != nil {
			log.Printf("[DesiredPhase] Warning: failed to update status: %v", err)
		}

		log.Printf("[DesiredPhase] Session %s/%s: transitioned to Stopping", sessionNamespace, name)
		// Don't clear desired-phase yet - the Stopping handler will do that after verifying cleanup
		return nil
	}

	// === STOPPING PHASE HANDLER ===
	// Complete the stop transition: verify cleanup and transition to Stopped
	if phase == "Stopping" {
		podName := fmt.Sprintf("%s-runner", name)
		_, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})

		if errors.IsNotFound(err) {
			// Pod is gone - safe to transition to Stopped
			log.Printf("[Stopping] Session %s/%s: pod deleted, transitioning to Stopped", sessionNamespace, name)

			// Set phase=Stopped explicitly
			statusPatch.SetField("phase", "Stopped")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			// Update progress-tracking conditions to reflect stopped state
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionPodCreated,
				Status:  "False",
				Reason:  "UserStopped",
				Message: "Pod deleted by user stop request",
			})
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionRunnerStarted,
				Status:  "False",
				Reason:  "UserStopped",
				Message: "Runner stopped by user",
			})

			if err := statusPatch.Apply(); err != nil {
				log.Printf("[Stopping] Warning: failed to update status: %v", err)
			}

			// Now clear the desired-phase annotation
			_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
			_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/stop-requested-at")

			log.Printf("[Stopping] Session %s/%s: transitioned to Stopped", sessionNamespace, name)
		} else if err != nil {
			// Error checking pod - log and retry next reconciliation
			log.Printf("[Stopping] Session %s/%s: error checking pod status: %v", sessionNamespace, name, err)
		} else {
			// Pod still exists - try to delete it again
			log.Printf("[Stopping] Session %s/%s: pod still exists, deleting", sessionNamespace, name)
			if err := deletePodAndPerPodService(sessionNamespace, podName, name); err != nil {
				log.Printf("[Stopping] Warning: failed to delete pod: %v", err)
			}
			// Will retry on next reconciliation
		}
		return nil
	}

	// === TEMP CONTENT POD RECONCILIATION ===
	// Manage temporary content pods for workspace access when runner is not active

	// Temp-content pods removed - users view artifacts directly from S3 bucket
	// Session state and artifacts persist in S3, accessible via bucket browser or CLI

	// Early exit for terminal phases - no reconciliation needed
	if phase == "Stopped" || phase == "Completed" || phase == "Failed" {
		return nil
	}

	// === CONTINUE WITH PHASE-BASED RECONCILIATION ===

	// Early exit: If desired-phase is "Stopped", do not recreate pods or reconcile
	// This prevents race conditions where the operator sees the pod deleted before phase is updated
	if desiredPhase == "Stopped" {
		log.Printf("Session %s has desired-phase=Stopped, skipping further reconciliation", name)
		return nil
	}

	// Handle Stopped phase - clean up running pod if it exists
	if phase == "Stopped" {
		log.Printf("Session %s is stopped, checking for running pod to clean up", name)
		podName := fmt.Sprintf("%s-runner", name)

		_, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err == nil {
			// Pod exists, delete it
			log.Printf("Pod %s is still active, cleaning up pod", podName)

			// Delete the pod
			deletePolicy := v1.DeletePropagationForeground
			err = config.K8sClient.CoreV1().Pods(sessionNamespace).Delete(context.TODO(), podName, v1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			if err != nil && !errors.IsNotFound(err) {
				log.Printf("Failed to delete pod %s: %v", podName, err)
			} else {
				log.Printf("Successfully deleted pod %s for stopped session", podName)
			}

			// Also delete any other pods labeled with this session (in case owner refs are lost)
			sessionPodSelector := fmt.Sprintf("agentic-session=%s", name)
			log.Printf("Deleting pods with agentic-session selector: %s", sessionPodSelector)
			err = config.K8sClient.CoreV1().Pods(sessionNamespace).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
				LabelSelector: sessionPodSelector,
			})
			if err != nil && !errors.IsNotFound(err) {
				log.Printf("Failed to delete session-labeled pods: %v (continuing anyway)", err)
			} else {
				log.Printf("Successfully deleted session-labeled pods")
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("Error checking pod %s: %v", podName, err)
		} else {
			log.Printf("Pod %s not found, already cleaned up", podName)
		}

		// Also cleanup ambient-vertex secret when session is stopped
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := deleteAmbientVertexSecret(deleteCtx, sessionNamespace); err != nil {
			log.Printf("Warning: Failed to cleanup %s secret from %s: %v", types.AmbientVertexSecretName, sessionNamespace, err)
			// Continue - session cleanup is still successful
		}

		// Cleanup Langfuse secret when session is stopped
		// This only deletes secrets copied by the operator (with CopiedFromAnnotation).
		// The platform-wide ambient-admin-langfuse-secret in the operator namespace is never deleted.
		if err := deleteAmbientLangfuseSecret(deleteCtx, sessionNamespace); err != nil {
			log.Printf("Warning: Failed to cleanup ambient-admin-langfuse-secret from %s: %v", sessionNamespace, err)
			// Continue - session cleanup is still successful
		}

		return nil
	}

	// Handle Running phase - check for generation changes (spec updates)
	if phase == "Running" {

		currentGeneration := currentObj.GetGeneration()
		observedGeneration := int64(0)
		if stMap != nil {
			if og, ok := stMap["observedGeneration"].(int64); ok {
				observedGeneration = og
			} else if og, ok := stMap["observedGeneration"].(float64); ok {
				observedGeneration = int64(og)
			}
		}

		if currentGeneration > observedGeneration {
			spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
			reposErr := reconcileSpecReposWithPatch(sessionNamespace, name, spec, currentObj, statusPatch)
			if reposErr != nil {
				log.Printf("[Reconcile] Failed to reconcile repos for %s/%s: %v", sessionNamespace, name, reposErr)
				// Don't update observedGeneration - will retry on next watch event
				statusPatch.AddCondition(conditionUpdate{
					Type:    "Reconciled",
					Status:  "False",
					Reason:  "RepoReconciliationFailed",
					Message: fmt.Sprintf("Failed to reconcile repos: %v", reposErr),
				})
				_ = statusPatch.Apply()
				return fmt.Errorf("repo reconciliation failed: %w", reposErr)
			}

			workflowErr := reconcileActiveWorkflowWithPatch(sessionNamespace, name, spec, currentObj, statusPatch)
			if workflowErr != nil {
				log.Printf("[Reconcile] Failed to reconcile workflow for %s/%s: %v", sessionNamespace, name, workflowErr)
				// Don't update observedGeneration - will retry on next watch event
				statusPatch.AddCondition(conditionUpdate{
					Type:    "Reconciled",
					Status:  "False",
					Reason:  "WorkflowReconciliationFailed",
					Message: fmt.Sprintf("Failed to reconcile workflow: %v", workflowErr),
				})
				_ = statusPatch.Apply()
				return fmt.Errorf("workflow reconciliation failed: %w", workflowErr)
			}

			// Update observedGeneration only if reconciliation succeeded
			statusPatch.SetField("observedGeneration", currentGeneration)
			statusPatch.AddCondition(conditionUpdate{
				Type:    "Reconciled",
				Status:  "True",
				Reason:  "SpecApplied",
				Message: fmt.Sprintf("Successfully reconciled generation %d", currentGeneration),
			})
			if err := statusPatch.Apply(); err != nil {
				log.Printf("[Reconcile] Warning: failed to apply status patch: %v", err)
			}
		}

		return nil
	}

	// Only process if status is Pending or Creating (to handle operator restarts)
	if phase != "Pending" && phase != "Creating" {
		return nil
	}

	// If in Creating phase, check if job exists
	if phase == "Creating" {
		podName := fmt.Sprintf("%s-runner", name)
		_, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err == nil {
			// Pod exists, start monitoring if not already running
			monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, podName)
			monitoredPodsMu.Lock()
			alreadyMonitoring := monitoredPods[monitorKey]
			if !alreadyMonitoring {
				monitoredPods[monitorKey] = true
				monitoredPodsMu.Unlock()
				log.Printf("Resuming monitoring for existing pod %s (session in Creating phase)", podName)
				go monitorPod(podName, name, sessionNamespace)
			} else {
				monitoredPodsMu.Unlock()
				log.Printf("Pod %s already being monitored, skipping duplicate", podName)
			}
			return nil
		} else if errors.IsNotFound(err) {
			// Pod doesn't exist but phase is Creating - check if this is due to a stop request
			if desiredPhase == "Stopped" {
				// Job already gone, can transition directly to Stopped (skip Stopping phase)
				log.Printf("Session %s in Creating phase but job not found and stop requested, transitioning to Stopped", name)
				// Set phase=Stopped explicitly
				statusPatch.SetField("phase", "Stopped")
				statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionReady,
					Status:  "False",
					Reason:  "UserStopped",
					Message: "User requested stop during pod creation",
				})
				// Update progress-tracking conditions
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionPodCreated,
					Status:  "False",
					Reason:  "UserStopped",
					Message: "Pod deleted by user stop request",
				})
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionRunnerStarted,
					Status:  "False",
					Reason:  "UserStopped",
					Message: "Runner stopped by user",
				})
				_ = statusPatch.Apply()
				_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
				_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/stop-requested-at")
				return nil
			}

			// Pod doesn't exist but phase is Creating - this is inconsistent state
			// Could happen if:
			// 1. Pod was manually deleted
			// 2. Operator crashed between pod creation and status update
			// 3. Session is being stopped and pod was deleted (stale event)

			// Before recreating, verify the session hasn't been stopped
			// Fetch fresh status to check for recent state changes
			freshObj, err := config.DynamicClient.Resource(types.GetAgenticSessionResource()).
				Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					log.Printf("Session %s was deleted, skipping recovery", name)
					return nil
				}
				log.Printf("Error fetching fresh status for %s: %v, will attempt recovery anyway", name, err)
			} else {
				// Check fresh phase - if it's Stopped/Stopping/Failed/Completed, don't recreate
				freshStatus, _, _ := unstructured.NestedMap(freshObj.Object, "status")
				freshPhase, _, _ := unstructured.NestedString(freshStatus, "phase")
				if freshPhase == "Stopped" || freshPhase == "Stopping" || freshPhase == "Failed" || freshPhase == "Completed" {
					log.Printf("Session %s is now in %s phase (stale Creating event), skipping pod recreation", name, freshPhase)
					return nil
				}
			}

			log.Printf("Session %s in Creating phase but pod not found, resetting to Pending and recreating", name)
			statusPatch.SetField("phase", "Pending")
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionPodCreated,
				Status:  "False",
				Reason:  "PodMissing",
				Message: "Pod not found, will recreate",
			})
			// Apply immediately and continue to Pending logic
			_ = statusPatch.ApplyAndReset()
			// Don't return - fall through to Pending logic to create pod
			_ = "Pending" // phase reset handled by status update
		} else {
			// Error checking pod - log and continue
			log.Printf("Error checking pod for Creating session %s: %v, will attempt recovery", name, err)
			// Fall through to Pending logic
			_ = "Pending" // phase reset handled by status update
		}
	}

	// Check for session continuation (parent session ID)
	parentSessionID := ""
	// Annotations already loaded above, reuse
	if val, ok := annotations["vteam.ambient-code/parent-session-id"]; ok {
		parentSessionID = strings.TrimSpace(val)
	}
	// Check environmentVariables as fallback
	if parentSessionID == "" {
		spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
		if envVars, found, _ := unstructured.NestedStringMap(spec, "environmentVariables"); found {
			if val, ok := envVars["PARENT_SESSION_ID"]; ok {
				parentSessionID = strings.TrimSpace(val)
			}
		}
	}

	// EmptyDir replaces PVC - session state persists in S3
	log.Printf("Session will use EmptyDir with S3 state persistence")

	// Load config for this session
	appConfig := config.LoadConfig()

	// Check for ambient-vertex secret in the operator's namespace and copy it if Vertex is enabled
	// This will be used to conditionally mount the secret as a volume
	ambientVertexSecretCopied := false
	operatorNamespace := appConfig.BackendNamespace // Assuming operator runs in same namespace as backend
	vertexEnabled := os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1"

	// Only attempt to copy the secret if Vertex AI is enabled
	if vertexEnabled {
		if ambientVertexSecret, err := config.K8sClient.CoreV1().Secrets(operatorNamespace).Get(context.TODO(), types.AmbientVertexSecretName, v1.GetOptions{}); err == nil {
			// Secret exists in operator namespace, copy it to the session namespace
			log.Printf("Found %s secret in %s, copying to %s", types.AmbientVertexSecretName, operatorNamespace, sessionNamespace)
			// Create context with timeout for secret copy operation
			copyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := copySecretToNamespace(copyCtx, ambientVertexSecret, sessionNamespace, currentObj); err != nil {
				return fmt.Errorf("failed to copy %s secret from %s to %s (CLAUDE_CODE_USE_VERTEX=1): %w", types.AmbientVertexSecretName, operatorNamespace, sessionNamespace, err)
			}
			ambientVertexSecretCopied = true
			log.Printf("Successfully copied %s secret to %s", types.AmbientVertexSecretName, sessionNamespace)
		} else if !errors.IsNotFound(err) {
			errMsg := fmt.Sprintf("Failed to check for %s secret: %v", types.AmbientVertexSecretName, err)
			statusPatch.SetField("phase", "Failed")
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionSecretsReady,
				Status:  "False",
				Reason:  "SecretCheckFailed",
				Message: errMsg,
			})
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReady,
				Status:  "False",
				Reason:  "VertexSecretError",
				Message: errMsg,
			})
			_ = statusPatch.Apply()
			return fmt.Errorf("failed to check for %s secret in %s (CLAUDE_CODE_USE_VERTEX=1): %w", types.AmbientVertexSecretName, operatorNamespace, err)
		} else {
			// Vertex enabled but secret not found - fail fast
			errMsg := fmt.Sprintf("CLAUDE_CODE_USE_VERTEX=1 but %s secret not found in namespace %s. Create it with: kubectl create secret generic %s --from-file=ambient-code-key.json=/path/to/sa.json -n %s",
				types.AmbientVertexSecretName, operatorNamespace, types.AmbientVertexSecretName, operatorNamespace)
			statusPatch.SetField("phase", "Failed")
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionSecretsReady,
				Status:  "False",
				Reason:  "VertexSecretMissing",
				Message: errMsg,
			})
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReady,
				Status:  "False",
				Reason:  "VertexSecretMissing",
				Message: "Vertex AI enabled but ambient-vertex secret not found",
			})
			_ = statusPatch.Apply()
			return fmt.Errorf("CLAUDE_CODE_USE_VERTEX=1 but %s secret not found in namespace %s", types.AmbientVertexSecretName, operatorNamespace)
		}
	} else {
		log.Printf("Vertex AI disabled (CLAUDE_CODE_USE_VERTEX=0), skipping %s secret copy", types.AmbientVertexSecretName)
	}

	// Check for Langfuse secret in the operator's namespace and copy it if enabled
	ambientLangfuseSecretCopied := false
	langfuseEnabled := os.Getenv("LANGFUSE_ENABLED") != "" && os.Getenv("LANGFUSE_ENABLED") != "0" && os.Getenv("LANGFUSE_ENABLED") != "false"

	if langfuseEnabled {
		if langfuseSecret, err := config.K8sClient.CoreV1().Secrets(operatorNamespace).Get(context.TODO(), "ambient-admin-langfuse-secret", v1.GetOptions{}); err == nil {
			// Secret exists in operator namespace, copy it to the session namespace
			log.Printf("Found ambient-admin-langfuse-secret in %s, copying to %s", operatorNamespace, sessionNamespace)
			// Create context with timeout for secret copy operation
			copyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := copySecretToNamespace(copyCtx, langfuseSecret, sessionNamespace, currentObj); err != nil {
				log.Printf("Warning: Failed to copy Langfuse secret: %v. Langfuse observability will be disabled for this session.", err)
			} else {
				ambientLangfuseSecretCopied = true
				log.Printf("Successfully copied Langfuse secret to %s", sessionNamespace)
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("Warning: Failed to check for Langfuse secret in %s: %v. Langfuse observability will be disabled for this session.", operatorNamespace, err)
		} else {
			// Langfuse enabled but secret not found - log warning and continue without Langfuse
			log.Printf("Warning: LANGFUSE_ENABLED is set but ambient-admin-langfuse-secret not found in namespace %s. Langfuse observability will be disabled for this session.", operatorNamespace)
		}
	} else {
		log.Printf("Langfuse disabled, skipping secret copy")
	}

	// Create a Kubernetes Pod for this AgenticSession
	podName := fmt.Sprintf("%s-runner", name)

	// Ensure runner token exists before creating pod
	// This handles cases where sessions are created directly via kubectl (bypassing the backend)
	// or when the backend failed to provision the token
	runnerTokenSecretName := fmt.Sprintf("ambient-runner-token-%s", name)
	if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Get(context.TODO(), runnerTokenSecretName, v1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Runner token secret %s not found, creating it now", runnerTokenSecretName)
			if err := regenerateRunnerToken(sessionNamespace, name, currentObj); err != nil {
				errMsg := fmt.Sprintf("Failed to provision runner token: %v", err)
				log.Print(errMsg)
				statusPatch.SetField("phase", "Failed")
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionReady,
					Status:  "False",
					Reason:  "TokenProvisionFailed",
					Message: errMsg,
				})
				_ = statusPatch.Apply()
				return fmt.Errorf("failed to provision runner token for session %s: %v", name, err)
			}
			log.Printf("Successfully provisioned runner token for session %s", name)
		} else {
			log.Printf("Warning: error checking runner token secret: %v", err)
		}
	}

	// Check if pod already exists in the session's namespace
	_, err = config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})
	if err == nil {
		log.Printf("Pod %s already exists for AgenticSession %s", podName, name)
		statusPatch.SetField("phase", "Creating")
		statusPatch.SetField("observedGeneration", currentObj.GetGeneration())
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionPodCreated,
			Status:  "True",
			Reason:  "PodExists",
			Message: "Runner pod already exists",
		})
		_ = statusPatch.Apply()
		// Clear desired-phase annotation if it exists (pod already created)
		_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
		return nil
	}

	// Extract spec information from the fresh object
	spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
	_ = reconcileSpecReposWithPatch(sessionNamespace, name, spec, currentObj, statusPatch)
	_ = reconcileActiveWorkflowWithPatch(sessionNamespace, name, spec, currentObj, statusPatch)
	prompt, _, _ := unstructured.NestedString(spec, "initialPrompt")
	timeout, _, _ := unstructured.NestedInt64(spec, "timeout")
	interactive, _, _ := unstructured.NestedBool(spec, "interactive")

	llmSettings, _, _ := unstructured.NestedMap(spec, "llmSettings")
	model, _, _ := unstructured.NestedString(llmSettings, "model")
	temperature, _, _ := unstructured.NestedFloat64(llmSettings, "temperature")
	maxTokens, _, _ := unstructured.NestedInt64(llmSettings, "maxTokens")

	// Hardcoded secret names (convention over configuration)
	const runnerSecretsName = "ambient-runner-secrets"               // ANTHROPIC_API_KEY only (ignored when Vertex enabled)
	const integrationSecretsName = "ambient-non-vertex-integrations" // GIT_*, JIRA_*, custom keys (optional)

	// Only check for runner secrets when Vertex is disabled
	// When Vertex is enabled, ambient-vertex secret is used instead
	if !vertexEnabled {
		if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Get(context.TODO(), runnerSecretsName, v1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				log.Printf("Error checking runner secret %s: %v", runnerSecretsName, err)
			} else {
				log.Printf("Runner secret %s missing in %s (Vertex disabled)", runnerSecretsName, sessionNamespace)
			}
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionSecretsReady,
				Status:  "False",
				Reason:  "RunnerSecretMissing",
				Message: fmt.Sprintf("Secret %s missing", runnerSecretsName),
			})
			_ = statusPatch.Apply()
			return fmt.Errorf("runner secret %s missing in namespace %s", runnerSecretsName, sessionNamespace)
		}
		log.Printf("Found runner secret %s in %s (Vertex disabled)", runnerSecretsName, sessionNamespace)
	} else {
		log.Printf("Vertex AI enabled, skipping runner secret %s validation", runnerSecretsName)
	}

	integrationSecretsExist := false
	if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Get(context.TODO(), integrationSecretsName, v1.GetOptions{}); err == nil {
		integrationSecretsExist = true
		log.Printf("Found %s secret in %s, will inject as env vars", integrationSecretsName, sessionNamespace)
	} else if !errors.IsNotFound(err) {
		log.Printf("Error checking for %s secret in %s: %v", integrationSecretsName, sessionNamespace, err)
	} else {
		log.Printf("No %s secret found in %s (optional, skipping)", integrationSecretsName, sessionNamespace)
	}

	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionSecretsReady,
		Status:  "True",
		Reason:  "AllRequiredSecretsFound",
		Message: "Runner secret available",
	})
	if integrationSecretsExist {
		statusPatch.AddCondition(conditionUpdate{
			Type:    "IntegrationSecretsReady",
			Status:  "True",
			Reason:  "OptionalSecretFound",
			Message: fmt.Sprintf("Secret %s present", integrationSecretsName),
		})
	}

	// Extract userContext for observability and auditing
	userID := ""
	userName := ""
	if userContext, found, _ := unstructured.NestedMap(spec, "userContext"); found {
		if v, ok := userContext["userId"].(string); ok {
			userID = strings.TrimSpace(v)
		}
		if v, ok := userContext["displayName"].(string); ok {
			userName = strings.TrimSpace(v)
		}
	}
	log.Printf("Session %s initiated by user: %s (userId: %s)", name, userName, userID)

	// NOTE: Google email no longer fetched by operator - runner fetches credentials at runtime
	// Runner will set USER_GOOGLE_EMAIL from backend API response in _populate_runtime_credentials()

	// Get S3 configuration for this project (from project secret or operator defaults)
	s3Endpoint, s3Bucket, s3AccessKey, s3SecretKey, err := getS3ConfigForProject(sessionNamespace, appConfig)
	if err != nil {
		log.Printf("Warning: S3 not available for project %s: %v (sessions will use ephemeral storage only)", sessionNamespace, err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    "S3Available",
			Status:  "False",
			Reason:  "NotConfigured",
			Message: fmt.Sprintf("S3 storage not configured: %v. Session state will not persist across pod restarts. Configure S3 in project settings.", err),
		})
		// Set empty values - init-hydrate and state-sync will skip S3 operations
		s3Endpoint = ""
		s3Bucket = ""
		s3AccessKey = ""
		s3SecretKey = ""
	} else {
		log.Printf("S3 configured for project %s: endpoint=%s, bucket=%s", sessionNamespace, s3Endpoint, s3Bucket)
		statusPatch.AddCondition(conditionUpdate{
			Type:    "S3Available",
			Status:  "True",
			Reason:  "Configured",
			Message: fmt.Sprintf("S3 storage configured: %s/%s", s3Endpoint, s3Bucket),
		})
	}

	// Create the Pod directly (no Job wrapper for faster startup)
	podSpec := corev1.PodSpec{
		RestartPolicy:                 corev1.RestartPolicyNever,
		TerminationGracePeriodSeconds: int64Ptr(30), // Allow time for state-sync final sync
		// Explicitly set service account for pod creation permissions
		AutomountServiceAccountToken: boolPtr(false),
		Volumes: []corev1.Volume{
			{
				Name: "workspace",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: resource.NewQuantity(10*1024*1024*1024, resource.BinarySI), // 10Gi
					},
				},
			},
		},

		// InitContainer to hydrate session state from S3
		InitContainers: []corev1.Container{
			{
				Name:            "init-hydrate",
				Image:           appConfig.StateSyncImage,
				ImagePullPolicy: appConfig.ImagePullPolicy,
				Command:         []string{"/usr/local/bin/hydrate.sh"},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
				},
				Env: func() []corev1.EnvVar {
					base := []corev1.EnvVar{
						{Name: "SESSION_NAME", Value: name},
						{Name: "NAMESPACE", Value: sessionNamespace},
						{Name: "S3_ENDPOINT", Value: s3Endpoint},
						{Name: "S3_BUCKET", Value: s3Bucket},
						{Name: "AWS_ACCESS_KEY_ID", Value: s3AccessKey},
						{Name: "AWS_SECRET_ACCESS_KEY", Value: s3SecretKey},
						// NOTE: GIT_USER_NAME and GIT_USER_EMAIL removed - auto-derived from GitHub/GitLab token via API
					}

					// Add repos JSON if present
					if repos, ok := spec["repos"].([]interface{}); ok && len(repos) > 0 {
						b, _ := json.Marshal(repos)
						base = append(base, corev1.EnvVar{Name: "REPOS_JSON", Value: string(b)})
					}

					// Add workflow info if present
					if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
						if gitURL, ok := workflow["gitUrl"].(string); ok && strings.TrimSpace(gitURL) != "" {
							base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_GIT_URL", Value: gitURL})
						}
						if branch, ok := workflow["branch"].(string); ok && strings.TrimSpace(branch) != "" {
							base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_BRANCH", Value: branch})
						}
						if path, ok := workflow["path"].(string); ok && strings.TrimSpace(path) != "" {
							base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_PATH", Value: path})
						}
					}

					// Add GitHub token for private repos
					secretName := ""
					if meta, ok := currentObj.Object["metadata"].(map[string]interface{}); ok {
						if anns, ok := meta["annotations"].(map[string]interface{}); ok {
							if v, ok := anns["ambient-code.io/runner-token-secret"].(string); ok && strings.TrimSpace(v) != "" {
								secretName = strings.TrimSpace(v)
							}
						}
					}
					if secretName == "" {
						secretName = fmt.Sprintf("ambient-runner-token-%s", name)
					}
					base = append(base, corev1.EnvVar{
						Name: "BOT_TOKEN",
						ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							Key:                  "k8s-token",
						}},
					})

					return base
				}(),
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace"},
					// SubPath mount for .claude so init container writes to same location as runner
					{Name: "workspace", MountPath: "/app/.claude", SubPath: ".claude"},
				},
			},
		},

		// Flip roles so the content writer is the main container that keeps the pod alive
		Containers: []corev1.Container{
			{
				Name:            "ambient-content",
				Image:           appConfig.ContentServiceImage,
				ImagePullPolicy: appConfig.ImagePullPolicy,
				Env: []corev1.EnvVar{
					{Name: "CONTENT_SERVICE_MODE", Value: "true"},
					{Name: "STATE_BASE_DIR", Value: "/workspace"},
				},
				// Import integration secrets as environment variables (GITHUB_TOKEN, GITLAB_TOKEN, etc.)
				EnvFrom: func() []corev1.EnvFromSource {
					if integrationSecretsExist {
						return []corev1.EnvFromSource{{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: integrationSecretsName},
							},
						}}
					}
					return nil
				}(),
				Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/health",
							Port: intstr.FromString("http"),
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
				VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
			},
			{
				Name:            "ambient-code-runner",
				Image:           appConfig.AmbientCodeRunnerImage,
				ImagePullPolicy: appConfig.ImagePullPolicy,
				// 🔒 Container-level security (SCC-compatible, no privileged capabilities)
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false), // Playwright needs to write temp files
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"}, // Drop all capabilities for security
					},
				},

				// Expose AG-UI server port for backend proxy
				Ports: []corev1.ContainerPort{{
					Name:          "agui",
					ContainerPort: 8001,
					Protocol:      corev1.ProtocolTCP,
				}},

				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace", ReadOnly: false},
					// Mount .claude directory for session state persistence (synced to S3)
					// This enables SDK's built-in resume functionality
					{Name: "workspace", MountPath: "/app/.claude", SubPath: ".claude", ReadOnly: false},
				},

				// Lifecycle hook to copy Google credentials from read-only secret mount to writable workspace
				Lifecycle: &corev1.Lifecycle{
					PostStart: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"/bin/sh", "-c",
								"mkdir -p /workspace/.google_workspace_mcp/credentials && " +
									"cp -f /app/.google_workspace_mcp/credentials/* /workspace/.google_workspace_mcp/credentials/ 2>/dev/null || true"},
						},
					},
				},

				Env: func() []corev1.EnvVar {
					base := []corev1.EnvVar{
						{Name: "DEBUG", Value: "true"},
						{Name: "INTERACTIVE", Value: fmt.Sprintf("%t", interactive)},
						{Name: "AGENTIC_SESSION_NAME", Value: name},
						{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
						// Provide session id and workspace path for the runner wrapper
						{Name: "SESSION_ID", Value: name},
						{Name: "PROJECT_NAME", Value: sessionNamespace}, // For runtime credential fetching
						{Name: "WORKSPACE_PATH", Value: "/workspace"},
						{Name: "ARTIFACTS_DIR", Value: "artifacts"},
						// AG-UI server port (must match containerPort and Service)
						{Name: "AGUI_PORT", Value: "8001"},
						// Google MCP credentials directory - uses writable workspace location
						// Credentials fetched at runtime by runner from backend API
						{Name: "GOOGLE_MCP_CREDENTIALS_DIR", Value: "/workspace/.google_workspace_mcp/credentials"},
						// Google OAuth client credentials for workspace-mcp
						{Name: "GOOGLE_OAUTH_CLIENT_ID", Value: os.Getenv("GOOGLE_OAUTH_CLIENT_ID")},
						{Name: "GOOGLE_OAUTH_CLIENT_SECRET", Value: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")},
						// NOTE: USER_GOOGLE_EMAIL set by runner at runtime from fetched credentials
					}

					// For e2e: use minimal MCP config (webfetch only, no credentials needed)
					if mcpConfigFile := os.Getenv("MCP_CONFIG_FILE"); strings.TrimSpace(mcpConfigFile) != "" {
						base = append(base, corev1.EnvVar{Name: "MCP_CONFIG_FILE", Value: mcpConfigFile})
					}

					// Add user context for observability and auditing (Langfuse userId, logs, etc.)
					if userID != "" {
						base = append(base, corev1.EnvVar{Name: "USER_ID", Value: userID})
					}
					if userName != "" {
						base = append(base, corev1.EnvVar{Name: "USER_NAME", Value: userName})
					}

					// Core session env vars
					base = append(base,
						corev1.EnvVar{Name: "INITIAL_PROMPT", Value: prompt},
						corev1.EnvVar{Name: "LLM_MODEL", Value: model},
						corev1.EnvVar{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
						corev1.EnvVar{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
						corev1.EnvVar{Name: "USE_AGUI", Value: "true"},
						corev1.EnvVar{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
						corev1.EnvVar{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", appConfig.BackendNamespace)},
						// LEGACY: WEBSOCKET_URL removed - runner now uses AG-UI server pattern (FastAPI)
						// Backend proxies to runner's HTTP endpoint instead of WebSocket
					)

					// Platform-wide Langfuse observability configuration
					// Uses secretKeyRef to prevent credential exposure in pod specs
					// Secret is copied to session namespace from operator namespace
					// All keys are optional to prevent pod startup failures if keys are missing
					if ambientLangfuseSecretCopied {
						base = append(base,
							corev1.EnvVar{
								Name: "LANGFUSE_ENABLED",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "ambient-admin-langfuse-secret"},
										Key:                  "LANGFUSE_ENABLED",
										Optional:             boolPtr(true),
									},
								},
							},
							corev1.EnvVar{
								Name: "LANGFUSE_HOST",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "ambient-admin-langfuse-secret"},
										Key:                  "LANGFUSE_HOST",
										Optional:             boolPtr(true),
									},
								},
							},
							corev1.EnvVar{
								Name: "LANGFUSE_PUBLIC_KEY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "ambient-admin-langfuse-secret"},
										Key:                  "LANGFUSE_PUBLIC_KEY",
										Optional:             boolPtr(true),
									},
								},
							},
							corev1.EnvVar{
								Name: "LANGFUSE_SECRET_KEY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: "ambient-admin-langfuse-secret"},
										Key:                  "LANGFUSE_SECRET_KEY",
										Optional:             boolPtr(true),
									},
								},
							},
						)
						log.Printf("Langfuse env vars configured via secretKeyRef for session %s", name)
					}

					// Add Vertex AI configuration only if enabled
					if vertexEnabled {
						base = append(base,
							corev1.EnvVar{Name: "CLAUDE_CODE_USE_VERTEX", Value: "1"},
							corev1.EnvVar{Name: "CLOUD_ML_REGION", Value: os.Getenv("CLOUD_ML_REGION")},
							corev1.EnvVar{Name: "ANTHROPIC_VERTEX_PROJECT_ID", Value: os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")},
							corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")},
						)
					} else {
						// Explicitly set to 0 when Vertex is disabled
						base = append(base, corev1.EnvVar{Name: "CLAUDE_CODE_USE_VERTEX", Value: "0"})
					}

					// Add PARENT_SESSION_ID if this is a continuation
					if parentSessionID != "" {
						base = append(base, corev1.EnvVar{Name: "PARENT_SESSION_ID", Value: parentSessionID})
						log.Printf("Session %s: passing PARENT_SESSION_ID=%s to runner", name, parentSessionID)
					}

					// Add IS_RESUME if this session has been started before
					// Check status.startTime - if present, this is a resume (pod recreate/restart)
					// This tells the runner to skip INITIAL_PROMPT and use continue_conversation
					if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
						if startTime, ok := status["startTime"].(string); ok && startTime != "" {
							base = append(base, corev1.EnvVar{Name: "IS_RESUME", Value: "true"})
							log.Printf("Session %s: marking as resume (IS_RESUME=true, startTime=%s)", name, startTime)
						}
					}

					// If backend annotated the session with a runner token secret, inject only BOT_TOKEN
					// Secret contains: 'k8s-token' (for CR updates)
					// Prefer annotated secret name; fallback to deterministic name
					secretName := ""
					if meta, ok := currentObj.Object["metadata"].(map[string]interface{}); ok {
						if anns, ok := meta["annotations"].(map[string]interface{}); ok {
							if v, ok := anns["ambient-code.io/runner-token-secret"].(string); ok && strings.TrimSpace(v) != "" {
								secretName = strings.TrimSpace(v)
							}
						}
					}
					if secretName == "" {
						secretName = fmt.Sprintf("ambient-runner-token-%s", name)
					}
					base = append(base, corev1.EnvVar{
						Name: "BOT_TOKEN",
						ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							Key:                  "k8s-token",
						}},
					})
					// Add CR-provided envs last (override base when same key)
					if spec, ok := currentObj.Object["spec"].(map[string]interface{}); ok {
						// Inject REPOS_JSON and MAIN_REPO_NAME from spec.repos and spec.mainRepoName if present
						if repos, ok := spec["repos"].([]interface{}); ok && len(repos) > 0 {
							// Use a minimal JSON serialization via fmt (we'll rely on client to pass REPOS_JSON too)
							// This ensures runner gets repos even if env vars weren't passed from frontend
							b, _ := json.Marshal(repos)
							base = append(base, corev1.EnvVar{Name: "REPOS_JSON", Value: string(b)})
						}
						if mrn, ok := spec["mainRepoName"].(string); ok && strings.TrimSpace(mrn) != "" {
							base = append(base, corev1.EnvVar{Name: "MAIN_REPO_NAME", Value: mrn})
						}
						// Inject MAIN_REPO_INDEX if provided
						if mriRaw, ok := spec["mainRepoIndex"]; ok {
							switch v := mriRaw.(type) {
							case int64:
								base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
							case int32:
								base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
							case int:
								base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
							case float64:
								base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", int64(v))})
							case string:
								if strings.TrimSpace(v) != "" {
									base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: v})
								}
							}
						}
						// Inject activeWorkflow environment variables if present
						if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
							if gitURL, ok := workflow["gitUrl"].(string); ok && strings.TrimSpace(gitURL) != "" {
								base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_GIT_URL", Value: gitURL})
							}
							if branch, ok := workflow["branch"].(string); ok && strings.TrimSpace(branch) != "" {
								base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_BRANCH", Value: branch})
							}
							if path, ok := workflow["path"].(string); ok && strings.TrimSpace(path) != "" {
								base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_PATH", Value: path})
							}
						}
						if envMap, ok := spec["environmentVariables"].(map[string]interface{}); ok {
							for k, v := range envMap {
								if vs, ok := v.(string); ok {
									// replace if exists
									replaced := false
									for i := range base {
										if base[i].Name == k {
											base[i].Value = vs
											replaced = true
											break
										}
									}
									if !replaced {
										base = append(base, corev1.EnvVar{Name: k, Value: vs})
									}
								}
							}
						}
					}

					return base
				}(),

				// Import secrets as environment variables
				// - integrationSecretsName: Only if exists (GIT_TOKEN, JIRA_*, custom keys)
				// - runnerSecretsName: Only when Vertex disabled (ANTHROPIC_API_KEY)
				// - ambient-langfuse-keys: Platform-wide Langfuse observability (LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, LANGFUSE_HOST, LANGFUSE_ENABLED)
				EnvFrom: func() []corev1.EnvFromSource {
					sources := []corev1.EnvFromSource{}

					// Only inject integration secrets if they exist (optional)
					if integrationSecretsExist {
						sources = append(sources, corev1.EnvFromSource{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: integrationSecretsName},
							},
						})
						log.Printf("Injecting integration secrets from '%s' for session %s", integrationSecretsName, name)
					} else {
						log.Printf("Skipping integration secrets '%s' for session %s (not found or not configured)", integrationSecretsName, name)
					}

					// Only inject runner secrets (ANTHROPIC_API_KEY) when Vertex is disabled
					if !vertexEnabled && runnerSecretsName != "" {
						sources = append(sources, corev1.EnvFromSource{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName},
							},
						})
						log.Printf("Injecting runner secrets from '%s' for session %s (Vertex disabled)", runnerSecretsName, name)
					} else if vertexEnabled && runnerSecretsName != "" {
						log.Printf("Skipping runner secrets '%s' for session %s (Vertex enabled)", runnerSecretsName, name)
					}

					return sources
				}(),

				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2000m"), // 2 cores for MCP + Claude SDK
						corev1.ResourceMemory: resource.MustParse("4Gi"),   // Increased for Playwright/Chromium + dev server
					},
				},
			},
			// S3 state-sync sidecar - syncs .claude/, artifacts/, uploads/ to S3
			{
				Name:            "state-sync",
				Image:           appConfig.StateSyncImage,
				ImagePullPolicy: appConfig.ImagePullPolicy,
				Command:         []string{"/usr/local/bin/sync.sh"},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: boolPtr(false),
					ReadOnlyRootFilesystem:   boolPtr(false),
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
				},
				Env: []corev1.EnvVar{
					{Name: "SESSION_NAME", Value: name},
					{Name: "NAMESPACE", Value: sessionNamespace},
					{Name: "S3_ENDPOINT", Value: s3Endpoint},
					{Name: "S3_BUCKET", Value: s3Bucket},
					{Name: "SYNC_INTERVAL", Value: "60"},
					{Name: "MAX_SYNC_SIZE", Value: "1073741824"}, // 1GB
					{Name: "AWS_ACCESS_KEY_ID", Value: s3AccessKey},
					{Name: "AWS_SECRET_ACCESS_KEY", Value: s3SecretKey},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace", ReadOnly: false},
					// SubPath mount for .claude so sync sidecar reads from same location as runner
					{Name: "workspace", MountPath: "/app/.claude", SubPath: ".claude", ReadOnly: false},
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"), // Increased from 200m for MCP startup
						corev1.ResourceMemory: resource.MustParse("1Gi"),   // Increased from 256Mi
					},
				},
			},
		},
	}

	if appConfig.PodFSGroup != nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{
			FSGroup:             appConfig.PodFSGroup,
			FSGroupChangePolicy: func() *corev1.PodFSGroupChangePolicy { p := corev1.FSGroupChangeOnRootMismatch; return &p }(),
		}
	}

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: sessionNamespace,
			Labels: map[string]string{
				"agentic-session": name,
				"app":             "ambient-code-runner",
			},
			// If you run a service mesh that injects sidecars and causes egress issues:
			// Annotations: map[string]string{"sidecar.istio.io/inject": "false"},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1alpha1",
					Kind:       "AgenticSession",
					Name:       currentObj.GetName(),
					UID:        currentObj.GetUID(),
					Controller: boolPtr(true),
					// Remove BlockOwnerDeletion to avoid permission issues
					// BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: podSpec,
	}

	// Note: No volume mounts needed for runner/integration secrets
	// All keys are injected as environment variables via EnvFrom above

	// If ambient-vertex secret was successfully copied, mount it as a volume
	if ambientVertexSecretCopied {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name:         "vertex",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: types.AmbientVertexSecretName}},
		})
		// Mount to the ambient-code-runner container by name
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == "ambient-code-runner" {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "vertex",
					MountPath: "/app/vertex",
					ReadOnly:  true,
				})
				log.Printf("Mounted %s secret to /app/vertex in runner container for session %s", types.AmbientVertexSecretName, name)
				break
			}
		}
	}

	// NOTE: Google credentials are now fetched at runtime via backend API
	// No longer mounting credentials.json as volume
	// This ensures tokens are always fresh and automatically refreshed
	log.Printf("Session %s will fetch Google credentials at runtime from backend API", name)

	// NOTE: Credentials are now fetched at runtime via backend API endpoints
	// No longer injecting Jira, GitLab, or GitHub credentials as env vars
	// This ensures fresh tokens for long-running sessions and automatic refresh
	log.Printf("Session %s will fetch credentials at runtime from backend API", name)

	// Do not mount runner Secret volume; runner fetches tokens on demand

	// Create the pod
	createdPod, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Create(context.TODO(), pod, v1.CreateOptions{})
	if err != nil {
		// If pod already exists, this is likely a race condition from duplicate watch events - not an error
		if errors.IsAlreadyExists(err) {
			log.Printf("Pod %s already exists (race condition), continuing", podName)
			// Clear desired-phase annotation since pod exists
			_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
			return nil
		}
		log.Printf("Failed to create pod %s: %v", podName, err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionPodCreated,
			Status:  "False",
			Reason:  "CreateFailed",
			Message: err.Error(),
		})
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "PodCreationFailed",
			Message: "Runner pod creation failed",
		})
		_ = statusPatch.Apply()
		return fmt.Errorf("failed to create pod: %v", err)
	}

	log.Printf("Created pod %s for AgenticSession %s", podName, name)
	statusPatch.SetField("phase", "Creating")
	statusPatch.SetField("observedGeneration", currentObj.GetGeneration())
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionPodCreated,
		Status:  "True",
		Reason:  "PodCreated",
		Message: "Runner pod created",
	})
	// Apply all accumulated status changes in a single API call
	if err := statusPatch.Apply(); err != nil {
		log.Printf("Warning: failed to apply status patch: %v", err)
	}

	// Clear desired-phase annotation now that pod is created
	// (This was deferred from the restart handler to avoid race conditions with stale events)
	_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
	log.Printf("[DesiredPhase] Cleared desired-phase annotation after successful pod creation")

	// Create a per-pod Service pointing to the content container
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("ambient-content-%s", name),
			Namespace: sessionNamespace,
			Labels:    map[string]string{"app": "ambient-code-runner", "agentic-session": name},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       podName,
				UID:        createdPod.UID,
				Controller: boolPtr(true),
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"agentic-session": name, "app": "ambient-code-runner"},
			Ports:    []corev1.ServicePort{{Port: 8080, TargetPort: intstr.FromString("http"), Protocol: corev1.ProtocolTCP, Name: "http"}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if _, serr := config.K8sClient.CoreV1().Services(sessionNamespace).Create(context.TODO(), svc, v1.CreateOptions{}); serr != nil && !errors.IsAlreadyExists(serr) {
		log.Printf("Failed to create per-pod content service for %s: %v", name, serr)
	}

	// Create AG-UI Service pointing to the runner's FastAPI server
	// Backend proxies AG-UI requests to this service endpoint
	aguiSvc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("session-%s", name),
			Namespace: sessionNamespace,
			Labels: map[string]string{
				"app":             "ambient-code",
				"agentic-session": name,
			},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       podName,
				UID:        createdPod.UID,
				Controller: boolPtr(true),
			}},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"agentic-session": name, "app": "ambient-code-runner"},
			Ports: []corev1.ServicePort{{
				Name:       "agui",
				Protocol:   corev1.ProtocolTCP,
				Port:       8001,
				TargetPort: intstr.FromInt(8001),
			}},
		},
	}
	if _, serr := config.K8sClient.CoreV1().Services(sessionNamespace).Create(context.TODO(), aguiSvc, v1.CreateOptions{}); serr != nil && !errors.IsAlreadyExists(serr) {
		log.Printf("Failed to create AG-UI service for %s: %v", name, serr)
	} else {
		log.Printf("Created AG-UI service session-%s for AgenticSession %s", name, name)
	}

	// Start monitoring the pod (only if not already being monitored)
	monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, podName)
	monitoredPodsMu.Lock()
	alreadyMonitoring := monitoredPods[monitorKey]
	if !alreadyMonitoring {
		monitoredPods[monitorKey] = true
		monitoredPodsMu.Unlock()
		go monitorPod(podName, name, sessionNamespace)
	} else {
		monitoredPodsMu.Unlock()
		log.Printf("Pod %s already being monitored, skipping duplicate goroutine", podName)
	}

	return nil
}

// reconcileSpecReposWithPatch is a version of reconcileSpecRepos that uses StatusPatch for batched updates.
// This is used during initial reconciliation to avoid triggering multiple watch events.
func reconcileSpecReposWithPatch(sessionNamespace, sessionName string, spec map[string]interface{}, session *unstructured.Unstructured, statusPatch *StatusPatch) error {
	repoSlice, found, _ := unstructured.NestedSlice(spec, "repos")
	if !found {
		statusPatch.DeleteField("reconciledRepos")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReposReconciled,
			Status:  "True",
			Reason:  "NoRepos",
			Message: "No repositories defined",
		})
		return nil
	}

	// Parse spec repos and deduplicate by URL (keep last occurrence)
	// When the same repo URL appears multiple times with different branches,
	// we only keep one entry (the last one) since we only have one physical clone
	specReposMap := make(map[string]map[string]string)
	for _, entry := range repoSlice {
		if repoMap, ok := entry.(map[string]interface{}); ok {
			url, _ := repoMap["url"].(string)
			if strings.TrimSpace(url) == "" {
				continue
			}
			branch := "main"
			if b, ok := repoMap["branch"].(string); ok && strings.TrimSpace(b) != "" {
				branch = b
			}
			// Use URL as key to deduplicate - last branch wins
			specReposMap[url] = map[string]string{
				"url":    url,
				"branch": branch,
			}
		}
	}

	// Convert map back to slice
	specRepos := make([]map[string]string, 0, len(specReposMap))
	for _, repo := range specReposMap {
		specRepos = append(specRepos, repo)
	}

	// Get current reconciled repos from status
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	reconciledReposRaw, _, _ := unstructured.NestedSlice(status, "reconciledRepos")
	reconciledRepos := make([]map[string]string, 0, len(reconciledReposRaw))
	for _, entry := range reconciledReposRaw {
		if repoMap, ok := entry.(map[string]interface{}); ok {
			url, _ := repoMap["url"].(string)
			branch, _ := repoMap["branch"].(string)
			if url != "" {
				reconciledRepos = append(reconciledRepos, map[string]string{
					"url":    url,
					"branch": branch,
				})
			}
		}
	}

	// Detect drift: repos added, removed, or branch changed
	toAdd := []map[string]string{}
	toRemove := []map[string]string{}
	branchChanged := []map[string]string{}

	// Find repos in spec but not in reconciled (need to add)
	// Also detect branch changes for existing repos
	for _, specRepo := range specRepos {
		found := false
		for _, reconciledRepo := range reconciledRepos {
			if specRepo["url"] == reconciledRepo["url"] {
				found = true
				// Check if branch changed
				if specRepo["branch"] != reconciledRepo["branch"] {
					branchChanged = append(branchChanged, specRepo)
				}
				break
			}
		}
		if !found {
			toAdd = append(toAdd, specRepo)
		}
	}

	// Find repos in reconciled but not in spec (need to remove)
	for _, reconciledRepo := range reconciledRepos {
		found := false
		for _, specRepo := range specRepos {
			if reconciledRepo["url"] == specRepo["url"] {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, reconciledRepo)
		}
	}

	if len(toAdd) == 0 && len(toRemove) == 0 && len(branchChanged) == 0 {
		return nil
	}

	// AG-UI pattern: Call runner's REST endpoints to update configuration
	// Runner will restart Claude SDK client with new repo configuration
	runnerBaseURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001", sessionName, sessionNamespace)

	// Add new repos
	for _, repo := range toAdd {
		repoName := deriveRepoNameFromURL(repo["url"])

		payload := map[string]interface{}{
			"url":    repo["url"],
			"branch": repo["branch"],
			"name":   repoName,
		}
		payloadBytes, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", runnerBaseURL+"/repos/add", bytes.NewReader(payloadBytes))
		if err != nil {
			log.Printf("[Reconcile] Failed to create repo add request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Reconcile] Failed to add repo via runner: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Reconcile] Runner returned %d for repo add", resp.StatusCode)
		}
	}

	// Handle branch changes for existing repos (checkout different branch)
	for _, repo := range branchChanged {
		repoName := deriveRepoNameFromURL(repo["url"])
		log.Printf("[Reconcile] Branch changed for repo '%s' to '%s', checking out new branch", repoName, repo["branch"])

		payload := map[string]interface{}{
			"url":    repo["url"],
			"branch": repo["branch"],
			"name":   repoName,
		}
		payloadBytes, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", runnerBaseURL+"/repos/add", bytes.NewReader(payloadBytes))
		if err != nil {
			log.Printf("[Reconcile] Failed to create branch change request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Reconcile] Failed to change branch via runner: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Reconcile] Runner returned %d for branch change", resp.StatusCode)
		}
	}

	// Remove repos
	for _, repo := range toRemove {
		repoName := deriveRepoNameFromURL(repo["url"])

		payload := map[string]interface{}{
			"name": repoName,
		}
		payloadBytes, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", runnerBaseURL+"/repos/remove", bytes.NewReader(payloadBytes))
		if err != nil {
			log.Printf("[Reconcile] Failed to create repo remove request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Reconcile] Failed to remove repo via runner: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[Reconcile] Runner returned %d for repo remove", resp.StatusCode)
		}
	}

	// Build simple reconciled status (frontend now polls runner directly for real-time branch info)
	reconciled := make([]interface{}, 0, len(specRepos))
	for _, repo := range specRepos {
		repoName := deriveRepoNameFromURL(repo["url"])
		reconciledEntry := map[string]interface{}{
			"url":      repo["url"],
			"name":     repoName,
			"branch":   repo["branch"], // Intended branch from spec (deprecated)
			"clonedAt": time.Now().UTC().Format(time.RFC3339),
			"status":   "Ready", // Simplified - frontend polls runner for detailed status
		}
		reconciled = append(reconciled, reconciledEntry)
	}
	statusPatch.SetField("reconciledRepos", reconciled)
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReposReconciled,
		Status:  "True",
		Reason:  "Reconciled",
		Message: fmt.Sprintf("Reconciled %d repos (added: %d, removed: %d, branch changed: %d)", len(specRepos), len(toAdd), len(toRemove), len(branchChanged)),
	})

	return nil
}

// reconcileActiveWorkflowWithPatch is a version of reconcileActiveWorkflow that uses StatusPatch for batched updates.
func reconcileActiveWorkflowWithPatch(sessionNamespace, sessionName string, spec map[string]interface{}, session *unstructured.Unstructured, statusPatch *StatusPatch) error {
	workflow, found, _ := unstructured.NestedMap(spec, "activeWorkflow")
	if !found || len(workflow) == 0 {
		statusPatch.DeleteField("reconciledWorkflow")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionWorkflowReconciled,
			Status:  "True",
			Reason:  "NotConfigured",
			Message: "No workflow selected",
		})
		return nil
	}

	gitURL, _ := workflow["gitUrl"].(string)
	branch := "main"
	if b, ok := workflow["branch"].(string); ok && strings.TrimSpace(b) != "" {
		branch = b
	}
	path, _ := workflow["path"].(string)

	if strings.TrimSpace(gitURL) == "" {
		return nil
	}

	// Get current reconciled workflow from status
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	reconciledWorkflowRaw, _, _ := unstructured.NestedMap(status, "reconciledWorkflow")
	reconciledGitURL, _ := reconciledWorkflowRaw["gitUrl"].(string)
	reconciledBranch, _ := reconciledWorkflowRaw["branch"].(string)
	reconciledPath, _ := reconciledWorkflowRaw["path"].(string)

	// Detect drift: workflow changed
	if reconciledGitURL == gitURL && reconciledBranch == branch && reconciledPath == path {
		return nil
	}

	// AG-UI pattern: Call runner's /workflow endpoint to update configuration
	// Runner will restart Claude SDK client with new workflow
	runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/workflow", sessionName, sessionNamespace)

	payload := map[string]interface{}{
		"gitUrl": gitURL,
		"branch": branch,
		"path":   path,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", runnerURL, bytes.NewReader(payloadBytes))
	if err != nil {
		log.Printf("[Reconcile] Failed to create workflow request: %v", err)
		return fmt.Errorf("failed to create workflow request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Reconcile] Failed to send workflow change to runner: %v", err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionWorkflowReconciled,
			Status:  "False",
			Reason:  "UpdateFailed",
			Message: fmt.Sprintf("Failed to notify runner: %v", err),
		})
		return fmt.Errorf("failed to update runner workflow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Reconcile] Runner returned non-200 for workflow change: %d - %s", resp.StatusCode, string(body))
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionWorkflowReconciled,
			Status:  "False",
			Reason:  "UpdateFailed",
			Message: fmt.Sprintf("Runner returned %d", resp.StatusCode),
		})
		return fmt.Errorf("runner workflow update failed: %d", resp.StatusCode)
	}

	// Update status to reflect the reconciled state (via statusPatch)
	statusPatch.SetField("reconciledWorkflow", map[string]interface{}{
		"gitUrl":    gitURL,
		"branch":    branch,
		"path":      path,
		"status":    "Active",
		"appliedAt": time.Now().UTC().Format(time.RFC3339),
	})
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionWorkflowReconciled,
		Status:  "True",
		Reason:  "Reconciled",
		Message: fmt.Sprintf("Switched to workflow %s@%s", gitURL, branch),
	})

	return nil
}

func monitorPod(podName, sessionName, sessionNamespace string) {
	monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, podName)

	// Remove from monitoring map when this goroutine exits
	defer func() {
		monitoredPodsMu.Lock()
		delete(monitoredPods, monitorKey)
		monitoredPodsMu.Unlock()
		log.Printf("Stopped monitoring pod %s (goroutine exiting)", podName)
	}()

	log.Printf("Starting pod monitoring for %s (session: %s/%s)", podName, sessionNamespace, sessionName)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Create status accumulator for this tick - all updates batched into single API call
		statusPatch := NewStatusPatch(sessionNamespace, sessionName)

		gvr := types.GetAgenticSessionResource()
		sessionObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s deleted; stopping job monitoring", sessionName)
				return
			}
			log.Printf("Failed to fetch AgenticSession %s: %v", sessionName, err)
			continue
		}

		// Check if session was stopped - exit monitor loop immediately
		sessionStatus, _, _ := unstructured.NestedMap(sessionObj.Object, "status")
		if sessionStatus != nil {
			if currentPhase, ok := sessionStatus["phase"].(string); ok && currentPhase == "Stopped" {
				log.Printf("AgenticSession %s was stopped; stopping pod monitoring", sessionName)
				return
			}
		}

		if err := ensureFreshRunnerToken(context.TODO(), sessionObj); err != nil {
			log.Printf("Failed to refresh runner token for %s/%s: %v", sessionNamespace, sessionName, err)
		}

		pod, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Pod %s deleted; stopping monitor", podName)
				return
			}
			log.Printf("Error fetching pod %s: %v", podName, err)
			continue
		}
		// Note: We don't store pod name in status (pods are ephemeral, can be recreated)
		// Use k8s-resources endpoint or kubectl for live pod info

		if pod.Spec.NodeName != "" {
			statusPatch.AddCondition(conditionUpdate{Type: conditionPodScheduled, Status: "True", Reason: "Scheduled", Message: fmt.Sprintf("Scheduled on %s", pod.Spec.NodeName)})
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			statusPatch.SetField("phase", "Completed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "Completed", Message: "Session finished"})
			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deletePodAndPerPodService(sessionNamespace, podName, sessionName)
			return
		}

		if pod.Status.Phase == corev1.PodFailed {
			// Collect detailed error message from pod and containers
			errorMsg := pod.Status.Message
			if errorMsg == "" {
				errorMsg = pod.Status.Reason
			}

			// Check init containers for errors
			for _, initStatus := range pod.Status.InitContainerStatuses {
				if initStatus.State.Terminated != nil && initStatus.State.Terminated.ExitCode != 0 {
					msg := fmt.Sprintf("Init container %s failed (exit %d): %s",
						initStatus.Name,
						initStatus.State.Terminated.ExitCode,
						initStatus.State.Terminated.Message)
					if initStatus.State.Terminated.Reason != "" {
						msg = fmt.Sprintf("%s - %s", msg, initStatus.State.Terminated.Reason)
					}
					errorMsg = msg
					break
				}
				if initStatus.State.Waiting != nil && initStatus.State.Waiting.Reason != "" {
					errorMsg = fmt.Sprintf("Init container %s: %s - %s",
						initStatus.Name,
						initStatus.State.Waiting.Reason,
						initStatus.State.Waiting.Message)
					break
				}
			}

			// Check main containers for errors if init passed
			if errorMsg == "" || errorMsg == "PodFailed" {
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
						errorMsg = fmt.Sprintf("Container %s failed (exit %d): %s - %s",
							containerStatus.Name,
							containerStatus.State.Terminated.ExitCode,
							containerStatus.State.Terminated.Reason,
							containerStatus.State.Terminated.Message)
						break
					}
					if containerStatus.State.Waiting != nil {
						errorMsg = fmt.Sprintf("Container %s: %s - %s",
							containerStatus.Name,
							containerStatus.State.Waiting.Reason,
							containerStatus.State.Waiting.Message)
						break
					}
				}
			}

			if errorMsg == "" {
				errorMsg = "Pod failed with unknown error"
			}

			log.Printf("Pod %s failed: %s", podName, errorMsg)
			statusPatch.SetField("phase", "Failed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "PodFailed", Message: errorMsg})
			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deletePodAndPerPodService(sessionNamespace, podName, sessionName)
			return
		}

		runner := getContainerStatusByName(pod, "ambient-code-runner")
		if runner == nil {
			// Apply any accumulated changes (e.g., PodScheduled) before continuing
			_ = statusPatch.Apply()
			continue
		}

		if runner.State.Running != nil {
			statusPatch.SetField("phase", "Running")
			statusPatch.AddCondition(conditionUpdate{Type: conditionRunnerStarted, Status: "True", Reason: "ContainerRunning", Message: "Runner container is executing"})
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "True", Reason: "Running", Message: "Session is running"})
			_ = statusPatch.Apply()
			continue
		}

		if runner.State.Waiting != nil {
			waiting := runner.State.Waiting
			errorStates := map[string]bool{"ImagePullBackOff": true, "ErrImagePull": true, "CrashLoopBackOff": true, "CreateContainerConfigError": true, "InvalidImageName": true}
			if errorStates[waiting.Reason] {
				msg := fmt.Sprintf("Runner waiting: %s - %s", waiting.Reason, waiting.Message)
				statusPatch.SetField("phase", "Failed")
				statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
				statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: waiting.Reason, Message: msg})
				_ = statusPatch.Apply()
				_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
				_ = deletePodAndPerPodService(sessionNamespace, podName, sessionName)
				return
			}
		}

		if runner.State.Terminated != nil {
			term := runner.State.Terminated
			now := time.Now().UTC().Format(time.RFC3339)

			statusPatch.SetField("completionTime", now)
			switch term.ExitCode {
			case 0:
				statusPatch.SetField("phase", "Completed")
				statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "Completed", Message: "Runner finished"})
			case 2:
				msg := fmt.Sprintf("Runner exited due to prerequisite failure: %s", term.Message)
				statusPatch.SetField("phase", "Failed")
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionReady,
					Status:  "False",
					Reason:  "PrerequisiteFailed",
					Message: msg,
				})
			default:
				msg := fmt.Sprintf("Runner exited with code %d: %s", term.ExitCode, term.Reason)
				if term.Message != "" {
					msg = fmt.Sprintf("%s - %s", msg, term.Message)
				}
				statusPatch.SetField("phase", "Failed")
				statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "RunnerExit", Message: msg})
			}

			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deletePodAndPerPodService(sessionNamespace, podName, sessionName)
			return
		}

		// Apply any accumulated changes at end of tick
		_ = statusPatch.Apply()
	}
}

// getContainerStatusByName returns the ContainerStatus for a given container name
func getContainerStatusByName(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

// getS3ConfigForProject reads S3 configuration from project's integration secret
// Falls back to operator defaults if not configured
func getS3ConfigForProject(namespace string, appConfig *config.Config) (endpoint, bucket, accessKey, secretKey string, err error) {
	// Try to read from project's ambient-non-vertex-integrations secret
	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(context.TODO(), "ambient-non-vertex-integrations", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return "", "", "", "", fmt.Errorf("failed to read project secret: %w", err)
	}

	// Read from project secret if available
	storageMode := "shared" // Default to shared cluster storage
	if secret != nil && secret.Data != nil {
		// Check storage mode (shared vs custom)
		if mode := string(secret.Data["STORAGE_MODE"]); mode != "" {
			storageMode = mode
		}

		// Only read custom S3 settings if in custom mode
		if storageMode == "custom" {
			if val := string(secret.Data["S3_ENDPOINT"]); val != "" {
				endpoint = val
			}
			if val := string(secret.Data["S3_BUCKET"]); val != "" {
				bucket = val
			}
			if val := string(secret.Data["S3_ACCESS_KEY"]); val != "" {
				accessKey = val
			}
			if val := string(secret.Data["S3_SECRET_KEY"]); val != "" {
				secretKey = val
			}
			log.Printf("Using custom S3 configuration for project %s", namespace)
		} else {
			log.Printf("Using shared cluster storage (MinIO) for project %s", namespace)
		}
	}

	// Use operator defaults (for shared mode or as fallback)
	if endpoint == "" {
		endpoint = appConfig.S3Endpoint
	}
	if bucket == "" {
		bucket = appConfig.S3Bucket
	}

	// If credentials still empty AND using default endpoint/bucket, use shared MinIO credentials
	// This implements "shared cluster storage" mode where users don't need to configure anything
	usingDefaults := endpoint == appConfig.S3Endpoint && bucket == appConfig.S3Bucket
	if (accessKey == "" || secretKey == "") && usingDefaults {
		// Look for minio-credentials secret in operator namespace
		minioSecret, err := config.K8sClient.CoreV1().Secrets(appConfig.BackendNamespace).Get(context.TODO(), "minio-credentials", v1.GetOptions{})
		if err == nil && minioSecret.Data != nil {
			if accessKey == "" {
				accessKey = string(minioSecret.Data["access-key"])
			}
			if secretKey == "" {
				secretKey = string(minioSecret.Data["secret-key"])
			}
			log.Printf("Using shared MinIO credentials for project %s (shared cluster storage mode)", namespace)
		} else {
			log.Printf("Warning: minio-credentials secret not found in namespace %s", appConfig.BackendNamespace)
		}
	}

	// Validate we have required config
	if endpoint == "" || bucket == "" {
		return "", "", "", "", fmt.Errorf("incomplete S3 configuration - endpoint and bucket required")
	}
	if accessKey == "" || secretKey == "" {
		return "", "", "", "", fmt.Errorf("incomplete S3 configuration - access key and secret key required")
	}

	log.Printf("S3 config for project %s: endpoint=%s, bucket=%s", namespace, endpoint, bucket)
	return endpoint, bucket, accessKey, secretKey, nil
}

// deleteJobAndPerJobService deletes the Job and its associated per-job Service
func deletePodAndPerPodService(namespace, podName, sessionName string) error {
	// Delete Service first (it has ownerRef to Pod, but delete explicitly just in case)
	svcName := fmt.Sprintf("ambient-content-%s", sessionName)
	if err := config.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), svcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-pod service %s/%s: %v", namespace, svcName, err)
	}

	// Delete AG-UI service
	aguiSvcName := fmt.Sprintf("session-%s", sessionName)
	if err := config.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), aguiSvcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete AG-UI service %s/%s: %v", namespace, aguiSvcName, err)
	}

	// Delete the Pod with background propagation
	policy := v1.DeletePropagationBackground
	if err := config.K8sClient.CoreV1().Pods(namespace).Delete(context.TODO(), podName, v1.DeleteOptions{PropagationPolicy: &policy}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete pod %s/%s: %v", namespace, podName, err)
		return err
	}

	// Delete the ambient-vertex secret if it was copied by the operator
	deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := deleteAmbientVertexSecret(deleteCtx, namespace); err != nil {
		log.Printf("Failed to delete %s secret from %s: %v", types.AmbientVertexSecretName, namespace, err)
		// Don't return error - this is a non-critical cleanup step
	}

	// Delete the Langfuse secret if it was copied by the operator
	// This only deletes secrets copied by the operator (with CopiedFromAnnotation).
	// The platform-wide ambient-admin-langfuse-secret in the operator namespace is never deleted.
	if err := deleteAmbientLangfuseSecret(deleteCtx, namespace); err != nil {
		log.Printf("Failed to delete ambient-admin-langfuse-secret from %s: %v", namespace, err)
		// Don't return error - this is a non-critical cleanup step
	}

	// NOTE: PVC is kept for all sessions and only deleted via garbage collection
	// when the session CR is deleted. This allows sessions to be restarted.

	return nil
}

// copySecretToNamespace copies a secret to a target namespace with owner references
func copySecretToNamespace(ctx context.Context, sourceSecret *corev1.Secret, targetNamespace string, ownerObj *unstructured.Unstructured) error {
	// Check if secret already exists in target namespace
	existingSecret, err := config.K8sClient.CoreV1().Secrets(targetNamespace).Get(ctx, sourceSecret.Name, v1.GetOptions{})
	secretExists := err == nil
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error checking for existing secret: %w", err)
	}

	// Determine if we should set Controller: true
	// For shared secrets (like ambient-vertex), don't set Controller: true if secret already exists
	// to avoid conflicts when multiple sessions use the same secret
	shouldSetController := true
	if secretExists {
		// Check if existing secret already has a controller reference
		for _, ownerRef := range existingSecret.OwnerReferences {
			if ownerRef.Controller != nil && *ownerRef.Controller {
				shouldSetController = false
				log.Printf("Secret %s already has a controller reference, adding non-controller reference instead", sourceSecret.Name)
				break
			}
		}
	}

	// Create owner reference
	newOwnerRef := v1.OwnerReference{
		APIVersion: ownerObj.GetAPIVersion(),
		Kind:       ownerObj.GetKind(),
		Name:       ownerObj.GetName(),
		UID:        ownerObj.GetUID(),
	}
	if shouldSetController {
		newOwnerRef.Controller = boolPtr(true)
	}

	// Create a new secret in the target namespace
	newSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      sourceSecret.Name,
			Namespace: targetNamespace,
			Labels:    sourceSecret.Labels,
			Annotations: map[string]string{
				types.CopiedFromAnnotation: fmt.Sprintf("%s/%s", sourceSecret.Namespace, sourceSecret.Name),
			},
			OwnerReferences: []v1.OwnerReference{newOwnerRef},
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	if secretExists {
		// Secret already exists, check if it needs to be updated
		log.Printf("Secret %s already exists in namespace %s, checking if update needed", sourceSecret.Name, targetNamespace)

		// Check if the existing secret has the correct owner reference
		hasOwnerRef := false
		for _, ownerRef := range existingSecret.OwnerReferences {
			if ownerRef.UID == ownerObj.GetUID() {
				hasOwnerRef = true
				break
			}
		}

		if hasOwnerRef {
			log.Printf("Secret %s already has correct owner reference, skipping", sourceSecret.Name)
			return nil
		}

		// Update the secret with owner reference using retry logic to handle race conditions
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Re-fetch the secret to get the latest version
			currentSecret, err := config.K8sClient.CoreV1().Secrets(targetNamespace).Get(ctx, sourceSecret.Name, v1.GetOptions{})
			if err != nil {
				return err
			}

			// Check again if there's already a controller reference (may have changed since last check)
			hasController := false
			for _, ownerRef := range currentSecret.OwnerReferences {
				if ownerRef.Controller != nil && *ownerRef.Controller {
					hasController = true
					break
				}
			}

			// Create a fresh owner reference based on current state
			// If there's already a controller, don't set Controller: true for the new reference
			ownerRefToAdd := newOwnerRef
			if hasController {
				ownerRefToAdd.Controller = nil
			}

			// Apply updates
			// Create a new slice to avoid mutating shared/cached data
			currentSecret.OwnerReferences = append([]v1.OwnerReference{}, currentSecret.OwnerReferences...)
			currentSecret.OwnerReferences = append(currentSecret.OwnerReferences, ownerRefToAdd)
			currentSecret.Data = sourceSecret.Data
			if currentSecret.Annotations == nil {
				currentSecret.Annotations = make(map[string]string)
			}
			currentSecret.Annotations[types.CopiedFromAnnotation] = fmt.Sprintf("%s/%s", sourceSecret.Namespace, sourceSecret.Name)

			// Attempt update
			_, err = config.K8sClient.CoreV1().Secrets(targetNamespace).Update(ctx, currentSecret, v1.UpdateOptions{})
			return err
		})
	}

	// Create the secret
	_, err = config.K8sClient.CoreV1().Secrets(targetNamespace).Create(ctx, newSecret, v1.CreateOptions{})
	return err
}

// deleteAmbientVertexSecret deletes the ambient-vertex secret from a namespace if it was copied
// and no other active sessions in the namespace still need it.
func deleteAmbientVertexSecret(ctx context.Context, namespace string) error {
	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(ctx, types.AmbientVertexSecretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Secret doesn't exist, nothing to do
			return nil
		}
		return fmt.Errorf("error checking for %s secret: %w", types.AmbientVertexSecretName, err)
	}

	// Check if this was a copied secret (has the annotation)
	if _, ok := secret.Annotations[types.CopiedFromAnnotation]; !ok {
		log.Printf("%s secret in namespace %s was not copied by operator, not deleting", types.AmbientVertexSecretName, namespace)
		return nil
	}

	// Check if there are other active sessions in this namespace that might need this secret
	// Don't delete the shared secret if other sessions are Running, Creating, or Pending
	gvr := types.GetAgenticSessionResource()
	sessions, err := config.DynamicClient.Resource(gvr).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Printf("Warning: failed to list sessions in namespace %s, skipping secret deletion: %v", namespace, err)
		return nil // Don't delete if we can't verify no other sessions need it
	}

	activeCount := 0
	for _, session := range sessions.Items {
		status, _, _ := unstructured.NestedMap(session.Object, "status")
		phase := ""
		if status != nil {
			if p, ok := status["phase"].(string); ok {
				phase = p
			}
		}
		// Count sessions that are active and might need the vertex secret
		if phase == "Running" || phase == "Creating" || phase == "Pending" {
			activeCount++
		}
	}

	if activeCount > 0 {
		log.Printf("Skipping %s secret deletion in namespace %s: %d active session(s) may still need it", types.AmbientVertexSecretName, namespace, activeCount)
		return nil
	}

	log.Printf("Deleting copied %s secret from namespace %s (no active sessions)", types.AmbientVertexSecretName, namespace)
	err = config.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, types.AmbientVertexSecretName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s secret: %w", types.AmbientVertexSecretName, err)
	}

	return nil
}

// deleteAmbientLangfuseSecret deletes the ambient-admin-langfuse-secret from a namespace if it was copied
// and no other active sessions in the namespace still need it.
func deleteAmbientLangfuseSecret(ctx context.Context, namespace string) error {
	const langfuseSecretName = "ambient-admin-langfuse-secret"
	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(ctx, langfuseSecretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Secret doesn't exist, nothing to do
			return nil
		}
		return fmt.Errorf("error checking for %s secret: %w", langfuseSecretName, err)
	}

	// Check if this was a copied secret (has the annotation)
	if _, ok := secret.Annotations[types.CopiedFromAnnotation]; !ok {
		log.Printf("%s secret in namespace %s was not copied by operator, not deleting", langfuseSecretName, namespace)
		return nil
	}

	// Check if there are other active sessions in this namespace that might need this secret
	// Don't delete the shared secret if other sessions are Running, Creating, or Pending
	gvr := types.GetAgenticSessionResource()
	sessions, err := config.DynamicClient.Resource(gvr).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Printf("Warning: failed to list sessions in namespace %s, skipping secret deletion: %v", namespace, err)
		return nil // Don't delete if we can't verify no other sessions need it
	}

	activeCount := 0
	for _, session := range sessions.Items {
		status, _, _ := unstructured.NestedMap(session.Object, "status")
		phase := ""
		if status != nil {
			if p, ok := status["phase"].(string); ok {
				phase = p
			}
		}
		// Count sessions that are active and might need the langfuse secret
		if phase == "Running" || phase == "Creating" || phase == "Pending" {
			activeCount++
		}
	}

	if activeCount > 0 {
		log.Printf("Skipping %s secret deletion in namespace %s: %d active session(s) may still need it", langfuseSecretName, namespace, activeCount)
		return nil
	}

	log.Printf("Deleting copied %s secret from namespace %s (no active sessions)", langfuseSecretName, namespace)
	err = config.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, langfuseSecretName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s secret: %w", langfuseSecretName, err)
	}

	return nil
}

// LEGACY: getBackendAPIURL removed - AG-UI migration
// Workflow and repo changes now call runner's REST endpoints directly

// deriveRepoNameFromURL extracts the repository name from a git URL
func deriveRepoNameFromURL(repoURL string) string {
	// Remove .git suffix
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Extract last path component
	parts := strings.Split(strings.TrimRight(repoURL, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "repo"
}

// pollRunnerReposStatus removed - frontend now polls runner directly via backend API
// for real-time branch information. Operator no longer needs to maintain this in CR status.

// regenerateRunnerToken provisions a fresh ServiceAccount, Role, RoleBinding, and token Secret for a session.
// This is called when restarting sessions to ensure fresh tokens.
func regenerateRunnerToken(sessionNamespace, sessionName string, session *unstructured.Unstructured) error {
	log.Printf("[TokenProvision] Regenerating runner token for %s/%s", sessionNamespace, sessionName)

	// Create owner reference
	ownerRef := v1.OwnerReference{
		APIVersion: session.GetAPIVersion(),
		Kind:       session.GetKind(),
		Name:       session.GetName(),
		UID:        session.GetUID(),
		Controller: boolPtr(true),
	}

	// Create ServiceAccount
	saName := fmt.Sprintf("ambient-session-%s", sessionName)
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:            saName,
			Namespace:       sessionNamespace,
			Labels:          map[string]string{"app": "ambient-runner"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
	}
	if _, err := config.K8sClient.CoreV1().ServiceAccounts(sessionNamespace).Create(context.TODO(), sa, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create SA: %w", err)
		}
		log.Printf("[TokenProvision] ServiceAccount %s already exists", saName)
	}

	// Create Role with least-privilege permissions
	roleName := fmt.Sprintf("ambient-session-%s-role", sessionName)
	role := &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:            roleName,
			Namespace:       sessionNamespace,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"vteam.ambient-code"},
				Resources: []string{"agenticsessions"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"selfsubjectaccessreviews"},
				Verbs:     []string{"create"},
			},
		},
	}
	if _, err := config.K8sClient.RbacV1().Roles(sessionNamespace).Create(context.TODO(), role, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing role to ensure latest permissions
			if _, err := config.K8sClient.RbacV1().Roles(sessionNamespace).Update(context.TODO(), role, v1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update Role: %w", err)
			}
			log.Printf("[TokenProvision] Updated existing Role %s", roleName)
		} else {
			return fmt.Errorf("create Role: %w", err)
		}
	}

	// Create RoleBinding
	rbName := fmt.Sprintf("ambient-session-%s-rb", sessionName)
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:            rbName,
			Namespace:       sessionNamespace,
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		RoleRef:  rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: roleName},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: saName, Namespace: sessionNamespace}},
	}
	if _, err := config.K8sClient.RbacV1().RoleBindings(sessionNamespace).Create(context.TODO(), rb, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create RoleBinding: %w", err)
		}
		log.Printf("[TokenProvision] RoleBinding %s already exists", rbName)
	}

	// Mint token
	tr := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tok, err := config.K8sClient.CoreV1().ServiceAccounts(sessionNamespace).CreateToken(context.TODO(), saName, tr, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("mint token: %w", err)
	}
	k8sToken := strings.TrimSpace(tok.Status.Token)
	if k8sToken == "" {
		return fmt.Errorf("received empty token for SA %s", saName)
	}

	// Store token in Secret
	secretName := fmt.Sprintf("ambient-runner-token-%s", sessionName)
	refreshedAt := time.Now().UTC().Format(time.RFC3339)
	sec := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:            secretName,
			Namespace:       sessionNamespace,
			Labels:          map[string]string{"app": "ambient-runner-token"},
			OwnerReferences: []v1.OwnerReference{ownerRef},
			Annotations: map[string]string{
				"ambient-code.io/token-refreshed-at": refreshedAt,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"k8s-token": []byte(k8sToken),
		},
	}

	// Create or update secret
	if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Create(context.TODO(), sec, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			existing, getErr := config.K8sClient.CoreV1().Secrets(sessionNamespace).Get(context.TODO(), secretName, v1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("get Secret for update: %w", getErr)
			}
			secretCopy := existing.DeepCopy()
			if secretCopy.Data == nil {
				secretCopy.Data = map[string][]byte{}
			}
			secretCopy.Data["k8s-token"] = []byte(k8sToken)
			if secretCopy.Annotations == nil {
				secretCopy.Annotations = map[string]string{}
			}
			secretCopy.Annotations["ambient-code.io/token-refreshed-at"] = refreshedAt
			if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Update(context.TODO(), secretCopy, v1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update Secret: %w", err)
			}
			log.Printf("[TokenProvision] Updated secret %s with fresh token", secretName)
		} else {
			return fmt.Errorf("create Secret: %w", err)
		}
	} else {
		log.Printf("[TokenProvision] Created secret %s with runner token", secretName)
	}

	// Annotate session with secret/SA names
	sessionAnnotations := session.GetAnnotations()
	if sessionAnnotations == nil {
		sessionAnnotations = make(map[string]string)
	}
	sessionAnnotations["ambient-code.io/runner-token-secret"] = secretName
	sessionAnnotations["ambient-code.io/runner-sa"] = saName
	if err := updateAnnotations(sessionNamespace, sessionName, sessionAnnotations); err != nil {
		log.Printf("[TokenProvision] Warning: failed to annotate session: %v", err)
		// Non-fatal - job will use default names
	}

	log.Printf("[TokenProvision] Successfully regenerated token for session %s/%s", sessionNamespace, sessionName)
	return nil
}

// NOTE: Sync functions below are now unused - credentials are fetched at runtime via backend API
// This supersedes PR #562's volume mounting approach with just-in-time credential fetching

// Helper functions
var (
	boolPtr  = func(b bool) *bool { return &b }
	int64Ptr = func(i int64) *int64 { return &i }
)
