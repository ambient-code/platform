package webhook

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	// MaxPayloadSize is the maximum allowed webhook payload size (10MB) (FR-005)
	MaxPayloadSize = 10 * 1024 * 1024 // 10MB
)

var (
	// ErrInvalidMethod is returned when the HTTP method is not POST
	ErrInvalidMethod = errors.New("invalid HTTP method, must be POST")
	// ErrInvalidContentType is returned when Content-Type is not application/json
	ErrInvalidContentType = errors.New("invalid Content-Type, must be application/json")
	// ErrPayloadTooLarge is returned when payload exceeds MaxPayloadSize
	ErrPayloadTooLarge = errors.New("payload too large, maximum 10MB")
	// ErrMissingHeaders is returned when required headers are missing
	ErrMissingHeaders = errors.New("missing required webhook headers")
)

// ValidateWebhookRequest validates the basic HTTP request requirements (FR-004, FR-005, FR-010)
// Returns the extracted headers and payload if valid
func ValidateWebhookRequest(r *http.Request) (*GitHubWebhookHeaders, []byte, error) {
	// 1. Validate HTTP method (FR-004)
	if r.Method != http.MethodPost {
		return nil, nil, ErrInvalidMethod
	}

	// 2. Validate Content-Type header (FR-004)
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, nil, ErrInvalidContentType
	}

	// 3. Extract required GitHub webhook headers (FR-006)
	headers := &GitHubWebhookHeaders{
		Signature:   r.Header.Get("X-Hub-Signature-256"),
		Event:       r.Header.Get("X-GitHub-Event"),
		DeliveryID:  r.Header.Get("X-GitHub-Delivery"),
		HookID:      r.Header.Get("X-GitHub-Hook-ID"),
		ContentType: contentType,
	}

	// Validate required headers are present
	if headers.Signature == "" || headers.Event == "" || headers.DeliveryID == "" {
		return nil, nil, ErrMissingHeaders
	}

	// 4. Read payload with size limit (FR-005)
	payload, err := readPayloadWithLimit(r.Body, MaxPayloadSize)
	if err != nil {
		return nil, nil, err
	}

	return headers, payload, nil
}

// readPayloadWithLimit reads the request body up to the specified limit
// Returns ErrPayloadTooLarge if the body exceeds the limit
func readPayloadWithLimit(body io.ReadCloser, maxSize int64) ([]byte, error) {
	// Use io.LimitReader to enforce size limit
	limitedReader := io.LimitReader(body, maxSize+1) // +1 to detect over-limit

	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Check if we exceeded the limit
	if int64(len(payload)) > maxSize {
		return nil, ErrPayloadTooLarge
	}

	return payload, nil
}

// IsSupportedEventType checks if the event type is supported (FR-012)
// Phase 1A only supports issue_comment on PRs
func IsSupportedEventType(eventType string) bool {
	supportedEvents := map[string]bool{
		"issue_comment": true,
		// Phase 1B will add: "pull_request": true,
		// Phase 1C will add: "workflow_run": true,
	}

	return supportedEvents[eventType]
}
