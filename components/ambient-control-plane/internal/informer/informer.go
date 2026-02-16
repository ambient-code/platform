package informer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
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
	Object    interface{}
	OldObject interface{}
}

type EventHandler func(ctx context.Context, event ResourceEvent) error

const defaultPageSize int32 = 100

var cycleCounter atomic.Uint64

type Informer struct {
	client        *openapi.APIClient
	pollInterval  time.Duration
	pageSize      int32
	handlers      map[string][]EventHandler
	mu            sync.RWMutex
	logger        zerolog.Logger
	sessionCache         map[string]openapi.Session
	workflowCache        map[string]openapi.Workflow
	taskCache            map[string]openapi.Task
	projectCache         map[string]openapi.Project
	projectSettingsCache map[string]openapi.ProjectSettings
}

func New(client *openapi.APIClient, pollInterval time.Duration, logger zerolog.Logger) *Informer {
	return &Informer{
		client:        client,
		pollInterval:  pollInterval,
		pageSize:      defaultPageSize,
		handlers:      make(map[string][]EventHandler),
		logger:        logger.With().Str("component", "informer").Logger(),
		sessionCache:         make(map[string]openapi.Session),
		workflowCache:        make(map[string]openapi.Workflow),
		taskCache:            make(map[string]openapi.Task),
		projectCache:         make(map[string]openapi.Project),
		projectSettingsCache: make(map[string]openapi.ProjectSettings),
	}
}

func (inf *Informer) RegisterHandler(resource string, handler EventHandler) {
	inf.mu.Lock()
	defer inf.mu.Unlock()
	inf.handlers[resource] = append(inf.handlers[resource], handler)
}

func (inf *Informer) Run(ctx context.Context) error {
	inf.logger.Info().
		Dur("poll_interval", inf.pollInterval).
		Msg("starting informer loop")

	if err := inf.syncAll(ctx); err != nil {
		inf.logger.Warn().Err(err).Msg("initial sync failed, will retry")
	}

	ticker := time.NewTicker(inf.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			inf.logger.Info().Msg("informer shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := inf.syncAll(ctx); err != nil {
				inf.logger.Error().Err(err).Msg("sync cycle failed")
			}
		}
	}
}

func (inf *Informer) syncAll(ctx context.Context) error {
	cycleID := fmt.Sprintf("cycle-%d", cycleCounter.Add(1))
	log := inf.logger.With().Str("cycle_id", cycleID).Logger()
	start := time.Now()
	log.Debug().Msg("sync cycle starting")

	if err := inf.syncSessions(ctx, cycleID); err != nil {
		log.Error().Err(err).Dur("elapsed", time.Since(start)).Msg("sync cycle failed at sessions")
		return err
	}
	if err := inf.syncWorkflows(ctx, cycleID); err != nil {
		log.Error().Err(err).Dur("elapsed", time.Since(start)).Msg("sync cycle failed at workflows")
		return err
	}
	if err := inf.syncTasks(ctx, cycleID); err != nil {
		log.Error().Err(err).Dur("elapsed", time.Since(start)).Msg("sync cycle failed at tasks")
		return err
	}
	if err := inf.syncProjects(ctx, cycleID); err != nil {
		log.Error().Err(err).Dur("elapsed", time.Since(start)).Msg("sync cycle failed at projects")
		return err
	}
	if err := inf.syncProjectSettings(ctx, cycleID); err != nil {
		log.Error().Err(err).Dur("elapsed", time.Since(start)).Msg("sync cycle failed at project_settings")
		return err
	}

	log.Debug().Dur("elapsed", time.Since(start)).Msg("sync cycle complete")
	return nil
}

func (inf *Informer) fetchAllSessions(ctx context.Context, cycleID string) ([]openapi.Session, error) {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "sessions").Logger()
	var allItems []openapi.Session
	var page int32 = 1
	for {
		log.Debug().Int32("page", page).Int32("page_size", inf.pageSize).Msg("fetching page")
		list, httpResp, err := inf.client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).
			Page(page).Size(inf.pageSize).Execute()
		if err != nil {
			log.Error().Err(err).Int32("page", page).Msg("fetch failed")
			return nil, err
		}
		log.Debug().
			Int32("page", page).
			Int("items_in_page", len(list.Items)).
			Int32("total", list.Total).
			Int("http_status", httpResp.StatusCode).
			Msg("page received")
		allItems = append(allItems, list.Items...)
		if int32(len(allItems)) >= list.Total || len(list.Items) == 0 {
			break
		}
		page++
	}
	log.Debug().Int("total_fetched", len(allItems)).Msg("fetch complete")
	return allItems, nil
}

func (inf *Informer) fetchAllWorkflows(ctx context.Context, cycleID string) ([]openapi.Workflow, error) {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "workflows").Logger()
	var allItems []openapi.Workflow
	var page int32 = 1
	for {
		log.Debug().Int32("page", page).Int32("page_size", inf.pageSize).Msg("fetching page")
		list, httpResp, err := inf.client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).
			Page(page).Size(inf.pageSize).Execute()
		if err != nil {
			log.Error().Err(err).Int32("page", page).Msg("fetch failed")
			return nil, err
		}
		log.Debug().
			Int32("page", page).
			Int("items_in_page", len(list.Items)).
			Int32("total", list.Total).
			Int("http_status", httpResp.StatusCode).
			Msg("page received")
		allItems = append(allItems, list.Items...)
		if int32(len(allItems)) >= list.Total || len(list.Items) == 0 {
			break
		}
		page++
	}
	log.Debug().Int("total_fetched", len(allItems)).Msg("fetch complete")
	return allItems, nil
}

func (inf *Informer) fetchAllTasks(ctx context.Context, cycleID string) ([]openapi.Task, error) {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "tasks").Logger()
	var allItems []openapi.Task
	var page int32 = 1
	for {
		log.Debug().Int32("page", page).Int32("page_size", inf.pageSize).Msg("fetching page")
		list, httpResp, err := inf.client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).
			Page(page).Size(inf.pageSize).Execute()
		if err != nil {
			log.Error().Err(err).Int32("page", page).Msg("fetch failed")
			return nil, err
		}
		log.Debug().
			Int32("page", page).
			Int("items_in_page", len(list.Items)).
			Int32("total", list.Total).
			Int("http_status", httpResp.StatusCode).
			Msg("page received")
		allItems = append(allItems, list.Items...)
		if int32(len(allItems)) >= list.Total || len(list.Items) == 0 {
			break
		}
		page++
	}
	log.Debug().Int("total_fetched", len(allItems)).Msg("fetch complete")
	return allItems, nil
}

func (inf *Informer) syncSessions(ctx context.Context, cycleID string) error {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "sessions").Logger()
	sessions, err := inf.fetchAllSessions(ctx, cycleID)
	if err != nil {
		return err
	}

	var added, modified, deleted int
	currentIDs := make(map[string]bool)
	for _, session := range sessions {
		id := session.GetId()
		currentIDs[id] = true

		if existing, found := inf.sessionCache[id]; found {
			if session.GetUpdatedAt() != existing.GetUpdatedAt() {
				inf.sessionCache[id] = session
				inf.dispatch(ctx, ResourceEvent{
					Type:      EventModified,
					Resource:  "sessions",
					Object:    session,
					OldObject: existing,
				})
				modified++
			}
		} else {
			inf.sessionCache[id] = session
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventAdded,
				Resource: "sessions",
				Object:   session,
			})
			added++
		}
	}

	for id, session := range inf.sessionCache {
		if !currentIDs[id] {
			delete(inf.sessionCache, id)
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventDeleted,
				Resource: "sessions",
				Object:   session,
			})
			deleted++
		}
	}

	log.Debug().
		Int("fetched", len(sessions)).
		Int("cached", len(inf.sessionCache)).
		Int("added", added).Int("modified", modified).Int("deleted", deleted).
		Msg("sync complete")
	return nil
}

func (inf *Informer) syncWorkflows(ctx context.Context, cycleID string) error {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "workflows").Logger()
	workflows, err := inf.fetchAllWorkflows(ctx, cycleID)
	if err != nil {
		return err
	}

	var added, modified, deleted int
	currentIDs := make(map[string]bool)
	for _, workflow := range workflows {
		id := workflow.GetId()
		currentIDs[id] = true

		if existing, found := inf.workflowCache[id]; found {
			if workflow.GetUpdatedAt() != existing.GetUpdatedAt() {
				inf.workflowCache[id] = workflow
				inf.dispatch(ctx, ResourceEvent{
					Type:      EventModified,
					Resource:  "workflows",
					Object:    workflow,
					OldObject: existing,
				})
				modified++
			}
		} else {
			inf.workflowCache[id] = workflow
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventAdded,
				Resource: "workflows",
				Object:   workflow,
			})
			added++
		}
	}

	for id, workflow := range inf.workflowCache {
		if !currentIDs[id] {
			delete(inf.workflowCache, id)
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventDeleted,
				Resource: "workflows",
				Object:   workflow,
			})
			deleted++
		}
	}

	log.Debug().
		Int("fetched", len(workflows)).
		Int("cached", len(inf.workflowCache)).
		Int("added", added).Int("modified", modified).Int("deleted", deleted).
		Msg("sync complete")
	return nil
}

func (inf *Informer) syncTasks(ctx context.Context, cycleID string) error {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "tasks").Logger()
	tasks, err := inf.fetchAllTasks(ctx, cycleID)
	if err != nil {
		return err
	}

	var added, modified, deleted int
	currentIDs := make(map[string]bool)
	for _, task := range tasks {
		id := task.GetId()
		currentIDs[id] = true

		if existing, found := inf.taskCache[id]; found {
			if task.GetUpdatedAt() != existing.GetUpdatedAt() {
				inf.taskCache[id] = task
				inf.dispatch(ctx, ResourceEvent{
					Type:      EventModified,
					Resource:  "tasks",
					Object:    task,
					OldObject: existing,
				})
				modified++
			}
		} else {
			inf.taskCache[id] = task
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventAdded,
				Resource: "tasks",
				Object:   task,
			})
			added++
		}
	}

	for id, task := range inf.taskCache {
		if !currentIDs[id] {
			delete(inf.taskCache, id)
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventDeleted,
				Resource: "tasks",
				Object:   task,
			})
			deleted++
		}
	}

	log.Debug().
		Int("fetched", len(tasks)).
		Int("cached", len(inf.taskCache)).
		Int("added", added).Int("modified", modified).Int("deleted", deleted).
		Msg("sync complete")
	return nil
}

func (inf *Informer) fetchAllProjects(ctx context.Context, cycleID string) ([]openapi.Project, error) {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "projects").Logger()
	var allItems []openapi.Project
	var page int32 = 1
	for {
		log.Debug().Int32("page", page).Int32("page_size", inf.pageSize).Msg("fetching page")
		list, httpResp, err := inf.client.DefaultAPI.ApiAmbientApiServerV1ProjectsGet(ctx).
			Page(page).Size(inf.pageSize).Execute()
		if err != nil {
			log.Error().Err(err).Int32("page", page).Msg("fetch failed")
			return nil, err
		}
		log.Debug().
			Int32("page", page).
			Int("items_in_page", len(list.Items)).
			Int32("total", list.Total).
			Int("http_status", httpResp.StatusCode).
			Msg("page received")
		allItems = append(allItems, list.Items...)
		if int32(len(allItems)) >= list.Total || len(list.Items) == 0 {
			break
		}
		page++
	}
	log.Debug().Int("total_fetched", len(allItems)).Msg("fetch complete")
	return allItems, nil
}

func (inf *Informer) fetchAllProjectSettings(ctx context.Context, cycleID string) ([]openapi.ProjectSettings, error) {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "project_settings").Logger()
	var allItems []openapi.ProjectSettings
	var page int32 = 1
	for {
		log.Debug().Int32("page", page).Int32("page_size", inf.pageSize).Msg("fetching page")
		list, httpResp, err := inf.client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(ctx).
			Page(page).Size(inf.pageSize).Execute()
		if err != nil {
			log.Error().Err(err).Int32("page", page).Msg("fetch failed")
			return nil, err
		}
		log.Debug().
			Int32("page", page).
			Int("items_in_page", len(list.Items)).
			Int32("total", list.Total).
			Int("http_status", httpResp.StatusCode).
			Msg("page received")
		allItems = append(allItems, list.Items...)
		if int32(len(allItems)) >= list.Total || len(list.Items) == 0 {
			break
		}
		page++
	}
	log.Debug().Int("total_fetched", len(allItems)).Msg("fetch complete")
	return allItems, nil
}

func (inf *Informer) syncProjects(ctx context.Context, cycleID string) error {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "projects").Logger()
	projects, err := inf.fetchAllProjects(ctx, cycleID)
	if err != nil {
		return err
	}

	var added, modified, deleted int
	currentIDs := make(map[string]bool)
	for _, project := range projects {
		id := project.GetId()
		currentIDs[id] = true

		if existing, found := inf.projectCache[id]; found {
			if project.GetUpdatedAt() != existing.GetUpdatedAt() {
				inf.projectCache[id] = project
				inf.dispatch(ctx, ResourceEvent{
					Type:      EventModified,
					Resource:  "projects",
					Object:    project,
					OldObject: existing,
				})
				modified++
			}
		} else {
			inf.projectCache[id] = project
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventAdded,
				Resource: "projects",
				Object:   project,
			})
			added++
		}
	}

	for id, project := range inf.projectCache {
		if !currentIDs[id] {
			delete(inf.projectCache, id)
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventDeleted,
				Resource: "projects",
				Object:   project,
			})
			deleted++
		}
	}

	log.Debug().
		Int("fetched", len(projects)).
		Int("cached", len(inf.projectCache)).
		Int("added", added).Int("modified", modified).Int("deleted", deleted).
		Msg("sync complete")
	return nil
}

func (inf *Informer) syncProjectSettings(ctx context.Context, cycleID string) error {
	log := inf.logger.With().Str("cycle_id", cycleID).Str("resource", "project_settings").Logger()
	settings, err := inf.fetchAllProjectSettings(ctx, cycleID)
	if err != nil {
		return err
	}

	var added, modified, deleted int
	currentIDs := make(map[string]bool)
	for _, ps := range settings {
		id := ps.GetId()
		currentIDs[id] = true

		if existing, found := inf.projectSettingsCache[id]; found {
			if ps.GetUpdatedAt() != existing.GetUpdatedAt() {
				inf.projectSettingsCache[id] = ps
				inf.dispatch(ctx, ResourceEvent{
					Type:      EventModified,
					Resource:  "project_settings",
					Object:    ps,
					OldObject: existing,
				})
				modified++
			}
		} else {
			inf.projectSettingsCache[id] = ps
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventAdded,
				Resource: "project_settings",
				Object:   ps,
			})
			added++
		}
	}

	for id, ps := range inf.projectSettingsCache {
		if !currentIDs[id] {
			delete(inf.projectSettingsCache, id)
			inf.dispatch(ctx, ResourceEvent{
				Type:     EventDeleted,
				Resource: "project_settings",
				Object:   ps,
			})
			deleted++
		}
	}

	log.Debug().
		Int("fetched", len(settings)).
		Int("cached", len(inf.projectSettingsCache)).
		Int("added", added).Int("modified", modified).Int("deleted", deleted).
		Msg("sync complete")
	return nil
}

func (inf *Informer) dispatch(ctx context.Context, event ResourceEvent) {
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
