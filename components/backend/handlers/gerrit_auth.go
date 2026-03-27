package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GerritCredentials represents cluster-level Gerrit credentials for a user instance
type GerritCredentials struct {
	UserID            string    `json:"userId"`
	InstanceName      string    `json:"instanceName"`                // User-assigned name (e.g., "openstack")
	URL               string    `json:"url"`                         // Gerrit instance base URL
	AuthMethod        string    `json:"authMethod"`                  // "http_basic" or "git_cookies"
	Username          string    `json:"username,omitempty"`          // For http_basic
	HTTPToken         string    `json:"httpToken,omitempty"`         // For http_basic
	GitcookiesContent string    `json:"gitcookiesContent,omitempty"` // For git_cookies
	UpdatedAt         time.Time `json:"updatedAt"`
}

const gerritSecretPrefix = "gerrit-credentials-"

var validInstanceNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

// validateGerritTokenFn is the function used to validate Gerrit credentials.
// It is a variable so tests can replace it with a stub.
var validateGerritTokenFn = ValidateGerritToken

// gerritSecretName returns the per-user Secret name for Gerrit credentials.
// Each user gets their own Secret to avoid cross-user contention and
// unbounded Secret size growth.
func gerritSecretName(userID string) string {
	return gerritSecretPrefix + userID
}

// validateGerritURL validates a Gerrit URL for SSRF protection.
// It ensures the scheme is HTTPS and the host does not resolve to
// loopback, private, link-local, or metadata-service addresses.
func validateGerritURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("URL must use HTTPS scheme")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL must include a hostname")
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("URL must not resolve to a private or loopback address")
		}
		// Block cloud metadata endpoints (169.254.169.254)
		if ip.Equal(net.ParseIP("169.254.169.254")) {
			return fmt.Errorf("URL must not resolve to a metadata service address")
		}
	}
	return nil
}

// ConnectGerrit handles POST /api/auth/gerrit/connect
// Validates and stores Gerrit credentials for a user instance
func ConnectGerrit(c *gin.Context) {
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
		InstanceName      string `json:"instanceName" binding:"required"`
		URL               string `json:"url" binding:"required"`
		AuthMethod        string `json:"authMethod" binding:"required"`
		Username          string `json:"username"`
		HTTPToken         string `json:"httpToken"`
		GitcookiesContent string `json:"gitcookiesContent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate instance name format
	if !validInstanceNameRegex.MatchString(req.InstanceName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance name must be lowercase alphanumeric with hyphens (2-63 chars)"})
		return
	}

	// Validate URL (SSRF protection)
	if err := validateGerritURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid Gerrit URL: %s", err.Error())})
		return
	}

	// Validate auth method, required fields, and reject mixed credentials
	switch req.AuthMethod {
	case "http_basic":
		if req.Username == "" || req.HTTPToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username and HTTP token are required for HTTP basic auth"})
			return
		}
		if req.GitcookiesContent != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gitcookiesContent must not be provided with http_basic auth"})
			return
		}
	case "git_cookies":
		if req.GitcookiesContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Gitcookies content is required for git_cookies auth"})
			return
		}
		if req.Username != "" || req.HTTPToken != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and httpToken must not be provided with git_cookies auth"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Auth method must be 'http_basic' or 'git_cookies'"})
		return
	}

	// Validate credentials against the Gerrit instance
	valid, err := validateGerritTokenFn(c.Request.Context(), req.URL, req.AuthMethod, req.Username, req.HTTPToken, req.GitcookiesContent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Validation failed: %s", err.Error())})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Gerrit credentials"})
		return
	}

	creds := &GerritCredentials{
		UserID:            userID,
		InstanceName:      req.InstanceName,
		URL:               req.URL,
		AuthMethod:        req.AuthMethod,
		Username:          req.Username,
		HTTPToken:         req.HTTPToken,
		GitcookiesContent: req.GitcookiesContent,
		UpdatedAt:         time.Now(),
	}

	if err := storeGerritCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store Gerrit credentials for user %s instance %s: %v", userID, req.InstanceName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gerrit credentials"})
		return
	}

	log.Printf("✓ Stored Gerrit credentials for user %s instance %s", userID, req.InstanceName)
	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("Gerrit instance '%s' connected successfully", req.InstanceName),
		"instanceName": req.InstanceName,
		"url":          req.URL,
		"authMethod":   req.AuthMethod,
	})
}

// GetGerritStatus handles GET /api/auth/gerrit/:instanceName/status
func GetGerritStatus(c *gin.Context) {
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

	instanceName := c.Param("instanceName")
	if instanceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance name is required"})
		return
	}

	creds, err := getGerritCredentials(c.Request.Context(), instanceName, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false})
			return
		}
		log.Printf("Failed to get Gerrit credentials for user %s instance %s: %v", userID, instanceName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Gerrit status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected":    true,
		"instanceName": creds.InstanceName,
		"url":          creds.URL,
		"authMethod":   creds.AuthMethod,
		"updatedAt":    creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectGerrit handles DELETE /api/auth/gerrit/:instanceName/disconnect
func DisconnectGerrit(c *gin.Context) {
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

	instanceName := c.Param("instanceName")
	if instanceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance name is required"})
		return
	}

	if err := deleteGerritCredentials(c.Request.Context(), instanceName, userID); err != nil {
		log.Printf("Failed to delete Gerrit credentials for user %s instance %s: %v", userID, instanceName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Gerrit"})
		return
	}

	log.Printf("✓ Deleted Gerrit credentials for user %s instance %s", userID, instanceName)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Gerrit instance '%s' disconnected successfully", instanceName)})
}

// ListGerritInstances handles GET /api/auth/gerrit/instances
// Returns all connected Gerrit instances for the authenticated user
func ListGerritInstances(c *gin.Context) {
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

	instances, err := listGerritCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to list Gerrit instances for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Gerrit instances"})
		return
	}

	result := make([]gin.H, 0, len(instances))
	for _, creds := range instances {
		result = append(result, gin.H{
			"connected":    true,
			"instanceName": creds.InstanceName,
			"url":          creds.URL,
			"authMethod":   creds.AuthMethod,
			"updatedAt":    creds.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{"instances": result})
}

// storeGerritCredentials stores Gerrit credentials in a per-user Secret
func storeGerritCredentials(ctx context.Context, creds *GerritCredentials) error {
	if creds == nil || creds.UserID == "" || creds.InstanceName == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	secretName := gerritSecretName(creds.UserID)

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                      "ambient-code",
							"ambient-code.io/provider": "gerrit",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
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
		secret.Data[creds.InstanceName] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// getGerritCredentials retrieves Gerrit credentials for a specific instance and user
func getGerritCredentials(ctx context.Context, instanceName, userID string) (*GerritCredentials, error) {
	if userID == "" || instanceName == "" {
		return nil, fmt.Errorf("userID and instanceName are required")
	}

	secretName := gerritSecretName(userID)
	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[instanceName]) == 0 {
		return nil, nil
	}

	var creds GerritCredentials
	if err := json.Unmarshal(secret.Data[instanceName], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// listGerritCredentials returns all Gerrit instances for a user.
// Each user has their own Secret, so all entries belong to the user.
// Results are sorted by instance name for deterministic ordering.
func listGerritCredentials(ctx context.Context, userID string) ([]*GerritCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	secretName := gerritSecretName(userID)
	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []*GerritCredentials
	for key, val := range secret.Data {
		var creds GerritCredentials
		if err := json.Unmarshal(val, &creds); err != nil {
			log.Printf("Failed to parse Gerrit credentials for key %s: %v", key, err)
			continue
		}
		result = append(result, &creds)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].InstanceName < result[j].InstanceName
	})

	return result, nil
}

// deleteGerritCredentials removes Gerrit credentials for a specific instance and user
func deleteGerritCredentials(ctx context.Context, instanceName, userID string) error {
	if userID == "" || instanceName == "" {
		return fmt.Errorf("userID and instanceName are required")
	}

	secretName := gerritSecretName(userID)

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[instanceName]) == 0 {
			return nil
		}

		delete(secret.Data, instanceName)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}
