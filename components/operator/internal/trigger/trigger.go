// Package trigger implements the session-trigger subcommand that creates AgenticSession CRs from scheduled session templates.
package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"ambient-code-operator/internal/types"
)

// RunSessionTrigger creates an AgenticSession CR from a scheduled session template and exits.
func RunSessionTrigger() {
	sessionTemplate := os.Getenv("SESSION_TEMPLATE")
	projectNamespace := os.Getenv("PROJECT_NAMESPACE")
	scheduledSessionName := os.Getenv("SCHEDULED_SESSION_NAME")

	if sessionTemplate == "" || projectNamespace == "" || scheduledSessionName == "" {
		log.Fatalf("Required environment variables SESSION_TEMPLATE, PROJECT_NAMESPACE, and SCHEDULED_SESSION_NAME must be set")
	}

	// Init K8s client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// Parse session template
	var template map[string]interface{}
	if err := json.Unmarshal([]byte(sessionTemplate), &template); err != nil {
		log.Fatalf("Failed to parse SESSION_TEMPLATE JSON: %v", err)
	}

	// Build session name and display name
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	namePrefix := scheduledSessionName
	if dn, ok := template["displayName"].(string); ok && dn != "" {
		namePrefix = sanitizeName(dn)
		// Set display name with human-readable timestamp, e.g. "Daily Jira Summary (Jan 1, 2026 - 00:00:00)"
		template["displayName"] = fmt.Sprintf("%s (%s)", dn, now.UTC().Format("Jan 2, 2006 - 15:04:05"))
	}
	sessionName := fmt.Sprintf("%s-%s", namePrefix, ts)

	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": projectNamespace,
				"labels": map[string]interface{}{
					"ambient-code.io/scheduled-session-name": scheduledSessionName,
					"ambient-code.io/scheduled-run":          "true",
				},
			},
			"spec": template,
		},
	}

	// Create via dynamic client
	gvr := types.GetAgenticSessionResource()
	_, err = dynamicClient.Resource(gvr).Namespace(projectNamespace).Create(context.TODO(), session, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create AgenticSession %s in namespace %s: %v", sessionName, projectNamespace, err)
	}

	log.Printf("Successfully created AgenticSession %s in namespace %s", sessionName, projectNamespace)
}

// sanitizeName converts a display name to a valid Kubernetes resource name prefix.
// Lowercases, replaces non-alphanumeric with hyphens, trims, and limits to 40 chars.
func sanitizeName(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
			result = append(result, ch)
		case ch >= 'A' && ch <= 'Z':
			result = append(result, ch+32) // lowercase
		default:
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	// Trim trailing hyphens
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	if len(result) > 40 {
		result = result[:40]
	}
	if len(result) == 0 {
		return "run"
	}
	return string(result)
}
