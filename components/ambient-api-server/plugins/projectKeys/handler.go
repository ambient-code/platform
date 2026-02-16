package projectKeys

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type projectKeyHandler struct {
	projectKey ProjectKeyService
	generic    services.GenericService
}

func NewProjectKeyHandler(projectKey ProjectKeyService, generic services.GenericService) *projectKeyHandler {
	return &projectKeyHandler{
		projectKey: projectKey,
		generic:    generic,
	}
}

func (h projectKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var projectKey openapi.ProjectKey
	cfg := &handlers.HandlerConfig{
		Body: &projectKey,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&projectKey, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectKeyModel := ConvertProjectKey(projectKey)
			projectKeyModel, err := h.projectKey.Create(ctx, projectKeyModel)
			if err != nil {
				return nil, err
			}
			return PresentProjectKey(projectKeyModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h projectKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var projectKeys []ProjectKey
			paging, err := h.generic.List(ctx, "id", listArgs, &projectKeys)
			if err != nil {
				return nil, err
			}
			projectKeyList := openapi.ProjectKeyList{
				Kind:  "ProjectKeyList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.ProjectKey{},
			}

			for _, projectKey := range projectKeys {
				converted := PresentProjectKey(&projectKey)
				projectKeyList.Items = append(projectKeyList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, projectKeyList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return projectKeyList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h projectKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			projectKey, err := h.projectKey.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentProjectKey(projectKey), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h projectKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.projectKey.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
