package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	// ErrInvalidSignature is returned when the webhook signature is invalid
	ErrInvalidSignature = errors.New("invalid webhook signature")
	// ErrMissingSignature is returned when the X-Hub-Signature-256 header is missing
	ErrMissingSignature = errors.New("missing X-Hub-Signature-256 header")
	// ErrInvalidSignatureFormat is returned when the signature format is invalid
	ErrInvalidSignatureFormat = errors.New("signature must start with 'sha256='")
)

// VerifySignature verifies the HMAC-SHA256 signature of a GitHub webhook payload
// using constant-time comparison to prevent timing attacks (FR-002, FR-003, FR-007)
//
// GitHub sends the signature in the X-Hub-Signature-256 header in the format:
// "sha256=<hex_encoded_signature>"
//
// This function:
// 1. Validates the signature header format
// 2. Computes the expected HMAC-SHA256 signature
// 3. Compares using constant-time comparison to prevent timing attacks
//
// Returns nil if signature is valid, error otherwise
func VerifySignature(signatureHeader string, payload []byte, secret string) error {
	// Validate signature header is present (FR-003)
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	// Validate signature format
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return ErrInvalidSignatureFormat
	}

	// Extract the hex-encoded signature
	providedSignatureHex := strings.TrimPrefix(signatureHeader, "sha256=")

	// Decode the provided signature from hex
	providedSignature, err := hex.DecodeString(providedSignatureHex)
	if err != nil {
		return ErrInvalidSignature
	}

	// Compute the expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := mac.Sum(nil)

	// Use constant-time comparison to prevent timing attacks (FR-007)
	// subtle.ConstantTimeCompare returns 1 if equal, 0 otherwise
	if subtle.ConstantTimeCompare(providedSignature, expectedSignature) != 1 {
		return ErrInvalidSignature
	}

	return nil
}

// ComputeSignature computes the HMAC-SHA256 signature for a payload
// This is primarily used for testing and local webhook signature generation
func ComputeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
