package reconciler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/auth"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
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
	baseURL  string
	provider auth.TokenProvider
	logger   zerolog.Logger
	mu       sync.Mutex
	clients  map[string]*sdkclient.Client
	tokens   map[string]string
}

func NewSDKClientFactory(baseURL string, provider auth.TokenProvider, logger zerolog.Logger) *SDKClientFactory {
	return &SDKClientFactory{
		baseURL:  baseURL,
		provider: provider,
		logger:   logger,
		clients:  make(map[string]*sdkclient.Client),
		tokens:   make(map[string]string),
	}
}

func (f *SDKClientFactory) Token(ctx context.Context) (string, error) {
	return f.provider.Token(ctx)
}

func (f *SDKClientFactory) ForProject(ctx context.Context, project string) (*sdkclient.Client, error) {
	token, err := f.provider.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving token for project %s: %w", project, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if c, ok := f.clients[project]; ok && f.tokens[project] == token {
		return c, nil
	}

	c, err := sdkclient.NewClient(f.baseURL, token, project, sdkclient.WithTimeout(sdkClientTimeout))
	if err != nil {
		return nil, fmt.Errorf("creating SDK client for project %s: %w", project, err)
	}
	f.clients[project] = c
	f.tokens[project] = token
	return c, nil
}

const (
	LabelManaged   = "ambient-code.io/managed"
	LabelProjectID = "ambient-code.io/project-id"
	LabelManagedBy = "ambient-code.io/managed-by"
)
