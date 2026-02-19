package projects

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = projectHandler{}

type projectHandler struct {
	project ProjectService
	generic services.GenericService
}

func NewProjectHandler(project ProjectService, generic services.GenericService) *projectHandler {
	return &projectHandler{
		project: project,
		generic: generic,
	}
}

func (h projectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var project openapi.Project
	cfg := &handlers.HandlerConfig{
		Body: &project,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&project, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectModel := ConvertProject(project)
			projectModel, err := h.project.Create(ctx, projectModel)
			if err != nil {
				return nil, err
			}
			return PresentProject(projectModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h projectHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.ProjectPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.project.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.DisplayName != nil {
				found.DisplayName = patch.DisplayName
			}
			if patch.Description != nil {
				found.Description = patch.Description
			}
			if patch.Labels != nil {
				found.Labels = patch.Labels
			}
			if patch.Annotations != nil {
				found.Annotations = patch.Annotations
			}
			if patch.Status != nil {
				found.Status = patch.Status
			}

			projectModel, err := h.project.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentProject(projectModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h projectHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var projects []Project
			paging, err := h.generic.List(ctx, "id", listArgs, &projects)
			if err != nil {
				return nil, err
			}
			projectList := openapi.ProjectList{
				Kind:  "ProjectList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Project{},
			}

			for _, project := range projects {
				converted := PresentProject(&project)
				projectList.Items = append(projectList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, projectList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return projectList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h projectHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			project, err := h.project.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentProject(project), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h projectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.project.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
