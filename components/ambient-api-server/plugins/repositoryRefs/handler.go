package repositoryRefs

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = repositoryRefHandler{}

type repositoryRefHandler struct {
	repositoryRef RepositoryRefService
	generic       services.GenericService
}

func NewRepositoryRefHandler(repositoryRef RepositoryRefService, generic services.GenericService) *repositoryRefHandler {
	return &repositoryRefHandler{
		repositoryRef: repositoryRef,
		generic:       generic,
	}
}

func (h repositoryRefHandler) Create(w http.ResponseWriter, r *http.Request) {
	var repositoryRef openapi.RepositoryRef
	cfg := &handlers.HandlerConfig{
		Body: &repositoryRef,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&repositoryRef, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			repositoryRefModel := ConvertRepositoryRef(repositoryRef)
			repositoryRefModel, err := h.repositoryRef.Create(ctx, repositoryRefModel)
			if err != nil {
				return nil, err
			}
			return PresentRepositoryRef(repositoryRefModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h repositoryRefHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.RepositoryRefPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.repositoryRef.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.Url != nil {
				found.Url = *patch.Url
			}
			if patch.Branch != nil {
				found.Branch = patch.Branch
			}
			if patch.Provider != nil {
				found.Provider = patch.Provider
			}
			if patch.Owner != nil {
				found.Owner = patch.Owner
			}
			if patch.RepoName != nil {
				found.RepoName = patch.RepoName
			}
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}

			repositoryRefModel, err := h.repositoryRef.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRepositoryRef(repositoryRefModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h repositoryRefHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var repositoryRefs []RepositoryRef
			paging, err := h.generic.List(ctx, "id", listArgs, &repositoryRefs)
			if err != nil {
				return nil, err
			}
			repositoryRefList := openapi.RepositoryRefList{
				Kind:  "RepositoryRefList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.RepositoryRef{},
			}

			for _, repositoryRef := range repositoryRefs {
				converted := PresentRepositoryRef(&repositoryRef)
				repositoryRefList.Items = append(repositoryRefList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, repositoryRefList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return repositoryRefList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h repositoryRefHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			repositoryRef, err := h.repositoryRef.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentRepositoryRef(repositoryRef), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h repositoryRefHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.repositoryRef.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
