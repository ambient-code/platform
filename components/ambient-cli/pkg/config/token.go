package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func RefreshAccessToken(issuerURL, clientID, refreshToken string) (accessToken, newRefreshToken string, err error) {
	tokenURL := strings.TrimRight(issuerURL, "/") + "/protocol/openid-connect/token"

	params := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.PostForm(tokenURL, params)
	if err != nil {
		return "", "", fmt.Errorf("POST to token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.ErrorDescription != "" {
			return "", "", fmt.Errorf("token refresh: %s", errResp.ErrorDescription)
		}
		return "", "", fmt.Errorf("token refresh: HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", "", fmt.Errorf("no access_token in refresh response")
	}

	return tokenResp.AccessToken, tokenResp.RefreshToken, nil
}
