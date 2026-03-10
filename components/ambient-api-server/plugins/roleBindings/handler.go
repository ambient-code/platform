package roleBindings

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = roleBindingHandler{}

type roleBindingHandler struct {
	roleBinding RoleBindingService
	generic     services.GenericService
}

func NewRoleBindingHandler(roleBinding RoleBindingService, generic services.GenericService) *roleBindingHandler {
	return &roleBindingHandler{
		roleBinding: roleBinding,
		generic:     generic,
	}
}

func (h roleBindingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var roleBinding openapi.RoleBinding
	cfg := &handlers.HandlerConfig{
		Body: &roleBinding,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&roleBinding, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			roleBindingModel := ConvertRoleBinding(roleBinding)
			roleBindingModel, err := h.roleBinding.Create(ctx, roleBindingModel)
			if err != nil {
				return nil, err
			}
			return PresentRoleBinding(roleBindingModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h roleBindingHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.RoleBindingPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.roleBinding.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.UserId != nil {
				found.UserId = *patch.UserId
			}
			if patch.RoleId != nil {
				found.RoleId = *patch.RoleId
			}
			if patch.Scope != nil {
				found.Scope = *patch.Scope
			}
			if patch.ScopeId != nil {
				found.ScopeId = patch.ScopeId
			}

			roleBindingModel, err := h.roleBinding.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRoleBinding(roleBindingModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h roleBindingHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var roleBindings []RoleBinding
			paging, err := h.generic.List(ctx, "id", listArgs, &roleBindings)
			if err != nil {
				return nil, err
			}
			roleBindingList := openapi.RoleBindingList{
				Kind:  "RoleBindingList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.RoleBinding{},
			}

			for _, roleBinding := range roleBindings {
				converted := PresentRoleBinding(&roleBinding)
				roleBindingList.Items = append(roleBindingList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, roleBindingList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return roleBindingList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h roleBindingHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			roleBinding, err := h.roleBinding.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentRoleBinding(roleBinding), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h roleBindingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.roleBinding.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
