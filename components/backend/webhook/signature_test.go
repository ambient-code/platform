package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// TestVerifySignature_Valid tests HMAC verification with valid signatures
func TestVerifySignature_Valid(t *testing.T) {
	secret := "test-webhook-secret"
	payload := []byte(`{"action":"created","comment":{"body":"@amber help"}}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	err := VerifySignature(validSignature, payload, secret)
	if err != nil {
		t.Errorf("Expected valid signature to pass, got error: %v", err)
	}
}

// TestVerifySignature_InvalidSignature tests rejection of invalid signatures
func TestVerifySignature_InvalidSignature(t *testing.T) {
	secret := "test-webhook-secret"
	payload := []byte(`{"action":"created"}`)

	testCases := []struct {
		name      string
		signature string
	}{
		{
			name:      "wrong signature",
			signature: "sha256=invalid0123456789abcdef",
		},
		{
			name:      "wrong secret used",
			signature: func() string {
				mac := hmac.New(sha256.New, []byte("wrong-secret"))
				mac.Write(payload)
				return "sha256=" + hex.EncodeToString(mac.Sum(nil))
			}(),
		},
		{
			name:      "missing sha256 prefix",
			signature: "invalid0123456789abcdef",
		},
		{
			name:      "empty signature",
			signature: "",
		},
		{
			name:      "malformed hex",
			signature: "sha256=notvalidhex",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifySignature(tc.signature, payload, secret)
			if err == nil {
				t.Errorf("Expected invalid signature to be rejected")
			}
		})
	}
}

// TestVerifySignature_ConstantTime tests that verification is constant-time
// This is critical to prevent timing attacks (FR-007)
func TestVerifySignature_ConstantTime(t *testing.T) {
	secret := "test-webhook-secret-for-timing-analysis"
	payload := []byte(`{"action":"created","comment":{"body":"test payload"}}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	// Create signatures with different numbers of matching prefix bytes
	signatures := []string{
		"0000000000000000000000000000000000000000000000000000000000000000", // 0 bytes match
		validSig[:2] + strings.Repeat("0", 62),                              // 1 byte matches
		validSig[:8] + strings.Repeat("0", 56),                              // 4 bytes match
		validSig[:32] + strings.Repeat("0", 32),                             // 16 bytes match
		validSig[:62] + "00",                                                // 31 bytes match
		validSig,                                                            // all bytes match (valid)
	}

	// Measure timing for each signature
	iterations := 1000
	timings := make([]time.Duration, len(signatures))

	for i, sig := range signatures {
		start := time.Now()
		for j := 0; j < iterations; j++ {
			_ = VerifySignature("sha256="+sig, payload, secret)
		}
		timings[i] = time.Since(start)
	}

	// Calculate variance - constant-time should have low variance
	var sum time.Duration
	for _, t := range timings {
		sum += t
	}
	mean := sum / time.Duration(len(timings))

	var variance float64
	for _, t := range timings {
		diff := float64(t - mean)
		variance += diff * diff
	}
	variance /= float64(len(timings))
	stddev := time.Duration(variance)

	// Constant-time comparison should have < 5% variance
	// (allows for normal system timing jitter)
	maxVariance := mean / 20 // 5%

	if stddev > maxVariance {
		t.Logf("Timing analysis:")
		for i, sig := range signatures {
			t.Logf("  Signature %d (%d bytes match): %v", i, countMatchingBytes(sig, validSig), timings[i])
		}
		t.Logf("  Mean: %v, StdDev: %v, Max variance: %v", mean, stddev, maxVariance)
		t.Errorf("Timing variance too high - may be vulnerable to timing attacks")
	}
}

// countMatchingBytes counts how many prefix bytes match
func countMatchingBytes(a, b string) int {
	count := 0
	for i := 0; i < len(a) && i < len(b); i += 2 {
		if a[i:i+2] == b[i:i+2] {
			count++
		} else {
			break
		}
	}
	return count
}

// TestVerifySignature_DifferentPayloads tests that payload changes invalidate signature
func TestVerifySignature_DifferentPayloads(t *testing.T) {
	secret := "test-webhook-secret"
	originalPayload := []byte(`{"action":"created"}`)

	// Generate signature for original payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(originalPayload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Try with modified payloads
	modifiedPayloads := [][]byte{
		[]byte(`{"action":"edited"}`),                // Changed field value
		[]byte(`{"action":"created","extra":"data"}`), // Added field
		[]byte(`{"action":"created"} `),              // Extra whitespace
		[]byte(`{"action":"created",}`),              // Extra comma
	}

	for i, modifiedPayload := range modifiedPayloads {
		t.Run("modified_"+string(rune(i)), func(t *testing.T) {
			err := VerifySignature(signature, modifiedPayload, secret)
			if err == nil {
				t.Errorf("Expected modified payload to fail verification")
			}
		})
	}
}

// TestVerifySignature_EmptyPayload tests edge case of empty payload
func TestVerifySignature_EmptyPayload(t *testing.T) {
	secret := "test-webhook-secret"
	payload := []byte{}

	// Generate valid signature for empty payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	err := VerifySignature(signature, payload, secret)
	if err != nil {
		t.Errorf("Expected empty payload with valid signature to pass, got error: %v", err)
	}
}

// TestVerifySignature_LargePayload tests with large payloads (near 10MB limit)
func TestVerifySignature_LargePayload(t *testing.T) {
	secret := "test-webhook-secret"
	// Create 5MB payload (well under 10MB limit)
	payload := make([]byte, 5*1024*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	start := time.Now()
	err := VerifySignature(signature, payload, secret)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected large payload with valid signature to pass, got error: %v", err)
	}

	// Verify performance - should complete in < 100ms even for large payload
	if duration > 100*time.Millisecond {
		t.Errorf("Signature verification took too long for large payload: %v", duration)
	}
}
