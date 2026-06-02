package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	podSyncInterval    = 15 * time.Second
	managedLabelFilter = LabelManaged + "=true," + LabelManagedBy + "=ambient-control-plane"
)

type PodStatusSyncer struct {
	factory            *SDKClientFactory
	kube               *kubeclient.KubeClient
	platformMode       string
	mppConfigNamespace string
	logger             zerolog.Logger
}

func NewPodStatusSyncer(factory *SDKClientFactory, kube *kubeclient.KubeClient, platformMode, mppConfigNamespace string, logger zerolog.Logger) *PodStatusSyncer {
	return &PodStatusSyncer{
		factory:            factory,
		kube:               kube,
		platformMode:       platformMode,
		mppConfigNamespace: mppConfigNamespace,
		logger:             logger.With().Str("component", "pod-status-syncer").Logger(),
	}
}

func (s *PodStatusSyncer) Run(ctx context.Context) error {
	s.logger.Info().Dur("interval", podSyncInterval).Msg("pod status syncer started")
	ticker := time.NewTicker(podSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("pod status syncer stopped")
			return ctx.Err()
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

func (s *PodStatusSyncer) syncOnce(ctx context.Context) {
	namespaces, err := s.listManagedNamespaces(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to list managed namespaces")
		return
	}

	for _, ns := range namespaces {
		s.syncNamespace(ctx, ns)
	}
}

func (s *PodStatusSyncer) listManagedNamespaces(ctx context.Context) ([]string, error) {
	if s.platformMode == "mpp" {
		return s.listMPPManagedNamespaces(ctx)
	}
	nsList, err := s.kube.ListNamespacesByLabel(ctx, managedLabelFilter)
	if err != nil {
		return nil, fmt.Errorf("listing managed namespaces: %w", err)
	}

	var names []string
	for _, ns := range nsList.Items {
		names = append(names, ns.GetName())
	}
	return names, nil
}

func (s *PodStatusSyncer) listMPPManagedNamespaces(ctx context.Context) ([]string, error) {
	tnList, err := s.kube.ListTenantNamespaces(ctx, s.mppConfigNamespace, managedLabelFilter)
	if err != nil {
		return nil, fmt.Errorf("listing managed TenantNamespaces in %s: %w", s.mppConfigNamespace, err)
	}

	var names []string
	for _, tn := range tnList.Items {
		names = append(names, "ambient-code--"+tn.GetName())
	}
	return names, nil
}

func (s *PodStatusSyncer) syncNamespace(ctx context.Context, namespace string) {
	pods, err := s.kube.ListPodsByLabel(ctx, namespace, managedLabelFilter)
	if err != nil {
		s.logger.Warn().Err(err).Str("namespace", namespace).Msg("failed to list pods")
		return
	}

	for i := range pods.Items {
		s.syncPod(ctx, namespace, &pods.Items[i])
	}
}

func (s *PodStatusSyncer) syncPod(ctx context.Context, namespace string, pod *unstructured.Unstructured) {
	labels := pod.GetLabels()
	sessionID := labels["ambient-code.io/session-id"]
	projectID := labels[LabelProjectID]
	if sessionID == "" || projectID == "" {
		return
	}

	podPhase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
	desiredSessionPhase := s.mapPodPhaseToSessionPhase(podPhase, pod)
	if desiredSessionPhase == "" {
		return
	}

	sdk, err := s.factory.ForProject(ctx, projectID)
	if err != nil {
		s.logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to get SDK client")
		return
	}

	session, err := sdk.Sessions().Get(ctx, sessionID)
	if err != nil {
		s.logger.Debug().Err(err).Str("session_id", sessionID).Msg("session not found in API")
		return
	}

	if isTerminalPhase(session.Phase) {
		return
	}

	if session.Phase == desiredSessionPhase {
		return
	}

	if session.Phase == PhaseStopping && desiredSessionPhase == PhaseCompleted {
		desiredSessionPhase = PhaseStopped
	}

	s.updateSessionPhase(ctx, sdk, session, desiredSessionPhase)
}

func (s *PodStatusSyncer) mapPodPhaseToSessionPhase(podPhase string, pod *unstructured.Unstructured) string {
	switch podPhase {
	case "Succeeded":
		return PhaseCompleted
	case "Failed":
		return PhaseFailed
	case "Pending":
		if s.hasContainerCrashLoop(pod) {
			return PhaseFailed
		}
		return ""
	case "Running":
		if s.hasContainerCrashLoop(pod) {
			return PhaseFailed
		}
		return ""
	default:
		return ""
	}
}

func (s *PodStatusSyncer) hasContainerCrashLoop(pod *unstructured.Unstructured) bool {
	statuses, found, _ := unstructured.NestedSlice(pod.Object, "status", "containerStatuses")
	if !found {
		return false
	}

	for _, cs := range statuses {
		csMap, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		restartCount, _, _ := unstructured.NestedInt64(csMap, "restartCount")
		if restartCount >= 5 {
			return true
		}
		waiting, found, _ := unstructured.NestedMap(csMap, "state", "waiting")
		if found {
			reason, _, _ := unstructured.NestedString(waiting, "reason")
			if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" {
				return true
			}
		}
		name, _, _ := unstructured.NestedString(csMap, "name")
		terminated, found, _ := unstructured.NestedMap(csMap, "state", "terminated")
		if found && name == "ambient-code-runner" {
			reason, _, _ := unstructured.NestedString(terminated, "reason")
			if reason == "OOMKilled" || reason == "Error" {
				return true
			}
		}
	}

	initStatuses, found, _ := unstructured.NestedSlice(pod.Object, "status", "initContainerStatuses")
	if !found {
		return false
	}

	for _, cs := range initStatuses {
		csMap, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		waiting, found, _ := unstructured.NestedMap(csMap, "state", "waiting")
		if found {
			reason, _, _ := unstructured.NestedString(waiting, "reason")
			if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" {
				return true
			}
		}
	}
	return false
}

func (s *PodStatusSyncer) updateSessionPhase(ctx context.Context, sdk *sdkclient.Client, session *types.Session, newPhase string) {
	patch := map[string]interface{}{"phase": newPhase}

	if newPhase == PhaseCompleted || newPhase == PhaseFailed || newPhase == PhaseStopped {
		now := time.Now()
		patch["completion_time"] = &now
	}

	if _, err := sdk.Sessions().UpdateStatus(ctx, session.ID, patch); err != nil {
		s.logger.Warn().Err(err).
			Str("session_id", session.ID).
			Str("from_phase", session.Phase).
			Str("to_phase", newPhase).
			Msg("failed to update session phase from pod status")
		return
	}

	s.logger.Info().
		Str("session_id", session.ID).
		Str("from_phase", session.Phase).
		Str("to_phase", newPhase).
		Msg("session phase updated from pod status")
}

func isTerminalPhase(phase string) bool {
	for _, tp := range TerminalPhases {
		if phase == tp {
			return true
		}
	}
	return false
}
