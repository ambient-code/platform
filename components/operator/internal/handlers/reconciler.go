// Package handlers provides exported reconciliation functions for the controller.
// These functions are called from the controller-runtime reconciler and contain
// the actual business logic for managing AgenticSession resources.
package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"ambient-code-operator/internal/config"
)

// ReconcilePendingSession handles the Pending phase - creates pod and services.
// This is the main entry point called from the controller for pending sessions.
//
// TODO(controller-runtime-migration): This is a transitional wrapper around the legacy
// handleAgenticSessionEvent() function (2,300+ lines). Future work should:
// 1. Extract phase-specific logic into separate functions (ReconcilePending, ReconcileRunning, etc.)
// 2. Use controller-runtime patterns (Patch, StatusWriter, etc.) instead of direct API calls
// 3. Remove handleAgenticSessionEvent() entirely
// This approach allows adopting controller-runtime framework without rewriting all logic at once.
func ReconcilePendingSession(ctx context.Context, session *unstructured.Unstructured, appConfig *config.Config) error {
	// Delegate to existing handleAgenticSessionEvent logic
	// This is a wrapper that allows the existing code to be called from the controller
	return handleAgenticSessionEvent(session)
}

// ResetToPending transitions a session back to Pending phase.
func ResetToPending(ctx context.Context, session *unstructured.Unstructured) error {
	namespace := session.GetNamespace()
	name := session.GetName()

	statusPatch := NewStatusPatch(namespace, name)
	statusPatch.SetField("phase", "Pending")
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionPodCreated,
		Status:  "False",
		Reason:  "PodMissing",
		Message: "Pod not found, will recreate",
	})

	return statusPatch.Apply()
}

// TransitionToStopped transitions a session to Stopped phase.
func TransitionToStopped(ctx context.Context, session *unstructured.Unstructured) error {
	namespace := session.GetNamespace()
	name := session.GetName()

	statusPatch := NewStatusPatch(namespace, name)
	statusPatch.SetField("phase", "Stopped")
	statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReady,
		Status:  "False",
		Reason:  "UserStopped",
		Message: "Session stopped by user",
	})
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
		return err
	}

	// Clear annotations
	_ = clearAnnotation(namespace, name, "ambient-code.io/desired-phase")
	_ = clearAnnotation(namespace, name, "ambient-code.io/stop-requested-at")

	// Cleanup secrets
	deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = deleteAmbientVertexSecret(deleteCtx, namespace)
	_ = deleteAmbientLangfuseSecret(deleteCtx, namespace)

	return nil
}

// TransitionToFailed transitions a session to Failed phase with an error message.
func TransitionToFailed(ctx context.Context, session *unstructured.Unstructured, errorMsg string) error {
	namespace := session.GetNamespace()
	name := session.GetName()

	statusPatch := NewStatusPatch(namespace, name)
	statusPatch.SetField("phase", "Failed")
	statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReady,
		Status:  "False",
		Reason:  "PodFailed",
		Message: errorMsg,
	})

	if err := statusPatch.Apply(); err != nil {
		return err
	}

	_ = ensureSessionIsInteractive(namespace, name)

	return nil
}

// InitiateStop starts the stop process for a running session.
func InitiateStop(ctx context.Context, session *unstructured.Unstructured) error {
	namespace := session.GetNamespace()
	name := session.GetName()
	podName := fmt.Sprintf("%s-runner", name)

	log.Printf("[Stop] Initiating stop for session %s/%s", namespace, name)

	// Delete the pod
	if err := DeletePodAndServices(ctx, namespace, podName, name); err != nil {
		log.Printf("[Stop] Warning: failed to delete pod: %v", err)
	}

	// Transition to Stopping phase
	statusPatch := NewStatusPatch(namespace, name)
	statusPatch.SetField("phase", "Stopping")
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReady,
		Status:  "False",
		Reason:  "Stopping",
		Message: "Session is stopping",
	})

	return statusPatch.Apply()
}

// ReconcileSpecChanges handles spec updates for a running session.
func ReconcileSpecChanges(ctx context.Context, session *unstructured.Unstructured) error {
	namespace := session.GetNamespace()
	name := session.GetName()

	spec, _, _ := unstructured.NestedMap(session.Object, "spec")
	statusPatch := NewStatusPatch(namespace, name)

	// Reconcile repos
	if err := reconcileSpecReposWithPatch(namespace, name, spec, session, statusPatch); err != nil {
		log.Printf("[Reconcile] Failed to reconcile repos for %s/%s: %v", namespace, name, err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReconciled,
			Status:  "False",
			Reason:  "RepoReconciliationFailed",
			Message: fmt.Sprintf("Failed to reconcile repos: %v", err),
		})
		_ = statusPatch.Apply()
		return err
	}

	// Reconcile workflow
	if err := reconcileActiveWorkflowWithPatch(namespace, name, spec, session, statusPatch); err != nil {
		log.Printf("[Reconcile] Failed to reconcile workflow for %s/%s: %v", namespace, name, err)
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReconciled,
			Status:  "False",
			Reason:  "WorkflowReconciliationFailed",
			Message: fmt.Sprintf("Failed to reconcile workflow: %v", err),
		})
		_ = statusPatch.Apply()
		return err
	}

	// Update observedGeneration
	statusPatch.SetField("observedGeneration", session.GetGeneration())
	statusPatch.AddCondition(conditionUpdate{
		Type:    conditionReconciled,
		Status:  "True",
		Reason:  "SpecApplied",
		Message: fmt.Sprintf("Successfully reconciled generation %d", session.GetGeneration()),
	})

	return statusPatch.Apply()
}

// UpdateSessionFromPodStatus updates the session status based on pod state.
func UpdateSessionFromPodStatus(ctx context.Context, session *unstructured.Unstructured, pod *corev1.Pod) error {
	namespace := session.GetNamespace()
	name := session.GetName()
	podName := pod.Name

	statusPatch := NewStatusPatch(namespace, name)

	// Check if pod is scheduled
	if pod.Spec.NodeName != "" {
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionPodScheduled,
			Status:  "True",
			Reason:  "Scheduled",
			Message: fmt.Sprintf("Scheduled on %s", pod.Spec.NodeName),
		})
	}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		statusPatch.SetField("phase", "Completed")
		statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "Completed",
			Message: "Session finished",
		})
		if err := statusPatch.Apply(); err != nil {
			return err
		}
		_ = ensureSessionIsInteractive(namespace, name)
		return DeletePodAndServices(ctx, namespace, podName, name)

	case corev1.PodFailed:
		errorMsg := collectPodErrorMessage(pod)
		log.Printf("Pod %s failed: %s", podName, errorMsg)
		statusPatch.SetField("phase", "Failed")
		statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "False",
			Reason:  "PodFailed",
			Message: errorMsg,
		})
		if err := statusPatch.Apply(); err != nil {
			return err
		}
		_ = ensureSessionIsInteractive(namespace, name)
		return DeletePodAndServices(ctx, namespace, podName, name)
	}

	// Check runner container status
	runner := getContainerStatusByName(pod, "ambient-code-runner")
	if runner == nil {
		return statusPatch.Apply()
	}

	if runner.State.Running != nil {
		statusPatch.SetField("phase", "Running")
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionRunnerStarted,
			Status:  "True",
			Reason:  "ContainerRunning",
			Message: "Runner container is executing",
		})
		statusPatch.AddCondition(conditionUpdate{
			Type:    conditionReady,
			Status:  "True",
			Reason:  "Running",
			Message: "Session is running",
		})
		return statusPatch.Apply()
	}

	if runner.State.Waiting != nil {
		waiting := runner.State.Waiting
		errorStates := map[string]bool{
			"ImagePullBackOff":           true,
			"ErrImagePull":               true,
			"CrashLoopBackOff":           true,
			"CreateContainerConfigError": true,
			"InvalidImageName":           true,
		}
		if errorStates[waiting.Reason] {
			msg := fmt.Sprintf("Runner waiting: %s - %s", waiting.Reason, waiting.Message)
			statusPatch.SetField("phase", "Failed")
			statusPatch.SetField("completionTime", time.Now().UTC().Format(time.RFC3339))
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReady,
				Status:  "False",
				Reason:  waiting.Reason,
				Message: msg,
			})
			if err := statusPatch.Apply(); err != nil {
				return err
			}
			_ = ensureSessionIsInteractive(namespace, name)
			return DeletePodAndServices(ctx, namespace, podName, name)
		}
	}

	if runner.State.Terminated != nil {
		term := runner.State.Terminated
		now := time.Now().UTC().Format(time.RFC3339)
		statusPatch.SetField("completionTime", now)

		switch term.ExitCode {
		case 0:
			statusPatch.SetField("phase", "Completed")
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReady,
				Status:  "False",
				Reason:  "Completed",
				Message: "Runner finished",
			})
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
			statusPatch.AddCondition(conditionUpdate{
				Type:    conditionReady,
				Status:  "False",
				Reason:  "RunnerExit",
				Message: msg,
			})
		}

		if err := statusPatch.Apply(); err != nil {
			return err
		}
		_ = ensureSessionIsInteractive(namespace, name)
		return DeletePodAndServices(ctx, namespace, podName, name)
	}

	return statusPatch.Apply()
}

// DeletePodAndServices deletes the pod and associated services.
func DeletePodAndServices(ctx context.Context, namespace, podName, sessionName string) error {
	return deletePodAndPerPodService(namespace, podName, sessionName)
}

// EnsureFreshRunnerToken refreshes the runner token if needed.
func EnsureFreshRunnerToken(ctx context.Context, session *unstructured.Unstructured) error {
	return ensureFreshRunnerToken(ctx, session)
}

// collectPodErrorMessage extracts detailed error information from a failed pod.
func collectPodErrorMessage(pod *corev1.Pod) string {
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
			return msg
		}
		if initStatus.State.Waiting != nil && initStatus.State.Waiting.Reason != "" {
			return fmt.Sprintf("Init container %s: %s - %s",
				initStatus.Name,
				initStatus.State.Waiting.Reason,
				initStatus.State.Waiting.Message)
		}
	}

	// Check main containers for errors if init passed
	if errorMsg == "" || errorMsg == "PodFailed" {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode != 0 {
				return fmt.Sprintf("Container %s failed (exit %d): %s - %s",
					containerStatus.Name,
					containerStatus.State.Terminated.ExitCode,
					containerStatus.State.Terminated.Reason,
					containerStatus.State.Terminated.Message)
			}
			if containerStatus.State.Waiting != nil {
				return fmt.Sprintf("Container %s: %s - %s",
					containerStatus.Name,
					containerStatus.State.Waiting.Reason,
					containerStatus.State.Waiting.Message)
			}
		}
	}

	if errorMsg == "" {
		errorMsg = "Pod failed with unknown error"
	}

	return errorMsg
}
