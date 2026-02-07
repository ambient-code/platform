package webhook

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var projectSettingsGVR = schema.GroupVersionResource{
	Group:    "vteam.ambient-code",
	Version:  "v1alpha1",
	Resource: "projectsettings",
}

// NamespaceResolver resolves GitHub repositories to authorized namespaces
// by querying ProjectSettings CRDs across the cluster
type NamespaceResolver struct {
	dynamicClient dynamic.Interface
}

// NewNamespaceResolver creates a new namespace resolver
func NewNamespaceResolver(dynamicClient dynamic.Interface) *NamespaceResolver {
	return &NamespaceResolver{
		dynamicClient: dynamicClient,
	}
}

// GetAuthorizedNamespace finds the namespace authorized for a given installation + repository
// Returns the namespace name or an error if not authorized
func (nr *NamespaceResolver) GetAuthorizedNamespace(
	ctx context.Context,
	installationID int64,
	repository string, // format: "owner/repo"
) (string, error) {
	// List all ProjectSettings across all namespaces
	projectSettingsList, err := nr.dynamicClient.Resource(projectSettingsGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list ProjectSettings: %w", err)
	}

	// Search for matching installation ID + repository
	for _, item := range projectSettingsList.Items {
		namespace := item.GetNamespace()

		// Extract githubInstallation from spec
		githubInstallation, found, err := unstructured.NestedMap(item.Object, "spec", "githubInstallation")
		if err != nil {
			continue // Skip if error accessing field
		}
		if !found {
			continue // Skip if no githubInstallation configured
		}

		// Check installation ID
		itemInstallationID, found, err := unstructured.NestedInt64(githubInstallation, "installationID")
		if err != nil || !found {
			continue // Skip if installationID not set
		}
		if itemInstallationID != installationID {
			continue // Wrong installation
		}

		// Check if repository is in authorized list
		repositories, found, err := unstructured.NestedStringSlice(githubInstallation, "repositories")
		if err != nil || !found {
			continue // Skip if repositories not set
		}

		for _, repo := range repositories {
			if repo == repository {
				// Found matching installation ID + repository
				return namespace, nil
			}
		}
	}

	// No matching ProjectSettings found
	return "", fmt.Errorf("repository %s not authorized for installation %d (no matching ProjectSettings)", repository, installationID)
}
