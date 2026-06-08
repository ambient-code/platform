// Package jwtauth provides JWT validation against OIDC providers using JWKS.
package jwtauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Claims struct {
	Sub               string
	Email             string
	PreferredUsername string
	Groups            []string
	Issuer            string
	Audience          []string
	ExpiresAt         time.Time
}

type Validator struct {
	jwksCache  *jwk.Cache
	jwksURL    string
	issuer     string
	altIssuers []string
	audience   string
}

func NewValidator(issuerURL, audience string) (*Validator, error) {
	if issuerURL == "" {
		return nil, fmt.Errorf("issuer URL is required")
	}

	discoveredIssuer, jwksURL, err := discoverOIDCConfig(issuerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	ctx := context.Background()
	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(5*time.Minute)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	return &Validator{
		jwksCache: cache,
		jwksURL:   jwksURL,
		issuer:    discoveredIssuer,
		audience:  audience,
	}, nil
}

func NewValidatorWithJWKSURL(jwksURL, issuer, audience string) (*Validator, error) {
	if jwksURL == "" {
		return nil, fmt.Errorf("JWKS URL is required")
	}

	ctx := context.Background()
	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(5*time.Minute)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	return &Validator{
		jwksCache: cache,
		jwksURL:   jwksURL,
		issuer:    issuer,
		audience:  audience,
	}, nil
}

// AddAltIssuer adds an alternative accepted issuer URL. Tokens signed by the
// same JWKS keys but with a different iss claim (e.g., the public URL of a
// Keycloak behind a port-forward) will be accepted.
func (v *Validator) AddAltIssuer(issuer string) {
	if issuer != "" && issuer != v.issuer {
		v.altIssuers = append(v.altIssuers, issuer)
	}
}

func (v *Validator) Validate(tokenString string) (*Claims, error) {
	keySet, err := v.jwksCache.Get(context.Background(), v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	// Verify signature and expiration, but validate issuer manually to support
	// multiple accepted issuers (internal + public URL in dev environments).
	opts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}
	if v.audience != "" {
		opts = append(opts, jwt.WithAudience(v.audience))
	}

	token, err := jwt.Parse([]byte(tokenString), opts...)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !v.isAcceptedIssuer(token.Issuer()) {
		return nil, fmt.Errorf("token validation failed: issuer %q not accepted", token.Issuer())
	}

	claims := &Claims{
		Sub:       token.Subject(),
		Issuer:    token.Issuer(),
		ExpiresAt: token.Expiration(),
	}

	if aud := token.Audience(); len(aud) > 0 {
		claims.Audience = aud
	}

	privateClaims := token.PrivateClaims()

	if email, ok := privateClaims["email"].(string); ok {
		claims.Email = email
	}

	if username, ok := privateClaims["preferred_username"].(string); ok {
		claims.PreferredUsername = username
	}

	if groups, ok := privateClaims["groups"]; ok {
		switch g := groups.(type) {
		case []interface{}:
			for _, item := range g {
				if s, ok := item.(string); ok {
					claims.Groups = append(claims.Groups, s)
				}
			}
		case []string:
			claims.Groups = g
		}
	}

	return claims, nil
}

func (v *Validator) isAcceptedIssuer(iss string) bool {
	if iss == v.issuer {
		return true
	}
	for _, alt := range v.altIssuers {
		if iss == alt {
			return true
		}
	}
	return false
}

func discoverOIDCConfig(issuerURL string) (discoveredIssuer string, jwksURL string, err error) {
	wellKnownURL := issuerURL + "/.well-known/openid-configuration"

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(wellKnownURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch OIDC configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("OIDC configuration returned status %d", resp.StatusCode)
	}

	var config struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return "", "", fmt.Errorf("failed to decode OIDC configuration: %w", err)
	}

	if config.JWKSURI == "" {
		return "", "", fmt.Errorf("OIDC configuration missing jwks_uri")
	}

	issuer := config.Issuer
	if issuer == "" {
		issuer = issuerURL
	}

	return issuer, config.JWKSURI, nil
}
