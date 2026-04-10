package tokenexchange

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("marshaling public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privKey, string(pubPEM)
}

func decryptBearer(t *testing.T, privKey *rsa.PrivateKey, bearer string) string {
	t.Helper()
	ciphertext, err := base64.StdEncoding.DecodeString(bearer)
	if err != nil {
		t.Fatalf("decoding bearer base64: %v", err)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, ciphertext, nil)
	if err != nil {
		t.Fatalf("decrypting bearer: %v", err)
	}
	return string(plaintext)
}

func newTokenServer(t *testing.T, privKey *rsa.PrivateKey, apiToken string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		bearer := auth[7:]
		sessionID := decryptBearer(t, privKey, bearer)
		if len(sessionID) < 8 {
			http.Error(w, "invalid session ID", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: apiToken})
	}))
}

func TestEncryptSessionID_Roundtrip(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	pubKey, err := parsePublicKey(pubPEM)
	if err != nil {
		t.Fatalf("parsePublicKey: %v", err)
	}

	sessionID := "test-session-abc123"
	encrypted, err := encryptSessionID(pubKey, sessionID)
	if err != nil {
		t.Fatalf("encryptSessionID: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, ciphertext, nil)
	if err != nil {
		t.Fatalf("RSA decrypt: %v", err)
	}
	if string(plaintext) != sessionID {
		t.Errorf("roundtrip got %q, want %q", string(plaintext), sessionID)
	}
}

func TestParsePublicKey_Valid(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	key, err := parsePublicKey(pubPEM)
	if err != nil {
		t.Fatalf("parsePublicKey: %v", err)
	}
	if key == nil {
		t.Fatal("parsePublicKey returned nil key")
	}
}

func TestParsePublicKey_InvalidPEM(t *testing.T) {
	_, err := parsePublicKey("not-a-pem-block")
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestParsePublicKey_NotRSA(t *testing.T) {
	_, err := parsePublicKey("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE\n-----END PUBLIC KEY-----\n")
	if err == nil {
		t.Fatal("expected error for non-RSA key")
	}
}

func TestValidateTokenURL(t *testing.T) {
	cases := []struct {
		url   string
		valid bool
	}{
		{"https://cp.example.com/token", true},
		{"http://localhost:8080/token", true},
		{"ftp://example.com/token", false},
		{"://missing-scheme", false},
		{"http://user:pass@example.com/token", false},
		{"", false},
	}
	for _, tc := range cases {
		err := validateTokenURL(tc.url)
		if (err == nil) != tc.valid {
			t.Errorf("validateTokenURL(%q): err=%v, wantValid=%v", tc.url, err, tc.valid)
		}
	}
}

func TestNew_ValidConfig(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	ex, err := New("https://cp.example.com/token", pubPEM, "test-session-12345678")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if ex == nil {
		t.Fatal("New returned nil")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	_, err := New("ftp://bad.example.com", pubPEM, "test-session-12345678")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNew_InvalidPublicKey(t *testing.T) {
	_, err := New("https://cp.example.com/token", "garbage", "test-session-12345678")
	if err == nil {
		t.Fatal("expected error for invalid public key")
	}
}

func TestFetchToken_Success(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	srv := newTokenServer(t, privKey, "fresh-api-token-xyz")
	defer srv.Close()

	ex, err := New(srv.URL+"/token", pubPEM, "session-abcdef12")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	token, err := ex.FetchToken()
	if err != nil {
		t.Fatalf("FetchToken: %v", err)
	}
	if token != "fresh-api-token-xyz" {
		t.Errorf("token = %q, want %q", token, "fresh-api-token-xyz")
	}
	if ex.Token() != "fresh-api-token-xyz" {
		t.Errorf("Token() = %q, want %q", ex.Token(), "fresh-api-token-xyz")
	}
}

func TestFetchToken_ServerError_Retries(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		bearer := r.Header.Get("Authorization")[7:]
		decryptBearer(t, privKey, bearer)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: "recovered-token"})
	}))
	defer srv.Close()

	ex, err := New(srv.URL+"/token", pubPEM, "session-retry-test1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	token, err := ex.FetchToken()
	if err != nil {
		t.Fatalf("FetchToken: %v", err)
	}
	if token != "recovered-token" {
		t.Errorf("token = %q, want %q", token, "recovered-token")
	}
	if callCount.Load() != 3 {
		t.Errorf("server called %d times, want 3", callCount.Load())
	}
}

func TestFetchToken_AllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "permanent failure", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, pubPEM := generateTestKeyPair(t)
	ex, err := New(srv.URL+"/token", pubPEM, "session-fail-test1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = ex.FetchToken()
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}
}

func TestFetchToken_EmptyTokenResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: ""})
	}))
	defer srv.Close()

	_, pubPEM := generateTestKeyPair(t)
	ex, err := New(srv.URL+"/token", pubPEM, "session-empty-test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = ex.FetchToken()
	if err == nil {
		t.Fatal("expected error for empty token response")
	}
}

func TestFetchToken_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not-json")
	}))
	defer srv.Close()

	_, pubPEM := generateTestKeyPair(t)
	ex, err := New(srv.URL+"/token", pubPEM, "session-badjson-x")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = ex.FetchToken()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchToken_OnRefreshCallback(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	srv := newTokenServer(t, privKey, "callback-token")
	defer srv.Close()

	ex, err := New(srv.URL+"/token", pubPEM, "session-callback1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var received string
	ex.OnRefresh(func(token string) {
		received = token
	})

	token, err := ex.FetchToken()
	if err != nil {
		t.Fatalf("FetchToken: %v", err)
	}
	if received != token {
		t.Errorf("OnRefresh received %q, want %q", received, token)
	}
}

func TestFetchToken_SendsCorrectBearer(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	sessionID := "session-bearer-check"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			t.Errorf("missing Bearer prefix in Authorization header")
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		bearer := auth[7:]
		decrypted := decryptBearer(t, privKey, bearer)
		if decrypted != sessionID {
			t.Errorf("decrypted session ID = %q, want %q", decrypted, sessionID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: "verified-token"})
	}))
	defer srv.Close()

	ex, err := New(srv.URL+"/token", pubPEM, sessionID)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = ex.FetchToken()
	if err != nil {
		t.Fatalf("FetchToken: %v", err)
	}
}

func TestToken_ReturnsEmptyBeforeFetch(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	ex, err := New("https://example.com/token", pubPEM, "session-nofetch1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := ex.Token(); got != "" {
		t.Errorf("Token() before fetch = %q, want empty", got)
	}
}

func TestStopBackgroundRefresh(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	srv := newTokenServer(t, privKey, "bg-token")
	defer srv.Close()

	ex, err := New(srv.URL+"/token", pubPEM, "session-stop-test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ex.StartBackgroundRefresh()
	ex.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestFetchToken_WrongKey_ServerRejects(t *testing.T) {
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating wrong key: %v", err)
	}
	wrongPubDER, _ := x509.MarshalPKIXPublicKey(&wrongKey.PublicKey)
	wrongPubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: wrongPubDER}))

	realKey, _ := generateTestKeyPair(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		bearer := auth[7:]
		ciphertext, err := base64.StdEncoding.DecodeString(bearer)
		if err != nil {
			http.Error(w, "bad base64", http.StatusUnauthorized)
			return
		}
		_, err = rsa.DecryptOAEP(sha256.New(), rand.Reader, realKey, ciphertext, nil)
		if err != nil {
			http.Error(w, "decryption failed", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: "should-not-get-this"})
	}))
	defer srv.Close()

	ex, err := New(srv.URL+"/token", wrongPubPEM, "session-wrongkey1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = ex.FetchToken()
	if err == nil {
		t.Fatal("expected error when encrypted with wrong key")
	}
}
