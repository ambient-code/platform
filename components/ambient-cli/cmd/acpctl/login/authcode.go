package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	defaultIssuerURL = "https://sso.redhat.com/auth/realms/redhat-external"
	// TODO(RHOAIENG-56155): replace with a dedicated acpctl public client ID once registered in the redhat-external realm.
	defaultClientID = "ocm-cli"
	callbackTimeout = 5 * time.Minute
	callbackHTML    = `<!DOCTYPE html><html><body>
<h2>Login successful</h2>
<p>You may close this tab and return to the terminal.</p>
</body></html>`
	callbackHTMLError = `<!DOCTYPE html><html><body>
<h2>Login failed</h2>
<p>%s</p>
<p>Please close this tab and check the terminal for details.</p>
</body></html>`
)

type authCodeResult struct {
	code  string
	state string
	err   error
}

type tokenResult struct {
	AccessToken  string
	RefreshToken string
}

func runAuthCodeFlow(issuerURL, clientID, clientSecret string) (*tokenResult, error) {
	issuerURL = strings.TrimRight(issuerURL, "/")
	authorizeURL := issuerURL + "/protocol/openid-connect/auth"
	tokenURL := issuerURL + "/protocol/openid-connect/token"

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start local callback listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	codeVerifier, codeChallenge, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate PKCE: %w", err)
	}

	authURL := buildAuthURL(authorizeURL, clientID, redirectURI, state, codeChallenge)

	resultCh := make(chan authCodeResult, 1)
	srv := &http.Server{
		Handler: callbackHandler(state, resultCh),
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Opening browser for authentication...\nIf the browser does not open, visit:\n\n  %s\n\n", authURL)
	_ = openBrowser(authURL)

	ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
	defer cancel()

	var result authCodeResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for authorization callback (%.0fs)", callbackTimeout.Seconds())
	}

	if result.err != nil {
		return nil, fmt.Errorf("authorization failed: %w", result.err)
	}

	tokens, err := exchangeCodeForTokens(tokenURL, clientID, clientSecret, result.code, redirectURI, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}

	return tokens, nil
}

func generateRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generatePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

func buildAuthURL(authorizeURL, clientID, redirectURI, state, codeChallenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return authorizeURL + "?" + params.Encode()
}

func callbackHandler(expectedState string, resultCh chan<- authCodeResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = errParam
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, callbackHTMLError, desc)
			resultCh <- authCodeResult{err: errors.New(desc)}
			return
		}

		gotState := q.Get("state")
		if gotState != expectedState {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, callbackHTMLError, "invalid state parameter")
			resultCh <- authCodeResult{err: errors.New("state mismatch: possible CSRF")}
			return
		}

		code := q.Get("code")
		if code == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, callbackHTMLError, "missing authorization code")
			resultCh <- authCodeResult{err: errors.New("missing authorization code in callback")}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, callbackHTML)
		resultCh <- authCodeResult{code: code, state: gotState}
	})
}

func exchangeCodeForToken(tokenURL, clientID, clientSecret, code, redirectURI, codeVerifier string) (string, error) {
	tokens, err := exchangeCodeForTokens(tokenURL, clientID, clientSecret, code, redirectURI, codeVerifier)
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

func exchangeCodeForTokens(tokenURL, clientID, clientSecret, code, redirectURI, codeVerifier string) (*tokenResult, error) {
	params := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {codeVerifier},
	}
	if clientSecret != "" {
		params.Set("client_secret", clientSecret)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.PostForm(tokenURL, params)
	if err != nil {
		return nil, fmt.Errorf("POST to token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, tokenEndpointError(resp.StatusCode, body)
	}

	return parseTokensResponse(body)
}

func tokenEndpointError(statusCode int, body []byte) error {
	var errResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil {
		if errResp.ErrorDescription != "" {
			return fmt.Errorf("token endpoint: %s", errResp.ErrorDescription)
		}
		if errResp.Error != "" {
			return fmt.Errorf("token endpoint: %s", errResp.Error)
		}
	}
	return fmt.Errorf("token endpoint returned HTTP %d", statusCode)
}

func parseTokenResponse(body []byte) (string, error) {
	tokens, err := parseTokensResponse(body)
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

func parseTokensResponse(body []byte) (*tokenResult, error) {
	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if resp.AccessToken == "" {
		return nil, errors.New("no access_token in token response")
	}
	return &tokenResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func openBrowser(target string) error {
	var cmd string
	var cmdArgs []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		cmdArgs = []string{target}
	case "windows":
		cmd = "rundll32"
		cmdArgs = []string{"url.dll,FileProtocolHandler", target}
	default:
		cmd = "xdg-open"
		cmdArgs = []string{target}
	}

	return exec.Command(cmd, cmdArgs...).Start()
}
