package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// feedbackLoopConfigMap is the ConfigMap name for feedback loop config and state.
	// Stores per-project configuration, deduplication state, and trigger history.
	// State is persisted here so that backend restarts do not lose deduplication
	// state or correction counts (FR-011).
	feedbackLoopConfigMap = "feedback-loop-state"

	defaultMinCorrections  = 2
	defaultTimeWindowHours = 24
)

// FeedbackLoopConfig holds per-project feedback loop settings.
type FeedbackLoopConfig struct {
	MinCorrections     int  `json:"minCorrections"`
	TimeWindowHours    int  `json:"timeWindowHours"`
	AutoTriggerEnabled bool `json:"autoTriggerEnabled"`
}

// FeedbackLoopHistoryEntry records a triggered improvement session.
type FeedbackLoopHistoryEntry struct {
	SessionName   string   `json:"sessionName"`
	CreatedAt     string   `json:"createdAt"`
	Source        string   `json:"source"`     // "event-driven" or "github-action"
	TargetType    string   `json:"targetType"` // "workflow" or "repo"
	TargetRepoURL string   `json:"targetRepoURL"`
	TargetBranch  string   `json:"targetBranch,omitempty"`
	TargetPath    string   `json:"targetPath,omitempty"`
	CorrectionIDs []string `json:"correctionIds"`
}

// defaultFeedbackLoopConfig returns the default configuration.
func defaultFeedbackLoopConfig() FeedbackLoopConfig {
	return FeedbackLoopConfig{
		MinCorrections:     defaultMinCorrections,
		TimeWindowHours:    defaultTimeWindowHours,
		AutoTriggerEnabled: true,
	}
}

// loadFeedbackLoopConfig reads the config from the ConfigMap. Returns defaults if not found.
func loadFeedbackLoopConfig(ctx context.Context, namespace string) (FeedbackLoopConfig, error) {
	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return defaultFeedbackLoopConfig(), nil
	}
	if err != nil {
		return FeedbackLoopConfig{}, err
	}

	configData, ok := cm.Data["config"]
	if !ok || configData == "" {
		return defaultFeedbackLoopConfig(), nil
	}

	var config FeedbackLoopConfig
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		log.Printf("Failed to parse feedback loop config in %s, using defaults: %v", namespace, err)
		return defaultFeedbackLoopConfig(), nil
	}

	return config, nil
}

// saveFeedbackLoopConfig persists config to the ConfigMap. Creates it if absent.
func saveFeedbackLoopConfig(ctx context.Context, namespace string, config FeedbackLoopConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      feedbackLoopConfigMap,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ambient-code",
					"app.kubernetes.io/component":  "feedback-loop",
				},
			},
			Data: map[string]string{
				"config": string(data),
			},
		}
		_, err = K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, newCM, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["config"] = string(data)
	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// loadFeedbackLoopHistory reads the history from the ConfigMap.
func loadFeedbackLoopHistory(ctx context.Context, namespace string) ([]FeedbackLoopHistoryEntry, error) {
	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return []FeedbackLoopHistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}

	historyData, ok := cm.Data["history"]
	if !ok || historyData == "" {
		return []FeedbackLoopHistoryEntry{}, nil
	}

	var entries []FeedbackLoopHistoryEntry
	if err := json.Unmarshal([]byte(historyData), &entries); err != nil {
		log.Printf("Failed to parse feedback loop history in %s: %v", namespace, err)
		return []FeedbackLoopHistoryEntry{}, nil
	}

	return entries, nil
}

// appendFeedbackLoopHistory adds an entry to the history in the ConfigMap.
func appendFeedbackLoopHistory(ctx context.Context, namespace string, entry FeedbackLoopHistoryEntry) error {
	entries, err := loadFeedbackLoopHistory(ctx, namespace)
	if err != nil {
		return err
	}

	entries = append(entries, entry)
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      feedbackLoopConfigMap,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ambient-code",
					"app.kubernetes.io/component":  "feedback-loop",
				},
			},
			Data: map[string]string{
				"history": string(data),
			},
		}
		_, err = K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, newCM, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["history"] = string(data)
	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// GetFeedbackLoopConfig handles GET /api/projects/:projectName/feedback-loop/config
func GetFeedbackLoopConfig(c *gin.Context) {
	namespace := sanitizeParam(c.Param("projectName"))

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	config, err := loadFeedbackLoopConfig(c.Request.Context(), namespace)
	if err != nil {
		log.Printf("Failed to load feedback loop config for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load feedback loop configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// PutFeedbackLoopConfig handles PUT /api/projects/:projectName/feedback-loop/config
func PutFeedbackLoopConfig(c *gin.Context) {
	namespace := sanitizeParam(c.Param("projectName"))

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	// Check admin permission (ability to patch ConfigMaps in namespace)
	allowed, err := checkConfigMapPermission(c.Request.Context(), reqK8s, namespace, "patch")
	if err != nil {
		log.Printf("Failed to check ConfigMap permissions for feedback loop in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permissions required to modify feedback loop configuration"})
		return
	}

	var req FeedbackLoopConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate
	if req.MinCorrections < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minCorrections must be >= 1"})
		return
	}
	if req.TimeWindowHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "timeWindowHours must be >= 1"})
		return
	}

	if err := saveFeedbackLoopConfig(c.Request.Context(), namespace, req); err != nil {
		log.Printf("Failed to save feedback loop config for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback loop configuration"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// GetFeedbackLoopHistory handles GET /api/projects/:projectName/feedback-loop/history
func GetFeedbackLoopHistory(c *gin.Context) {
	namespace := sanitizeParam(c.Param("projectName"))

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	entries, err := loadFeedbackLoopHistory(c.Request.Context(), namespace)
	if err != nil {
		log.Printf("Failed to load feedback loop history for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load feedback loop history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": entries})
}
