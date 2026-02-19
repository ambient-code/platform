package workflows

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = workflowHandler{}

type workflowHandler struct {
	workflow WorkflowService
	generic  services.GenericService
}

func NewWorkflowHandler(workflow WorkflowService, generic services.GenericService) *workflowHandler {
	return &workflowHandler{
		workflow: workflow,
		generic:  generic,
	}
}

func (h workflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	var workflow openapi.Workflow
	cfg := &handlers.HandlerConfig{
		Body: &workflow,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&workflow, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			workflowModel := ConvertWorkflow(workflow)
			workflowModel, err := h.workflow.Create(ctx, workflowModel)
			if err != nil {
				return nil, err
			}
			return PresentWorkflow(workflowModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h workflowHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.WorkflowPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.workflow.Get(ctx, id)
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
			if patch.AgentId != nil {
				found.AgentId = patch.AgentId
			}
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}
			if patch.Branch != nil {
				found.Branch = patch.Branch
			}
			if patch.Path != nil {
				found.Path = patch.Path
			}

			workflowModel, err := h.workflow.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentWorkflow(workflowModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h workflowHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var workflows []Workflow
			paging, err := h.generic.List(ctx, "id", listArgs, &workflows)
			if err != nil {
				return nil, err
			}
			workflowList := openapi.WorkflowList{
				Kind:  "WorkflowList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Workflow{},
			}

			for _, workflow := range workflows {
				converted := PresentWorkflow(&workflow)
				workflowList.Items = append(workflowList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, workflowList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return workflowList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h workflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			workflow, err := h.workflow.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentWorkflow(workflow), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h workflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.workflow.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
