package informer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient/platform/components/ambient-control-plane/internal/watcher"
	"github.com/rs/zerolog"
)

type EventType string

const (
	EventAdded    EventType = "ADDED"
	EventModified EventType = "MODIFIED"
	EventDeleted  EventType = "DELETED"
)

type ResourceEvent struct {
	Type      EventType
	Resource  string
	Object    any
	OldObject any
}

type EventHandler func(ctx context.Context, event ResourceEvent) error

type Informer struct {
	sdk          *sdkclient.Client
	watchManager *watcher.WatchManager
	handlers     map[string][]EventHandler
	mu           sync.RWMutex
	logger       zerolog.Logger
	eventCh      chan ResourceEvent

	sessionCache         map[string]types.Session
	projectCache         map[string]types.Project
	projectSettingsCache map[string]types.ProjectSettings
}

func New(sdk *sdkclient.Client, watchManager *watcher.WatchManager, logger zerolog.Logger) *Informer {
	return &Informer{
		sdk:                  sdk,
		watchManager:         watchManager,
		handlers:             make(map[string][]EventHandler),
		logger:               logger.With().Str("component", "informer").Logger(),
		eventCh:              make(chan ResourceEvent, 256),
		sessionCache:         make(map[string]types.Session),
		projectCache:         make(map[string]types.Project),
		projectSettingsCache: make(map[string]types.ProjectSettings),
	}
}

func (inf *Informer) RegisterHandler(resource string, handler EventHandler) {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	inf.handlers[resource] = append(inf.handlers[resource], handler)
}

func (inf *Informer) Run(ctx context.Context) error {
	inf.logger.Info().Msg("performing initial list sync")

	if err := inf.initialSync(ctx); err != nil {
		inf.logger.Warn().Err(err).Msg("initial sync failed, will rely on watch events")
	}

	go inf.dispatchLoop(ctx)

	inf.wireWatchHandlers()

	inf.logger.Info().Msg("starting gRPC watch streams")
	inf.watchManager.Run(ctx)

	return ctx.Err()
}

func (inf *Informer) dispatchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-inf.eventCh:
			inf.mu.RLock()
			handlers := inf.handlers[event.Resource]
			inf.mu.RUnlock()

			for _, handler := range handlers {
				if err := handler(ctx, event); err != nil {
					inf.logger.Error().
						Err(err).
						Str("resource", event.Resource).
						Str("event_type", string(event.Type)).
						Msg("handler failed")
				}
			}
		}
	}
}

func (inf *Informer) initialSync(ctx context.Context) error {
	var errs []string
	if err := inf.syncSessions(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial session sync failed")
		errs = append(errs, err.Error())
	}
	if err := inf.syncProjects(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial project sync failed")
		errs = append(errs, err.Error())
	}
	if err := inf.syncProjectSettings(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial project_settings sync failed")
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("initial sync failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (inf *Informer) syncSessions(ctx context.Context) error {
	opts := &types.ListOptions{Size: 100, Page: 1}
	var allSessions []types.Session
	for {
		list, err := inf.sdk.Sessions().List(ctx, opts)
		if err != nil {
			return fmt.Errorf("list sessions page %d: %w", opts.Page, err)
		}
		allSessions = append(allSessions, list.Items...)
		if len(allSessions) >= list.Total || len(list.Items) == 0 {
			break
		}
		opts.Page++
	}

	inf.mu.Lock()
	for _, session := range allSessions {
		inf.sessionCache[session.ID] = session
	}
	inf.mu.Unlock()

	for _, session := range allSessions {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "sessions",
			Object:   session,
		})
	}

	inf.logger.Info().Int("count", len(allSessions)).Msg("initial session sync complete")
	return nil
}

func (inf *Informer) syncProjects(ctx context.Context) error {
	opts := &types.ListOptions{Size: 100, Page: 1}
	var allProjects []types.Project
	for {
		list, err := inf.sdk.Projects().List(ctx, opts)
		if err != nil {
			return fmt.Errorf("list projects page %d: %w", opts.Page, err)
		}
		allProjects = append(allProjects, list.Items...)
		if len(allProjects) >= list.Total || len(list.Items) == 0 {
			break
		}
		opts.Page++
	}

	inf.mu.Lock()
	for _, project := range allProjects {
		inf.projectCache[project.ID] = project
	}
	inf.mu.Unlock()

	for _, project := range allProjects {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "projects",
			Object:   project,
		})
	}

	inf.logger.Info().Int("count", len(allProjects)).Msg("initial project sync complete")
	return nil
}

func (inf *Informer) syncProjectSettings(ctx context.Context) error {
	opts := &types.ListOptions{Size: 100, Page: 1}
	var allSettings []types.ProjectSettings
	for {
		list, err := inf.sdk.ProjectSettings().List(ctx, opts)
		if err != nil {
			return fmt.Errorf("list project_settings page %d: %w", opts.Page, err)
		}
		allSettings = append(allSettings, list.Items...)
		if len(allSettings) >= list.Total || len(list.Items) == 0 {
			break
		}
		opts.Page++
	}

	inf.mu.Lock()
	for _, ps := range allSettings {
		inf.projectSettingsCache[ps.ID] = ps
	}
	inf.mu.Unlock()

	for _, ps := range allSettings {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "project_settings",
			Object:   ps,
		})
	}

	inf.logger.Info().Int("count", len(allSettings)).Msg("initial project_settings sync complete")
	return nil
}

func (inf *Informer) wireWatchHandlers() {
	inf.watchManager.RegisterHandler("sessions", func(ctx context.Context, we watcher.WatchEvent) error {
		return inf.handleSessionWatch(ctx, we)
	})
	inf.watchManager.RegisterHandler("projects", func(ctx context.Context, we watcher.WatchEvent) error {
		return inf.handleProjectWatch(ctx, we)
	})
	inf.watchManager.RegisterHandler("project_settings", func(ctx context.Context, we watcher.WatchEvent) error {
		return inf.handleProjectSettingsWatch(ctx, we)
	})
}

func (inf *Informer) handleSessionWatch(ctx context.Context, we watcher.WatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		pbSession, ok := we.Object.(*pb.Session)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for sessions", we.Object)
		}
		session := protoSessionToSDK(pbSession)
		inf.sessionCache[session.ID] = session
		event = ResourceEvent{Type: EventAdded, Resource: "sessions", Object: session}

	case watcher.EventUpdated:
		pbSession, ok := we.Object.(*pb.Session)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for sessions", we.Object)
		}
		session := protoSessionToSDK(pbSession)
		old := inf.sessionCache[session.ID]
		inf.sessionCache[session.ID] = session
		event = ResourceEvent{Type: EventModified, Resource: "sessions", Object: session, OldObject: old}

	case watcher.EventDeleted:
		if old, found := inf.sessionCache[we.ResourceID]; found {
			delete(inf.sessionCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "sessions", Object: old}
		}
	}
	inf.mu.Unlock()

	if event.Resource != "" {
		inf.dispatchBlocking(ctx, event)
	}
	return nil
}

func (inf *Informer) handleProjectWatch(ctx context.Context, we watcher.WatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		pbProject, ok := we.Object.(*pb.Project)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for projects", we.Object)
		}
		project := protoProjectToSDK(pbProject)
		inf.projectCache[project.ID] = project
		event = ResourceEvent{Type: EventAdded, Resource: "projects", Object: project}

	case watcher.EventUpdated:
		pbProject, ok := we.Object.(*pb.Project)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for projects", we.Object)
		}
		project := protoProjectToSDK(pbProject)
		old := inf.projectCache[project.ID]
		inf.projectCache[project.ID] = project
		event = ResourceEvent{Type: EventModified, Resource: "projects", Object: project, OldObject: old}

	case watcher.EventDeleted:
		if old, found := inf.projectCache[we.ResourceID]; found {
			delete(inf.projectCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "projects", Object: old}
		}
	}
	inf.mu.Unlock()

	if event.Resource != "" {
		inf.dispatchBlocking(ctx, event)
	}
	return nil
}

func (inf *Informer) handleProjectSettingsWatch(ctx context.Context, we watcher.WatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		pbPS, ok := we.Object.(*pb.ProjectSettings)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for project_settings", we.Object)
		}
		ps := protoProjectSettingsToSDK(pbPS)
		inf.projectSettingsCache[ps.ID] = ps
		event = ResourceEvent{Type: EventAdded, Resource: "project_settings", Object: ps}

	case watcher.EventUpdated:
		pbPS, ok := we.Object.(*pb.ProjectSettings)
		if !ok {
			inf.mu.Unlock()
			return fmt.Errorf("unexpected watch event object type %T for project_settings", we.Object)
		}
		ps := protoProjectSettingsToSDK(pbPS)
		old := inf.projectSettingsCache[ps.ID]
		inf.projectSettingsCache[ps.ID] = ps
		event = ResourceEvent{Type: EventModified, Resource: "project_settings", Object: ps, OldObject: old}

	case watcher.EventDeleted:
		if old, found := inf.projectSettingsCache[we.ResourceID]; found {
			delete(inf.projectSettingsCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "project_settings", Object: old}
		}
	}
	inf.mu.Unlock()

	if event.Resource != "" {
		inf.dispatchBlocking(ctx, event)
	}
	return nil
}

func (inf *Informer) dispatchBlocking(ctx context.Context, event ResourceEvent) {
	select {
	case inf.eventCh <- event:
	case <-ctx.Done():
	}
}

func protoSessionToSDK(s *pb.Session) types.Session {
	if s == nil {
		return types.Session{}
	}
	session := types.Session{
		Name:                 s.GetName(),
		Prompt:               s.GetPrompt(),
		RepoURL:              s.GetRepoUrl(),
		Repos:                s.GetRepos(),
		LlmModel:             s.GetLlmModel(),
		LlmTemperature:       s.GetLlmTemperature(),
		LlmMaxTokens:         int(s.GetLlmMaxTokens()),
		Timeout:              int(s.GetTimeout()),
		ProjectID:            s.GetProjectId(),
		WorkflowID:           s.GetWorkflowId(),
		BotAccountName:       s.GetBotAccountName(),
		Labels:               s.GetLabels(),
		Annotations:          s.GetAnnotations(),
		ResourceOverrides:    s.GetResourceOverrides(),
		EnvironmentVariables: s.GetEnvironmentVariables(),
		CreatedByUserID:      s.GetCreatedByUserId(),
		AssignedUserID:       s.GetAssignedUserId(),
		ParentSessionID:      s.GetParentSessionId(),
		Phase:                s.GetPhase(),
		KubeCrName:           s.GetKubeCrName(),
		KubeCrUid:            s.GetKubeCrUid(),
		KubeNamespace:        s.GetKubeNamespace(),
		SdkSessionID:         s.GetSdkSessionId(),
		SdkRestartCount:      int(s.GetSdkRestartCount()),
		Conditions:           s.GetConditions(),
		ReconciledRepos:      s.GetReconciledRepos(),
		ReconciledWorkflow:   s.GetReconciledWorkflow(),
	}
	if m := s.GetMetadata(); m != nil {
		session.ID = m.GetId()
		if m.GetCreatedAt() != nil {
			t := m.GetCreatedAt().AsTime()
			session.CreatedAt = &t
		}
		if m.GetUpdatedAt() != nil {
			t := m.GetUpdatedAt().AsTime()
			session.UpdatedAt = &t
		}
	}
	if s.GetStartTime() != nil {
		t := s.GetStartTime().AsTime()
		session.StartTime = &t
	}
	if s.GetCompletionTime() != nil {
		t := s.GetCompletionTime().AsTime()
		session.CompletionTime = &t
	}
	return session
}

func protoProjectToSDK(p *pb.Project) types.Project {
	if p == nil {
		return types.Project{}
	}
	project := types.Project{
		Name:        p.GetName(),
		DisplayName: p.GetDisplayName(),
		Description: p.GetDescription(),
		Labels:      p.GetLabels(),
		Annotations: p.GetAnnotations(),
		Status:      p.GetStatus(),
	}
	if m := p.GetMetadata(); m != nil {
		project.ID = m.GetId()
		if m.GetCreatedAt() != nil {
			t := m.GetCreatedAt().AsTime()
			project.CreatedAt = &t
		}
		if m.GetUpdatedAt() != nil {
			t := m.GetUpdatedAt().AsTime()
			project.UpdatedAt = &t
		}
	}
	return project
}

func protoProjectSettingsToSDK(ps *pb.ProjectSettings) types.ProjectSettings {
	if ps == nil {
		return types.ProjectSettings{}
	}
	settings := types.ProjectSettings{
		ProjectID:    ps.GetProjectId(),
		GroupAccess:  ps.GetGroupAccess(),
		Repositories: ps.GetRepositories(),
	}
	if m := ps.GetMetadata(); m != nil {
		settings.ID = m.GetId()
		if m.GetCreatedAt() != nil {
			t := m.GetCreatedAt().AsTime()
			settings.CreatedAt = &t
		}
		if m.GetUpdatedAt() != nil {
			t := m.GetUpdatedAt().AsTime()
			settings.UpdatedAt = &t
		}
	}
	return settings
}
