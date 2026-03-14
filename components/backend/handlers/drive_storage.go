package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ambient-code-backend/models"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	driveIntegrationConfigMapPrefix = "drive-integration-"
	driveTokensSecretPrefix         = "drive-tokens-"

	// ConfigMap data keys.
	configKeyIntegration = "integration"
	configKeyFileGrants  = "file-grants"

	// Secret data keys.
	secretKeyAccessToken  = "access-token"
	secretKeyRefreshToken = "refresh-token"
	secretKeyExpiresAt    = "expires-at"
)

// DriveStorage handles persistence of DriveIntegration and FileGrant data
// using Kubernetes ConfigMaps and Secrets as the backing store.
type DriveStorage struct {
	clientset kubernetes.Interface
	namespace string
}

// NewDriveStorage creates a new DriveStorage instance.
func NewDriveStorage(clientset kubernetes.Interface, namespace string) *DriveStorage {
	return &DriveStorage{
		clientset: clientset,
		namespace: namespace,
	}
}

// configMapName returns the deterministic ConfigMap name for a given
// project and user combination.
func configMapName(projectName, userID string) string {
	return driveIntegrationConfigMapPrefix + projectName + "-" + userID
}

// secretName returns the deterministic Secret name for a given
// project and user combination.
func secretName(projectName, userID string) string {
	return driveTokensSecretPrefix + projectName + "-" + userID
}

// ---------------------------------------------------------------------------
// Integration CRUD
// ---------------------------------------------------------------------------

// GetIntegration retrieves a DriveIntegration from its backing ConfigMap.
// Returns nil and no error when the ConfigMap does not exist.
func (s *DriveStorage) GetIntegration(ctx context.Context, projectName, userID string) (*models.DriveIntegration, error) {
	name := configMapName(projectName, userID)
	cm, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get ConfigMap %s: %w", name, err)
	}

	raw, ok := cm.Data[configKeyIntegration]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s is missing key %q", name, configKeyIntegration)
	}

	var integration models.DriveIntegration
	if err := json.Unmarshal([]byte(raw), &integration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal integration from ConfigMap %s: %w", name, err)
	}

	return &integration, nil
}

// SaveIntegration persists a DriveIntegration to a ConfigMap. If the ConfigMap
// already exists it is updated; otherwise a new one is created.
func (s *DriveStorage) SaveIntegration(ctx context.Context, integration *models.DriveIntegration) error {
	name := configMapName(integration.ProjectName, integration.UserID)

	integrationJSON, err := json.Marshal(integration)
	if err != nil {
		return fmt.Errorf("failed to marshal integration: %w", err)
	}

	existing, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("failed to check for existing ConfigMap %s: %w", name, err)
		}

		// ConfigMap does not exist -- create it.
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: s.namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "platform-backend",
					"app.kubernetes.io/component":  "drive-integration",
					"platform/project":             integration.ProjectName,
					"platform/user":                integration.UserID,
				},
			},
			Data: map[string]string{
				configKeyIntegration: string(integrationJSON),
			},
		}
		if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create ConfigMap %s: %w", name, err)
		}
		return nil
	}

	// ConfigMap exists -- update it, preserving any existing file-grants data.
	if existing.Data == nil {
		existing.Data = make(map[string]string)
	}
	existing.Data[configKeyIntegration] = string(integrationJSON)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update ConfigMap %s: %w", name, err)
	}
	return nil
}

// DeleteIntegration removes the ConfigMap that backs a DriveIntegration. If the
// ConfigMap does not exist the call is a no-op.
func (s *DriveStorage) DeleteIntegration(ctx context.Context, projectName, userID string) error {
	name := configMapName(projectName, userID)
	err := s.clientset.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ConfigMap %s: %w", name, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// FileGrant operations
// ---------------------------------------------------------------------------

// ListFileGrants returns all FileGrants stored in the ConfigMap identified by
// the given integrationID. The integrationID is used to locate the ConfigMap
// by scanning for a matching integration payload. In practice, callers should
// first resolve projectName/userID from the integration and use those to build
// the ConfigMap name. This implementation searches the integration JSON stored
// inside each candidate ConfigMap.
//
// For efficiency the caller should provide the integrationID that corresponds
// to a known projectName/userID pair. This method lists ConfigMaps with the
// drive-integration label and finds the matching one.
func (s *DriveStorage) ListFileGrants(ctx context.Context, integrationID string) ([]models.FileGrant, error) {
	cm, err := s.findConfigMapByIntegrationID(ctx, integrationID)
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return nil, fmt.Errorf("no ConfigMap found for integration %s", integrationID)
	}

	return parseFileGrants(cm)
}

// UpdateFileGrants replaces the file-grants data in the ConfigMap that belongs
// to the given integrationID.
func (s *DriveStorage) UpdateFileGrants(ctx context.Context, integrationID string, grants []models.FileGrant) error {
	cm, err := s.findConfigMapByIntegrationID(ctx, integrationID)
	if err != nil {
		return err
	}
	if cm == nil {
		return fmt.Errorf("no ConfigMap found for integration %s", integrationID)
	}

	grantsJSON, err := json.Marshal(grants)
	if err != nil {
		return fmt.Errorf("failed to marshal file grants: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[configKeyFileGrants] = string(grantsJSON)

	if _, err := s.clientset.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update file grants in ConfigMap %s: %w", cm.Name, err)
	}
	return nil
}

// findConfigMapByIntegrationID locates the ConfigMap whose integration JSON
// contains the given ID. Returns nil (without error) when no match is found.
func (s *DriveStorage) findConfigMapByIntegrationID(ctx context.Context, integrationID string) (*corev1.ConfigMap, error) {
	list, err := s.clientset.CoreV1().ConfigMaps(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/component=drive-integration",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list drive-integration ConfigMaps: %w", err)
	}

	for i := range list.Items {
		cm := &list.Items[i]
		raw, ok := cm.Data[configKeyIntegration]
		if !ok {
			continue
		}
		var integration models.DriveIntegration
		if err := json.Unmarshal([]byte(raw), &integration); err != nil {
			continue
		}
		if integration.ID == integrationID {
			return cm, nil
		}
	}
	return nil, nil
}

// parseFileGrants extracts the FileGrant slice from a ConfigMap. An absent
// file-grants key is treated as an empty list (not an error).
func parseFileGrants(cm *corev1.ConfigMap) ([]models.FileGrant, error) {
	raw, ok := cm.Data[configKeyFileGrants]
	if !ok || raw == "" {
		return []models.FileGrant{}, nil
	}

	var grants []models.FileGrant
	if err := json.Unmarshal([]byte(raw), &grants); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file grants from ConfigMap %s: %w", cm.Name, err)
	}
	return grants, nil
}

// ---------------------------------------------------------------------------
// Token operations (stored in Kubernetes Secrets)
// ---------------------------------------------------------------------------

// SaveTokens persists OAuth tokens in a Kubernetes Secret. If the Secret
// already exists it is updated; otherwise a new one is created.
func (s *DriveStorage) SaveTokens(ctx context.Context, integration *models.DriveIntegration, accessToken, refreshToken string, expiresAt time.Time) error {
	name := secretName(integration.ProjectName, integration.UserID)

	data := map[string][]byte{
		secretKeyAccessToken:  []byte(accessToken),
		secretKeyRefreshToken: []byte(refreshToken),
		secretKeyExpiresAt:    []byte(expiresAt.UTC().Format(time.RFC3339)),
	}

	existing, err := s.clientset.CoreV1().Secrets(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("failed to check for existing Secret %s: %w", name, err)
		}

		// Secret does not exist -- create it.
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: s.namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "platform-backend",
					"app.kubernetes.io/component":  "drive-tokens",
					"platform/project":             integration.ProjectName,
					"platform/user":                integration.UserID,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		}
		if _, err := s.clientset.CoreV1().Secrets(s.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create Secret %s: %w", name, err)
		}
		return nil
	}

	// Secret exists -- update it.
	existing.Data = data
	if _, err := s.clientset.CoreV1().Secrets(s.namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update Secret %s: %w", name, err)
	}
	return nil
}

// GetTokens retrieves OAuth tokens from the Kubernetes Secret for the given
// project and user. Returns an error if the Secret does not exist.
func (s *DriveStorage) GetTokens(ctx context.Context, projectName, userID string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	name := secretName(projectName, userID)
	secret, getErr := s.clientset.CoreV1().Secrets(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if getErr != nil {
		err = fmt.Errorf("failed to get Secret %s: %w", name, getErr)
		return
	}

	accessToken = string(secret.Data[secretKeyAccessToken])
	refreshToken = string(secret.Data[secretKeyRefreshToken])

	rawExpiry := string(secret.Data[secretKeyExpiresAt])
	if rawExpiry != "" {
		expiresAt, err = time.Parse(time.RFC3339, rawExpiry)
		if err != nil {
			err = fmt.Errorf("failed to parse expires-at from Secret %s: %w", name, err)
			return
		}
	}

	return
}

// DeleteTokens removes the Kubernetes Secret that stores OAuth tokens for the
// given project and user. If the Secret does not exist the call is a no-op.
func (s *DriveStorage) DeleteTokens(ctx context.Context, projectName, userID string) error {
	name := secretName(projectName, userID)
	err := s.clientset.CoreV1().Secrets(s.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Secret %s: %w", name, err)
	}
	return nil
}
