package tasks

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = taskHandler{}

type taskHandler struct {
	task    TaskService
	generic services.GenericService
}

func NewTaskHandler(task TaskService, generic services.GenericService) *taskHandler {
	return &taskHandler{
		task:    task,
		generic: generic,
	}
}

func (h taskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var task openapi.Task
	cfg := &handlers.HandlerConfig{
		Body: &task,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&task, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			taskModel := ConvertTask(task)
			taskModel, err := h.task.Create(ctx, taskModel)
			if err != nil {
				return nil, err
			}
			return PresentTask(taskModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h taskHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.TaskPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.task.Get(ctx, id)
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

			taskModel, err := h.task.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentTask(taskModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h taskHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var tasks []Task
			paging, err := h.generic.List(ctx, "id", listArgs, &tasks)
			if err != nil {
				return nil, err
			}
			taskList := openapi.TaskList{
				Kind:  "TaskList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Task{},
			}

			for _, task := range tasks {
				converted := PresentTask(&task)
				taskList.Items = append(taskList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, taskList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return taskList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h taskHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			task, err := h.task.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentTask(task), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h taskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.task.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
