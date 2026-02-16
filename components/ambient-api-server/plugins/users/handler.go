package users

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = userHandler{}

type userHandler struct {
	user    UserService
	generic services.GenericService
}

func NewUserHandler(user UserService, generic services.GenericService) *userHandler {
	return &userHandler{
		user:    user,
		generic: generic,
	}
}

func (h userHandler) Create(w http.ResponseWriter, r *http.Request) {
	var user openapi.User
	cfg := &handlers.HandlerConfig{
		Body: &user,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&user, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			userModel := ConvertUser(user)
			userModel, err := h.user.Create(ctx, userModel)
			if err != nil {
				return nil, err
			}
			return PresentUser(userModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h userHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.UserPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.user.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Username != nil {
				found.Username = *patch.Username
			}
			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.Groups != nil {
				found.Groups = patch.Groups
			}

			userModel, err := h.user.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentUser(userModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h userHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var users []User
			paging, err := h.generic.List(ctx, "id", listArgs, &users)
			if err != nil {
				return nil, err
			}
			userList := openapi.UserList{
				Kind:  "UserList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.User{},
			}

			for _, user := range users {
				converted := PresentUser(&user)
				userList.Items = append(userList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, userList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return userList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h userHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			user, err := h.user.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentUser(user), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h userHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.user.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
