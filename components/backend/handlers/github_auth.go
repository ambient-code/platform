package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for GitHub auth (set from main package)
var (
	K8sClient          kubernetes.Interface
	Namespace          string
	GithubTokenManager GithubTokenManagerInterface

	// GetGitHubTokenRepo is a dependency-injectable function for getting GitHub tokens in repo operations
	// Tests can override this to provide mock implementations
	// Signature: func(context.Context, kubernetes.Interface, dynamic.Interface, string, string) (string, error)
	GetGitHubTokenRepo func(context.Context, kubernetes.Interface, dynamic.Interface, string, string) (string, error)

	// DoGitHubRequest is a dependency-injectable function for making GitHub API requests
	// Tests can override this to provide mock implementations
	// Signature: func(context.Context, string, string, string, string, io.Reader) (*http.Response, error)
	// If nil, falls back to doGitHubRequest
	DoGitHubRequest func(context.Context, string, string, string, string, io.Reader) (*http.Response, error)
)

// WrapGitHubTokenForRepo wraps git.GetGitHubToken to accept kubernetes.Interface instead of *kubernetes.Clientset
// This allows dependency injection while maintaining compatibility with git.GetGitHubToken
func WrapGitHubTokenForRepo(originalFunc func(context.Context, *kubernetes.Clientset, dynamic.Interface, string, string) (string, error)) func(context.Context, kubernetes.Interface, dynamic.Interface, string, string) (string, error) {
	return func(ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface, project, userID string) (string, error) {
		// Type assert to *kubernetes.Clientset for git.GetGitHubToken
		var k8sClient *kubernetes.Clientset
		if k8s != nil {
			if concrete, ok := k8s.(*kubernetes.Clientset); ok {
				k8sClient = concrete
			} else {
				return "", fmt.Errorf("kubernetes client is not a *Clientset (got %T)", k8s)
			}
		}
		return originalFunc(ctx, k8sClient, dyn, project, userID)
	}
}

// GithubTokenManagerInterface defines the interface for GitHub token management
type GithubTokenManagerInterface interface {
	GenerateJWT() (string, error)
}

// GitHubAppInstallation represents a GitHub App installation for a user
type GitHubAppInstallation struct {
	UserID         string    `json:"userId"`
	GitHubUserID   string    `json:"githubUserId"`
	InstallationID int64     `json:"installationId"`
	Host           string    `json:"host"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// GitHubPATCredentials represents a GitHub Personal Access Token for a user
type GitHubPATCredentials struct {
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetToken implements the interface for git package
func (g *GitHubPATCredentials) GetToken() string {
	return g.Token
}

// GetInstallationID implements the interface for git package
func (g *GitHubAppInstallation) GetInstallationID() int64 {
	return g.InstallationID
}

// GetHost implements the interface for git package
func (g *GitHubAppInstallation) GetHost() string {
	return g.Host
}

// helper: resolve GitHub API base URL from host
func githubAPIBaseURL(host string) string {
	if host == "" || host == "github.com" {
		return "https://api.github.com"
	}
	// GitHub Enterprise default
	return fmt.Sprintf("https://%s/api/v3", host)
}

// doGitHubRequest executes an HTTP request to the GitHub API
func doGitHubRequest(ctx context.Context, method string, url string, authHeader string, accept string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "vTeam-Backend")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	// Optional If-None-Match can be set by callers via context
	if v := ctx.Value("ifNoneMatch"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			req.Header.Set("If-None-Match", s)
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// ===== OAuth during installation (user verification) =====

// signState signs a payload with HMAC SHA-256
func signState(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// HandleGitHubUserOAuthCallback handles GET /auth/github/user/callback
func HandleGitHubUserOAuthCallback(c *gin.Context) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	stateSecret := os.Getenv("GITHUB_STATE_SECRET")
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(stateSecret) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth not configured"})
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	// Defaults when no state provided
	var retB64 string
	var instID int64
	// Validate state if present
	if state != "" {
		raw, err := base64.RawURLEncoding.DecodeString(state)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		parts := strings.SplitN(string(raw), ".", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		payload, sig := parts[0], parts[1]
		if signState(stateSecret, payload) != sig {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state signature"})
			return
		}
		fields := strings.Split(payload, ":")
		if len(fields) != 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad state payload"})
			return
		}
		userInState := fields[0]
		ts := fields[1]
		retB64 = fields[3]
		instB64 := fields[4]
		if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
			if time.Since(time.Unix(sec, 0)) > 10*time.Minute {
				c.JSON(http.StatusBadRequest, gin.H{"error": "state expired"})
				return
			}
		}
		// Confirm current session user matches state user
		userID, _ := c.Get("userID")
		if userID == nil || userInState != userID.(string) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user mismatch"})
			return
		}
		// Decode installation id from state
		instBytes, _ := base64.RawURLEncoding.DecodeString(instB64)
		instStr := string(instBytes)
		instID, _ = strconv.ParseInt(instStr, 10, 64)
	} else {
		// No state (install started outside our UI). Require user session and read installation_id from query.
		userID, _ := c.Get("userID")
		if userID == nil || strings.TrimSpace(userID.(string)) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
			return
		}
		instStr := c.Query("installation_id")
		var err error
		instID, err = strconv.ParseInt(instStr, 10, 64)
		if err != nil || instID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid installation id"})
			return
		}
	}
	// Exchange code → user token
	token, err := exchangeOAuthCodeForUserToken(clientID, clientSecret, code)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "oauth exchange failed"})
		return
	}
	// Verify ownership: GET /user/installations includes the installation
	owns, login, err := userOwnsInstallation(token, instID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "verification failed"})
		return
	}
	if !owns {
		c.JSON(http.StatusForbidden, gin.H{"error": "installation not owned by user"})
		return
	}
	// Store mapping
	installation := GitHubAppInstallation{
		UserID:         c.GetString("userID"),
		GitHubUserID:   login,
		InstallationID: instID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	// Redirect back to return_to if present
	retURL := "/integrations"
	if retB64 != "" {
		if b, err := base64.RawURLEncoding.DecodeString(retB64); err == nil {
			retURL = string(b)
		}
	}
	if retURL == "" {
		retURL = "/integrations"
	}
	c.Redirect(http.StatusFound, retURL)
}

func exchangeOAuthCodeForUserToken(clientID, clientSecret, code string) (string, error) {
	reqBody := strings.NewReader(fmt.Sprintf("client_id=%s&client_secret=%s&code=%s", clientID, clientSecret, code))
	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", reqBody)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("empty token")
	}
	return parsed.AccessToken, nil
}

func userOwnsInstallation(userToken string, installationID int64) (bool, string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/installations", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+userToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	var data struct {
		Installations []struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
			} `json:"account"`
		} `json:"installations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, "", err
	}
	for _, inst := range data.Installations {
		if inst.ID == installationID {
			return true, inst.Account.Login, nil
		}
	}
	return false, "", nil
}

// storeGitHubInstallation persists the GitHub App installation mapping
func storeGitHubInstallation(ctx context.Context, projectName string, installation *GitHubAppInstallation) error {
	if installation == nil || installation.UserID == "" {
		return fmt.Errorf("invalid installation payload")
	}
	// Cluster-scoped by server namespace; ignore projectName for storage
	const cmName = "github-app-installations"
	for i := 0; i < 3; i++ { // retry on conflict
		cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// create
				cm = &corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: cmName, Namespace: Namespace}, Data: map[string]string{}}
				if _, cerr := K8sClient.CoreV1().ConfigMaps(Namespace).Create(ctx, cm, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create ConfigMap: %w", cerr)
				}
				// fetch again to get resourceVersion
				cm, err = K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch ConfigMap after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get ConfigMap: %w", err)
			}
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		b, err := json.Marshal(installation)
		if err != nil {
			return fmt.Errorf("failed to marshal installation: %w", err)
		}
		cm.Data[installation.UserID] = string(b)
		if _, uerr := K8sClient.CoreV1().ConfigMaps(Namespace).Update(ctx, cm, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update ConfigMap: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update ConfigMap after retries")
}

// GetGitHubInstallation retrieves GitHub App installation for a user
func GetGitHubInstallation(ctx context.Context, userID string) (*GitHubAppInstallation, error) {
	const cmName = "github-app-installations"
	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("installation not found")
		}
		return nil, fmt.Errorf("failed to read ConfigMap: %w", err)
	}
	if cm.Data == nil {
		return nil, fmt.Errorf("installation not found")
	}
	raw, ok := cm.Data[userID]
	if !ok || raw == "" {
		return nil, fmt.Errorf("installation not found")
	}
	var inst GitHubAppInstallation
	if err := json.Unmarshal([]byte(raw), &inst); err != nil {
		return nil, fmt.Errorf("failed to decode installation: %w", err)
	}
	return &inst, nil
}

// deleteGitHubInstallation removes the user mapping from ConfigMap
func deleteGitHubInstallation(ctx context.Context, userID string) error {
	const cmName = "github-app-installations"
	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(ctx, cmName, v1.GetOptions{})
	if err != nil {
		return err
	}
	if cm.Data == nil {
		return nil
	}
	delete(cm.Data, userID)
	_, uerr := K8sClient.CoreV1().ConfigMaps(Namespace).Update(ctx, cm, v1.UpdateOptions{})
	return uerr
}

// ===== Global, non-project-scoped endpoints =====

// LinkGitHubInstallationGlobal handles POST /auth/github/install
// Links the current SSO user to a GitHub App installation ID.
func LinkGitHubInstallationGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	var req struct {
		InstallationID int64 `json:"installationId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	installation := GitHubAppInstallation{
		UserID:         userID.(string),
		InstallationID: req.InstallationID,
		Host:           "github.com",
		UpdatedAt:      time.Now(),
	}
	// Best-effort: enrich with GitHub account login for the installation
	if GithubTokenManager != nil {
		if jwt, err := GithubTokenManager.GenerateJWT(); err == nil {
			api := githubAPIBaseURL(installation.Host)
			url := fmt.Sprintf("%s/app/installations/%d", api, req.InstallationID)
			resp, err := doGitHubRequest(c.Request.Context(), http.MethodGet, url, "Bearer "+jwt, "", nil)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var instObj map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&instObj); err == nil {
						if acct, ok := instObj["account"].(map[string]interface{}); ok {
							if login, ok := acct["login"].(string); ok {
								installation.GitHubUserID = login
							}
						}
					}
				}
			}
		}
	}
	if err := storeGitHubInstallation(c.Request.Context(), "", &installation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub App installation linked successfully", "installationId": req.InstallationID})
}

// GetGitHubStatusGlobal handles GET /auth/github/status
// Returns both GitHub App and PAT status
func GetGitHubStatusGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}

	userIDStr := userID.(string)
	response := gin.H{
		"installed": false,
		"pat":       gin.H{"configured": false},
	}

	// Check GitHub App installation
	inst, err := GetGitHubInstallation(c.Request.Context(), userIDStr)
	if err == nil && inst != nil {
		response["installed"] = true
		response["installationId"] = inst.InstallationID
		response["host"] = inst.Host
		response["githubUserId"] = inst.GitHubUserID
		response["userId"] = inst.UserID
		response["updatedAt"] = inst.UpdatedAt.Format(time.RFC3339)
	}

	// Check GitHub PAT
	patCreds, err := GetGitHubPATCredentials(c.Request.Context(), userIDStr)
	if err == nil && patCreds != nil {
		response["pat"] = gin.H{
			"configured": true,
			"updatedAt":  patCreds.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Determine active method
	if patCreds != nil {
		response["active"] = "pat"
	} else if inst != nil {
		response["active"] = "app"
	}

	c.JSON(http.StatusOK, response)
}

// DisconnectGitHubGlobal handles POST /auth/github/disconnect
func DisconnectGitHubGlobal(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil || strings.TrimSpace(userID.(string)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user identity"})
		return
	}
	if err := deleteGitHubInstallation(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlink installation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "GitHub account disconnected"})
}

// ============================================================================
// GitHub Personal Access Token (PAT) Management
// ============================================================================

// SaveGitHubPAT handles POST /api/auth/github/pat
// Saves user's GitHub Personal Access Token at cluster level
func SaveGitHubPAT(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate token format (GitHub PATs start with ghp_, gho_, ghu_, ghs_, or github_pat_)
	if !strings.HasPrefix(req.Token, "ghp_") && !strings.HasPrefix(req.Token, "gho_") &&
		!strings.HasPrefix(req.Token, "ghu_") && !strings.HasPrefix(req.Token, "ghs_") &&
		!strings.HasPrefix(req.Token, "github_pat_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GitHub token format"})
		return
	}

	// Store credentials
	creds := &GitHubPATCredentials{
		UserID:    userID,
		Token:     req.Token,
		UpdatedAt: time.Now(),
	}

	if err := storeGitHubPATCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store GitHub PAT for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save GitHub PAT"})
		return
	}

	log.Printf("✓ Stored GitHub PAT for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "GitHub PAT saved successfully"})
}

// GetGitHubPATStatus handles GET /api/auth/github/pat/status
// Returns whether user has a PAT configured
func GetGitHubPATStatus(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	creds, err := GetGitHubPATCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get GitHub PAT for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check GitHub PAT status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"configured": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": true,
		"updatedAt":  creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteGitHubPAT handles DELETE /api/auth/github/pat
// Removes user's GitHub PAT
func DeleteGitHubPAT(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	if err := DeleteGitHubPATCredentials(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to delete GitHub PAT for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove GitHub PAT"})
		return
	}

	log.Printf("✓ Deleted GitHub PAT for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "GitHub PAT removed successfully"})
}

// storeGitHubPATCredentials stores GitHub PAT in cluster-level Secret
func storeGitHubPATCredentials(ctx context.Context, creds *GitHubPATCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "github-pat-credentials"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create Secret
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                      "ambient-code",
							"ambient-code.io/provider": "github",
							"ambient-code.io/type":     "pat",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				// Fetch again to get resourceVersion
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		b, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		secret.Data[creds.UserID] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// GetGitHubPATCredentials retrieves cluster-level GitHub PAT for a user
func GetGitHubPATCredentials(ctx context.Context, userID string) (*GitHubPATCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	const secretName = "github-pat-credentials"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // User hasn't configured PAT
		}
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[userID]) == 0 {
		return nil, nil // User hasn't configured PAT
	}

	var creds GitHubPATCredentials
	if err := json.Unmarshal(secret.Data[userID], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// DeleteGitHubPATCredentials removes GitHub PAT for a user
func DeleteGitHubPATCredentials(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	const secretName = "github-pat-credentials"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Secret doesn't exist, nothing to delete
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[userID]) == 0 {
			return nil // User's credentials don't exist
		}

		delete(secret.Data, userID)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}
