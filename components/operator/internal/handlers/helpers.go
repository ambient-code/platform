package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// Progress tracking conditions - these track the session's lifecycle stages
	conditionReady                     = "Ready"
	conditionSecretsReady              = "SecretsReady"
	conditionPodCreated                = "PodCreated"
	conditionPodScheduled              = "PodScheduled"
	conditionRunnerStarted             = "RunnerStarted"
	conditionReposReconciled           = "ReposReconciled"
	conditionWorkflowReconciled        = "WorkflowReconciled"
	conditionReconciled                = "Reconciled"
	runnerTokenSecretAnnotation        = "ambient-code.io/runner-token-secret"
	runnerServiceAccountAnnotation     = "ambient-code.io/runner-sa"
	runnerTokenRefreshedAtAnnotation   = "ambient-code.io/token-refreshed-at"
	runnerTokenRefreshTTL              = 45 * time.Minute
	defaultRunnerTokenSecretPrefix     = "ambient-runner-token-"
	defaultSessionServiceAccountPrefix = "ambient-session-"
)

type conditionUpdate struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// StatusPatch accumulates status field updates and condition changes
// before applying them in a single batch update. This reduces watch events
// by batching multiple status changes into one API call.
type StatusPatch struct {
	Fields     map[string]interface{}
	Conditions []conditionUpdate
	Deletions  map[string]bool // Fields to delete
	Namespace  string
	Name       string
}

// NewStatusPatch creates a new status accumulator for the given session.
func NewStatusPatch(namespace, name string) *StatusPatch {
	return &StatusPatch{
		Fields:     make(map[string]interface{}),
		Conditions: make([]conditionUpdate, 0),
		Deletions:  make(map[string]bool),
		Namespace:  namespace,
		Name:       name,
	}
}

// SetField queues a field update to be applied when Apply() is called.
func (sp *StatusPatch) SetField(key string, value interface{}) {
	delete(sp.Deletions, key) // Remove from deletions if it was there
	sp.Fields[key] = value
}

// DeleteField queues a field deletion to be applied when Apply() is called.
func (sp *StatusPatch) DeleteField(key string) {
	delete(sp.Fields, key) // Remove from fields if it was there
	sp.Deletions[key] = true
}

// AddCondition queues a condition update to be applied when Apply() is called.
func (sp *StatusPatch) AddCondition(cond conditionUpdate) {
	sp.Conditions = append(sp.Conditions, cond)
}

// HasChanges returns true if there are any pending changes to apply.
func (sp *StatusPatch) HasChanges() bool {
	return len(sp.Fields) > 0 || len(sp.Conditions) > 0 || len(sp.Deletions) > 0
}

// Apply executes all accumulated changes in a single API call.
// Returns nil if there are no changes to apply.
func (sp *StatusPatch) Apply() error {
	if !sp.HasChanges() {
		return nil // No changes to apply
	}

	return mutateAgenticSessionStatus(sp.Namespace, sp.Name, func(status map[string]interface{}) {
		// Apply field deletions first
		for key := range sp.Deletions {
			delete(status, key)
		}

		// Apply field updates
		for key, value := range sp.Fields {
			status[key] = value
		}

		// Apply condition updates
		for _, cond := range sp.Conditions {
			setCondition(status, cond)
		}
	})
}

// ApplyAndReset applies all changes and resets the patch for reuse.
// This is useful when you need to apply changes mid-reconciliation
// (e.g., before returning early) but want to continue accumulating.
func (sp *StatusPatch) ApplyAndReset() error {
	err := sp.Apply()
	sp.Fields = make(map[string]interface{})
	sp.Conditions = make([]conditionUpdate, 0)
	sp.Deletions = make(map[string]bool)
	return err
}

// mutateAgenticSessionStatus loads the AgenticSession, applies the mutator to the status map, and persists the result.
func mutateAgenticSessionStatus(sessionNamespace, name string, mutator func(status map[string]interface{})) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping status update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}

	status, ok := obj.Object["status"].(map[string]interface{})
	if !ok {
		status = make(map[string]interface{})
		obj.Object["status"] = status
	}

	mutator(status)

	// Phase is set explicitly by callers - no derivation needed

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s was deleted during status update, skipping", name)
			return nil
		}
		return fmt.Errorf("failed to update AgenticSession status: %w", err)
	}

	return nil
}

// ensureSessionIsInteractive forces spec.interactive=true so sessions can be restarted.
func ensureSessionIsInteractive(sessionNamespace, name string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping interactive update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to read spec for AgenticSession %s: %w", name, err)
	}
	if !found {
		log.Printf("AgenticSession %s has no spec; cannot update interactive flag", name)
		return nil
	}

	if interactive, _, _ := unstructured.NestedBool(spec, "interactive"); interactive {
		return nil
	}

	if err := unstructured.SetNestedField(obj.Object, true, "spec", "interactive"); err != nil {
		return fmt.Errorf("failed to set interactive flag for %s: %w", name, err)
	}

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to persist interactive flag for %s: %w", name, err)
	}

	return nil
}

// updateAnnotations updates annotations on the AgenticSession CR.
func updateAnnotations(sessionNamespace, name string, annotations map[string]string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping annotation update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	obj.SetAnnotations(annotations)

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to update annotations for %s: %w", name, err)
	}

	return nil
}

// clearAnnotation removes a specific annotation from the AgenticSession CR.
func clearAnnotation(sessionNamespace, name, annotationKey string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		return nil
	}

	if _, exists := annotations[annotationKey]; !exists {
		return nil
	}

	delete(annotations, annotationKey)
	obj.SetAnnotations(annotations)

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to clear annotation %s for %s: %w", annotationKey, name, err)
	}

	return nil
}

// setCondition upserts a condition entry on the provided status map.
func setCondition(status map[string]interface{}, update conditionUpdate) {
	now := time.Now().UTC().Format(time.RFC3339)
	conditions, _ := status["conditions"].([]interface{})
	updated := false

	for i, c := range conditions {
		if existing, ok := c.(map[string]interface{}); ok {
			if strings.EqualFold(existing["type"].(string), update.Type) {
				if existing["status"] != update.Status {
					existing["lastTransitionTime"] = now
				}
				existing["status"] = update.Status
				if update.Reason != "" {
					existing["reason"] = update.Reason
				}
				if update.Message != "" {
					existing["message"] = update.Message
				}
				conditions[i] = existing
				updated = true
				break
			}
		}
	}

	if !updated {
		newCond := map[string]interface{}{
			"type":               update.Type,
			"status":             update.Status,
			"reason":             update.Reason,
			"message":            update.Message,
			"lastTransitionTime": now,
		}
		conditions = append(conditions, newCond)
	}

	status["conditions"] = conditions
}

// ensureFreshRunnerToken refreshes the runner SA token if it is older than the allowed TTL.
func ensureFreshRunnerToken(ctx context.Context, session *unstructured.Unstructured) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	namespace := session.GetNamespace()
	if namespace == "" {
		return fmt.Errorf("session namespace is empty")
	}

	annotations := session.GetAnnotations()
	secretName := strings.TrimSpace(annotations[runnerTokenSecretAnnotation])
	if secretName == "" {
		secretName = fmt.Sprintf("%s%s", defaultRunnerTokenSecretPrefix, session.GetName())
	}

	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to fetch runner token secret %s/%s: %w", namespace, secretName, err)
	}

	if secret.Annotations != nil {
		if refreshedAtStr := secret.Annotations[runnerTokenRefreshedAtAnnotation]; refreshedAtStr != "" {
			if refreshedAt, parseErr := time.Parse(time.RFC3339, refreshedAtStr); parseErr == nil {
				if time.Since(refreshedAt) < runnerTokenRefreshTTL {
					return nil
				}
			}
		}
	}

	saName := strings.TrimSpace(annotations[runnerServiceAccountAnnotation])
	if saName == "" {
		saName = fmt.Sprintf("%s%s", defaultSessionServiceAccountPrefix, session.GetName())
	}

	tokenReq := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tokenResp, err := config.K8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saName, tokenReq, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to mint token for %s/%s: %w", namespace, saName, err)
	}
	token := strings.TrimSpace(tokenResp.Status.Token)
	if token == "" {
		return fmt.Errorf("received empty token for %s/%s", namespace, saName)
	}

	secretCopy := secret.DeepCopy()
	if secretCopy.Data == nil {
		secretCopy.Data = map[string][]byte{}
	}
	secretCopy.Data["k8s-token"] = []byte(token)
	if secretCopy.Annotations == nil {
		secretCopy.Annotations = map[string]string{}
	}
	secretCopy.Annotations[runnerTokenRefreshedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	if _, err := config.K8sClient.CoreV1().Secrets(namespace).Update(ctx, secretCopy, v1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update runner token secret %s/%s: %w", namespace, secretName, err)
	}

	log.Printf("Refreshed runner token for session %s/%s", namespace, session.GetName())
	return nil
}
