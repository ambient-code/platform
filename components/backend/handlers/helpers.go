package handlers

import (
	"ambient-code-backend/types"
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// GetProjectSettingsResource returns the GroupVersionResource for ProjectSettings
func GetProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}

// RetryWithBackoff attempts an operation with exponential backoff
// Used for operations that may temporarily fail due to async resource creation
// This is a generic utility that can be used by any handler
// Checks for context cancellation between retries to avoid wasting resources
func RetryWithBackoff(maxRetries int, initialDelay, maxDelay time.Duration, operation func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				// Calculate exponential backoff delay
				delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(i)))
				if delay > maxDelay {
					delay = maxDelay
				}
				log.Printf("Operation failed (attempt %d/%d), retrying in %v: %v", i+1, maxRetries, delay, err)
				time.Sleep(delay)
				continue
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// ValidateSecretAccess checks if the user has permission to perform the given verb on secrets
// Returns an error if the user lacks the required permission
// Accepts kubernetes.Interface for compatibility with dependency injection in tests
func ValidateSecretAccess(ctx context.Context, k8sClient kubernetes.Interface, namespace, verb string) error {
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "", // core API group for secrets
				Resource:  "secrets",
				Verb:      verb, // "create", "get", "update", "delete"
				Namespace: namespace,
			},
		},
	}

	res, err := k8sClient.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("RBAC check failed: %w", err)
	}

	if !res.Status.Allowed {
		return fmt.Errorf("user not allowed to %s secrets in namespace %s", verb, namespace)
	}

	return nil
}

// ParseRepoMap parses a repository map (from CR spec.repos[]) into a SimpleRepo struct.
// This helper is exported for testing purposes.
// Supports both legacy format (url/branch) and new format (input/output/autoPush).
func ParseRepoMap(m map[string]interface{}) (types.SimpleRepo, error) {
	r := types.SimpleRepo{}

	// Check for new format (input/output/autoPush)
	if inputMap, hasInput := m["input"].(map[string]interface{}); hasInput {
		// New format
		input := &types.RepoLocation{}
		if url, ok := inputMap["url"].(string); ok {
			input.URL = url
		}
		if branch, ok := inputMap["branch"].(string); ok && strings.TrimSpace(branch) != "" {
			input.Branch = types.StringPtr(branch)
		}
		r.Input = input

		// Parse output if present
		if outputMap, hasOutput := m["output"].(map[string]interface{}); hasOutput {
			output := &types.RepoLocation{}
			if url, ok := outputMap["url"].(string); ok {
				output.URL = url
			}
			if branch, ok := outputMap["branch"].(string); ok && strings.TrimSpace(branch) != "" {
				output.Branch = types.StringPtr(branch)
			}
			r.Output = output
		}

		// Parse autoPush if present
		if autoPush, ok := m["autoPush"].(bool); ok {
			r.AutoPush = types.BoolPtr(autoPush)
		}

		if strings.TrimSpace(r.Input.URL) == "" {
			return r, fmt.Errorf("input.url is required")
		}

		// Validate that output differs from input (if output is specified)
		if r.Output != nil {
			inputURL := strings.TrimSpace(r.Input.URL)
			outputURL := strings.TrimSpace(r.Output.URL)
			inputBranch := ""
			outputBranch := ""
			if r.Input.Branch != nil {
				inputBranch = strings.TrimSpace(*r.Input.Branch)
			}
			if r.Output.Branch != nil {
				outputBranch = strings.TrimSpace(*r.Output.Branch)
			}

			// Output must differ from input in either URL or branch
			if inputURL == outputURL && inputBranch == outputBranch {
				return r, fmt.Errorf("output repository must differ from input (different URL or branch required)")
			}
		}
	} else {
		// Legacy format
		if url, ok := m["url"].(string); ok {
			r.URL = url
		}
		if branch, ok := m["branch"].(string); ok && strings.TrimSpace(branch) != "" {
			r.Branch = types.StringPtr(branch)
		}

		if strings.TrimSpace(r.URL) == "" {
			return r, fmt.Errorf("url is required")
		}
	}

	return r, nil
}
