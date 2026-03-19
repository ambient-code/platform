package informer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/watcher"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
)

type EventType string

const (
	EventAdded    EventType = "ADDED"
	EventModified EventType = "MODIFIED"
	EventDeleted  EventType = "DELETED"
)

// ResourceObject represents a union type for all resources handled by the informer
type ResourceObject struct {
	Session         *types.Session
	Project         *types.Project
	ProjectSettings *types.ProjectSettings
}

// GetID returns the ID of the contained resource
func (r ResourceObject) GetID() string {
	switch {
	case r.Session != nil:
		return r.Session.ID
	case r.Project != nil:
		return r.Project.ID
	case r.ProjectSettings != nil:
		return r.ProjectSettings.ID
	default:
		return ""
	}
}

// GetResourceType returns the resource type string
func (r ResourceObject) GetResourceType() string {
	switch {
	case r.Session != nil:
		return "sessions"
	case r.Project != nil:
		return "projects"
	case r.ProjectSettings != nil:
		return "project_settings"
	default:
		return ""
	}
}

// IsEmpty returns true if the ResourceObject contains no resource
func (r ResourceObject) IsEmpty() bool {
	return r.Session == nil && r.Project == nil && r.ProjectSettings == nil
}

// Constructor helpers for creating ResourceObjects
func NewSessionObject(s types.Session) ResourceObject {
	return ResourceObject{Session: &s}
}

func NewProjectObject(p types.Project) ResourceObject {
	return ResourceObject{Project: &p}
}

func NewProjectSettingsObject(ps types.ProjectSettings) ResourceObject {
	return ResourceObject{ProjectSettings: &ps}
}

type ResourceEvent struct {
	Type      EventType
	Resource  string
	Object    ResourceObject
	OldObject ResourceObject
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
	if err := inf.syncProjects(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial project sync failed")
		errs = append(errs, err.Error())
	}
	if err := inf.syncProjectSettings(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial project_settings sync failed")
		errs = append(errs, err.Error())
	}
	if err := inf.syncSessions(ctx); err != nil {
		inf.logger.Error().Err(err).Msg("initial session sync failed")
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

	func() {
		inf.mu.Lock()
		defer inf.mu.Unlock()
		for _, session := range allSessions {
			inf.sessionCache[session.ID] = session
		}
	}()

	for _, session := range allSessions {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "sessions",
			Object:   NewSessionObject(session),
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

	func() {
		inf.mu.Lock()
		defer inf.mu.Unlock()
		for _, project := range allProjects {
			inf.projectCache[project.ID] = project
		}
	}()

	for _, project := range allProjects {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "projects",
			Object:   NewProjectObject(project),
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

	func() {
		inf.mu.Lock()
		defer inf.mu.Unlock()
		for _, ps := range allSettings {
			inf.projectSettingsCache[ps.ID] = ps
		}
	}()

	for _, ps := range allSettings {
		inf.dispatchBlocking(ctx, ResourceEvent{
			Type:     EventAdded,
			Resource: "project_settings",
			Object:   NewProjectSettingsObject(ps),
		})
	}

	inf.logger.Info().Int("count", len(allSettings)).Msg("initial project_settings sync complete")
	return nil
}

func (inf *Informer) wireWatchHandlers() {
	inf.watchManager.RegisterSessionHandler(func(ctx context.Context, we watcher.SessionWatchEvent) error {
		return inf.handleSessionWatch(ctx, we)
	})
	inf.watchManager.RegisterProjectHandler(func(ctx context.Context, we watcher.ProjectWatchEvent) error {
		return inf.handleProjectWatch(ctx, we)
	})
	inf.watchManager.RegisterProjectSettingsHandler(func(ctx context.Context, we watcher.ProjectSettingsWatchEvent) error {
		return inf.handleProjectSettingsWatch(ctx, we)
	})
}

func (inf *Informer) handleSessionWatch(ctx context.Context, we watcher.SessionWatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		session := protoSessionToSDK(we.Session)
		inf.sessionCache[session.ID] = session
		event = ResourceEvent{Type: EventAdded, Resource: "sessions", Object: NewSessionObject(session)}

	case watcher.EventUpdated:
		session := protoSessionToSDK(we.Session)
		old := inf.sessionCache[session.ID]
		inf.sessionCache[session.ID] = session
		event = ResourceEvent{Type: EventModified, Resource: "sessions", Object: NewSessionObject(session), OldObject: NewSessionObject(old)}

	case watcher.EventDeleted:
		if old, found := inf.sessionCache[we.ResourceID]; found {
			delete(inf.sessionCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "sessions", Object: NewSessionObject(old)}
		} else {
			inf.logger.Warn().Str("resource_id", we.ResourceID).Msg("session DELETE event for unknown resource; dispatching tombstone")
			tombstone := types.Session{}
			tombstone.ID = we.ResourceID
			event = ResourceEvent{Type: EventDeleted, Resource: "sessions", Object: NewSessionObject(tombstone)}
		}
	}
	inf.mu.Unlock()

	if event.Resource != "" {
		inf.dispatchBlocking(ctx, event)
	}
	return nil
}

func (inf *Informer) handleProjectWatch(ctx context.Context, we watcher.ProjectWatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		project := protoProjectToSDK(we.Project)
		inf.projectCache[project.ID] = project
		event = ResourceEvent{Type: EventAdded, Resource: "projects", Object: NewProjectObject(project)}

	case watcher.EventUpdated:
		project := protoProjectToSDK(we.Project)
		old := inf.projectCache[project.ID]
		inf.projectCache[project.ID] = project
		event = ResourceEvent{Type: EventModified, Resource: "projects", Object: NewProjectObject(project), OldObject: NewProjectObject(old)}

	case watcher.EventDeleted:
		if old, found := inf.projectCache[we.ResourceID]; found {
			delete(inf.projectCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "projects", Object: NewProjectObject(old)}
		} else {
			inf.logger.Warn().Str("resource_id", we.ResourceID).Msg("project DELETE event for unknown resource; dispatching tombstone")
			tombstone := types.Project{}
			tombstone.ID = we.ResourceID
			event = ResourceEvent{Type: EventDeleted, Resource: "projects", Object: NewProjectObject(tombstone)}
		}
	}
	inf.mu.Unlock()

	if event.Resource != "" {
		inf.dispatchBlocking(ctx, event)
	}
	return nil
}

func (inf *Informer) handleProjectSettingsWatch(ctx context.Context, we watcher.ProjectSettingsWatchEvent) error {
	var event ResourceEvent

	inf.mu.Lock()
	switch we.Type {
	case watcher.EventCreated:
		ps := protoProjectSettingsToSDK(we.ProjectSettings)
		inf.projectSettingsCache[ps.ID] = ps
		event = ResourceEvent{Type: EventAdded, Resource: "project_settings", Object: NewProjectSettingsObject(ps)}

	case watcher.EventUpdated:
		ps := protoProjectSettingsToSDK(we.ProjectSettings)
		old := inf.projectSettingsCache[ps.ID]
		inf.projectSettingsCache[ps.ID] = ps
		event = ResourceEvent{Type: EventModified, Resource: "project_settings", Object: NewProjectSettingsObject(ps), OldObject: NewProjectSettingsObject(old)}

	case watcher.EventDeleted:
		if old, found := inf.projectSettingsCache[we.ResourceID]; found {
			delete(inf.projectSettingsCache, we.ResourceID)
			event = ResourceEvent{Type: EventDeleted, Resource: "project_settings", Object: NewProjectSettingsObject(old)}
		} else {
			inf.logger.Warn().Str("resource_id", we.ResourceID).Msg("project_settings DELETE event for unknown resource; dispatching tombstone")
			tombstone := types.ProjectSettings{}
			tombstone.ID = we.ResourceID
			event = ResourceEvent{Type: EventDeleted, Resource: "project_settings", Object: NewProjectSettingsObject(tombstone)}
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
