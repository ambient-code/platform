package tokenserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/auth"
	"github.com/rs/zerolog"
)

type tokenResponse struct {
	Token string `json:"token"`
}

type handler struct {
	tokenProvider auth.TokenProvider
	privateKey    *rsa.PrivateKey
	logger        zerolog.Logger
}

func (h *handler) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ciphertext, err := extractBearerToken(r)
	if err != nil {
		h.logger.Warn().Err(err).Msg("token request: missing or malformed Authorization header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := h.decryptSessionID(ciphertext)
	if err != nil {
		h.logger.Warn().Err(err).Msg("token request: session ID decryption failed")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !isValidSessionID(sessionID) {
		h.logger.Warn().Str("session_id", sessionID).Msg("token request: decrypted value does not match session ID pattern")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	apiToken, err := h.tokenProvider.Token(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("token request: failed to mint API token")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("session_id", sessionID).Msg("token request: issued fresh API token")

	resp := tokenResponse{Token: apiToken}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Warn().Err(err).Msg("token request: failed to write response")
	}
}

func (h *handler) decryptSessionID(ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64-decoding bearer token: %w", err)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, h.privateKey, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("RSA decryption failed: %w", err)
	}
	return string(plaintext), nil
}

func isValidSessionID(sessionID string) bool {
	return len(sessionID) >= 8 && !strings.ContainsAny(sessionID, " \t\n\r")
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header missing")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("Authorization header must use Bearer scheme")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
