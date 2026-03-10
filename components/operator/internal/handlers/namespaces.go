package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"ambient-code-operator/internal/config"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

// WatchNamespaces watches for managed namespace events
func WatchNamespaces() {
	for {
		watcher, err := config.K8sClient.CoreV1().Namespaces().Watch(context.TODO(), v1.ListOptions{
			LabelSelector: "ambient-code.io/managed=true",
		})
		if err != nil {
			log.Printf("Failed to create namespace watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for managed namespaces...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				namespace := event.Object.(*corev1.Namespace)
				log.Printf("Detected new managed namespace: %s", namespace.Name)

				// Auto-create ProjectSettings for this namespace
				if err := createDefaultProjectSettings(namespace.Name); err != nil {
					log.Printf("Error creating default ProjectSettings for namespace %s: %v", namespace.Name, err)
				}

				// Auto-create empty ambient-runner-secrets for this namespace
				if err := createDefaultRunnerSecrets(namespace.Name); err != nil {
					log.Printf("Error creating default ambient-runner-secrets for namespace %s: %v", namespace.Name, err)
				}

				// PVC creation removed - sessions now use EmptyDir with S3 state persistence
				log.Printf("Namespace %s ready (using EmptyDir + S3 for session storage)", namespace.Name)
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for namespaces: %v", obj)
			}
		}

		log.Println("Namespace watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

// createDefaultRunnerSecrets creates an empty ambient-runner-secrets for the namespace
// This prevents session reconciliation failures when no API keys are configured yet
func createDefaultRunnerSecrets(namespaceName string) error {
	const secretName = "ambient-runner-secrets"

	// Check if ambient-runner-secrets already exists in this namespace
	_, err := config.K8sClient.CoreV1().Secrets(namespaceName).Get(context.TODO(), secretName, v1.GetOptions{})
	if err == nil {
		log.Printf("ambient-runner-secrets already exists in namespace %s", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking existing ambient-runner-secrets: %v", err)
	}

	// Create empty ambient-runner-secrets (users will populate via UI later)
	defaultSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"app": "ambient-runner-secrets",
			},
			Annotations: map[string]string{
				"ambient-code.io/runner-secret": "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			// Create with placeholder - users must set real API keys via Project Settings UI
			"ANTHROPIC_API_KEY": []byte(""),
		},
	}

	_, err = config.K8sClient.CoreV1().Secrets(namespaceName).Create(context.TODO(), defaultSecret, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create default ambient-runner-secrets: %v", err)
	}

	log.Printf("Created default ambient-runner-secrets for namespace %s", namespaceName)
	return nil
}
