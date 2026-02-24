package sessions

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var safeProjectIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var _ handlers.RestHandler = sessionHandler{}

type sessionHandler struct {
	session SessionService
	generic services.GenericService
}

func NewSessionHandler(session SessionService, generic services.GenericService) *sessionHandler {
	return &sessionHandler{
		session: session,
		generic: generic,
	}
}

func (h sessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var session openapi.Session
	cfg := &handlers.HandlerConfig{
		Body: &session,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&session, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sessionModel := ConvertSession(session)
			if username := auth.GetUsernameFromContext(ctx); username != "" {
				sessionModel.CreatedByUserId = &username
			}
			sessionModel, err := h.session.Create(ctx, sessionModel)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h sessionHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.SessionPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.RepoUrl != nil {
				found.RepoUrl = patch.RepoUrl
			}
			if patch.Prompt != nil {
				found.Prompt = patch.Prompt
			}
			if patch.AssignedUserId != nil {
				found.AssignedUserId = patch.AssignedUserId
			}
			if patch.WorkflowId != nil {
				found.WorkflowId = patch.WorkflowId
			}
			if patch.Repos != nil {
				found.Repos = patch.Repos
			}
			if patch.Timeout != nil {
				found.Timeout = patch.Timeout
			}
			if patch.LlmModel != nil {
				found.LlmModel = patch.LlmModel
			}
			if patch.LlmTemperature != nil {
				found.LlmTemperature = patch.LlmTemperature
			}
			if patch.LlmMaxTokens != nil {
				found.LlmMaxTokens = patch.LlmMaxTokens
			}
			if patch.ParentSessionId != nil {
				found.ParentSessionId = patch.ParentSessionId
			}
			if patch.BotAccountName != nil {
				found.BotAccountName = patch.BotAccountName
			}
			if patch.ResourceOverrides != nil {
				found.ResourceOverrides = patch.ResourceOverrides
			}
			if patch.EnvironmentVariables != nil {
				found.EnvironmentVariables = patch.EnvironmentVariables
			}
			if patch.Labels != nil {
				found.SessionLabels = patch.Labels
			}
			if patch.Annotations != nil {
				found.SessionAnnotations = patch.Annotations
			}

			sessionModel, err := h.session.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h sessionHandler) PatchStatus(w http.ResponseWriter, r *http.Request) {
	var patch SessionStatusPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.UpdateStatus(ctx, id, &patch)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h sessionHandler) Start(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.Start(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) Stop(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.Stop(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			if projectID := r.URL.Query().Get("project_id"); projectID != "" {
				if !safeProjectIDPattern.MatchString(projectID) {
					return nil, errors.Validation("invalid project_id format")
				}
				projectFilter := fmt.Sprintf("project_id = '%s'", projectID)
				if listArgs.Search != "" {
					listArgs.Search = fmt.Sprintf("%s and (%s)", projectFilter, listArgs.Search)
				} else {
					listArgs.Search = projectFilter
				}
			}
			var sessions []Session
			paging, err := h.generic.List(ctx, "id", listArgs, &sessions)
			if err != nil {
				return nil, err
			}
			sessionList := openapi.SessionList{
				Kind:  "SessionList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Session{},
			}

			for _, session := range sessions {
				converted := PresentSession(&session)
				sessionList.Items = append(sessionList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, sessionList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return sessionList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h sessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentSession(session), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.session.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
