package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// FeatureFlagOverridesConfigMap is the name of the ConfigMap storing workspace-specific flag overrides
	FeatureFlagOverridesConfigMap = "feature-flag-overrides"
)

var (
	unleashAdminURL          = os.Getenv("UNLEASH_ADMIN_URL")           // e.g., https://unleash.example.com
	unleashAdminToken        = os.Getenv("UNLEASH_ADMIN_TOKEN")         // Admin API token
	unleashProject           = os.Getenv("UNLEASH_PROJECT")             // Unleash project ID (default: "default")
	unleashEnv               = os.Getenv("UNLEASH_ENVIRONMENT")         // Environment (default: "development")
	unleashWorkspaceTagType  = os.Getenv("UNLEASH_WORKSPACE_TAG_TYPE")  // Tag type for workspace-configurable flags (default: "scope")
	unleashWorkspaceTagValue = os.Getenv("UNLEASH_WORKSPACE_TAG_VALUE") // Tag value for workspace-configurable flags (default: "workspace")
)

// FeatureToggle represents a feature toggle with workspace override status
type FeatureToggle struct {
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	Enabled         bool       `json:"enabled"`
	Type            string     `json:"type,omitempty"` // release, experiment, operational, etc.
	Stale           bool       `json:"stale,omitempty"`
	Tags            []Tag      `json:"tags,omitempty"`
	Environments    []EnvState `json:"environments,omitempty"`
	Source          string     `json:"source"`                    // "workspace-override" or "unleash"
	OverrideEnabled *bool      `json:"overrideEnabled,omitempty"` // nil if no override, true/false if overridden
}

// Tag represents an Unleash tag
type Tag struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// EnvState represents the state of a toggle in an environment
type EnvState struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// unleashFeaturesResponse is the response from Unleash Admin API for listing features
type unleashFeaturesResponse struct {
	Features []unleashFeature `json:"features"`
}

// unleashFeature is a single feature from Unleash Admin API
type unleashFeature struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Type         string               `json:"type"`
	Stale        bool                 `json:"stale"`
	Tags         []Tag                `json:"tags"`
	Environments []unleashEnvironment `json:"environments"`
}

// unleashEnvironment represents an environment in the Unleash list response
type unleashEnvironment struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// OverrideRequest represents a request to set a feature flag override
type OverrideRequest struct {
	Enabled bool `json:"enabled"`
}

// sanitizeParam strips newlines and carriage returns from user-controlled URL
// params to prevent log injection attacks.
func sanitizeParam(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}

// getWorkspaceOverrides reads the feature-flag-overrides ConfigMap for a namespace
// Uses the provided K8s client (should be user-scoped for RBAC enforcement)
func getWorkspaceOverrides(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (map[string]string, error) {
	cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil // No overrides ConfigMap exists
	}
	if err != nil {
		return nil, err
	}
	return cm.Data, nil
}

// checkConfigMapPermission verifies the user has permission to perform the specified verb on the ConfigMap
func checkConfigMapPermission(ctx context.Context, reqK8s kubernetes.Interface, namespace, verb string) (bool, error) {
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Resource:  "configmaps",
				Verb:      verb,
				Namespace: namespace,
				Name:      FeatureFlagOverridesConfigMap,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return res.Status.Allowed, nil
}

// isWorkspaceConfigurable checks if a feature flag has the workspace-configurable tag.
// Only flags with this tag are shown in the workspace admin UI.
// Flags without this tag are platform-only and can only be managed via Unleash UI.
func isWorkspaceConfigurable(tags []Tag) bool {
	tagType := getWorkspaceTagType()
	tagValue := getWorkspaceTagValue()

	for _, tag := range tags {
		if tag.Type == tagType && tag.Value == tagValue {
			return true
		}
	}
	return false
}

func getWorkspaceTagType() string {
	if unleashWorkspaceTagType == "" {
		unleashWorkspaceTagType = os.Getenv("UNLEASH_WORKSPACE_TAG_TYPE")
	}
	if unleashWorkspaceTagType == "" {
		return "scope"
	}
	return unleashWorkspaceTagType
}

func getWorkspaceTagValue() string {
	if unleashWorkspaceTagValue == "" {
		unleashWorkspaceTagValue = os.Getenv("UNLEASH_WORKSPACE_TAG_VALUE")
	}
	if unleashWorkspaceTagValue == "" {
		return "workspace"
	}
	return unleashWorkspaceTagValue
}

// ListFeatureFlags handles GET /api/projects/:projectName/feature-flags
// Lists all feature flags from Unleash with workspace override status
func ListFeatureFlags(c *gin.Context) {
	ctx := context.Background()
	namespace := sanitizeParam(c.Param("projectName"))

	// Verify user has project access first (uses user token)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	// Get workspace overrides using user-scoped client for RBAC enforcement
	overrides, err := getWorkspaceOverrides(ctx, reqK8s, namespace)
	if err != nil {
		log.Printf("Failed to get workspace overrides for %s: %v", namespace, err)
		// Continue - overrides are optional (user may not have ConfigMap read permission)
	}

	// Check if Unleash Admin is configured
	if getUnleashAdminURL() == "" || getUnleashAdminToken() == "" {
		// Return just workspace overrides if Unleash not configured
		if len(overrides) > 0 {
			features := make([]FeatureToggle, 0, len(overrides))
			for flagName, value := range overrides {
				enabled := value == "true"
				features = append(features, FeatureToggle{
					Name:            flagName,
					Enabled:         enabled,
					Source:          "workspace-override",
					OverrideEnabled: &enabled,
				})
			}
			c.JSON(http.StatusOK, gin.H{"features": features})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Unleash Admin API not configured",
			"message": "Set UNLEASH_ADMIN_URL and UNLEASH_ADMIN_TOKEN environment variables",
		})
		return
	}

	url := fmt.Sprintf("%s/api/admin/projects/%s/features",
		strings.TrimSuffix(getUnleashAdminURL(), "/"),
		url.PathEscape(getUnleashProject()))

	resp, err := unleashAdminRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to connect to Unleash Admin API: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect to Unleash"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Unleash response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unleash Admin API returned %d: %s", resp.StatusCode, string(body))
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch feature flags from Unleash"})
		return
	}

	// Parse and transform response
	var unleashResp unleashFeaturesResponse
	if err := json.Unmarshal(body, &unleashResp); err != nil {
		log.Printf("Failed to parse Unleash response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	// Transform to our format with workspace override status.
	// Only include flags that are workspace-configurable (have the required tag).
	targetEnv := getUnleashEnv()

	var features []FeatureToggle
	for _, f := range unleashResp.Features {
		if !isWorkspaceConfigurable(f.Tags) {
			continue
		}

		unleashEnabled := false
		envStates := make([]EnvState, 0, len(f.Environments))
		for _, env := range f.Environments {
			envStates = append(envStates, EnvState(env))
			if env.Name == targetEnv {
				unleashEnabled = env.Enabled
			}
		}

		// Check for workspace override
		var overrideEnabled *bool
		effectiveEnabled := unleashEnabled
		source := "unleash"

		if overrides != nil {
			if override, exists := overrides[f.Name]; exists {
				enabled := override == "true"
				overrideEnabled = &enabled
				effectiveEnabled = enabled
				source = "workspace-override"
			}
		}

		features = append(features, FeatureToggle{
			Name:            f.Name,
			Description:     f.Description,
			Enabled:         effectiveEnabled,
			Type:            f.Type,
			Stale:           f.Stale,
			Tags:            f.Tags,
			Environments:    envStates,
			Source:          source,
			OverrideEnabled: overrideEnabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{"features": features})
}

// EvaluateFeatureFlag handles GET /api/projects/:projectName/feature-flags/evaluate/:flagName
// Evaluates a feature flag for a workspace using three-state logic:
// 1. Check ConfigMap override - if set, return that value
// 2. Fall back to Unleash default
func EvaluateFeatureFlag(c *gin.Context) {
	ctx := context.Background()
	namespace := sanitizeParam(c.Param("projectName"))
	flagName := sanitizeParam(c.Param("flagName"))

	// Verify user has project access first
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flag name is required"})
		return
	}

	// 1. Check ConfigMap for workspace override using user-scoped client
	overrides, err := getWorkspaceOverrides(ctx, reqK8s, namespace)
	if err != nil {
		log.Printf("Failed to get workspace overrides for %s: %v", namespace, err)
		// Continue to Unleash fallback
	}

	if overrides != nil {
		if override, exists := overrides[flagName]; exists {
			enabled := override == "true"
			c.JSON(http.StatusOK, gin.H{
				"flag":    flagName,
				"enabled": enabled,
				"source":  "workspace-override",
			})
			return
		}
	}

	// 2. Fall back to Unleash Client SDK (generates metrics for flag evaluation)
	// This uses the initialized Unleash Go SDK which properly tracks usage metrics.
	// The SDK was initialized in main.go via featureflags.Init().
	enabled := FeatureEnabledForRequest(c, flagName)
	c.JSON(http.StatusOK, gin.H{
		"flag":    flagName,
		"enabled": enabled,
		"source":  "unleash",
	})
}

// GetFeatureFlag handles GET /api/projects/:projectName/feature-flags/:flagName
// Gets details for a specific feature toggle
func GetFeatureFlag(c *gin.Context) {
	// Verify user has project access first
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	flagName := sanitizeParam(c.Param("flagName"))
	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flag name is required"})
		return
	}

	// Check if Unleash Admin is configured
	if getUnleashAdminURL() == "" || getUnleashAdminToken() == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Unleash Admin API not configured",
			"message": "Set UNLEASH_ADMIN_URL and UNLEASH_ADMIN_TOKEN environment variables",
		})
		return
	}

	url := fmt.Sprintf("%s/api/admin/projects/%s/features/%s",
		strings.TrimSuffix(getUnleashAdminURL(), "/"),
		url.PathEscape(getUnleashProject()),
		url.PathEscape(flagName))

	resp, err := unleashAdminRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to connect to Unleash Admin API: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect to Unleash"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Unleash response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unleash Admin API returned %d for flag %s: %s", resp.StatusCode, flagName, string(body))
		if resp.StatusCode == http.StatusNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feature flag not found"})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch feature flag from Unleash"})
		}
		return
	}

	c.Data(resp.StatusCode, "application/json", body)
}

// setFlagOverride is the shared implementation for setting a feature flag override value.
// It handles auth, RBAC check, ConfigMap creation if needed, and the write.
// responseMsg is the message returned in the JSON response (e.g. "Override set", "Feature enabled").
func setFlagOverride(c *gin.Context, value string, responseMsg string) {
	ctx := context.Background()
	namespace := sanitizeParam(c.Param("projectName"))
	flagName := sanitizeParam(c.Param("flagName"))

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flag name is required"})
		return
	}

	// Check if ConfigMap exists to determine required verb
	cm, err := reqK8s.CoreV1().ConfigMaps(namespace).Get(ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
	configMapExists := !errors.IsNotFound(err)
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to get feature flag overrides ConfigMap in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set feature flag override"})
		return
	}

	// Check user has permission to create/update ConfigMaps in namespace
	verb := "update"
	if !configMapExists {
		verb = "create"
	}
	allowed, err := checkConfigMapPermission(ctx, reqK8s, namespace, verb)
	if err != nil {
		log.Printf("Failed to check ConfigMap permissions in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to modify feature flags"})
		return
	}

	// Use service account for the write (after validation)
	if !configMapExists {
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      FeatureFlagOverridesConfigMap,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ambient-code",
					"app.kubernetes.io/component":  "feature-flags",
				},
			},
			Data: map[string]string{},
		}
		cm, err = K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, newCM, metav1.CreateOptions{})
		if errors.IsAlreadyExists(err) {
			// Another concurrent request created the ConfigMap; fetch it and proceed to update.
			cm, err = K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
		}
		if err != nil {
			log.Printf("Failed to create feature flag overrides ConfigMap in %s: %v", namespace, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set feature flag override"})
			return
		}
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[flagName] = value

	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update feature flag overrides ConfigMap in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set feature flag override"})
		return
	}

	enabled := value == "true"
	log.Printf("Feature flag override: %s=%s in workspace %s", flagName, value, namespace)
	c.JSON(http.StatusOK, gin.H{
		"message": responseMsg,
		"flag":    flagName,
		"enabled": enabled,
		"source":  "workspace-override",
	})
}

// SetFeatureFlagOverride handles PUT /api/projects/:projectName/feature-flags/:flagName/override
// Sets a workspace-scoped override for a feature flag
func SetFeatureFlagOverride(c *gin.Context) {
	var req OverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	setFlagOverride(c, strconv.FormatBool(req.Enabled), "Override set")
}

// DeleteFeatureFlagOverride handles DELETE /api/projects/:projectName/feature-flags/:flagName/override
// Removes a workspace-scoped override, reverting to Unleash default
func DeleteFeatureFlagOverride(c *gin.Context) {
	ctx := context.Background()
	namespace := sanitizeParam(c.Param("projectName"))
	flagName := sanitizeParam(c.Param("flagName"))

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	if flagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Flag name is required"})
		return
	}

	cm, err := reqK8s.CoreV1().ConfigMaps(namespace).Get(ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		c.JSON(http.StatusOK, gin.H{
			"message": "No override to remove",
			"flag":    flagName,
		})
		return
	}
	if err != nil {
		log.Printf("Failed to get feature flag overrides ConfigMap in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get overrides"})
		return
	}

	allowed, err := checkConfigMapPermission(ctx, reqK8s, namespace, "update")
	if err != nil {
		log.Printf("Failed to check ConfigMap permissions in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to modify feature flags"})
		return
	}

	if cm.Data != nil {
		delete(cm.Data, flagName)
	}

	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update feature flag overrides ConfigMap in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove override"})
		return
	}

	log.Printf("Feature flag override removed: %s in workspace %s", flagName, namespace)
	c.JSON(http.StatusOK, gin.H{
		"message": "Override removed",
		"flag":    flagName,
		"source":  "unleash",
	})
}

// EnableFeatureFlag handles POST /api/projects/:projectName/feature-flags/:flagName/enable
// Sets a workspace override to enable the feature flag
func EnableFeatureFlag(c *gin.Context) {
	setFlagOverride(c, "true", "Feature enabled")
}

// DisableFeatureFlag handles POST /api/projects/:projectName/feature-flags/:flagName/disable
// Sets a workspace override to disable the feature flag
func DisableFeatureFlag(c *gin.Context) {
	setFlagOverride(c, "false", "Feature disabled")
}

// unleashAdminRequest makes an authenticated request to the Unleash Admin API
func unleashAdminRequest(method, url string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", getUnleashAdminToken())
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

func getUnleashAdminURL() string {
	if unleashAdminURL == "" {
		unleashAdminURL = os.Getenv("UNLEASH_ADMIN_URL")
	}
	return unleashAdminURL
}

func getUnleashAdminToken() string {
	if unleashAdminToken == "" {
		unleashAdminToken = os.Getenv("UNLEASH_ADMIN_TOKEN")
	}
	return unleashAdminToken
}

func getUnleashProject() string {
	if unleashProject == "" {
		unleashProject = os.Getenv("UNLEASH_PROJECT")
	}
	if unleashProject == "" {
		return "default"
	}
	return unleashProject
}

func getUnleashEnv() string {
	if unleashEnv == "" {
		unleashEnv = os.Getenv("UNLEASH_ENVIRONMENT")
	}
	if unleashEnv == "" {
		return "development"
	}
	return unleashEnv
}
