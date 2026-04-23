package tokenserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

type staticTokenProvider struct{ token string }

func (s *staticTokenProvider) Token(_ context.Context) (string, error) {
	return s.token, nil
}

func newTestHandler(t *testing.T) (*handler, *rsa.PrivateKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	h := &handler{
		tokenProvider: &staticTokenProvider{token: "test-api-token"},
		privateKey:    privKey,
		logger:        zerolog.Nop(),
	}
	return h, privKey
}

func encryptSessionID(t *testing.T, pubKey *rsa.PublicKey, sessionID string) string {
	t.Helper()
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(sessionID), nil)
	if err != nil {
		t.Fatalf("encrypting session ID: %v", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func TestHandleToken_Success(t *testing.T) {
	h, privKey := newTestHandler(t)
	bearer := encryptSessionID(t, &privKey.PublicKey, "abc123session")

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d — body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestHandleToken_MissingAuthHeader(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleToken_WrongBearerScheme(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleToken_InvalidBase64(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("Authorization", "Bearer not-valid-base64!!!")
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleToken_WrongKey(t *testing.T) {
	h, _ := newTestHandler(t)

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating other RSA key: %v", err)
	}
	bearer := encryptSessionID(t, &otherKey.PublicKey, "abc123session")

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleToken_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/token", nil)
	rr := httptest.NewRecorder()

	h.handleToken(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestIsValidSessionID(t *testing.T) {
	cases := []struct {
		id    string
		valid bool
	}{
		{"abc12345", true},
		{"3BurtLWQNFMLp61XAGFKILYiHoN", true},
		{"short", false},
		{"has space", false},
		{"has\nnewline", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isValidSessionID(tc.id)
		if got != tc.valid {
			t.Errorf("isValidSessionID(%q) = %v, want %v", tc.id, got, tc.valid)
		}
	}
}

func TestDecryptSessionID_RoundTrip(t *testing.T) {
	h, privKey := newTestHandler(t)
	want := "my-session-id-xyz"
	bearer := encryptSessionID(t, &privKey.PublicKey, want)

	got, err := h.decryptSessionID(bearer)
	if err != nil {
		t.Fatalf("decryptSessionID() error: %v", err)
	}
	if got != want {
		t.Errorf("decryptSessionID() = %q, want %q", got, want)
	}
}
