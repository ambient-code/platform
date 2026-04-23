package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGeneratePKCE_LengthAndUniqueness(t *testing.T) {
	v1, c1, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE: %v", err)
	}
	if v1 == "" || c1 == "" {
		t.Fatal("expected non-empty verifier and challenge")
	}

	v2, c2, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE second call: %v", err)
	}
	if v1 == v2 {
		t.Error("expected unique verifiers across calls")
	}
	if c1 == c2 {
		t.Error("expected unique challenges across calls")
	}
}

func TestGeneratePKCE_ValidBase64URL(t *testing.T) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE: %v", err)
	}

	if len(verifier) < 40 {
		t.Errorf("verifier too short: %d chars", len(verifier))
	}
	if len(challenge) < 40 {
		t.Errorf("challenge too short: %d chars", len(challenge))
	}

	const base64URLChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for _, c := range verifier + challenge {
		if !strings.ContainsRune(base64URLChars, c) {
			t.Errorf("unexpected character in PKCE output: %q", c)
		}
	}
}

func TestGenerateRandomState_Uniqueness(t *testing.T) {
	s1, err := generateRandomState()
	if err != nil {
		t.Fatalf("generateRandomState: %v", err)
	}
	s2, err := generateRandomState()
	if err != nil {
		t.Fatalf("generateRandomState second call: %v", err)
	}
	if s1 == s2 {
		t.Error("expected unique states across calls")
	}
	if len(s1) < 20 {
		t.Errorf("state too short: %q", s1)
	}
}

func TestBuildAuthURL_ContainsRequiredParams(t *testing.T) {
	authURL := buildAuthURL(
		"https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/auth",
		"acpctl",
		"http://127.0.0.1:12345/callback",
		"test-state",
		"test-challenge",
	)

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}

	q := parsed.Query()
	checks := map[string]string{
		"response_type":         "code",
		"client_id":             "acpctl",
		"redirect_uri":          "http://127.0.0.1:12345/callback",
		"state":                 "test-state",
		"code_challenge":        "test-challenge",
		"code_challenge_method": "S256",
	}
	for param, want := range checks {
		if got := q.Get(param); got != want {
			t.Errorf("param %q: got %q, want %q", param, got, want)
		}
	}
}

func TestCallbackHandler_ValidCode(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("expected-state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=expected-state&code=authcode123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := <-resultCh
	if result.err != nil {
		t.Errorf("unexpected error: %v", result.err)
	}
	if result.code != "authcode123" {
		t.Errorf("expected code %q, got %q", "authcode123", result.code)
	}
}

func TestCallbackHandler_StateMismatch(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("expected-state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=wrong-state&code=authcode123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	result := <-resultCh
	if result.err == nil {
		t.Fatal("expected error for state mismatch")
	}
	if !strings.Contains(result.err.Error(), "CSRF") {
		t.Errorf("expected CSRF mention in error, got: %v", result.err)
	}
}

func TestCallbackHandler_OAuthError(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("expected-state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied&error_description=User+denied+access", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	result := <-resultCh
	if result.err == nil {
		t.Fatal("expected error for OAuth error response")
	}
	if !strings.Contains(result.err.Error(), "User denied access") {
		t.Errorf("expected error description in error, got: %v", result.err)
	}
}

func TestCallbackHandler_MissingCode(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("expected-state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=expected-state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	result := <-resultCh
	if result.err == nil {
		t.Fatal("expected error for missing code")
	}
}

func TestCallbackHandler_WrongPath(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("expected-state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/other", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for wrong path, got %d", w.Code)
	}
	if len(resultCh) != 0 {
		t.Error("expected no result sent for wrong path")
	}
}

func TestParseTokenResponse_JSON(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"access_token": "my-access-token",
		"token_type":   "Bearer",
	})

	token, err := parseTokenResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-access-token" {
		t.Errorf("expected %q, got %q", "my-access-token", token)
	}
}

func TestParseTokenResponse_MissingAccessToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"token_type": "Bearer"})

	_, err := parseTokenResponse(body)
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
}

func TestParseTokenResponse_InvalidJSON(t *testing.T) {
	_, err := parseTokenResponse([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestTokenEndpointError_WithDescription(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"error":             "invalid_client",
		"error_description": "Client authentication failed",
	})

	err := tokenEndpointError(http.StatusUnauthorized, body)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Client authentication failed") {
		t.Errorf("expected description in error, got: %v", err)
	}
}

func TestTokenEndpointError_ErrorOnly(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"error": "invalid_grant"})

	err := tokenEndpointError(http.StatusBadRequest, body)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("expected error code in message, got: %v", err)
	}
}

func TestTokenEndpointError_NonJSON(t *testing.T) {
	err := tokenEndpointError(http.StatusServiceUnavailable, []byte("Service Unavailable"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected HTTP status in fallback error, got: %v", err)
	}
}

func TestExchangeCodeForToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "mycode" {
			t.Errorf("expected code=mycode, got %q", r.FormValue("code"))
		}
		if r.FormValue("code_verifier") != "myverifier" {
			t.Errorf("expected code_verifier=myverifier, got %q", r.FormValue("code_verifier"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"returned-token","token_type":"Bearer"}`)
	}))
	defer srv.Close()

	token, err := exchangeCodeForToken(srv.URL, "client-id", "", "mycode", "http://127.0.0.1:9/callback", "myverifier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "returned-token" {
		t.Errorf("expected %q, got %q", "returned-token", token)
	}
}

func TestExchangeCodeForToken_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"invalid_client","error_description":"Bad credentials"}`)
	}))
	defer srv.Close()

	_, err := exchangeCodeForToken(srv.URL, "bad-client", "", "code", "http://127.0.0.1:9/callback", "verifier")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "Bad credentials") {
		t.Errorf("expected error description in error, got: %v", err)
	}
}

func TestParseTokensResponse_WithRefreshToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"access_token":  "my-access-token",
		"refresh_token": "my-refresh-token",
		"token_type":    "Bearer",
	})

	result, err := parseTokensResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "my-access-token" {
		t.Errorf("expected %q, got %q", "my-access-token", result.AccessToken)
	}
	if result.RefreshToken != "my-refresh-token" {
		t.Errorf("expected %q, got %q", "my-refresh-token", result.RefreshToken)
	}
}

func TestParseTokensResponse_NoRefreshToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"access_token": "my-access-token",
		"token_type":   "Bearer",
	})

	result, err := parseTokensResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "my-access-token" {
		t.Errorf("expected %q, got %q", "my-access-token", result.AccessToken)
	}
	if result.RefreshToken != "" {
		t.Errorf("expected empty refresh token, got %q", result.RefreshToken)
	}
}

func TestParseTokensResponse_MissingAccessToken(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"refresh_token": "refresh-only"})

	_, err := parseTokensResponse(body)
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
}

func TestExchangeCodeForTokens_ReturnsRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"access-123","refresh_token":"refresh-456","token_type":"Bearer"}`)
	}))
	defer srv.Close()

	result, err := exchangeCodeForTokens(srv.URL, "client-id", "", "mycode", "http://127.0.0.1:9/callback", "myverifier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "access-123" {
		t.Errorf("expected %q, got %q", "access-123", result.AccessToken)
	}
	if result.RefreshToken != "refresh-456" {
		t.Errorf("expected %q, got %q", "refresh-456", result.RefreshToken)
	}
}

func TestCallbackHandler_OAuthErrorFallsBackToErrorCode(t *testing.T) {
	resultCh := make(chan authCodeResult, 1)
	handler := callbackHandler("state", resultCh)

	req := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	result := <-resultCh
	if result.err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(result.err.Error(), "access_denied") {
		t.Errorf("expected error code as fallback, got: %v", result.err)
	}
}
