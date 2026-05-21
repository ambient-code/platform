package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/ambient-code/platform/components/ambient-mcp/tokenexchange"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: credential-entrypoint <command> [args...]\n")
		os.Exit(1)
	}

	tokenURL := os.Getenv("AMBIENT_CP_TOKEN_URL")
	publicKey := os.Getenv("AMBIENT_CP_TOKEN_PUBLIC_KEY")
	sessionID := os.Getenv("SESSION_ID")
	apiURL := os.Getenv("AMBIENT_API_URL")
	provider := os.Getenv("CREDENTIAL_PROVIDER")

	if tokenURL == "" || publicKey == "" || sessionID == "" {
		fmt.Fprintf(os.Stderr, "AMBIENT_CP_TOKEN_URL, AMBIENT_CP_TOKEN_PUBLIC_KEY, SESSION_ID required\n")
		os.Exit(1)
	}

	exchanger, err := tokenexchange.New(tokenURL, publicKey, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "token exchange init failed: %v\n", err)
		os.Exit(1)
	}

	bearerToken, err := exchanger.FetchToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "token fetch failed: %v\n", err)
		os.Exit(1)
	}

	if apiURL != "" && provider != "" {
		if err := fetchAndSetCredential(bearerToken, apiURL, provider); err != nil {
			fmt.Fprintf(os.Stderr, "credential fetch failed: %v\n", err)
		}
	}

	exchanger.OnRefresh(func(newToken string) {
		if apiURL != "" && provider != "" {
			if err := fetchAndSetCredential(newToken, apiURL, provider); err != nil {
				fmt.Fprintf(os.Stderr, "credential refresh failed: %v\n", err)
			}
		}
	})
	exchanger.StartBackgroundRefresh()
	defer exchanger.Stop()

	execCommand(os.Args[1:])
}

func fetchAndSetCredential(bearerToken, apiURL, provider string) error {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("parse API URL: %w", err)
	}
	hostname := parsed.Hostname()
	if !strings.HasSuffix(hostname, ".svc.cluster.local") &&
		!strings.HasSuffix(hostname, ".svc") &&
		hostname != "localhost" &&
		hostname != "127.0.0.1" {
		return fmt.Errorf("refusing to send credentials to external host: %s", hostname)
	}

	credentialIDs := map[string]string{}
	if raw := os.Getenv("CREDENTIAL_IDS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &credentialIDs); err != nil {
			return fmt.Errorf("parse CREDENTIAL_IDS: %w", err)
		}
	}

	credID := credentialIDs[provider]
	if credID == "" {
		return fmt.Errorf("no credential ID for provider %s in CREDENTIAL_IDS", provider)
	}

	credURL := fmt.Sprintf("%s/api/ambient/v1/credentials/%s/token",
		strings.TrimRight(apiURL, "/"), credID)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, credURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("credential request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read credential response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("credential fetch HTTP %d (response length: %d)", resp.StatusCode, len(body))
	}

	var credData map[string]interface{}
	if err := json.Unmarshal(body, &credData); err != nil {
		return fmt.Errorf("parse credential response: %w", err)
	}

	setCredentialEnv(provider, credData)
	return nil
}

func setCredentialEnv(provider string, data map[string]interface{}) {
	switch provider {
	case "github":
		if token, ok := data["token"].(string); ok && token != "" {
			os.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", token)
		}
	case "jira":
		if token, ok := data["apiToken"].(string); ok {
			os.Setenv("JIRA_API_TOKEN", token)
		}
		if url, ok := data["url"].(string); ok {
			os.Setenv("JIRA_URL", url)
		}
		if email, ok := data["email"].(string); ok {
			os.Setenv("JIRA_USERNAME", email)
		}
	case "kubeconfig":
		if token, ok := data["token"].(string); ok && token != "" {
			path := "/tmp/.ambient_kubeconfig"
			if err := os.WriteFile(path, []byte(token), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "write kubeconfig failed: %v\n", err)
			}
			os.Setenv("KUBECONFIG", path)
		}
	case "google":
		if token, ok := data["accessToken"].(string); ok && token != "" {
			os.Setenv("GOOGLE_ACCESS_TOKEN", token)
		}
	}
}

func execCommand(args []string) {
	binary, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", args[0])
		os.Exit(1)
	}
	if err := syscall.Exec(binary, args, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec failed: %v\n", err)
		os.Exit(1)
	}
}
