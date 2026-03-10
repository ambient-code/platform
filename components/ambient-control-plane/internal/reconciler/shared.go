package reconciler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
)

const (
	ConditionReady              = "Ready"
	ConditionSecretsReady       = "SecretsReady"
	ConditionPodCreated         = "PodCreated"
	ConditionPodScheduled       = "PodScheduled"
	ConditionRunnerStarted      = "RunnerStarted"
	ConditionReposReconciled    = "ReposReconciled"
	ConditionWorkflowReconciled = "WorkflowReconciled"
	ConditionReconciled         = "Reconciled"
)

const (
	sdkClientTimeout = 30 * time.Second
	maxUpdateRetries = 3
)

const (
	PhasePending   = "Pending"
	PhaseCreating  = "Creating"
	PhaseRunning   = "Running"
	PhaseStopping  = "Stopping"
	PhaseStopped   = "Stopped"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)

var TerminalPhases = []string{
	PhaseStopped,
	PhaseCompleted,
	PhaseFailed,
}

type Reconciler interface {
	Resource() string
	Reconcile(ctx context.Context, event informer.ResourceEvent) error
}

type SDKClientFactory struct {
	baseURL string
	token   string
	logger  zerolog.Logger
	mu      sync.Mutex
	clients map[string]*sdkclient.Client
}

func NewSDKClientFactory(baseURL, token string, logger zerolog.Logger) *SDKClientFactory {
	return &SDKClientFactory{
		baseURL: baseURL,
		token:   token,
		logger:  logger,
		clients: make(map[string]*sdkclient.Client),
	}
}

func (f *SDKClientFactory) Token() string {
	return f.token
}

func (f *SDKClientFactory) ForProject(project string) (*sdkclient.Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.clients[project]; ok {
		return c, nil
	}
	c, err := sdkclient.NewClient(f.baseURL, f.token, project, sdkclient.WithTimeout(sdkClientTimeout))
	if err != nil {
		return nil, fmt.Errorf("creating SDK client for project %s: %w", project, err)
	}
	f.clients[project] = c
	return c, nil
}

func namespaceForSession(session types.Session) string {
	if session.ProjectID != "" {
		return strings.ToLower(session.ProjectID)
	}
	if session.KubeNamespace != "" {
		return session.KubeNamespace
	}
	return "default"
}

const (
	LabelManaged   = "ambient-code.io/managed"
	LabelProjectID = "ambient-code.io/project-id"
	LabelManagedBy = "ambient-code.io/managed-by"
)
