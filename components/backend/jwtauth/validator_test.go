package jwtauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func setupTestServer(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	key, err := jwk.FromRaw(privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := key.Set(jwk.KeyIDKey, "test-key-1"); err != nil {
		t.Fatal(err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatal(err)
	}
	if err := key.Set(jwk.KeyUsageKey, "sig"); err != nil {
		t.Fatal(err)
	}

	keySet := jwk.NewSet()
	if err := keySet.AddKey(key); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	var serverURL string

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := map[string]string{
			"issuer":   serverURL,
			"jwks_uri": serverURL + "/jwks",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL

	return privateKey, server
}

func signToken(t *testing.T, privateKey *rsa.PrivateKey, token jwt.Token) string {
	t.Helper()

	signingKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := signingKey.Set(jwk.KeyIDKey, "test-key-1"); err != nil {
		t.Fatal(err)
	}
	if err := signingKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatal(err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signingKey))
	if err != nil {
		t.Fatal(err)
	}
	return string(signed)
}

func TestValidate(t *testing.T) {
	privateKey, server := setupTestServer(t)
	defer server.Close()

	validator, err := NewValidator(server.URL, "ambient-frontend")
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	tests := []struct {
		name        string
		buildToken  func() string
		wantErr     bool
		checkClaims func(*testing.T, *Claims)
	}{
		{
			name: "valid token with all claims",
			buildToken: func() string {
				tok, _ := jwt.NewBuilder().
					Subject("f:abc:jsell").
					Issuer(server.URL).
					Audience([]string{"ambient-frontend"}).
					Expiration(time.Now().Add(5*time.Minute)).
					IssuedAt(time.Now()).
					Claim("email", "jsell@redhat.com").
					Claim("preferred_username", "jsell").
					Claim("groups", []string{"ambient-users", "team-ambient"}).
					Build()
				return signToken(t, privateKey, tok)
			},
			wantErr: false,
			checkClaims: func(t *testing.T, c *Claims) {
				if c.Sub != "f:abc:jsell" {
					t.Errorf("Sub = %q, want %q", c.Sub, "f:abc:jsell")
				}
				if c.Email != "jsell@redhat.com" {
					t.Errorf("Email = %q, want %q", c.Email, "jsell@redhat.com")
				}
				if c.PreferredUsername != "jsell" {
					t.Errorf("PreferredUsername = %q, want %q", c.PreferredUsername, "jsell")
				}
				if len(c.Groups) != 2 || c.Groups[0] != "ambient-users" {
					t.Errorf("Groups = %v, want [ambient-users, team-ambient]", c.Groups)
				}
				if c.Issuer != server.URL {
					t.Errorf("Issuer = %q, want %q", c.Issuer, server.URL)
				}
			},
		},
		{
			name: "expired token",
			buildToken: func() string {
				tok, _ := jwt.NewBuilder().
					Subject("expired-user").
					Issuer(server.URL).
					Audience([]string{"ambient-frontend"}).
					Expiration(time.Now().Add(-1 * time.Hour)).
					IssuedAt(time.Now().Add(-2 * time.Hour)).
					Build()
				return signToken(t, privateKey, tok)
			},
			wantErr: true,
		},
		{
			name: "wrong issuer",
			buildToken: func() string {
				tok, _ := jwt.NewBuilder().
					Subject("wrong-issuer-user").
					Issuer("https://evil.example.com").
					Audience([]string{"ambient-frontend"}).
					Expiration(time.Now().Add(5 * time.Minute)).
					IssuedAt(time.Now()).
					Build()
				return signToken(t, privateKey, tok)
			},
			wantErr: true,
		},
		{
			name: "wrong audience",
			buildToken: func() string {
				tok, _ := jwt.NewBuilder().
					Subject("wrong-audience-user").
					Issuer(server.URL).
					Audience([]string{"wrong-audience"}).
					Expiration(time.Now().Add(5 * time.Minute)).
					IssuedAt(time.Now()).
					Build()
				return signToken(t, privateKey, tok)
			},
			wantErr: true,
		},
		{
			name: "tampered signature",
			buildToken: func() string {
				otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				tok, _ := jwt.NewBuilder().
					Subject("tampered-user").
					Issuer(server.URL).
					Audience([]string{"ambient-frontend"}).
					Expiration(time.Now().Add(5 * time.Minute)).
					IssuedAt(time.Now()).
					Build()
				return signToken(t, otherKey, tok)
			},
			wantErr: true,
		},
		{
			name: "malformed token",
			buildToken: func() string {
				return "not.a.jwt"
			},
			wantErr: true,
		},
		{
			name: "empty token",
			buildToken: func() string {
				return ""
			},
			wantErr: true,
		},
		{
			name: "token with minimal claims",
			buildToken: func() string {
				tok, _ := jwt.NewBuilder().
					Subject("minimal-user").
					Issuer(server.URL).
					Audience([]string{"ambient-frontend"}).
					Expiration(time.Now().Add(5 * time.Minute)).
					IssuedAt(time.Now()).
					Build()
				return signToken(t, privateKey, tok)
			},
			wantErr: false,
			checkClaims: func(t *testing.T, c *Claims) {
				if c.Sub != "minimal-user" {
					t.Errorf("Sub = %q, want %q", c.Sub, "minimal-user")
				}
				if c.Email != "" {
					t.Errorf("Email = %q, want empty", c.Email)
				}
				if c.PreferredUsername != "" {
					t.Errorf("PreferredUsername = %q, want empty", c.PreferredUsername)
				}
				if len(c.Groups) != 0 {
					t.Errorf("Groups = %v, want empty", c.Groups)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenStr := tt.buildToken()
			claims, err := validator.Validate(tokenStr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkClaims != nil {
				tt.checkClaims(t, claims)
			}
		})
	}
}

func TestNewValidator_MissingIssuer(t *testing.T) {
	_, err := NewValidator("", "audience")
	if err == nil {
		t.Fatal("expected error for empty issuer URL")
	}
}

func TestNewValidator_BadIssuer(t *testing.T) {
	_, err := NewValidator("http://localhost:1/nonexistent", "audience")
	if err == nil {
		t.Fatal("expected error for unreachable issuer")
	}
}

func TestNewValidatorWithJWKSURL(t *testing.T) {
	privateKey, server := setupTestServer(t)
	defer server.Close()

	validator, err := NewValidatorWithJWKSURL(server.URL+"/jwks", server.URL, "ambient-frontend")
	if err != nil {
		t.Fatalf("NewValidatorWithJWKSURL: %v", err)
	}

	tok, _ := jwt.NewBuilder().
		Subject("direct-jwks-user").
		Issuer(server.URL).
		Audience([]string{"ambient-frontend"}).
		Expiration(time.Now().Add(5 * time.Minute)).
		IssuedAt(time.Now()).
		Build()

	tokenStr := signToken(t, privateKey, tok)
	claims, err := validator.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.Sub != "direct-jwks-user" {
		t.Errorf("Sub = %q, want %q", claims.Sub, "direct-jwks-user")
	}
}

func TestDiscoverJWKSURL(t *testing.T) {
	mux := http.NewServeMux()
	var serverURL string

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jwks_uri": "%s/jwks", "issuer": "%s"}`, serverURL, serverURL)
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	serverURL = server.URL

	issuer, jwksURL, err := discoverOIDCConfig(server.URL)
	if err != nil {
		t.Fatalf("discoverOIDCConfig: %v", err)
	}
	if issuer != server.URL {
		t.Errorf("issuer = %q, want %q", issuer, server.URL)
	}
	expected := server.URL + "/jwks"
	if jwksURL != expected {
		t.Errorf("jwksURL = %q, want %q", jwksURL, expected)
	}
}
