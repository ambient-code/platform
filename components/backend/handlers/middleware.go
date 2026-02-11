package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Dependencies injected from main package
var (
	BaseKubeConfig *rest.Config
	K8sClientMw    kubernetes.Interface
)

// Helper functions and types
var (
	BoolPtr   = func(b bool) *bool { return &b }
	StringPtr = func(s string) *string { return &s }
)

// Kubernetes DNS-1123 label validation (namespace, service account names)
var kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// isValidKubernetesName validates that a string is a valid Kubernetes DNS-1123 label
// Returns false if:
//   - name is empty (prevents empty string injection)
//   - name exceeds 63 characters
//   - name contains invalid characters (not lowercase alphanumeric or '-')
//   - name starts or ends with '-' (enforced by regex)
func isValidKubernetesName(name string) bool {
	// Explicit length check: reject empty strings and names > 63 chars
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	return kubernetesNameRegex.MatchString(name)
}

// isValidRBACSubject validates RBAC subject identifiers (user/group names in RoleBindings)
// RBAC subjects can contain colons (e.g., system:authenticated, system:serviceaccount:ns:name)
// Unlike Kubernetes resource names, RBAC subjects follow a different format (RFC 1123 subdomain)
// Returns false if:
//   - name is empty (prevents empty string injection)
//   - name exceeds 253 characters (max length for DNS subdomain)
//   - name contains control characters (prevents injection attacks)
func isValidRBACSubject(name string) bool {
	// Max length for RBAC subjects is 253 chars (same as DNS subdomain)
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	// Reject control characters and newlines for security
	for _, r := range name {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

// ContentListItem represents a content list item for file browsing
type ContentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// getK8sClientsDefault is the production implementation of GetK8sClientsForRequest
func getK8sClientsDefault(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
	token, tokenSource, hasAuthHeader, hasFwdToken := extractRequestToken(c)

	// SECURITY: No authentication bypass in production code.
	// All requests must provide a valid user token. No environment variable checks.
	// No fallback to service account credentials.

	if token != "" && BaseKubeConfig != nil {
		cfg := *BaseKubeConfig
		cfg.BearerToken = token
		// Ensure we do NOT fall back to the in-cluster SA token or other auth providers
		cfg.BearerTokenFile = ""
		cfg.AuthProvider = nil
		cfg.ExecProvider = nil
		cfg.Username = ""
		cfg.Password = ""

		kc, err1 := kubernetes.NewForConfig(&cfg)
		dc, err2 := dynamic.NewForConfig(&cfg)

		if err1 == nil && err2 == nil {

			// Best-effort update last-used for service account tokens
			updateAccessKeyLastUsedAnnotation(c)
			return kc, dc
		}
		// Token provided but client build failed – treat as invalid token
		log.Printf("Failed to build user-scoped k8s clients (source=%s tokenLen=%d) typedErr=%v dynamicErr=%v for %s", tokenSource, len(token), err1, err2, c.FullPath())
		return nil, nil
	}

	if token != "" && BaseKubeConfig == nil {
		// Token was provided but the backend is misconfigured; don't pretend it's a missing token.
		log.Printf("Cannot build user-scoped k8s clients: BaseKubeConfig is nil (source=%s tokenLen=%d) for %s", tokenSource, len(token), c.FullPath())
		return nil, nil
	}

	// No token provided (or headers present but parsed to empty token)
	log.Printf("No user token found for %s (tokenSource=%s hasAuthHeader=%t hasFwdToken=%t)", c.FullPath(), tokenSource, hasAuthHeader, hasFwdToken)
	return nil, nil
}

// extractRequestToken extracts a caller token from request headers with consistent semantics across
// production and test builds.
//
// Supported sources (in priority order):
//  1. Authorization: Bearer <token>  (or raw token)
//  2. X-Forwarded-Access-Token: <token>
//
// Returns:
//   - token: trimmed token ("" if none)
//   - tokenSource: "authorization", "x-forwarded-access-token", or "none"
//   - hasAuthHeader/hasFwdToken: basic presence booleans (safe for logging; never log token content)
func extractRequestToken(c *gin.Context) (token string, tokenSource string, hasAuthHeader bool, hasFwdToken bool) {
	rawAuth := c.GetHeader("Authorization")
	rawFwd := c.GetHeader("X-Forwarded-Access-Token")

	hasAuthHeader = strings.TrimSpace(rawAuth) != ""
	hasFwdToken = strings.TrimSpace(rawFwd) != ""

	// Prefer X-Forwarded-Access-Token (set by trusted OAuth proxy infrastructure).
	// This takes priority because the OAuth proxy explicitly passes the validated
	// access token here, while the Authorization header may come from untrusted
	// sources (e.g., CopilotKit runtime forwarding browser headers that contain
	// OAuth session tokens rather than valid K8s API tokens).
	if strings.TrimSpace(rawFwd) != "" {
		tokenSource = "x-forwarded-access-token"
		token = strings.TrimSpace(rawFwd)
	}

	// Fallback to Authorization header (Bearer <token> or raw token)
	if strings.TrimSpace(token) == "" && strings.TrimSpace(rawAuth) != "" {
		tokenSource = "authorization"
		parts := strings.SplitN(rawAuth, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = strings.TrimSpace(parts[1])
		} else {
			token = strings.TrimSpace(rawAuth)
		}
	}

	if strings.TrimSpace(token) == "" {
		// Preserve the source if the header existed but was malformed/empty after parsing.
		if hasFwdToken {
			return "", "x-forwarded-access-token", hasAuthHeader, hasFwdToken
		}
		if hasAuthHeader {
			return "", "authorization", hasAuthHeader, hasFwdToken
		}
		return "", "none", hasAuthHeader, hasFwdToken
	}
	return token, tokenSource, hasAuthHeader, hasFwdToken
}

// updateAccessKeyLastUsedAnnotation attempts to update the ServiceAccount's last-used annotation
// when the incoming token is a ServiceAccount JWT. Uses the backend service account client strictly
// for this telemetry update and only for SAs labeled app=ambient-access-key. Best-effort; errors ignored.
//
// RBAC:
// This function intentionally uses the backend service account (K8sClientMw) instead of user credentials
// because it updates platform-managed telemetry metadata (last-used timestamp) that users should not control.
//
// - Only updates ServiceAccounts with label app=ambient-access-key (line check below)
// - Only updates the last-used-at annotation (no other metadata changes)
// - Best-effort operation with all errors ignored (cannot disrupt user requests)
func updateAccessKeyLastUsedAnnotation(c *gin.Context) {
	// Parse Authorization header
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return
	}

	// Decode JWT payload (second segment)
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return
	}
	payloadB64 := segs[1]
	// JWT uses base64url without padding; add padding if necessary
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	// Expect sub like: system:serviceaccount:<namespace>:<sa-name>
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return
	}
	ns := parts2[0]
	saName := parts2[1]

	// Backend client must exist
	if K8sClientMw == nil {
		return
	}

	// Ensure the SA is an Ambient access key (label check) before writing
	saObj, err := K8sClientMw.CoreV1().ServiceAccounts(ns).Get(c.Request.Context(), saName, v1.GetOptions{})
	if err != nil {
		return
	}
	if saObj.Labels == nil || saObj.Labels["app"] != "ambient-access-key" {
		return
	}

	// Patch the annotation
	now := time.Now().Format(time.RFC3339)
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"ambient-code.io/last-used-at": now,
			},
		},
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return
	}
	_, err = K8sClientMw.CoreV1().ServiceAccounts(ns).Patch(c.Request.Context(), saName, types.MergePatchType, b, v1.PatchOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to update last-used annotation for SA %s/%s: %v", ns, saName, err)
	}
}

// ExtractServiceAccountFromAuth extracts namespace and ServiceAccount name from the Authorization Bearer JWT 'sub' claim
// Also checks X-Remote-User header for service account format (OpenShift OAuth proxy format)
// Returns (namespace, saName, true) when a SA subject is present, otherwise ("","",false)
func ExtractServiceAccountFromAuth(c *gin.Context) (string, string, bool) {
	// Check X-Remote-User header (OpenShift OAuth proxy format)
	// This is a production feature, not just for tests
	remoteUser := c.GetHeader("X-Remote-User")
	if remoteUser != "" {
		const prefix = "system:serviceaccount:"
		if strings.HasPrefix(remoteUser, prefix) {
			rest := strings.TrimPrefix(remoteUser, prefix)
			parts := strings.SplitN(rest, ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], true
			}
		}
	}

	// Standard Authorization Bearer JWT parsing
	rawAuth := c.GetHeader("Authorization")
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", "", false
	}
	segs := strings.Split(token, ".")
	if len(segs) < 2 {
		return "", "", false
	}
	payloadB64 := segs[1]
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", "", false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", false
	}
	sub, _ := payload["sub"].(string)
	const prefix = "system:serviceaccount:"
	if !strings.HasPrefix(sub, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(sub, prefix)
	parts2 := strings.SplitN(rest, ":", 2)
	if len(parts2) != 2 {
		return "", "", false
	}
	return parts2[0], parts2[1], true
}

// ValidateProjectContext is middleware for project context validation
func ValidateProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow token via query parameter for websocket/agent callers
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			if qp := strings.TrimSpace(c.Query("token")); qp != "" {
				c.Request.Header.Set("Authorization", "Bearer "+qp)
			}
		}

		// SECURITY: Authentication is always required - no bypass mechanism
		// Require user/API key token; do not fall back to service account
		if c.GetHeader("Authorization") == "" && c.GetHeader("X-Forwarded-Access-Token") == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
			c.Abort()
			return
		}
		reqK8s, _ := GetK8sClientsForRequest(c)
		if reqK8s == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
			c.Abort()
			return
		}
		// Prefer project from route param; fallback to header for backward compatibility
		projectHeader := c.Param("projectName")
		if projectHeader == "" {
			projectHeader = c.GetHeader("X-OpenShift-Project")
		}
		if projectHeader == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Project is required in path /api/projects/:projectName or X-OpenShift-Project header"})
			c.Abort()
			return
		}

		// Validate namespace name to prevent injection attacks
		if !isValidKubernetesName(projectHeader) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name format"})
			c.Abort()
			return
		}

		// Ensure the caller has at least list permission on agenticsessions in the namespace
		ssar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:     "vteam.ambient-code",
					Resource:  "agenticsessions",
					Verb:      "list",
					Namespace: projectHeader,
				},
			},
		}
		res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
		if err != nil {
			log.Printf("validateProjectContext: SSAR failed for %s: %v", projectHeader, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to perform access review"})
			c.Abort()
			return
		}
		if !res.Status.Allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to access project"})
			c.Abort()
			return
		}

		// Store project in context for handlers
		c.Set("project", projectHeader)
		c.Next()
	}
}

// SECURITY: Removed the previous local-dev authentication bypass helpers.
// The removed implementation relied on environment variables (test/dev flags)
// which could be accidentally set in production, creating an authentication bypass risk.
//
// Production code must NEVER bypass authentication based on environment variables.
// All requests require valid user tokens. No exceptions.
//
// For local development, use proper authentication tokens or configure the cluster
// to allow unauthenticated access only in development namespaces (not via code).
