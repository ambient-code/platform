package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TokenExpiry(tokenStr string) (time.Time, error) {
	if strings.HasPrefix(tokenStr, "sha256~") {
		return time.Time{}, nil
	}

	// ParseUnverified is intentional: the CLI only reads claims (e.g. exp) for
	// local display and cannot verify the server's signing key.
	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse token: %w", err)
	}

	exp, ok := claims["exp"]
	if !ok {
		return time.Time{}, nil
	}

	expFloat, ok := exp.(float64)
	if !ok {
		return time.Time{}, fmt.Errorf("token 'exp' claim is not a number")
	}

	return time.Unix(int64(expFloat), 0), nil
}

func IsTokenExpired(tokenStr string) (bool, error) {
	expiry, err := TokenExpiry(tokenStr)
	if err != nil {
		return false, err
	}

	if expiry.IsZero() {
		return false, nil
	}

	return time.Now().After(expiry), nil
}
