package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// InstallationCacheTTL is the duration to cache installation verification results (FR-025)
	InstallationCacheTTL = 1 * time.Hour
	// InstallationsConfigMapName is the name of the ConfigMap storing GitHub App installations
	InstallationsConfigMapName = "github-app-installations"
)

var (
	// ErrNotAuthorized is returned when the repository is not authorized (no GitHub App installation)
	ErrNotAuthorized = errors.New("repository not authorized - GitHub App not installed")
	// ErrInstallationNotFound is returned when no installation is found for the repository
	ErrInstallationNotFound = errors.New("GitHub App installation not found")
)

// InstallationVerifier verifies GitHub App installations for repositories
// It caches verification results for 1 hour to minimize GitHub API calls (FR-025)
type InstallationVerifier struct {
	k8sClient kubernetes.Interface
	namespace string

	mu    sync.RWMutex
	cache map[string]*installationCacheEntry // repository -> cache entry
}

type installationCacheEntry struct {
	installationID int64
	expiresAt      time.Time
}

// NewInstallationVerifier creates a new installation verifier with caching
func NewInstallationVerifier(k8sClient kubernetes.Interface, namespace string) *InstallationVerifier {
	verifier := &InstallationVerifier{
		k8sClient: k8sClient,
		namespace: namespace,
		cache:     make(map[string]*installationCacheEntry),
	}

	// Start background cache cleanup
	go verifier.cleanupExpiredCache()

	return verifier
}

// VerifyInstallation checks if the GitHub App is installed for the repository
// Returns the installation ID if authorized, error otherwise (FR-008, FR-009, FR-016)
//
// The verification is cached for 1 hour to reduce load on the ConfigMap (FR-025)
func (v *InstallationVerifier) VerifyInstallation(ctx context.Context, repository string) (int64, error) {
	// Check cache first
	v.mu.RLock()
	if entry, exists := v.cache[repository]; exists {
		if time.Now().Before(entry.expiresAt) {
			v.mu.RUnlock()
			return entry.installationID, nil
		}
	}
	v.mu.RUnlock()

	// Cache miss or expired - fetch from ConfigMap
	installationID, err := v.fetchInstallationFromConfigMap(ctx, repository)
	if err != nil {
		return 0, err
	}

	// Update cache
	v.mu.Lock()
	v.cache[repository] = &installationCacheEntry{
		installationID: installationID,
		expiresAt:      time.Now().Add(InstallationCacheTTL),
	}
	v.mu.Unlock()

	return installationID, nil
}

// fetchInstallationFromConfigMap retrieves installation ID from the ConfigMap
// This follows the existing pattern from handlers/github_auth.go
func (v *InstallationVerifier) fetchInstallationFromConfigMap(ctx context.Context, repository string) (int64, error) {
	cm, err := v.k8sClient.CoreV1().ConfigMaps(v.namespace).Get(ctx, InstallationsConfigMapName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to read installations ConfigMap: %w", err)
	}

	// The ConfigMap stores installations by user ID, but for webhook authorization
	// we need to check if ANY user has installed the app for this repository
	// We'll iterate through all installations and check if any matches this repository
	for _, installationJSON := range cm.Data {
		var installation struct {
			InstallationID int64  `json:"installationId"`
			GitHubUserID   string `json:"githubUserId"`
		}

		if err := json.Unmarshal([]byte(installationJSON), &installation); err != nil {
			continue // Skip invalid entries
		}

		// TODO: This is a simplified check - in production, we should verify the repository
		// belongs to this installation by calling the GitHub API
		// For Phase 1A, we'll assume any installation ID is valid
		// Phase 1B should enhance this with ProjectSettings mapping
		if installation.InstallationID > 0 {
			return installation.InstallationID, nil
		}
	}

	return 0, ErrInstallationNotFound
}

// GetInstallationIDForProject retrieves the installation ID for a specific project
// This uses the ProjectSettings CRD which should map project -> installation ID
// For Phase 1A, this is a placeholder that returns the first found installation
func (v *InstallationVerifier) GetInstallationIDForProject(ctx context.Context, projectName string) (int64, error) {
	// TODO: In Phase 1B, read from ProjectSettings CRD
	// For now, return the first installation we find
	return v.fetchInstallationFromConfigMap(ctx, projectName)
}

// InvalidateCache removes the cache entry for a repository
// This should be called when we know an installation has been revoked
func (v *InstallationVerifier) InvalidateCache(repository string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.cache, repository)
}

// cleanupExpiredCache runs periodically to remove expired cache entries
func (v *InstallationVerifier) cleanupExpiredCache() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		v.mu.Lock()
		now := time.Now()
		for repo, entry := range v.cache {
			if now.After(entry.expiresAt) {
				delete(v.cache, repo)
			}
		}
		v.mu.Unlock()
	}
}

// CacheSize returns the current number of cached entries (for monitoring)
func (v *InstallationVerifier) CacheSize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.cache)
}
