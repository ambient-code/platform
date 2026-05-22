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
			fmt.Fprintf(os.Stderr, "credential fetch failed for %s: %v\n", provider, err)
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

	baseURL := strings.TrimRight(apiURL, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	credTokenURL := fmt.Sprintf("%s/api/ambient/v1/credentials/%s/token", baseURL, credID)
	tokenData, err := fetchJSON(client, credTokenURL, bearerToken)
	if err != nil {
		return fmt.Errorf("credential token fetch: %w", err)
	}

	metaData, err := fetchJSON(client, fmt.Sprintf("%s/api/ambient/v1/credentials/%s", baseURL, credID), bearerToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "credential metadata fetch failed (non-fatal): %v\n", err)
		metaData = map[string]interface{}{}
	}

	for k, v := range metaData {
		if _, exists := tokenData[k]; !exists {
			tokenData[k] = v
		}
	}

	setCredentialEnv(provider, tokenData)
	return nil
}

func fetchJSON(client *http.Client, fetchURL, bearerToken string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d (response length: %d)", resp.StatusCode, len(body))
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return data, nil
}

func setCredentialEnv(provider string, data map[string]interface{}) {
	switch provider {
	case "github":
		if token, ok := data["token"].(string); ok && token != "" {
			os.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", token)
		}
	case "jira":
		token, _ := data["apiToken"].(string)
		if token == "" {
			token, _ = data["token"].(string)
		}
		if token != "" {
			os.Setenv("JIRA_API_TOKEN", token)
		}
		if jiraURL, ok := data["url"].(string); ok && jiraURL != "" {
			os.Setenv("JIRA_URL", jiraURL)
		}
		if email, ok := data["email"].(string); ok && email != "" {
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
