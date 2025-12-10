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
	"ambient-code-operator/internal/services"
	"ambient-code-operator/internal/types"

	authnv1 "k8s.io/api/authentication/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

// Track which jobs are currently being monitored to prevent duplicate goroutines
var (
	monitoredJobs   = make(map[string]bool)
	monitoredJobsMu sync.Mutex
)

// WatchAgenticSessions watches for AgenticSession custom resources and creates jobs
func WatchAgenticSessions() {
	gvr := types.GetAgenticSessionResource()

	for {
		// Watch AgenticSessions across all namespaces
		watcher, err := config.DynamicClient.Resource(gvr).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create AgenticSession watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for AgenticSession events across all namespaces...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Only process resources in managed namespaces
				ns := obj.GetNamespace()
				if ns == "" {
					continue
				}
				nsObj, err := config.K8sClient.CoreV1().Namespaces().Get(context.TODO(), ns, v1.GetOptions{})
				if err != nil {
					log.Printf("Failed to get namespace %s: %v", ns, err)
					continue
				}
				if nsObj.Labels["ambient-code.io/managed"] != "true" {
					// Skip unmanaged namespaces
					continue
				}

				// Add small delay to avoid race conditions with rapid create/delete cycles
				time.Sleep(100 * time.Millisecond)

				if err := handleAgenticSessionEvent(obj); err != nil {
					log.Printf("Error handling AgenticSession event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				sessionName := obj.GetName()
				sessionNamespace := obj.GetNamespace()
				log.Printf("AgenticSession %s/%s deleted", sessionNamespace, sessionName)

				// Cancel any ongoing job monitoring for this session
				// (We could implement this with a context cancellation if needed)
				// OwnerReferences handle cleanup of per-session resources
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for AgenticSession: %v", obj)
			}
		}

		log.Println("AgenticSession watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

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

		// Delete temp pod if it exists (to free PVC for job)
		tempPodName := fmt.Sprintf("temp-content-%s", name)
		if _, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), tempPodName, v1.GetOptions{}); err == nil {
			log.Printf("[DesiredPhase] Deleting temp pod %s to free PVC for job", tempPodName)
			if err := config.K8sClient.CoreV1().Pods(sessionNamespace).Delete(context.TODO(), tempPodName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				log.Printf("[DesiredPhase] Warning: failed to delete temp pod: %v", err)
			}
			// Clear temp pod annotations
			_ = clearAnnotation(sessionNamespace, name, tempContentRequestedAnnotation)
			_ = clearAnnotation(sessionNamespace, name, tempContentLastAccessedAnnotation)
		}

		// Delete old job if it exists (from previous run)
		jobName := fmt.Sprintf("%s-job", name)
		_, err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err == nil {
			log.Printf("[DesiredPhase] Cleaning up old job %s before restart", jobName)
			if err := deleteJobAndPerJobService(sessionNamespace, jobName, name); err != nil {
				log.Printf("[DesiredPhase] Warning: failed to cleanup old job: %v", err)
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

		// Delete running job (this triggers pod deletion via OwnerReferences)
		jobName := fmt.Sprintf("%s-job", name)
		if err := deleteJobAndPerJobService(sessionNamespace, jobName, name); err != nil {
			log.Printf("[DesiredPhase] Warning: failed to delete job: %v", err)
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
		jobName := fmt.Sprintf("%s-job", name)
		_, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})

		if errors.IsNotFound(err) {
			// Job is gone - safe to transition to Stopped
			log.Printf("[Stopping] Session %s/%s: job deleted, transitioning to Stopped", sessionNamespace, name)

			// Set phase=Stopped explicitly
			statusPatch.SetField("phase", "Stopped")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			// Update progress-tracking conditions to reflect stopped state
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionJobCreated,
				Status:  "False",
				Reason:  "UserStopped",
				Message: "Job deleted by user stop request",
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
			// Error checking job - log and retry next reconciliation
			log.Printf("[Stopping] Session %s/%s: error checking job status: %v", sessionNamespace, name, err)
		} else {
			// Job still exists - try to delete it again
			log.Printf("[Stopping] Session %s/%s: job still exists, deleting", sessionNamespace, name)
			if err := deleteJobAndPerJobService(sessionNamespace, jobName, name); err != nil {
				log.Printf("[Stopping] Warning: failed to delete job: %v", err)
			}
			// Will retry on next reconciliation
		}
		return nil
	}

	// === TEMP CONTENT POD RECONCILIATION ===
	// Manage temporary content pods for workspace access on stopped sessions

	tempContentRequested := annotations != nil && annotations[tempContentRequestedAnnotation] == "true"
	tempPodName := fmt.Sprintf("temp-content-%s", name)

	// Only manage temp pods for stopped/completed/failed sessions
	if phase == "Stopped" || phase == "Completed" || phase == "Failed" {
		if tempContentRequested {
			// User wants workspace access - ensure temp pod exists
			if err := reconcileTempContentPodWithPatch(sessionNamespace, name, tempPodName, currentObj, statusPatch); err != nil {
				log.Printf("[TempPod] Failed to reconcile temp pod: %v", err)
			}
		} else {
			// Temp pod not requested - delete if it exists
			_, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), tempPodName, v1.GetOptions{})
			if err == nil {
				log.Printf("[TempPod] Deleting unrequested temp pod: %s", tempPodName)
				if err := config.K8sClient.CoreV1().Pods(sessionNamespace).Delete(context.TODO(), tempPodName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
					log.Printf("[TempPod] Failed to delete temp pod: %v", err)
				} else {
					statusPatch.AddCondition(conditionUpdate{
						Type:    conditionTempContentPodReady,
						Status:  "False",
						Reason:  "NotRequested",
						Message: "Temp pod removed (not requested)",
					})
				}
			}
		}
		// Apply temp pod status changes and return (no further reconciliation needed for stopped sessions)
		if statusPatch.HasChanges() {
			if err := statusPatch.Apply(); err != nil {
				log.Printf("[TempPod] Warning: failed to apply status patch: %v", err)
			}
		}
		return nil
	}

	// === CONTINUE WITH PHASE-BASED RECONCILIATION ===

	// Early exit: If desired-phase is "Stopped", do not recreate jobs or reconcile
	// This prevents race conditions where the operator sees the job deleted before phase is updated
	if desiredPhase == "Stopped" {
		log.Printf("Session %s has desired-phase=Stopped, skipping further reconciliation", name)
		return nil
	}

	// Handle Stopped phase - clean up running job if it exists
	if phase == "Stopped" {
		log.Printf("Session %s is stopped, checking for running job to clean up", name)
		jobName := fmt.Sprintf("%s-job", name)

		job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err == nil {
			// Job exists, check if it's still running or needs cleanup
			if job.Status.Active > 0 || (job.Status.Succeeded == 0 && job.Status.Failed == 0) {
				log.Printf("Job %s is still active, cleaning up job and pods", jobName)

				// First, delete the job itself with foreground propagation
				deletePolicy := v1.DeletePropagationForeground
				err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Delete(context.TODO(), jobName, v1.DeleteOptions{
					PropagationPolicy: &deletePolicy,
				})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete job %s: %v", jobName, err)
				} else {
					log.Printf("Successfully deleted job %s for stopped session", jobName)
				}

				// Then, explicitly delete all pods for this job (by job-name label)
				podSelector := fmt.Sprintf("job-name=%s", jobName)
				log.Printf("Deleting pods with job-name selector: %s", podSelector)
				err = config.K8sClient.CoreV1().Pods(sessionNamespace).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
					LabelSelector: podSelector,
				})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete pods for job %s: %v (continuing anyway)", jobName, err)
				} else {
					log.Printf("Successfully deleted pods for job %s", jobName)
				}

				// Also delete any pods labeled with this session (in case owner refs are lost)
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
			} else {
				log.Printf("Job %s already completed (Succeeded: %d, Failed: %d), no cleanup needed", jobName, job.Status.Succeeded, job.Status.Failed)
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("Error checking job %s: %v", jobName, err)
		} else {
			log.Printf("Job %s not found, already cleaned up", jobName)
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
		log.Printf("[Reconcile] Session %s/%s is Running, checking for spec changes", sessionNamespace, name)

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
			log.Printf("[Reconcile] Session %s/%s: detected spec change (generation %d > observed %d), reconciling repos and workflow",
				sessionNamespace, name, currentGeneration, observedGeneration)

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
			log.Printf("[Reconcile] Session %s/%s: updated observedGeneration to %d after successful reconciliation", sessionNamespace, name, currentGeneration)
		} else {
			log.Printf("[Reconcile] Session %s/%s: no spec changes detected (generation %d == observed %d)", sessionNamespace, name, currentGeneration, observedGeneration)
		}

		return nil
	}

	// Only process if status is Pending or Creating (to handle operator restarts)
	if phase != "Pending" && phase != "Creating" {
		return nil
	}

	// If in Creating phase, check if job exists
	if phase == "Creating" {
		jobName := fmt.Sprintf("%s-job", name)
		_, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err == nil {
			// Job exists, start monitoring if not already running
			monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, jobName)
			monitoredJobsMu.Lock()
			alreadyMonitoring := monitoredJobs[monitorKey]
			if !alreadyMonitoring {
				monitoredJobs[monitorKey] = true
				monitoredJobsMu.Unlock()
				log.Printf("Resuming monitoring for existing job %s (session in Creating phase)", jobName)
				go monitorJob(jobName, name, sessionNamespace)
			} else {
				monitoredJobsMu.Unlock()
				log.Printf("Job %s already being monitored, skipping duplicate", jobName)
			}
			return nil
		} else if errors.IsNotFound(err) {
			// Job doesn't exist but phase is Creating - check if this is due to a stop request
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
					Message: "User requested stop during job creation",
				})
				// Update progress-tracking conditions
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionJobCreated,
					Status:  "False",
					Reason:  "UserStopped",
					Message: "Job deleted by user stop request",
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

			// Job doesn't exist but phase is Creating - this is inconsistent state
			// Could happen if:
			// 1. Job was manually deleted
			// 2. Operator crashed between job creation and status update
			// 3. Session is being stopped and job was deleted (stale event)

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
					log.Printf("Session %s is now in %s phase (stale Creating event), skipping job recreation", name, freshPhase)
					return nil
				}
			}

			log.Printf("Session %s in Creating phase but job not found, resetting to Pending and recreating", name)
			statusPatch.SetField("phase", "Pending")
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionJobCreated,
				Status:  "False",
				Reason:  "JobMissing",
				Message: "Job not found, will recreate",
			})
			// Apply immediately and continue to Pending logic
			_ = statusPatch.ApplyAndReset()
			// Don't return - fall through to Pending logic to create job
			_ = "Pending" // phase reset handled by status update
		} else {
			// Error checking job - log and continue
			log.Printf("Error checking job for Creating session %s: %v, will attempt recovery", name, err)
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

	// Determine PVC name and owner references
	var pvcName string
	var ownerRefs []v1.OwnerReference
	reusingPVC := false

	if parentSessionID != "" {
		// Continuation: reuse parent's PVC
		pvcName = fmt.Sprintf("ambient-workspace-%s", parentSessionID)
		reusingPVC = true
		log.Printf("Session continuation: reusing PVC %s from parent session %s", pvcName, parentSessionID)
		// No owner refs - we don't own the parent's PVC
	} else {
		// New session: create fresh PVC with owner refs
		pvcName = fmt.Sprintf("ambient-workspace-%s", name)
		ownerRefs = []v1.OwnerReference{
			{
				APIVersion: "vteam.ambient-code/v1",
				Kind:       "AgenticSession",
				Name:       currentObj.GetName(),
				UID:        currentObj.GetUID(),
				Controller: boolPtr(true),
				// BlockOwnerDeletion intentionally omitted to avoid permission issues
			},
		}
	}

	// Ensure PVC exists (skip for continuation if parent's PVC should exist)
	if !reusingPVC {
		if err := services.EnsureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
			log.Printf("Failed to ensure session PVC %s in %s: %v", pvcName, sessionNamespace, err)
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionPVCReady,
				Status:  "False",
				Reason:  "ProvisioningFailed",
				Message: err.Error(),
			})
		} else {
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionPVCReady,
				Status:  "True",
				Reason:  "Bound",
				Message: fmt.Sprintf("PVC %s ready", pvcName),
			})
		}
	} else {
		// Verify parent's PVC exists
		if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(sessionNamespace).Get(context.TODO(), pvcName, v1.GetOptions{}); err != nil {
			log.Printf("Warning: Parent PVC %s not found for continuation session %s: %v", pvcName, name, err)
			// Fall back to creating new PVC with current session's owner refs
			pvcName = fmt.Sprintf("ambient-workspace-%s", name)
			ownerRefs = []v1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1",
					Kind:       "AgenticSession",
					Name:       currentObj.GetName(),
					UID:        currentObj.GetUID(),
					Controller: boolPtr(true),
				},
			}
			if err := services.EnsureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
				log.Printf("Failed to create fallback PVC %s: %v", pvcName, err)
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionPVCReady,
					Status:  "False",
					Reason:  "ProvisioningFailed",
					Message: err.Error(),
				})
			} else {
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionPVCReady,
					Status:  "True",
					Reason:  "Bound",
					Message: fmt.Sprintf("PVC %s ready", pvcName),
				})
			}
		} else {
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionPVCReady,
				Status:  "True",
				Reason:  "Reused",
				Message: fmt.Sprintf("Reused PVC %s from parent session", pvcName),
			})
		}
	}

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

	// Create a Kubernetes Job for this AgenticSession
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job already exists in the session's namespace
	_, err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err == nil {
		log.Printf("Job %s already exists for AgenticSession %s", jobName, name)
		statusPatch.SetField("phase", "Creating")
		statusPatch.SetField("observedGeneration", currentObj.GetGeneration())
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionJobCreated,
			Status:  "True",
			Reason:  "JobExists",
			Message: "Runner job already exists",
		})
		_ = statusPatch.Apply()
		// Clear desired-phase annotation if it exists (job already created)
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

	// Extract repos configuration (simplified format: url and branch)
	type RepoConfig struct {
		URL    string
		Branch string
	}

	var repos []RepoConfig

	// Read simplified repos[] array format
	if reposArr, found, _ := unstructured.NestedSlice(spec, "repos"); found && len(reposArr) > 0 {
		repos = make([]RepoConfig, 0, len(reposArr))
		for _, repoItem := range reposArr {
			if repoMap, ok := repoItem.(map[string]interface{}); ok {
				repo := RepoConfig{}
				if url, ok := repoMap["url"].(string); ok {
					repo.URL = url
				}
				if branch, ok := repoMap["branch"].(string); ok {
					repo.Branch = branch
				} else {
					repo.Branch = "main"
				}
				if repo.URL != "" {
					repos = append(repos, repo)
				}
			}
		}
	} else {
		// Fallback to old format for backward compatibility (input/output structure)
		inputRepo, _, _ := unstructured.NestedString(spec, "inputRepo")
		inputBranch, _, _ := unstructured.NestedString(spec, "inputBranch")
		if v, found, _ := unstructured.NestedString(spec, "input", "repo"); found && strings.TrimSpace(v) != "" {
			inputRepo = v
		}
		if v, found, _ := unstructured.NestedString(spec, "input", "branch"); found && strings.TrimSpace(v) != "" {
			inputBranch = v
		}
		if inputRepo != "" {
			if inputBranch == "" {
				inputBranch = "main"
			}
			repos = []RepoConfig{{
				URL:    inputRepo,
				Branch: inputBranch,
			}}
		}
	}

	// Get first repo for backward compatibility env vars (first repo is always main repo)
	var inputRepo, inputBranch, outputRepo, outputBranch string
	if len(repos) > 0 {
		inputRepo = repos[0].URL
		inputBranch = repos[0].Branch
		outputRepo = repos[0].URL // Output same as input in simplified format
		outputBranch = repos[0].Branch
	}

	// Read autoPushOnComplete flag
	autoPushOnComplete, _, _ := unstructured.NestedBool(spec, "autoPushOnComplete")

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

	// Determine runner token secret name for volume mount
	runnerTokenSecretName := ""
	if annotations := currentObj.GetAnnotations(); annotations != nil {
		if v, ok := annotations["ambient-code.io/runner-token-secret"]; ok && strings.TrimSpace(v) != "" {
			runnerTokenSecretName = strings.TrimSpace(v)
		}
	}
	if runnerTokenSecretName == "" {
		runnerTokenSecretName = fmt.Sprintf("ambient-runner-token-%s", name)
	}

	// Create the Job
	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      jobName,
			Namespace: sessionNamespace,
			Labels: map[string]string{
				"agentic-session": name,
				"app":             "ambient-code-runner",
			},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1",
					Kind:       "AgenticSession",
					Name:       currentObj.GetName(),
					UID:        currentObj.GetUID(),
					Controller: boolPtr(true),
					// Remove BlockOwnerDeletion to avoid permission issues
					// BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          int32Ptr(3),
			ActiveDeadlineSeconds: int64Ptr(14400), // 4 hour timeout for safety
			// Auto-cleanup finished Jobs if TTL controller is enabled in the cluster
			TTLSecondsAfterFinished: int32Ptr(600),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"agentic-session": name,
						"app":             "ambient-code-runner",
					},
					// If you run a service mesh that injects sidecars and causes egress issues for Jobs:
					// Annotations: map[string]string{"sidecar.istio.io/inject": "false"},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					// Explicitly set service account for pod creation permissions
					AutomountServiceAccountToken: boolPtr(false),
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
						{
							Name: "runner-token",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: runnerTokenSecretName,
								},
							},
						},
					},

					// InitContainer to ensure workspace directory structure exists
					InitContainers: []corev1.Container{
						{
							Name:  "init-workspace",
							Image: "registry.access.redhat.com/ubi8/ubi-minimal:latest",
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("mkdir -p /workspace/sessions/%s/workspace && chmod 777 /workspace/sessions/%s/workspace && echo 'Workspace initialized'", name, name),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
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

							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace", ReadOnly: false},
								// Mount .claude directory for session state persistence
								// This enables SDK's built-in resume functionality
								{Name: "workspace", MountPath: "/app/.claude", SubPath: fmt.Sprintf("sessions/%s/.claude", name), ReadOnly: false},
								// Mount runner token secret as volume for dynamic token refresh
								{Name: "runner-token", MountPath: "/app/runner-token", ReadOnly: true},
							},

							Env: func() []corev1.EnvVar {
								base := []corev1.EnvVar{
									{Name: "DEBUG", Value: "true"},
									{Name: "INTERACTIVE", Value: fmt.Sprintf("%t", interactive)},
									{Name: "AGENTIC_SESSION_NAME", Value: name},
									{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
									// Provide session id and workspace path for the runner wrapper
									{Name: "SESSION_ID", Value: name},
									{Name: "WORKSPACE_PATH", Value: fmt.Sprintf("/workspace/sessions/%s/workspace", name)},
									{Name: "ARTIFACTS_DIR", Value: "_artifacts"},
								}

								// Add user context for observability and auditing (Langfuse userId, logs, etc.)
								if userID != "" {
									base = append(base, corev1.EnvVar{Name: "USER_ID", Value: userID})
								}
								if userName != "" {
									base = append(base, corev1.EnvVar{Name: "USER_NAME", Value: userName})
								}

								// Add per-repo environment variables (simplified format)
								for i, repo := range repos {
									base = append(base,
										corev1.EnvVar{Name: fmt.Sprintf("REPO_%d_URL", i), Value: repo.URL},
										corev1.EnvVar{Name: fmt.Sprintf("REPO_%d_BRANCH", i), Value: repo.Branch},
									)
								}

								// Backward compatibility: set INPUT_REPO_URL/OUTPUT_REPO_URL from main repo
								base = append(base,
									corev1.EnvVar{Name: "INPUT_REPO_URL", Value: inputRepo},
									corev1.EnvVar{Name: "INPUT_BRANCH", Value: inputBranch},
									corev1.EnvVar{Name: "OUTPUT_REPO_URL", Value: outputRepo},
									corev1.EnvVar{Name: "OUTPUT_BRANCH", Value: outputBranch},
									corev1.EnvVar{Name: "INITIAL_PROMPT", Value: prompt},
									corev1.EnvVar{Name: "LLM_MODEL", Value: model},
									corev1.EnvVar{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
									corev1.EnvVar{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
									corev1.EnvVar{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
									corev1.EnvVar{Name: "AUTO_PUSH_ON_COMPLETE", Value: fmt.Sprintf("%t", autoPushOnComplete)},
									corev1.EnvVar{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", appConfig.BackendNamespace)},
									// WebSocket URL used by runner-shell to connect back to backend
									corev1.EnvVar{Name: "WEBSOCKET_URL", Value: fmt.Sprintf("ws://backend-service.%s.svc.cluster.local:8080/api/projects/%s/sessions/%s/ws", appConfig.BackendNamespace, sessionNamespace, name)},
									// S3 disabled; backend persists messages
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
								// Inject BOT_TOKEN_PATH pointing to mounted secret volume
								// Token is mounted from runnerTokenSecretName at /app/runner-token
								// This allows the runner to read refreshed tokens without pod restart
								base = append(base, corev1.EnvVar{
									Name:  "BOT_TOKEN_PATH",
									Value: "/app/runner-token/k8s-token",
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

							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			},
		},
	}

	// Note: No volume mounts needed for runner/integration secrets
	// All keys are injected as environment variables via EnvFrom above

	// If ambient-vertex secret was successfully copied, mount it as a volume
	if ambientVertexSecretCopied {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name:         "vertex",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: types.AmbientVertexSecretName}},
		})
		// Mount to the ambient-code-runner container by name
		for i := range job.Spec.Template.Spec.Containers {
			if job.Spec.Template.Spec.Containers[i].Name == "ambient-code-runner" {
				job.Spec.Template.Spec.Containers[i].VolumeMounts = append(job.Spec.Template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "vertex",
					MountPath: "/app/vertex",
					ReadOnly:  true,
				})
				log.Printf("Mounted %s secret to /app/vertex in runner container for session %s", types.AmbientVertexSecretName, name)
				break
			}
		}
	}

	// Do not mount runner Secret volume; runner fetches tokens on demand

	// Create the job
	createdJob, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
		// If job already exists, this is likely a race condition from duplicate watch events - not an error
		if errors.IsAlreadyExists(err) {
			log.Printf("Job %s already exists (race condition), continuing", jobName)
			// Clear desired-phase annotation since job exists
			_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
			return nil
		}
		log.Printf("Failed to create job %s: %v", jobName, err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionJobCreated,
			Status:  "False",
			Reason:  "CreateFailed",
			Message: err.Error(),
		})
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "JobCreationFailed",
			Message: "Runner job creation failed",
		})
		_ = statusPatch.Apply()
		return fmt.Errorf("failed to create job: %v", err)
	}

	log.Printf("Created job %s for AgenticSession %s", jobName, name)
	statusPatch.SetField("phase", "Creating")
	statusPatch.SetField("observedGeneration", currentObj.GetGeneration())
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionJobCreated,
		Status:  "True",
		Reason:  "JobCreated",
		Message: "Runner job created",
	})
	// Apply all accumulated status changes in a single API call
	if err := statusPatch.Apply(); err != nil {
		log.Printf("Warning: failed to apply status patch: %v", err)
	}

	// Clear desired-phase annotation now that job is created
	// (This was deferred from the restart handler to avoid race conditions with stale events)
	_ = clearAnnotation(sessionNamespace, name, "ambient-code.io/desired-phase")
	log.Printf("[DesiredPhase] Cleared desired-phase annotation after successful job creation")

	// Create a per-job Service pointing to the content container
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("ambient-content-%s", name),
			Namespace: sessionNamespace,
			Labels:    map[string]string{"app": "ambient-code-runner", "agentic-session": name},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "batch/v1",
				Kind:       "Job",
				Name:       jobName,
				UID:        createdJob.UID,
				Controller: boolPtr(true),
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"job-name": jobName},
			Ports:    []corev1.ServicePort{{Port: 8080, TargetPort: intstr.FromString("http"), Protocol: corev1.ProtocolTCP, Name: "http"}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if _, serr := config.K8sClient.CoreV1().Services(sessionNamespace).Create(context.TODO(), svc, v1.CreateOptions{}); serr != nil && !errors.IsAlreadyExists(serr) {
		log.Printf("Failed to create per-job content service for %s: %v", name, serr)
	}

	// Start monitoring the job (only if not already being monitored)
	monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, jobName)
	monitoredJobsMu.Lock()
	alreadyMonitoring := monitoredJobs[monitorKey]
	if !alreadyMonitoring {
		monitoredJobs[monitorKey] = true
		monitoredJobsMu.Unlock()
		go monitorJob(jobName, name, sessionNamespace)
	} else {
		monitoredJobsMu.Unlock()
		log.Printf("Job %s already being monitored, skipping duplicate goroutine", jobName)
	}

	return nil
}

// reconcileSpecReposWithPatch is a version of reconcileSpecRepos that uses StatusPatch for batched updates.
// This is used during initial reconciliation to avoid triggering multiple watch events.
func reconcileSpecReposWithPatch(sessionNamespace, sessionName string, spec map[string]interface{}, session *unstructured.Unstructured, statusPatch *StatusPatch) error {
	repoSlice, found, _ := unstructured.NestedSlice(spec, "repos")
	if !found {
		log.Printf("[Reconcile] Session %s/%s: no repos defined in spec", sessionNamespace, sessionName)
		statusPatch.DeleteField("reconciledRepos")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReposReconciled,
			Status:  "True",
			Reason:  "NoRepos",
			Message: "No repositories defined",
		})
		return nil
	}

	// Parse spec repos
	specRepos := make([]map[string]string, 0, len(repoSlice))
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
			specRepos = append(specRepos, map[string]string{
				"url":    url,
				"branch": branch,
			})
		}
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

	// Detect drift: repos added or removed
	toAdd := []map[string]string{}
	toRemove := []map[string]string{}

	// Find repos in spec but not in reconciled (need to add)
	for _, specRepo := range specRepos {
		found := false
		for _, reconciledRepo := range reconciledRepos {
			if specRepo["url"] == reconciledRepo["url"] {
				found = true
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

	if len(toAdd) == 0 && len(toRemove) == 0 {
		log.Printf("[Reconcile] Session %s/%s: repos already reconciled (%d repos)", sessionNamespace, sessionName, len(specRepos))
		return nil
	}

	log.Printf("[Reconcile] Session %s/%s: detected repo drift - adding %d, removing %d", sessionNamespace, sessionName, len(toAdd), len(toRemove))

	// Send WebSocket messages via backend to trigger runner actions
	backendURL := getBackendAPIURL(sessionNamespace)

	// Add repos
	for _, repo := range toAdd {
		repoName := deriveRepoNameFromURL(repo["url"])
		log.Printf("[Reconcile] Session %s/%s: sending repo_added message for %s (%s@%s)", sessionNamespace, sessionName, repoName, repo["url"], repo["branch"])
		if err := sendWebSocketMessageViaBackend(sessionNamespace, sessionName, backendURL, map[string]interface{}{
			"type":   "repo_added",
			"url":    repo["url"],
			"branch": repo["branch"],
			"name":   repoName,
		}); err != nil {
			log.Printf("[Reconcile] Failed to send repo_added message: %v", err)
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReposReconciled,
				Status:  "False",
				Reason:  "MessageFailed",
				Message: fmt.Sprintf("Failed to notify runner: %v", err),
			})
			return fmt.Errorf("failed to send repo_added message: %w", err)
		}
	}

	// Remove repos
	for _, repo := range toRemove {
		repoName := deriveRepoNameFromURL(repo["url"])
		log.Printf("[Reconcile] Session %s/%s: sending repo_removed message for %s", sessionNamespace, sessionName, repoName)
		if err := sendWebSocketMessageViaBackend(sessionNamespace, sessionName, backendURL, map[string]interface{}{
			"type": "repo_removed",
			"name": repoName,
		}); err != nil {
			log.Printf("[Reconcile] Failed to send repo_removed message: %v", err)
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReposReconciled,
				Status:  "False",
				Reason:  "MessageFailed",
				Message: fmt.Sprintf("Failed to notify runner: %v", err),
			})
			return fmt.Errorf("failed to send repo_removed message: %w", err)
		}
	}

	// Update status to reflect the reconciled state (via statusPatch)
	reconciled := make([]interface{}, 0, len(specRepos))
	for _, repo := range specRepos {
		reconciled = append(reconciled, map[string]interface{}{
			"url":      repo["url"],
			"branch":   repo["branch"],
			"status":   "Ready",
			"clonedAt": time.Now().UTC().Format(time.RFC3339),
		})
	}
	statusPatch.SetField("reconciledRepos", reconciled)
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReposReconciled,
		Status:  "True",
		Reason:  "Reconciled",
		Message: fmt.Sprintf("Reconciled %d repos (added: %d, removed: %d)", len(specRepos), len(toAdd), len(toRemove)),
	})

	log.Printf("[Reconcile] Session %s/%s: successfully reconciled repos", sessionNamespace, sessionName)
	return nil
}

// reconcileActiveWorkflowWithPatch is a version of reconcileActiveWorkflow that uses StatusPatch for batched updates.
func reconcileActiveWorkflowWithPatch(sessionNamespace, sessionName string, spec map[string]interface{}, session *unstructured.Unstructured, statusPatch *StatusPatch) error {
	workflow, found, _ := unstructured.NestedMap(spec, "activeWorkflow")
	if !found || len(workflow) == 0 {
		log.Printf("[Reconcile] Session %s/%s: no workflow defined in spec", sessionNamespace, sessionName)
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
		log.Printf("[Reconcile] Session %s/%s: workflow gitUrl is empty", sessionNamespace, sessionName)
		return nil
	}

	// Get current reconciled workflow from status
	status, _, _ := unstructured.NestedMap(session.Object, "status")
	reconciledWorkflowRaw, _, _ := unstructured.NestedMap(status, "reconciledWorkflow")
	reconciledGitURL, _ := reconciledWorkflowRaw["gitUrl"].(string)
	reconciledBranch, _ := reconciledWorkflowRaw["branch"].(string)

	// Detect drift: workflow changed
	if reconciledGitURL == gitURL && reconciledBranch == branch {
		log.Printf("[Reconcile] Session %s/%s: workflow already reconciled (%s@%s)", sessionNamespace, sessionName, gitURL, branch)
		return nil
	}

	log.Printf("[Reconcile] Session %s/%s: detected workflow drift - switching from %s@%s to %s@%s",
		sessionNamespace, sessionName, reconciledGitURL, reconciledBranch, gitURL, branch)

	// Send WebSocket message via backend to trigger runner workflow switch
	backendURL := getBackendAPIURL(sessionNamespace)
	log.Printf("[Reconcile] Session %s/%s: sending workflow_change message for %s@%s (path: %s)", sessionNamespace, sessionName, gitURL, branch, path)

	if err := sendWebSocketMessageViaBackend(sessionNamespace, sessionName, backendURL, map[string]interface{}{
		"type":   "workflow_change",
		"gitUrl": gitURL,
		"branch": branch,
		"path":   path,
	}); err != nil {
		log.Printf("[Reconcile] Failed to send workflow_change message: %v", err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionWorkflowReconciled,
			Status:  "False",
			Reason:  "MessageFailed",
			Message: fmt.Sprintf("Failed to notify runner: %v", err),
		})
		return fmt.Errorf("failed to send workflow_selected message: %w", err)
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

	log.Printf("[Reconcile] Session %s/%s: successfully reconciled workflow", sessionNamespace, sessionName)
	return nil
}

func monitorJob(jobName, sessionName, sessionNamespace string) {
	monitorKey := fmt.Sprintf("%s/%s", sessionNamespace, jobName)

	// Remove from monitoring map when this goroutine exits
	defer func() {
		monitoredJobsMu.Lock()
		delete(monitoredJobs, monitorKey)
		monitoredJobsMu.Unlock()
		log.Printf("Stopped monitoring job %s (goroutine exiting)", jobName)
	}()

	log.Printf("Starting job monitoring for %s (session: %s/%s)", jobName, sessionNamespace, sessionName)
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
				log.Printf("AgenticSession %s was stopped; stopping job monitoring", sessionName)
				return
			}
		}

		if err := ensureFreshRunnerToken(context.TODO(), sessionObj); err != nil {
			log.Printf("Failed to refresh runner token for %s/%s: %v", sessionNamespace, sessionName, err)
		}

		job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Job %s deleted; stopping monitor", jobName)
				return
			}
			log.Printf("Error fetching job %s: %v", jobName, err)
			continue
		}

		pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)})
		if err != nil {
			log.Printf("Failed to list pods for job %s: %v", jobName, err)
			continue
		}

		if job.Status.Succeeded > 0 {
			statusPatch.SetField("phase", "Completed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "Completed", Message: "Session finished"})
			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		if job.Spec.BackoffLimit != nil && job.Status.Failed >= *job.Spec.BackoffLimit {
			statusPatch.SetField("phase", "Failed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "BackoffLimitExceeded", Message: "Runner failed repeatedly"})
			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		if len(pods.Items) == 0 {
			if job.Status.Active == 0 && job.Status.Succeeded == 0 && job.Status.Failed == 0 {
				statusPatch.SetField("phase", "Failed")
				statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
				statusPatch.AddCondition(conditionUpdate{
					Type:    conditionReady,
					Status:  "False",
					Reason:  "PodMissing",
					Message: "Runner pod missing",
				})
				_ = statusPatch.Apply()
				_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
				_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
				return
			}
			continue
		}

		pod := pods.Items[0]
		// Note: We don't store pod name in status (pods are ephemeral, can be recreated)
		// Use k8s-resources endpoint or kubectl for live pod info

		if pod.Spec.NodeName != "" {
			statusPatch.AddCondition(conditionUpdate{Type: conditionPodScheduled, Status: "True", Reason: "Scheduled", Message: fmt.Sprintf("Scheduled on %s", pod.Spec.NodeName)})
		}

		if pod.Status.Phase == corev1.PodFailed {
			statusPatch.SetField("phase", "Failed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{Type: conditionReady, Status: "False", Reason: "PodFailed", Message: pod.Status.Message})
			_ = statusPatch.Apply()
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		runner := getContainerStatusByName(&pod, "ambient-code-runner")
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
				_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
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
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
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

// deleteJobAndPerJobService deletes the Job and its associated per-job Service
func deleteJobAndPerJobService(namespace, jobName, sessionName string) error {
	// Delete Service first (it has ownerRef to Job, but delete explicitly just in case)
	svcName := fmt.Sprintf("ambient-content-%s", sessionName)
	if err := config.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), svcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-job service %s/%s: %v", namespace, svcName, err)
	}

	// Delete the Job with background propagation
	policy := v1.DeletePropagationBackground
	if err := config.K8sClient.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, v1.DeleteOptions{PropagationPolicy: &policy}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete job %s/%s: %v", namespace, jobName, err)
		return err
	}

	// Proactively delete Pods for this Job
	if pods, err := config.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)}); err == nil {
		for i := range pods.Items {
			p := pods.Items[i]
			if err := config.K8sClient.CoreV1().Pods(namespace).Delete(context.TODO(), p.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				log.Printf("Failed to delete pod %s/%s for job %s: %v", namespace, p.Name, jobName, err)
			}
		}
	} else if !errors.IsNotFound(err) {
		log.Printf("Failed to list pods for job %s/%s: %v", namespace, jobName, err)
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

// CleanupExpiredTempContentPods removes temporary content pods that have exceeded their TTL
func CleanupExpiredTempContentPods() {
	log.Println("Starting temp content pod cleanup goroutine")
	for {
		time.Sleep(1 * time.Minute)

		// List all temp content pods across all namespaces
		pods, err := config.K8sClient.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{
			LabelSelector: "app=temp-content-service",
		})
		if err != nil {
			log.Printf("[TempPodCleanup] Failed to list temp content pods: %v", err)
			continue
		}

		gvr := types.GetAgenticSessionResource()
		for _, pod := range pods.Items {
			sessionName := pod.Labels["agentic-session"]
			if sessionName == "" {
				log.Printf("[TempPodCleanup] Temp pod %s has no agentic-session label, skipping", pod.Name)
				continue
			}

			// Check if session still exists
			session, err := config.DynamicClient.Resource(gvr).Namespace(pod.Namespace).Get(context.TODO(), sessionName, v1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					// Session deleted, delete temp pod
					log.Printf("[TempPodCleanup] Session %s/%s gone, deleting orphaned temp pod %s", pod.Namespace, sessionName, pod.Name)
					if err := config.K8sClient.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
						log.Printf("[TempPodCleanup] Failed to delete orphaned temp pod: %v", err)
					}
				}
				continue
			}

			// Get last-accessed timestamp from session annotation
			annotations := session.GetAnnotations()
			lastAccessedStr := annotations[tempContentLastAccessedAnnotation]
			if lastAccessedStr == "" {
				// Fall back to pod created-at if no last-accessed
				lastAccessedStr = pod.Annotations["ambient-code.io/created-at"]
			}

			if lastAccessedStr == "" {
				log.Printf("[TempPodCleanup] No timestamp for temp pod %s, skipping", pod.Name)
				continue
			}

			lastAccessed, err := time.Parse(time.RFC3339, lastAccessedStr)
			if err != nil {
				log.Printf("[TempPodCleanup] Failed to parse timestamp for pod %s: %v", pod.Name, err)
				continue
			}

			// Delete if inactive for > 10 minutes
			if time.Since(lastAccessed) > tempContentInactivityTTL {
				log.Printf("[TempPodCleanup] Deleting inactive temp pod %s/%s (last accessed: %v ago)",
					pod.Namespace, pod.Name, time.Since(lastAccessed))

				if err := config.K8sClient.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
					log.Printf("[TempPodCleanup] Failed to delete temp pod: %v", err)
					continue
				}

				// Update condition
				_ = mutateAgenticSessionStatus(pod.Namespace, sessionName, func(status map[string]interface{}) {
					setCondition(status, conditionUpdate{
						Type:    conditionTempContentPodReady,
						Status:  "False",
						Reason:  "Expired",
						Message: fmt.Sprintf("Temp pod deleted due to inactivity (%v)", time.Since(lastAccessed)),
					})
				})

				// Clear temp-content-requested annotation
				delete(annotations, tempContentRequestedAnnotation)
				delete(annotations, tempContentLastAccessedAnnotation)
				_ = updateAnnotations(pod.Namespace, sessionName, annotations)
			}
		}
	}
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

	log.Printf("Deleting copied %s secret from namespace %s", types.AmbientVertexSecretName, namespace)
	err = config.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, types.AmbientVertexSecretName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s secret: %w", types.AmbientVertexSecretName, err)
	}

	return nil
}

// deleteAmbientLangfuseSecret deletes the ambient-admin-langfuse-secret from a namespace if it was copied
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

	log.Printf("Deleting copied %s secret from namespace %s", langfuseSecretName, namespace)
	err = config.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, langfuseSecretName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s secret: %w", langfuseSecretName, err)
	}

	return nil
}

// reconcileTempContentPodWithPatch is a version of reconcileTempContentPod that uses StatusPatch for batched updates.
func reconcileTempContentPodWithPatch(sessionNamespace, sessionName, tempPodName string, session *unstructured.Unstructured, statusPatch *StatusPatch) error {
	// Check if pod already exists
	tempPod, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Get(context.TODO(), tempPodName, v1.GetOptions{})

	if errors.IsNotFound(err) {
		// Create temp pod
		log.Printf("[TempPod] Creating temp content pod for workspace access: %s/%s", sessionNamespace, tempPodName)

		pvcName := fmt.Sprintf("ambient-workspace-%s", sessionName)
		appConfig := config.LoadConfig()

		pod := &corev1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      tempPodName,
				Namespace: sessionNamespace,
				Labels: map[string]string{
					"app":             "temp-content-service",
					"agentic-session": sessionName,
				},
				Annotations: map[string]string{
					"ambient-code.io/created-at": time.Now().UTC().Format(time.RFC3339),
				},
				OwnerReferences: []v1.OwnerReference{{
					APIVersion: session.GetAPIVersion(),
					Kind:       session.GetKind(),
					Name:       session.GetName(),
					UID:        session.GetUID(),
					Controller: boolPtr(true),
				}},
			},
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name:            "content",
					Image:           appConfig.ContentServiceImage,
					ImagePullPolicy: appConfig.ImagePullPolicy,
					Env: []corev1.EnvVar{
						{Name: "CONTENT_SERVICE_MODE", Value: "true"},
						{Name: "STATE_BASE_DIR", Value: "/workspace"},
					},
					Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "workspace",
						MountPath: "/workspace",
					}},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/health",
								Port: intstr.FromString("http"),
							},
						},
						InitialDelaySeconds: 3,
						PeriodSeconds:       3,
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				}},
			},
		}

		if _, err := config.K8sClient.CoreV1().Pods(sessionNamespace).Create(context.TODO(), pod, v1.CreateOptions{}); err != nil {
			log.Printf("[TempPod] Failed to create temp pod: %v", err)
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionTempContentPodReady,
				Status:  "False",
				Reason:  "CreationFailed",
				Message: fmt.Sprintf("Failed to create temp pod: %v", err),
			})
			return fmt.Errorf("failed to create temp pod: %w", err)
		}

		log.Printf("[TempPod] Created temp pod %s", tempPodName)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionTempContentPodReady,
			Status:  "Unknown",
			Reason:  "Provisioning",
			Message: "Temp content pod starting",
		})
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to check temp pod: %w", err)
	}

	// Temp pod exists, check readiness
	if tempPod.Status.Phase == corev1.PodRunning {
		ready := false
		for _, cond := range tempPod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}

		if ready {
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionTempContentPodReady,
				Status:  "True",
				Reason:  "Ready",
				Message: "Temp content pod is ready for workspace access",
			})
		} else {
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionTempContentPodReady,
				Status:  "Unknown",
				Reason:  "NotReady",
				Message: "Temp content pod not ready yet",
			})
		}
	} else if tempPod.Status.Phase == corev1.PodFailed {
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionTempContentPodReady,
			Status:  "False",
			Reason:  "PodFailed",
			Message: fmt.Sprintf("Temp content pod failed: %s", tempPod.Status.Message),
		})
	}

	return nil
}

// getBackendAPIURL returns the backend API URL for the given namespace
func getBackendAPIURL(namespace string) string {
	appConfig := config.LoadConfig()
	return fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", appConfig.BackendNamespace)
}

// sendWebSocketMessageViaBackend sends a WebSocket message to the runner via the backend's message endpoint
func sendWebSocketMessageViaBackend(namespace, sessionName, backendURL string, message map[string]interface{}) error {
	// The backend exposes POST /api/projects/:project/sessions/:sessionName/messages
	// Format: { "type": "repo_added", "payload": {...}, ...other fields }
	// Backend will extract "type" and wrap remaining fields under "payload" if needed
	url := fmt.Sprintf("%s/projects/%s/sessions/%s/messages", backendURL, namespace, sessionName)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use operator's service account token for authentication
	// The backend accepts internal calls from the operator namespace
	// Get the operator's SA token from the mounted service account
	tokenBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err == nil && len(tokenBytes) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(tokenBytes)))
	} else {
		log.Printf("[WebSocket] Warning: could not read operator SA token, request may fail: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[WebSocket] Successfully sent message type=%s to session %s/%s via backend", message["type"], namespace, sessionName)
	return nil
}

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

// Helper functions
var (
	boolPtr  = func(b bool) *bool { return &b }
	int32Ptr = func(i int32) *int32 { return &i }
	int64Ptr = func(i int64) *int64 { return &i }
)
