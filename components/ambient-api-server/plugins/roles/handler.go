package roles

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = roleHandler{}

type roleHandler struct {
	role    RoleService
	generic services.GenericService
}

func NewRoleHandler(role RoleService, generic services.GenericService) *roleHandler {
	return &roleHandler{
		role:    role,
		generic: generic,
	}
}

func (h roleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var role openapi.Role
	cfg := &handlers.HandlerConfig{
		Body: &role,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&role, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			roleModel := ConvertRole(role)
			roleModel, err := h.role.Create(ctx, roleModel)
			if err != nil {
				return nil, err
			}
			return PresentRole(roleModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h roleHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.RolePatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.role.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.DisplayName != nil {
				found.DisplayName = patch.DisplayName
			}
			if patch.Description != nil {
				found.Description = patch.Description
			}
			if patch.Permissions != nil {
				_ = json.Unmarshal([]byte(*patch.Permissions), &found.Permissions)
			}

			roleModel, err := h.role.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRole(roleModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h roleHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var roles []Role
			paging, err := h.generic.List(ctx, "id", listArgs, &roles)
			if err != nil {
				return nil, err
			}
			roleList := openapi.RoleList{
				Kind:  "RoleList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Role{},
			}

			for _, role := range roles {
				converted := PresentRole(&role)
				roleList.Items = append(roleList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, roleList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return roleList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h roleHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			role, err := h.role.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentRole(role), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h roleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.role.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
