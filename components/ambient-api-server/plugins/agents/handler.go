package agents

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = agentHandler{}

type agentHandler struct {
	agent   AgentService
	generic services.GenericService
}

func NewAgentHandler(agent AgentService, generic services.GenericService) *agentHandler {
	return &agentHandler{
		agent:   agent,
		generic: generic,
	}
}

func (h agentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var agent openapi.Agent
	cfg := &handlers.HandlerConfig{
		Body: &agent,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&agent, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			agentModel := ConvertAgent(agent)
			agentModel, err := h.agent.Create(ctx, agentModel)
			if err != nil {
				return nil, err
			}
			return PresentAgent(agentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h agentHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.AgentPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.agent.Get(ctx, id)
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
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}

			agentModel, err := h.agent.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentAgent(agentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h agentHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var agents []Agent
			paging, err := h.generic.List(ctx, "id", listArgs, &agents)
			if err != nil {
				return nil, err
			}
			agentList := openapi.AgentList{
				Kind:  "AgentList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Agent{},
			}

			for _, agent := range agents {
				converted := PresentAgent(&agent)
				agentList.Items = append(agentList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, agentList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return agentList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h agentHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			agent, err := h.agent.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentAgent(agent), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h agentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.agent.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
