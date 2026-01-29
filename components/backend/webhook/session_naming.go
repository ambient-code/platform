package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateSessionName creates a deterministic session name from webhook context (FR-024)
//
// The session name format is: webhook-{owner}-{repo}-pr{number}-{delivery-hash}
// or: webhook-{owner}-{repo}-issue{number}-{delivery-hash}
//
// This ensures:
// 1. Session names are deterministic - same webhook = same name
// 2. Kubernetes will reject duplicate session creation attempts on pod restart
// 3. Webhook retries don't create duplicate sessions
// 4. Session names are DNS-1123 compliant (lowercase, alphanumeric, hyphens)
//
// Example: webhook-acme-backend-pr42-a1b2c3d4
func GenerateSessionName(repository string, prNumber *int, issueNumber *int, deliveryID string) string {
	// Extract owner and repo from "owner/repo" format
	parts := strings.Split(repository, "/")
	owner := "unknown"
	repo := "unknown"
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	}

	// Sanitize owner and repo for DNS-1123 compliance
	owner = sanitizeDNS(owner)
	repo = sanitizeDNS(repo)

	// Hash the delivery ID to get a short, deterministic suffix
	hash := sha256.Sum256([]byte(deliveryID))
	hashHex := hex.EncodeToString(hash[:])[:8] // Use first 8 chars

	// Determine the identifier (PR or issue number)
	identifier := "unknown"
	if prNumber != nil {
		identifier = fmt.Sprintf("pr%d", *prNumber)
	} else if issueNumber != nil {
		identifier = fmt.Sprintf("issue%d", *issueNumber)
	}

	// Construct the session name
	sessionName := fmt.Sprintf("webhook-%s-%s-%s-%s", owner, repo, identifier, hashHex)

	// Ensure total length doesn't exceed Kubernetes name limit (63 characters)
	if len(sessionName) > 63 {
		// Truncate the middle part but keep the hash for uniqueness
		maxOwnerRepoLen := 63 - len("webhook-") - len(identifier) - len(hashHex) - 3 // 3 hyphens
		combinedOwnerRepo := owner + "-" + repo
		if len(combinedOwnerRepo) > maxOwnerRepoLen {
			combinedOwnerRepo = combinedOwnerRepo[:maxOwnerRepoLen]
		}
		sessionName = fmt.Sprintf("webhook-%s-%s-%s", combinedOwnerRepo, identifier, hashHex)
	}

	return sessionName
}

// sanitizeDNS converts a string to be DNS-1123 compliant
// - Lowercase only
// - Alphanumeric and hyphens only
// - No leading/trailing hyphens
func sanitizeDNS(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace invalid characters with hyphens
	var builder strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}

	result := builder.String()

	// Remove leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Collapse multiple consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	return result
}

// GenerateSessionNameForWorkflow creates a session name for workflow_run events
// Format: webhook-{owner}-{repo}-workflow{run-id}-{delivery-hash}
func GenerateSessionNameForWorkflow(repository string, workflowRunID int64, deliveryID string) string {
	// Extract owner and repo
	parts := strings.Split(repository, "/")
	owner := "unknown"
	repo := "unknown"
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	}

	// Sanitize for DNS-1123
	owner = sanitizeDNS(owner)
	repo = sanitizeDNS(repo)

	// Hash delivery ID
	hash := sha256.Sum256([]byte(deliveryID))
	hashHex := hex.EncodeToString(hash[:])[:8]

	// Construct name
	identifier := fmt.Sprintf("wf%d", workflowRunID)
	sessionName := fmt.Sprintf("webhook-%s-%s-%s-%s", owner, repo, identifier, hashHex)

	// Ensure length limit
	if len(sessionName) > 63 {
		maxOwnerRepoLen := 63 - len("webhook-") - len(identifier) - len(hashHex) - 3
		combinedOwnerRepo := owner + "-" + repo
		if len(combinedOwnerRepo) > maxOwnerRepoLen {
			combinedOwnerRepo = combinedOwnerRepo[:maxOwnerRepoLen]
		}
		sessionName = fmt.Sprintf("webhook-%s-%s-%s", combinedOwnerRepo, identifier, hashHex)
	}

	return sessionName
}
