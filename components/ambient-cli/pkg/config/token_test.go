package config

import (
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
