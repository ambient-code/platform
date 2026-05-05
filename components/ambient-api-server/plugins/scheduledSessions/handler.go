package scheduledSessions

import (
	"encoding/json"
	"net/http"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/gorilla/mux"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
)

type scheduledSessionHandler struct {
	svc ScheduledSessionService
}

func NewScheduledSessionHandler(svc ScheduledSessionService) *scheduledSessionHandler {
	return &scheduledSessionHandler{svc: svc}
}

// List — GET /api/ambient/v1/projects/{project_id}/scheduled-sessions
func (h *scheduledSessionHandler) List(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			list, err := h.svc.ListByProject(ctx, projectId)
			if err != nil {
				return nil, err
			}
			result := openapi.ScheduledSessionList{
				Kind:  "ScheduledSessionList",
				Page:  1,
				Size:  int32(len(list)),
				Total: int32(len(list)),
				Items: make([]openapi.ScheduledSession, 0, len(list)),
			}
			for _, ss := range list {
				result.Items = append(result.Items, PresentScheduledSession(ss))
			}
			return result, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

// Get — GET /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}
func (h *scheduledSessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ss, err := h.svc.Get(r.Context(), id)
			if err != nil {
				return nil, err
			}
			return PresentScheduledSession(ss), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Create — POST /api/ambient/v1/projects/{project_id}/scheduled-sessions
func (h *scheduledSessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	var body openapi.ScheduledSession
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&body, "Id", "id"),
			func() *errors.ServiceError {
				if body.Name == "" {
					return errors.Validation("name is required")
				}
				if body.Schedule == "" {
					return errors.Validation("schedule is required")
				}
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			body.ProjectId = projectId
			ss := ConvertScheduledSession(body)
			created, err := h.svc.Create(r.Context(), ss)
			if err != nil {
				return nil, err
			}
			return PresentScheduledSession(created), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

// Patch — PATCH /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}
func (h *scheduledSessionHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var body openapi.ScheduledSessionPatchRequest
	cfg := &handlers.HandlerConfig{
		Body:       &body,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			patch := &ScheduledSessionPatch{
				Name:              body.Name,
				Description:       body.Description,
				AgentId:           body.AgentId,
				Schedule:          body.Schedule,
				Timezone:          body.Timezone,
				Enabled:           body.Enabled,
				SessionPrompt:     body.SessionPrompt,
				Timeout:           body.Timeout,
				InactivityTimeout: body.InactivityTimeout,
				StopOnRunFinished: body.StopOnRunFinished,
				RunnerType:        body.RunnerType,
			}
			updated, err := h.svc.Patch(r.Context(), id, patch)
			if err != nil {
				return nil, err
			}
			return PresentScheduledSession(updated), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

// Delete — DELETE /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}
func (h *scheduledSessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			if err := h.svc.Delete(r.Context(), id); err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

// Suspend — POST /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}/suspend
func (h *scheduledSessionHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ss, err := h.svc.Suspend(r.Context(), id)
			if err != nil {
				return nil, err
			}
			return PresentScheduledSession(ss), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Resume — POST /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}/resume
func (h *scheduledSessionHandler) Resume(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ss, err := h.svc.Resume(r.Context(), id)
			if err != nil {
				return nil, err
			}
			return PresentScheduledSession(ss), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Trigger — POST /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}/trigger
func (h *scheduledSessionHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			if err := h.svc.Trigger(r.Context(), id); err != nil {
				return nil, err
			}
			return map[string]string{"status": "triggered"}, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Runs — GET /api/ambient/v1/projects/{project_id}/scheduled-sessions/{id}/runs
func (h *scheduledSessionHandler) Runs(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			// Returns sessions triggered by this scheduled session.
			// In production, query sessions where source_scheduled_session_id = id.
			// For now, return empty list so the endpoint works and is testable.
			return map[string]interface{}{
				"kind":  "SessionList",
				"page":  1,
				"size":  0,
				"total": 0,
				"items": []interface{}{},
			}, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

// writeJSON is a helper for action endpoints that don't use the handler framework.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
