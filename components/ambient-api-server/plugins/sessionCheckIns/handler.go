package sessionCheckIns

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = sessionCheckInHandler{}

type sessionCheckInHandler struct {
	sessionCheckIn SessionCheckInService
	generic        services.GenericService
}

func NewSessionCheckInHandler(sessionCheckIn SessionCheckInService, generic services.GenericService) *sessionCheckInHandler {
	return &sessionCheckInHandler{
		sessionCheckIn: sessionCheckIn,
		generic:        generic,
	}
}

func (h sessionCheckInHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sessionCheckIn openapi.SessionCheckIn
	cfg := &handlers.HandlerConfig{
		Body: &sessionCheckIn,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&sessionCheckIn, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sessionCheckInModel := ConvertSessionCheckIn(sessionCheckIn)
			sessionCheckInModel, err := h.sessionCheckIn.Create(ctx, sessionCheckInModel)
			if err != nil {
				return nil, err
			}
			return PresentSessionCheckIn(sessionCheckInModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h sessionCheckInHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.SessionCheckInPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.sessionCheckIn.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.SessionId != nil {
				found.SessionId = *patch.SessionId
			}
			if patch.AgentId != nil {
				found.AgentId = *patch.AgentId
			}
			if patch.Summary != nil {
				found.Summary = patch.Summary
			}
			if patch.Branch != nil {
				found.Branch = patch.Branch
			}
			if patch.Worktree != nil {
				found.Worktree = patch.Worktree
			}
			if patch.Pr != nil {
				found.Pr = patch.Pr
			}
			if patch.Phase != nil {
				found.Phase = patch.Phase
			}
			if patch.TestCount != nil {
				testCountVal := int(*patch.TestCount)
				found.TestCount = &testCountVal
			}
			if patch.NextSteps != nil {
				found.NextSteps = patch.NextSteps
			}

			sessionCheckInModel, err := h.sessionCheckIn.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentSessionCheckIn(sessionCheckInModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h sessionCheckInHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var sessionCheckIns []SessionCheckIn
			paging, err := h.generic.List(ctx, "id", listArgs, &sessionCheckIns)
			if err != nil {
				return nil, err
			}
			sessionCheckInList := openapi.SessionCheckInList{
				Kind:  "SessionCheckInList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.SessionCheckIn{},
			}

			for _, sessionCheckIn := range sessionCheckIns {
				converted := PresentSessionCheckIn(&sessionCheckIn)
				sessionCheckInList.Items = append(sessionCheckInList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, sessionCheckInList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return sessionCheckInList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h sessionCheckInHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			sessionCheckIn, err := h.sessionCheckIn.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentSessionCheckIn(sessionCheckIn), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionCheckInHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.sessionCheckIn.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
