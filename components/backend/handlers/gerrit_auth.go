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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GerritCredentials represents cluster-level Gerrit credentials for a single instance
type GerritCredentials struct {
	UserID            string    `json:"userId"`
	InstanceName      string    `json:"instanceName"`
	URL               string    `json:"url"`
	AuthMethod        string    `json:"authMethod"`
	Username          string    `json:"username,omitempty"`
	HTTPToken         string    `json:"httpToken,omitempty"`
	GitcookiesContent string    `json:"gitcookiesContent,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// instanceNameRegex validates Gerrit instance names: 2-63 chars, lowercase alphanumeric + hyphens
var instanceNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

const (
	gerritAuthHTTPBasic  = "http_basic"
	gerritAuthGitCookies = "git_cookies"
)

// blockedCIDRs are IPv4 ranges blocked for SSRF protection beyond what
// net.IP.IsPrivate/IsLoopback/etc. already cover.
var blockedCIDRs = func() []net.IPNet {
	cidrs := []string{
		"100.64.0.0/10",   // CGNAT (RFC 6598)
		"192.0.2.0/24",    // TEST-NET-1 (RFC 5737)
		"198.51.100.0/24", // TEST-NET-2 (RFC 5737)
		"203.0.113.0/24",  // TEST-NET-3 (RFC 5737)
		"198.18.0.0/15",   // Benchmarking (RFC 2544)
		"240.0.0.0/4",     // Reserved
	}
	nets := make([]net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, n, _ := net.ParseCIDR(cidr)
		nets[i] = *n
	}
	return nets
}()

// validateGerritURL validates a Gerrit URL, enforcing HTTPS and a resolvable public hostname.
// The actual IP-level SSRF check happens in ssrfSafeTransport at dial time.
func validateGerritURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("URL must use HTTPS scheme")
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must include a hostname")
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname")
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isPrivateOrBlocked(ip) {
			return fmt.Errorf("URL resolves to a blocked IP address")
		}
	}

	return nil
}

func isPrivateOrBlocked(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	ip4 := ip.To4()
	if ip4 != nil {
		for _, cidr := range blockedCIDRs {
			if cidr.Contains(ip4) {
				return true
			}
		}
		// Cloud metadata endpoint
		if ip4.Equal(net.IP{169, 254, 169, 254}) {
			return true
		}
	}

	return false
}

// ssrfSafeTransport returns an http.Transport with a custom DialContext that
// resolves the hostname, checks each resolved IP against isPrivateOrBlocked,
// and dials the resolved IP directly (not the original hostname) to prevent
// DNS rebinding attacks (TOCTOU between validation and dial).
func ssrfSafeTransport() *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}

			ips, err := net.DefaultResolver.LookupHost(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve host: %w", err)
			}

			var allowedIP string
			for _, ipStr := range ips {
				ip := net.ParseIP(ipStr)
				if ip == nil {
					continue
				}
				if isPrivateOrBlocked(ip) {
					return nil, fmt.Errorf("connection to blocked IP address denied")
				}
				if allowedIP == "" {
					allowedIP = ipStr
				}
			}

			if allowedIP == "" {
				return nil, fmt.Errorf("no valid IP address resolved")
			}

			// Dial the resolved IP directly to eliminate DNS rebinding window
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(allowedIP, port))
		},
	}
}

func ConnectGerrit(c *gin.Context) {
	if !FeatureEnabledForRequest(c, "gerrit.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

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

	// Validate instance name
	if !instanceNameRegex.MatchString(req.InstanceName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance name: must be 2-63 lowercase alphanumeric characters or hyphens, starting and ending with alphanumeric"})
		return
	}

	// Validate URL (SSRF protection)
	if err := validateGerritURL(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid Gerrit URL: %s", err.Error())})
		return
	}

	// Validate auth method
	if req.AuthMethod != gerritAuthHTTPBasic && req.AuthMethod != gerritAuthGitCookies {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authMethod must be 'http_basic' or 'git_cookies'"})
		return
	}

	if req.HTTPToken != "" && req.GitcookiesContent != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot provide both httpToken and gitcookiesContent"})
		return
	}

	switch req.AuthMethod {
	case gerritAuthHTTPBasic:
		if req.Username == "" || req.HTTPToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and httpToken are required for http_basic auth"})
			return
		}
	case gerritAuthGitCookies:
		if req.GitcookiesContent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gitcookiesContent is required for git_cookies auth"})
			return
		}
	}

	// Validate credentials against Gerrit
	valid, err := validateGerritTokenFn(c.Request.Context(), req.URL, req.AuthMethod, req.Username, req.HTTPToken, req.GitcookiesContent)
	if err != nil {
		log.Printf("Gerrit credential validation failed for user %s instance %s: %v", userID, req.InstanceName, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to validate Gerrit credentials: %s", err.Error())})
		return
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Gerrit credentials"})
		return
	}

	// Store credentials
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

	log.Printf("Stored Gerrit credentials for user %s instance %s (authMethod=%s, hasToken=%t)",
		userID, req.InstanceName, req.AuthMethod, len(req.HTTPToken) > 0 || len(req.GitcookiesContent) > 0)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Gerrit instance connected successfully",
		"instanceName": req.InstanceName,
		"url":          req.URL,
		"authMethod":   req.AuthMethod,
	})
}

func GetGerritStatus(c *gin.Context) {
	if !FeatureEnabledForRequest(c, "gerrit.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

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
	if !instanceNameRegex.MatchString(instanceName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance name"})
		return
	}

	creds, err := getGerritCredentials(c.Request.Context(), userID, instanceName)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false, "instanceName": instanceName})
			return
		}
		log.Printf("Failed to get Gerrit credentials for user %s instance %s: %v", userID, instanceName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Gerrit status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false, "instanceName": instanceName})
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

func DisconnectGerrit(c *gin.Context) {
	if !FeatureEnabledForRequest(c, "gerrit.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

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
	if !instanceNameRegex.MatchString(instanceName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance name"})
		return
	}

	if err := deleteGerritCredentials(c.Request.Context(), userID, instanceName); err != nil {
		log.Printf("Failed to delete Gerrit credentials for user %s instance %s: %v", userID, instanceName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Gerrit instance"})
		return
	}

	log.Printf("Deleted Gerrit credentials for user %s instance %s", userID, instanceName)
	c.JSON(http.StatusOK, gin.H{"message": "Gerrit instance disconnected successfully"})
}

func ListGerritInstances(c *gin.Context) {
	if !FeatureEnabledForRequest(c, "gerrit.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

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

	// Sort by instance name for consistent ordering
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].InstanceName < instances[j].InstanceName
	})

	result := make([]gin.H, 0, len(instances))
	for _, inst := range instances {
		entry := gin.H{
			"instanceName": inst.InstanceName,
			"url":          inst.URL,
			"authMethod":   inst.AuthMethod,
			"connected":    true,
			"updatedAt":    inst.UpdatedAt.Format(time.RFC3339),
		}
		result = append(result, entry)
	}

	c.JSON(http.StatusOK, gin.H{"instances": result})
}

// gerritSecretName returns the K8s Secret name for a user's Gerrit credentials
func gerritSecretName(userID string) string {
	return fmt.Sprintf("gerrit-credentials-%s", userID)
}

// storeGerritCredentials stores Gerrit credentials for a single instance in the user's Secret
func storeGerritCredentials(ctx context.Context, creds *GerritCredentials) error {
	if creds == nil || creds.UserID == "" || creds.InstanceName == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	secretName := gerritSecretName(creds.UserID)

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
							"ambient-code.io/provider": "gerrit",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				created, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{})
				if cerr != nil {
					if errors.IsAlreadyExists(cerr) {
						continue // retry — concurrent create
					}
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				secret = created
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
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// getGerritCredentials retrieves credentials for a single Gerrit instance
func getGerritCredentials(ctx context.Context, userID, instanceName string) (*GerritCredentials, error) {
	if userID == "" || instanceName == "" {
		return nil, fmt.Errorf("userID and instanceName are required")
	}

	secretName := gerritSecretName(userID)

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[instanceName]) == 0 {
		return nil, nil // Instance not configured
	}

	var creds GerritCredentials
	if err := json.Unmarshal(secret.Data[instanceName], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// listGerritCredentials retrieves all Gerrit instances for a user
func listGerritCredentials(ctx context.Context, userID string) ([]GerritCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	secretName := gerritSecretName(userID)

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // No instances configured
		}
		return nil, fmt.Errorf("failed to get Secret: %w", err)
	}

	if secret.Data == nil {
		return nil, nil
	}

	var instances []GerritCredentials
	for key, data := range secret.Data {
		var creds GerritCredentials
		if err := json.Unmarshal(data, &creds); err != nil {
			log.Printf("Failed to parse Gerrit credentials for instance %s: %v", key, err)
			continue
		}
		instances = append(instances, creds)
	}

	return instances, nil
}

// deleteGerritCredentials removes a single Gerrit instance's credentials
func deleteGerritCredentials(ctx context.Context, userID, instanceName string) error {
	if userID == "" || instanceName == "" {
		return fmt.Errorf("userID and instanceName are required")
	}

	secretName := gerritSecretName(userID)

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Secret doesn't exist, nothing to delete
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[instanceName]) == 0 {
			return nil // Instance credentials don't exist
		}

		delete(secret.Data, instanceName)

		// If no more instances, delete the entire Secret
		if len(secret.Data) == 0 {
			if derr := K8sClient.CoreV1().Secrets(Namespace).Delete(ctx, secretName, v1.DeleteOptions{}); derr != nil && !errors.IsNotFound(derr) {
				return fmt.Errorf("failed to delete empty Secret: %w", derr)
			}
			return nil
		}

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

// parseGitcookies parses gitcookies content and extracts the cookie for the given Gerrit URL.
// Each line follows the Netscape cookie format: host\tsubdomain_flag\tpath\tsecure\texpiry\tname\tvalue
// Subdomain flag "TRUE" means wildcard match (URL host is a subdomain of cookie host),
// "FALSE" means exact match.
func parseGitcookies(gerritURL, content string) (string, error) {
	parsed, err := url.Parse(gerritURL)
	if err != nil {
		return "", fmt.Errorf("invalid Gerrit URL: %w", err)
	}
	targetHost := parsed.Hostname()

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		cookieHost := fields[0]
		subdomainFlag := strings.ToUpper(fields[1])
		cookieName := fields[5]
		cookieValue := fields[6]

		var matched bool
		if subdomainFlag == "TRUE" {
			// Wildcard: target host must be the cookie host or a subdomain of it
			matched = targetHost == cookieHost || strings.HasSuffix(targetHost, "."+cookieHost)
		} else {
			// Exact match
			matched = targetHost == cookieHost
		}

		if matched {
			return fmt.Sprintf("%s=%s", cookieName, cookieValue), nil
		}
	}

	return "", fmt.Errorf("no matching cookie found for host %s", targetHost)
}
