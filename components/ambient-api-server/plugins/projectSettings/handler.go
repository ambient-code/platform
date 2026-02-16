package projectSettings

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = projectSettingsHandler{}

type projectSettingsHandler struct {
	projectSettings ProjectSettingsService
	generic         services.GenericService
}

func NewProjectSettingsHandler(ps ProjectSettingsService, generic services.GenericService) *projectSettingsHandler {
	return &projectSettingsHandler{
		projectSettings: ps,
		generic:         generic,
	}
}

func (h projectSettingsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var ps openapi.ProjectSettings
	cfg := &handlers.HandlerConfig{
		Body: &ps,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&ps, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			psModel := ConvertProjectSettings(ps)
			psModel, err := h.projectSettings.Create(ctx, psModel)
			if err != nil {
				return nil, err
			}
			return PresentProjectSettings(psModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h projectSettingsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.ProjectSettingsPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.projectSettings.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.ProjectId != nil {
				found.ProjectId = *patch.ProjectId
			}
			if patch.GroupAccess != nil {
				found.GroupAccess = patch.GroupAccess
			}
			if patch.RunnerSecrets != nil {
				found.RunnerSecrets = patch.RunnerSecrets
			}
			if patch.Repositories != nil {
				found.Repositories = patch.Repositories
			}

			psModel, err := h.projectSettings.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentProjectSettings(psModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h projectSettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var items []ProjectSettings
			paging, err := h.generic.List(ctx, "id", listArgs, &items)
			if err != nil {
				return nil, err
			}
			list := openapi.ProjectSettingsList{
				Kind:  "ProjectSettingsList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.ProjectSettings{},
			}

			for _, item := range items {
				converted := PresentProjectSettings(&item)
				list.Items = append(list.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, list.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return list, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h projectSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			ps, err := h.projectSettings.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentProjectSettings(ps), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h projectSettingsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.projectSettings.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
