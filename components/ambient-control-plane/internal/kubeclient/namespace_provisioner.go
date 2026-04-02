package kubeclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var tenantNamespaceGVR = schema.GroupVersionResource{
	Group:    "tenant.paas.redhat.com",
	Version:  "v1alpha1",
	Resource: "tenantnamespaces",
}

type NamespaceProvisioner interface {
	NamespaceName(projectID string) string
	ProvisionNamespace(ctx context.Context, name string, labels map[string]string) error
	DeprovisionNamespace(ctx context.Context, name string) error
}

type StandardNamespaceProvisioner struct {
	kube   *KubeClient
	logger zerolog.Logger
}

func (p *StandardNamespaceProvisioner) NamespaceName(projectID string) string {
	return strings.ToLower(projectID)
}

func NewStandardNamespaceProvisioner(kube *KubeClient, logger zerolog.Logger) *StandardNamespaceProvisioner {
	return &StandardNamespaceProvisioner{
		kube:   kube,
		logger: logger.With().Str("provisioner", "standard").Logger(),
	}
}

func (p *StandardNamespaceProvisioner) ProvisionNamespace(ctx context.Context, name string, labels map[string]string) error {
	if _, err := p.kube.GetNamespace(ctx, name); err == nil {
		p.logger.Debug().Str("namespace", name).Msg("namespace already exists")
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking namespace %s: %w", name, err)
	}

	labelMap := make(map[string]interface{}, len(labels))
	for k, v := range labels {
		labelMap[k] = v
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   name,
				"labels": labelMap,
			},
		},
	}

	if _, err := p.kube.CreateNamespace(ctx, ns); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating namespace %s: %w", name, err)
	}

	p.logger.Info().Str("namespace", name).Msg("namespace created")
	return nil
}

func (p *StandardNamespaceProvisioner) DeprovisionNamespace(ctx context.Context, name string) error {
	if err := p.kube.DeleteNamespace(ctx, name); err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("deleting namespace %s: %w", name, err)
	}
	p.logger.Info().Str("namespace", name).Msg("namespace deleted")
	return nil
}

type MPPNamespaceProvisioner struct {
	kube            *KubeClient
	configNamespace string
	readyTimeout    time.Duration
	logger          zerolog.Logger
}

func NewMPPNamespaceProvisioner(kube *KubeClient, configNamespace string, logger zerolog.Logger) *MPPNamespaceProvisioner {
	return &MPPNamespaceProvisioner{
		kube:            kube,
		configNamespace: configNamespace,
		readyTimeout:    60 * time.Second,
		logger:          logger.With().Str("provisioner", "mpp").Logger(),
	}
}

const mppNamespacePrefix = "ambient-code--"

func (p *MPPNamespaceProvisioner) instanceID(namespaceName string) string {
	if len(namespaceName) > len(mppNamespacePrefix) && namespaceName[:len(mppNamespacePrefix)] == mppNamespacePrefix {
		return namespaceName[len(mppNamespacePrefix):]
	}
	return namespaceName
}

func (p *MPPNamespaceProvisioner) namespaceName(instanceID string) string {
	return mppNamespacePrefix + instanceID
}

func (p *MPPNamespaceProvisioner) NamespaceName(projectID string) string {
	return mppNamespacePrefix + strings.ToLower(projectID)
}

func (p *MPPNamespaceProvisioner) ProvisionNamespace(ctx context.Context, name string, _ map[string]string) error {
	instanceID := p.instanceID(name)
	fullNamespace := p.namespaceName(instanceID)

	existing, err := p.kube.dynamic.Resource(tenantNamespaceGVR).Namespace(p.configNamespace).Get(ctx, instanceID, metav1.GetOptions{})
	if err == nil {
		p.logger.Debug().Str("instance_id", instanceID).Str("namespace", fullNamespace).
			Str("resource_version", existing.GetResourceVersion()).
			Msg("TenantNamespace already exists")
		return p.waitForNamespaceActive(ctx, fullNamespace)
	}
	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking TenantNamespace %s: %w", instanceID, err)
	}

	tn := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "tenant.paas.redhat.com/v1alpha1",
			"kind":       "TenantNamespace",
			"metadata": map[string]interface{}{
				"name":      instanceID,
				"namespace": p.configNamespace,
				"labels": map[string]interface{}{
					"tenant.paas.redhat.com/namespace-type": "runtime",
					"tenant.paas.redhat.com/tenant":         "ambient-code",
				},
			},
			"spec": map[string]interface{}{
				"network": map[string]interface{}{
					"security-zone": "internal",
				},
				"type": "runtime",
			},
		},
	}

	if _, err := p.kube.dynamic.Resource(tenantNamespaceGVR).Namespace(p.configNamespace).Create(ctx, tn, metav1.CreateOptions{}); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating TenantNamespace %s in %s: %w", instanceID, p.configNamespace, err)
	}

	p.logger.Info().Str("instance_id", instanceID).Str("namespace", fullNamespace).Msg("TenantNamespace created")
	return p.waitForNamespaceActive(ctx, fullNamespace)
}

func (p *MPPNamespaceProvisioner) waitForNamespaceActive(ctx context.Context, name string) error {
	deadline := time.Now().Add(p.readyTimeout)
	for {
		ns, err := p.kube.GetNamespace(ctx, name)
		if err == nil {
			phase, _, _ := unstructured.NestedString(ns.Object, "status", "phase")
			if phase == "Active" {
				p.logger.Info().Str("namespace", name).Msg("namespace is Active")
				return nil
			}
			p.logger.Debug().Str("namespace", name).Str("phase", phase).Msg("waiting for namespace Active")
		} else if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("checking namespace %s status: %w", name, err)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("namespace %s did not become Active within %s", name, p.readyTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func (p *MPPNamespaceProvisioner) DeprovisionNamespace(ctx context.Context, name string) error {
	instanceID := p.instanceID(name)

	if err := p.kube.dynamic.Resource(tenantNamespaceGVR).Namespace(p.configNamespace).Delete(ctx, instanceID, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("deleting TenantNamespace %s: %w", instanceID, err)
	}

	p.logger.Info().Str("instance_id", instanceID).Str("namespace", name).Msg("TenantNamespace deleted")
	return nil
}
