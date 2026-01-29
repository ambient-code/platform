// Package services provides reusable infrastructure services for the operator.
package services

import (
	"context"

	"ambient-code-operator/internal/config"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureProjectWorkspacePVC creates a per-namespace PVC for runner workspace if missing
func EnsureProjectWorkspacePVC(namespace string) error {
	// Check if PVC exists
	if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), "ambient-workspace", v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-workspace",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-workspace"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}
	if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// EnsureContentService deploys a per-namespace backend instance in CONTENT_SERVICE_MODE
func EnsureContentService(namespace string) error {
	// removed: per-namespace content service no longer managed by operator
	return nil
}

// EnsureSessionWorkspacePVC is deprecated - sessions now use EmptyDir with S3 state persistence
// Kept for backward compatibility but returns nil immediately
func EnsureSessionWorkspacePVC(namespace, pvcName string, ownerRefs []v1.OwnerReference) error {
	// DEPRECATED: Per-session PVCs have been replaced with EmptyDir + S3 state sync
	// This function is kept for backward compatibility but does nothing
	return nil
}
