// Package handlers provides HTTP handlers for push notification management
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GetVapidPublicKey returns the VAPID public key for push notifications
func GetVapidPublicKey(c *gin.Context) {
	// Get VAPID public key from environment variable or config
	publicKey := os.Getenv("VAPID_PUBLIC_KEY")
	if publicKey == "" {
		// For development, we'll use a placeholder
		// In production, this should be generated and stored securely
		log.Println("Warning: VAPID_PUBLIC_KEY not set, push notifications will not work")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Push notifications are not configured",
		})
		return
	}

	c.JSON(http.StatusOK, types.VapidPublicKeyResponse{
		PublicKey: publicKey,
	})
}

// CreatePushSubscription creates a new push notification subscription for a project
func CreatePushSubscription(c *gin.Context) {
	projectName := c.Param("projectName")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	// Parse request body
	var req types.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Get user context from middleware
	userCtx, exists := c.Get("userContext")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}
	user := userCtx.(types.UserContext)

	// Get user-scoped K8s clients
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Check if user has access to the project
	ctx := c.Request.Context()
	if !hasProjectAccess(ctx, reqK8s, projectName) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to project"})
		return
	}

	// Create UserSubscription
	subscription := types.UserSubscription{
		ID:           uuid.New().String(),
		ProjectName:  projectName,
		UserID:       user.UserID,
		Subscription: req.Subscription,
		Preferences:  req.Preferences,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store subscription in ConfigMap
	if err := storePushSubscription(ctx, reqDyn, projectName, &subscription); err != nil {
		log.Printf("Failed to store push subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, subscription)
}

// GetCurrentPushSubscription returns the current user's push subscription for a project
func GetCurrentPushSubscription(c *gin.Context) {
	projectName := c.Param("projectName")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name is required"})
		return
	}

	// Get user context from middleware
	userCtx, exists := c.Get("userContext")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}
	user := userCtx.(types.UserContext)

	// Get user-scoped K8s clients
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Check if user has access to the project
	ctx := c.Request.Context()
	if !hasProjectAccess(ctx, reqK8s, projectName) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to project"})
		return
	}

	// Get subscription from ConfigMap
	subscription, err := getPushSubscription(ctx, reqDyn, projectName, user.UserID)
	if err != nil {
		log.Printf("Failed to get push subscription: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// UpdatePushSubscription updates notification preferences for a subscription
func UpdatePushSubscription(c *gin.Context) {
	projectName := c.Param("projectName")
	subscriptionID := c.Param("subscriptionId")

	if projectName == "" || subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name and subscription ID are required"})
		return
	}

	// Parse request body
	var req types.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Get user context from middleware
	userCtx, exists := c.Get("userContext")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}
	user := userCtx.(types.UserContext)

	// Get user-scoped K8s clients
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Check if user has access to the project
	ctx := c.Request.Context()
	if !hasProjectAccess(ctx, reqK8s, projectName) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to project"})
		return
	}

	// Get existing subscription
	subscription, err := getPushSubscription(ctx, reqDyn, projectName, user.UserID)
	if err != nil {
		log.Printf("Failed to get push subscription: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Verify subscription belongs to user
	if subscription.ID != subscriptionID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Update preferences
	subscription.Preferences = req.Preferences
	subscription.UpdatedAt = time.Now()

	// Store updated subscription
	if err := storePushSubscription(ctx, reqDyn, projectName, subscription); err != nil {
		log.Printf("Failed to update push subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// DeletePushSubscription deletes a push notification subscription
func DeletePushSubscription(c *gin.Context) {
	projectName := c.Param("projectName")
	subscriptionID := c.Param("subscriptionId")

	if projectName == "" || subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name and subscription ID are required"})
		return
	}

	// Get user context from middleware
	userCtx, exists := c.Get("userContext")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}
	user := userCtx.(types.UserContext)

	// Get user-scoped K8s clients
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Check if user has access to the project
	ctx := c.Request.Context()
	if !hasProjectAccess(ctx, reqK8s, projectName) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to project"})
		return
	}

	// Get existing subscription to verify ownership
	subscription, err := getPushSubscription(ctx, reqDyn, projectName, user.UserID)
	if err != nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}

	// Verify subscription belongs to user
	if subscription.ID != subscriptionID {
		c.JSON(http.StatusNoContent, nil)
		return
	}

	// Delete subscription from ConfigMap
	if err := deletePushSubscription(ctx, reqDyn, projectName, user.UserID); err != nil {
		log.Printf("Failed to delete push subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subscription"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Helper functions for storing/retrieving subscriptions

func storePushSubscription(ctx context.Context, dynClient dynamic.Interface, projectName string, subscription *types.UserSubscription) error {
	configMapName := "push-subscriptions"

	// Get or create ConfigMap
	configMap, err := dynClient.Resource(configMapGVR()).Namespace(projectName).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		// Create new ConfigMap
		configMap = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      configMapName,
					"namespace": projectName,
				},
				"data": map[string]interface{}{},
			},
		}
	}

	// Serialize subscription to JSON
	data, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	// Store in ConfigMap data
	dataMap, _, _ := unstructured.NestedMap(configMap.Object, "data")
	if dataMap == nil {
		dataMap = make(map[string]interface{})
	}
	dataMap[subscription.UserID] = string(data)
	if err := unstructured.SetNestedMap(configMap.Object, dataMap, "data"); err != nil {
		return fmt.Errorf("failed to set data map: %w", err)
	}

	// Create or update ConfigMap
	if configMap.GetResourceVersion() == "" {
		_, err = dynClient.Resource(configMapGVR()).Namespace(projectName).Create(ctx, configMap, metav1.CreateOptions{})
	} else {
		_, err = dynClient.Resource(configMapGVR()).Namespace(projectName).Update(ctx, configMap, metav1.UpdateOptions{})
	}

	return err
}

func getPushSubscription(ctx context.Context, dynClient dynamic.Interface, projectName, userID string) (*types.UserSubscription, error) {
	configMapName := "push-subscriptions"

	configMap, err := dynClient.Resource(configMapGVR()).Namespace(projectName).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	dataMap, found, _ := unstructured.NestedMap(configMap.Object, "data")
	if !found || dataMap == nil {
		return nil, fmt.Errorf("no subscriptions found")
	}

	subscriptionJSON, ok := dataMap[userID].(string)
	if !ok {
		return nil, fmt.Errorf("subscription not found for user")
	}

	var subscription types.UserSubscription
	if err := json.Unmarshal([]byte(subscriptionJSON), &subscription); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	return &subscription, nil
}

func deletePushSubscription(ctx context.Context, dynClient dynamic.Interface, projectName, userID string) error {
	configMapName := "push-subscriptions"

	configMap, err := dynClient.Resource(configMapGVR()).Namespace(projectName).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	dataMap, found, _ := unstructured.NestedMap(configMap.Object, "data")
	if !found || dataMap == nil {
		return nil
	}

	delete(dataMap, userID)
	if err := unstructured.SetNestedMap(configMap.Object, dataMap, "data"); err != nil {
		return fmt.Errorf("failed to set data map: %w", err)
	}

	_, err = dynClient.Resource(configMapGVR()).Namespace(projectName).Update(ctx, configMap, metav1.UpdateOptions{})
	return err
}

func configMapGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}
}

func hasProjectAccess(ctx context.Context, clientset kubernetes.Interface, projectName string) bool {
	// Check RBAC permissions using SelfSubjectAccessReview
	ssar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace: projectName,
				Verb:      "list",
				Resource:  "pods",
			},
		},
	}

	result, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to check project access: %v", err)
		return false
	}

	return result.Status.Allowed
}
