package permissions

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = permissionHandler{}

type permissionHandler struct {
	permission PermissionService
	generic    services.GenericService
}

func NewPermissionHandler(permission PermissionService, generic services.GenericService) *permissionHandler {
	return &permissionHandler{
		permission: permission,
		generic:    generic,
	}
}

func (h permissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var permission openapi.Permission
	cfg := &handlers.HandlerConfig{
		Body: &permission,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&permission, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			permissionModel := ConvertPermission(permission)
			permissionModel, err := h.permission.Create(ctx, permissionModel)
			if err != nil {
				return nil, err
			}
			return PresentPermission(permissionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h permissionHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.PermissionPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.permission.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.SubjectType != nil {
				found.SubjectType = *patch.SubjectType
			}
			if patch.SubjectName != nil {
				found.SubjectName = *patch.SubjectName
			}
			if patch.Role != nil {
				found.Role = *patch.Role
			}
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}

			permissionModel, err := h.permission.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentPermission(permissionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h permissionHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var permissions []Permission
			paging, err := h.generic.List(ctx, "id", listArgs, &permissions)
			if err != nil {
				return nil, err
			}
			permissionList := openapi.PermissionList{
				Kind:  "PermissionList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Permission{},
			}

			for _, permission := range permissions {
				converted := PresentPermission(&permission)
				permissionList.Items = append(permissionList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, permissionList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return permissionList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h permissionHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			permission, err := h.permission.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentPermission(permission), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h permissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.permission.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
