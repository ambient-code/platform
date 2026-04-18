package agents

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

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
			projectID := mux.Vars(r)["id"]
			agentModel := ConvertAgent(agent)
			agentModel.ProjectId = projectID
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
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["agent_id"]
			found, err := h.agent.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if found.ProjectId != projectID {
				return nil, errors.Forbidden("agent does not belong to this project")
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.Prompt != nil {
				found.Prompt = patch.Prompt
			}
			if patch.Labels != nil {
				found.Labels = patch.Labels
			}
			if patch.Annotations != nil {
				found.Annotations = patch.Annotations
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
			projectID := mux.Vars(r)["id"]

			if !validIDPattern.MatchString(projectID) {
				return nil, errors.Validation("invalid project id")
			}

			listArgs := services.NewListArguments(r.URL.Query())
			projectFilter := "project_id = '" + projectID + "'"
			if listArgs.Search != "" {
				listArgs.Search = projectFilter + " and (" + listArgs.Search + ")"
			} else {
				listArgs.Search = projectFilter
			}

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
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["agent_id"]
			ctx := r.Context()
			agent, err := h.agent.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if agent.ProjectId != projectID {
				return nil, errors.Forbidden("agent does not belong to this project")
			}

			return PresentAgent(agent), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h agentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["agent_id"]
			ctx := r.Context()
			agent, getErr := h.agent.Get(ctx, id)
			if getErr != nil {
				return nil, getErr
			}
			if agent.ProjectId != projectID {
				return nil, errors.Forbidden("agent does not belong to this project")
			}
			err := h.agent.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
