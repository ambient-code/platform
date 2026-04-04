package tokenserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/auth"
	"github.com/rs/zerolog"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	runnerSAPrefix = "system:serviceaccount:"
	sessionSAInfix = ":session-"
	sessionSASuffix = "-sa"
	tokenReviewTimeout = 10 * time.Second
)

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type handler struct {
	tokenProvider auth.TokenProvider
	k8sClient     kubernetes.Interface
	logger        zerolog.Logger
}

func (h *handler) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	saToken, err := extractBearerToken(r)
	if err != nil {
		h.logger.Warn().Err(err).Msg("token request: missing or malformed Authorization header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	username, err := h.validateSAToken(r.Context(), saToken)
	if err != nil {
		h.logger.Warn().Err(err).Msg("token request: SA token validation failed")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !isRunnerSA(username) {
		h.logger.Warn().Str("username", username).Msg("token request: username does not match runner SA pattern")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	apiToken, err := h.tokenProvider.Token(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Str("username", username).Msg("token request: failed to mint API token")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info().Str("username", username).Msg("token request: issued fresh API token")

	resp := tokenResponse{Token: apiToken}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Warn().Err(err).Msg("token request: failed to write response")
	}
}

func (h *handler) validateSAToken(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, tokenReviewTimeout)
	defer cancel()

	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := h.k8sClient.AuthenticationV1().TokenReviews().Create(ctx, tr, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("TokenReview API call failed: %w", err)
	}
	if !result.Status.Authenticated {
		return "", fmt.Errorf("token not authenticated: %s", result.Status.Error)
	}

	return result.Status.User.Username, nil
}

func isRunnerSA(username string) bool {
	if !strings.HasPrefix(username, runnerSAPrefix) {
		return false
	}
	rest := strings.TrimPrefix(username, runnerSAPrefix)
	idx := strings.Index(rest, sessionSAInfix)
	if idx < 0 {
		return false
	}
	return strings.HasSuffix(rest, sessionSASuffix)
}

func extractBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("Authorization header missing")
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("Authorization header must use Bearer scheme")
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}
	return token, nil
}
