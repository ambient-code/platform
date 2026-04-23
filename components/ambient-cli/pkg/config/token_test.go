package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func makeJWT(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret"))
	return signed
}

func TestTokenExpiryValid(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	token := makeJWT(jwt.MapClaims{"exp": float64(future.Unix())})

	exp, err := TokenExpiry(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.IsZero() {
		t.Fatal("expected non-zero expiry")
	}
	diff := exp.Sub(future)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expiry mismatch: got %v, want ~%v", exp, future)
	}
}

func TestTokenExpiryNoClaim(t *testing.T) {
	token := makeJWT(jwt.MapClaims{"sub": "user123"})

	exp, err := TokenExpiry(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exp.IsZero() {
		t.Errorf("expected zero time for missing exp, got %v", exp)
	}
}

func TestTokenExpirySHA256Prefix(t *testing.T) {
	exp, err := TokenExpiry("sha256~abcdef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exp.IsZero() {
		t.Errorf("expected zero time for sha256~ token, got %v", exp)
	}
}

func TestTokenExpiryInvalidToken(t *testing.T) {
	_, err := TokenExpiry("not.a.jwt.token.at.all")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestIsTokenExpiredTrue(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	token := makeJWT(jwt.MapClaims{"exp": float64(past.Unix())})

	expired, err := IsTokenExpired(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expired {
		t.Error("expected token to be expired")
	}
}

func TestIsTokenExpiredFalse(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	token := makeJWT(jwt.MapClaims{"exp": float64(future.Unix())})

	expired, err := IsTokenExpired(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expired {
		t.Error("expected token to not be expired")
	}
}

func TestIsTokenExpiredNoExp(t *testing.T) {
	token := makeJWT(jwt.MapClaims{"sub": "user123"})

	expired, err := IsTokenExpired(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expired {
		t.Error("expected non-expired for token without exp claim")
	}
}

func TestRefreshAccessToken_Success(t *testing.T) {
	newAccessToken := makeJWT(jwt.MapClaims{"exp": float64(time.Now().Add(1 * time.Hour).Unix())})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("refresh_token") != "my-refresh-token" {
			t.Errorf("expected refresh_token=my-refresh-token, got %q", r.FormValue("refresh_token"))
		}
		if r.FormValue("client_id") != "test-client" {
			t.Errorf("expected client_id=test-client, got %q", r.FormValue("client_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"new-refresh"}`, newAccessToken)
	}))
	defer srv.Close()

	access, refresh, err := RefreshAccessToken(srv.URL, "test-client", "my-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access != newAccessToken {
		t.Errorf("access token mismatch")
	}
	if refresh != "new-refresh" {
		t.Errorf("expected new-refresh, got %q", refresh)
	}
}

func TestRefreshAccessToken_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"Token is expired"}`)
	}))
	defer srv.Close()

	_, _, err := RefreshAccessToken(srv.URL, "test-client", "bad-refresh")
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
}

func TestRefreshAccessToken_NoAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"refresh_token":"new-refresh"}`)
	}))
	defer srv.Close()

	_, _, err := RefreshAccessToken(srv.URL, "test-client", "my-refresh")
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
}

func TestGetTokenWithRefresh_ValidToken(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "")
	validToken := makeJWT(jwt.MapClaims{"exp": float64(time.Now().Add(1 * time.Hour).Unix())})
	cfg := &Config{AccessToken: validToken}

	token, err := cfg.GetTokenWithRefresh()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != validToken {
		t.Error("expected valid token returned as-is")
	}
}

func TestGetTokenWithRefresh_ExpiredNoRefreshToken(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "")
	expiredToken := makeJWT(jwt.MapClaims{"exp": float64(time.Now().Add(-1 * time.Hour).Unix())})
	cfg := &Config{AccessToken: expiredToken}

	token, err := cfg.GetTokenWithRefresh()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != expiredToken {
		t.Error("expected expired token returned when no refresh available")
	}
}

func TestGetTokenWithRefresh_ExpiredWithRefresh(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "")
	expiredToken := makeJWT(jwt.MapClaims{"exp": float64(time.Now().Add(-1 * time.Hour).Unix())})
	newAccessToken := makeJWT(jwt.MapClaims{"exp": float64(time.Now().Add(1 * time.Hour).Unix())})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal(map[string]string{
			"access_token":  newAccessToken,
			"refresh_token": "rotated-refresh",
		})
		w.Write(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("AMBIENT_CONFIG", dir+"/config.json")

	cfg := &Config{
		AccessToken:  expiredToken,
		RefreshToken: "old-refresh",
		IssuerURL:    srv.URL,
		ClientID:     "test-client",
	}

	token, err := cfg.GetTokenWithRefresh()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != newAccessToken {
		t.Error("expected refreshed access token")
	}
	if cfg.RefreshToken != "rotated-refresh" {
		t.Errorf("expected rotated refresh token, got %q", cfg.RefreshToken)
	}
}

func TestGetTokenWithRefresh_EnvOverride(t *testing.T) {
	t.Setenv("AMBIENT_TOKEN", "env-token-at-least-20chars")
	cfg := &Config{AccessToken: "config-token"}

	token, err := cfg.GetTokenWithRefresh()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token-at-least-20chars" {
		t.Errorf("expected env token, got %q", token)
	}
}
