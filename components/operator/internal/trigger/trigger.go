// Package trigger implements the session-trigger subcommand that creates AgenticSession CRs from scheduled session templates.
package trigger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"ambient-code-operator/internal/handlers"
	"ambient-code-operator/internal/types"
)

// RunSessionTrigger creates an AgenticSession CR from a scheduled session template and exits.
// If REUSE_LAST_SESSION is true, it attempts to reuse the most recent session instead of creating a new one.
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

	// Check if reuse mode is enabled
	reuseLastSession := strings.EqualFold(strings.TrimSpace(os.Getenv("REUSE_LAST_SESSION")), "true")

	if reuseLastSession {
		reused, err := tryReuseLastSession(dynamicClient, projectNamespace, scheduledSessionName, template)
		if err != nil {
			// Don't fall through to create — the reuse may have partially succeeded
			log.Fatalf("Failed to reuse last session for %s: %v", scheduledSessionName, err)
		}
		if reused {
			return
		}
		// No reusable session found — fall through to create a new one
	}

	createNewSession(dynamicClient, projectNamespace, scheduledSessionName, template)
}

// tryReuseLastSession finds the most recent session for this scheduled session and either
// sends a follow-up prompt (if running) or resumes it (if stopped/completed).
// Returns true if a session was reused, false if a new session should be created.
func tryReuseLastSession(dynamicClient dynamic.Interface, namespace, scheduledSessionName string, template map[string]interface{}) (bool, error) {
	gvr := types.GetAgenticSessionResource()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("ambient-code.io/scheduled-session-name=%s", scheduledSessionName),
	})
	if err != nil {
		return false, fmt.Errorf("failed to list sessions: %v", err)
	}

	if len(list.Items) == 0 {
		log.Printf("No previous sessions found for scheduled session %s, creating new", scheduledSessionName)
		return false, nil
	}

	var latest *unstructured.Unstructured
	var latestTime time.Time
	for i := range list.Items {
		item := &list.Items[i]
		ct := item.GetCreationTimestamp().Time
		if latest == nil || ct.After(latestTime) {
			latest = item
			latestTime = ct
		}
	}

	phase := ""
	if status, ok := latest.Object["status"].(map[string]interface{}); ok {
		if p, ok := status["phase"].(string); ok {
			phase = p
		}
	}

	sessionName := latest.GetName()
	prompt := ""
	if p, ok := template["initialPrompt"].(string); ok {
		prompt = p
	}

	log.Printf("Found latest session %s with phase %s for scheduled session %s", sessionName, phase, scheduledSessionName)

	switch phase {
	case "Running":
		// Session is running — send a follow-up prompt via the runner's AG-UI endpoint
		if prompt == "" {
			log.Printf("Session %s is running but no prompt to send, skipping", sessionName)
			return true, nil
		}
		log.Printf("Session %s is running, sending follow-up prompt (%d chars)", sessionName, len(prompt))
		if err := sendFollowUpPrompt(namespace, sessionName, prompt); err != nil {
			return false, fmt.Errorf("failed to send follow-up to running session %s: %v", sessionName, err)
		}
		return true, nil

	case "Stopped", "Completed":
		// Session is stopped/completed — resume it with the prompt
		log.Printf("Session %s is %s, resuming with prompt", sessionName, phase)
		if err := resumeSessionWithPrompt(dynamicClient, namespace, sessionName, prompt); err != nil {
			return false, fmt.Errorf("failed to resume session %s: %v", sessionName, err)
		}
		return true, nil

	default:
		// Pending, Creating, Stopping, Failed — create a new session
		log.Printf("Session %s is in phase %s (not reusable), creating new session", sessionName, phase)
		return false, nil
	}
}

// sendFollowUpPrompt sends a prompt to a running session via the runner's AG-UI endpoint.
func sendFollowUpPrompt(namespace, sessionName, prompt string) error {
	runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:%d/", sessionName, namespace, handlers.DefaultRunnerPort)

	input := map[string]interface{}{
		"threadId": sessionName,
		"runId":    fmt.Sprintf("scheduled-%d", time.Now().Unix()),
		"messages": []map[string]interface{}{
			{
				"id":      fmt.Sprintf("scheduled-msg-%d", time.Now().UnixNano()),
				"role":    "user",
				"content": prompt,
			},
		},
	}

	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal run input: %v", err)
	}

	req, err := http.NewRequest("POST", runnerURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to runner: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("runner returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent follow-up prompt to session %s (status %d)", sessionName, resp.StatusCode)
	return nil
}

// resumeSessionWithPrompt resumes a stopped/completed session by setting annotations
// that the operator will reconcile. It also updates the initial prompt so the runner
// auto-executes it on startup.
func resumeSessionWithPrompt(dynamicClient dynamic.Interface, namespace, sessionName, prompt string) error {
	gvr := types.GetAgenticSessionResource()

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		item, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, sessionName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get session: %v", err)
		}

		annotations := item.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["ambient-code.io/desired-phase"] = "Running"
		annotations["ambient-code.io/start-requested-at"] = time.Now().Format(time.RFC3339)

		if prompt != "" {
			spec, _ := item.Object["spec"].(map[string]interface{})
			if spec == nil {
				spec = map[string]interface{}{}
			}
			spec["initialPrompt"] = prompt
			item.Object["spec"] = spec
			// Only force prompt execution when there's a prompt to execute
			annotations["ambient-code.io/force-execute-prompt"] = "true"
		}

		item.SetAnnotations(annotations)

		_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, item, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		log.Printf("Successfully set resume annotations on session %s", sessionName)
		return nil
	})
}

// createNewSession creates a new AgenticSession CR (original behavior).
func createNewSession(dynamicClient dynamic.Interface, namespace, scheduledSessionName string, template map[string]interface{}) {
	// Build session name and display name.
	// The most restrictive derived K8s resource name is the Service:
	//   "session-" (8 chars) + sessionName ≤ 63  →  sessionName ≤ 55
	// sanitizeName caps at 40 chars, so namePrefix + "-" + timestamp (10)
	// yields at most 51 chars — well within the 55-char budget.
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	namePrefix := sanitizeName(scheduledSessionName)
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
				"namespace": namespace,
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, session, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create AgenticSession %s in namespace %s: %v", sessionName, namespace, err)
	}

	log.Printf("Successfully created AgenticSession %s in namespace %s", sessionName, namespace)
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
	if len(result) > 40 {
		result = result[:40]
	}
	// Trim trailing hyphens (must be after truncation, which can reintroduce them)
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	if len(result) == 0 {
		return "run"
	}
	return string(result)
}
