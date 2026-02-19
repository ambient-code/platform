package workflowTasks

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = workflowTaskHandler{}

type workflowTaskHandler struct {
	workflowTask WorkflowTaskService
	generic      services.GenericService
}

func NewWorkflowTaskHandler(workflowTask WorkflowTaskService, generic services.GenericService) *workflowTaskHandler {
	return &workflowTaskHandler{
		workflowTask: workflowTask,
		generic:      generic,
	}
}

func (h workflowTaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var workflowTask openapi.WorkflowTask
	cfg := &handlers.HandlerConfig{
		Body: &workflowTask,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&workflowTask, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			workflowTaskModel := ConvertWorkflowTask(workflowTask)
			workflowTaskModel, err := h.workflowTask.Create(ctx, workflowTaskModel)
			if err != nil {
				return nil, err
			}
			return PresentWorkflowTask(workflowTaskModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h workflowTaskHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.WorkflowTaskPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.workflowTask.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.WorkflowId != nil {
				found.WorkflowId = *patch.WorkflowId
			}
			if patch.TaskId != nil {
				found.TaskId = *patch.TaskId
			}
			if patch.Position != nil {
				found.Position = int(*patch.Position)
			}

			workflowTaskModel, err := h.workflowTask.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentWorkflowTask(workflowTaskModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h workflowTaskHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var workflowTasks []WorkflowTask
			paging, err := h.generic.List(ctx, "id", listArgs, &workflowTasks)
			if err != nil {
				return nil, err
			}
			workflowTaskList := openapi.WorkflowTaskList{
				Kind:  "WorkflowTaskList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.WorkflowTask{},
			}

			for _, workflowTask := range workflowTasks {
				converted := PresentWorkflowTask(&workflowTask)
				workflowTaskList.Items = append(workflowTaskList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, workflowTaskList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return workflowTaskList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h workflowTaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			workflowTask, err := h.workflowTask.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentWorkflowTask(workflowTask), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h workflowTaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.workflowTask.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
