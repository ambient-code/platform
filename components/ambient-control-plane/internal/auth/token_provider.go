package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type StaticTokenProvider struct {
	token string
}

func NewStaticTokenProvider(token string) *StaticTokenProvider {
	return &StaticTokenProvider{token: token}
}

func (p *StaticTokenProvider) Token(_ context.Context) (string, error) {
	if p.token == "" {
		return "", fmt.Errorf("static token is empty")
	}
	return p.token, nil
}

type OIDCTokenProvider struct {
	cfg    clientcredentials.Config
	mu     sync.Mutex
	cached *oauth2.Token
	logger zerolog.Logger
}

func NewOIDCTokenProvider(tokenURL, clientID, clientSecret string, logger zerolog.Logger) *OIDCTokenProvider {
	return &OIDCTokenProvider{
		cfg: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
		},
		logger: logger.With().Str("component", "oidc-token-provider").Logger(),
	}
}

func (p *OIDCTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cached != nil && p.cached.Valid() && time.Until(p.cached.Expiry) > 30*time.Second {
		return p.cached.AccessToken, nil
	}

	p.logger.Info().Msg("fetching new OIDC token via client credentials")

	tok, err := p.cfg.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching OIDC token: %w", err)
	}

	p.cached = tok
	p.logger.Info().Time("expiry", tok.Expiry).Msg("OIDC token acquired")
	return tok.AccessToken, nil
}
