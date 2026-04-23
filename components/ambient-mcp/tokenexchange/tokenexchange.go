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
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	fetchAttempts  = 3
	fetchTimeout   = 10 * time.Second
	refreshPeriod  = 5 * time.Minute
	initialBackoff = 1 * time.Second
)

type Exchanger struct {
	tokenURL     string
	publicKey    *rsa.PublicKey
	sessionID    string
	httpClient   *http.Client
	mu           sync.RWMutex
	currentToken string
	onRefresh    func(string)
	stopCh       chan struct{}
	startOnce    sync.Once
	stopOnce     sync.Once
}

type tokenResponse struct {
	Token string `json:"token"`
}

func New(tokenURL, publicKeyPEM, sessionID string) (*Exchanger, error) {
	if err := validateTokenURL(tokenURL); err != nil {
		return nil, err
	}

	pubKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &Exchanger{
		tokenURL:   tokenURL,
		publicKey:  pubKey,
		sessionID:  sessionID,
		httpClient: &http.Client{Timeout: fetchTimeout},
		stopCh:     make(chan struct{}),
	}, nil
}

func (e *Exchanger) OnRefresh(fn func(string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onRefresh = fn
}

func (e *Exchanger) FetchToken() (string, error) {
	bearer, err := encryptSessionID(e.publicKey, e.sessionID)
	if err != nil {
		return "", fmt.Errorf("encrypt session ID: %w", err)
	}

	var lastErr error
	for attempt := range fetchAttempts {
		if attempt > 0 {
			time.Sleep(initialBackoff * time.Duration(1<<(attempt-1)))
		}

		token, err := e.doFetch(bearer)
		if err != nil {
			lastErr = err
			continue
		}

		e.mu.Lock()
		e.currentToken = token
		callback := e.onRefresh
		e.mu.Unlock()

		if callback != nil {
			callback(token)
		}

		return token, nil
	}

	return "", fmt.Errorf("token endpoint unreachable after %d attempts: %w", fetchAttempts, lastErr)
}

func (e *Exchanger) Token() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentToken
}

func (e *Exchanger) StartBackgroundRefresh() {
	e.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(refreshPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if _, err := e.FetchToken(); err != nil {
						fmt.Fprintf(os.Stderr, "background token refresh failed: %v\n", err)
					}
				case <-e.stopCh:
					return
				}
			}
		}()
	})
}

func (e *Exchanger) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopCh)
	})
}

func (e *Exchanger) doFetch(bearer string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, e.tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if tokenResp.Token == "" {
		return "", fmt.Errorf("token response missing 'token' field")
	}

	return tokenResp.Token, nil
}

func encryptSessionID(pubKey *rsa.PublicKey, sessionID string) (string, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(sessionID), nil)
	if err != nil {
		return "", fmt.Errorf("RSA-OAEP encrypt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA (got %T)", pub)
	}

	return rsaPub, nil
}

func validateTokenURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse token URL: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("invalid token URL scheme %q (must be http or https)", scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("token URL has no host")
	}
	if parsed.User != nil {
		return fmt.Errorf("token URL must not contain credentials")
	}
	return nil
}
