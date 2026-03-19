package projectAgents

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = projectAgentHandler{}

type projectAgentHandler struct {
	projectAgent ProjectAgentService
	generic      services.GenericService
}

func NewProjectAgentHandler(projectAgent ProjectAgentService, generic services.GenericService) *projectAgentHandler {
	return &projectAgentHandler{
		projectAgent: projectAgent,
		generic:      generic,
	}
}

func (h projectAgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var projectAgent openapi.ProjectAgent
	cfg := &handlers.HandlerConfig{
		Body: &projectAgent,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&projectAgent, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectAgentModel := ConvertProjectAgent(projectAgent)
			projectAgentModel, err := h.projectAgent.Create(ctx, projectAgentModel)
			if err != nil {
				return nil, err
			}
			return PresentProjectAgent(projectAgentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h projectAgentHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.ProjectAgentPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.projectAgent.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.AgentVersion != nil {
				agentVersionVal := int(*patch.AgentVersion)
				found.AgentVersion = &agentVersionVal
			}

			projectAgentModel, err := h.projectAgent.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentProjectAgent(projectAgentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h projectAgentHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var projectAgents []ProjectAgent
			paging, err := h.generic.List(ctx, "id", listArgs, &projectAgents)
			if err != nil {
				return nil, err
			}
			projectAgentList := openapi.ProjectAgentList{
				Kind:  "ProjectAgentList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.ProjectAgent{},
			}

			for _, projectAgent := range projectAgents {
				converted := PresentProjectAgent(&projectAgent)
				projectAgentList.Items = append(projectAgentList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, projectAgentList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return projectAgentList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h projectAgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			projectAgent, err := h.projectAgent.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentProjectAgent(projectAgent), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h projectAgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.projectAgent.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
