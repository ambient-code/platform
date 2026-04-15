package repoFindings

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type repoFindingHandler struct {
	service RepoFindingService
	generic services.GenericService
}

func NewRepoFindingHandler(svc RepoFindingService, generic services.GenericService) *repoFindingHandler {
	return &repoFindingHandler{
		service: svc,
		generic: generic,
	}
}

func (h repoFindingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body RepoFindingAPI
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&body, "ID", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			model := ConvertRepoFinding(body)
			model, err := h.service.Create(ctx, model)
			if err != nil {
				return nil, err
			}
			return PresentRepoFinding(model), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h repoFindingHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch RepoFindingPatchRequest
	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Status != nil {
				found.Status = *patch.Status
			}
			if patch.Severity != nil {
				found.Severity = patch.Severity
			}
			if patch.Title != nil {
				found.Title = *patch.Title
			}
			if patch.Body != nil {
				found.Body = *patch.Body
			}
			if patch.ResolvedBy != nil {
				found.ResolvedBy = patch.ResolvedBy
			}
			if patch.ResolvedReason != nil {
				found.ResolvedReason = patch.ResolvedReason
			}

			model, err := h.service.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRepoFinding(model), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h repoFindingHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())

			// Findings must be scoped to an intelligence record to prevent
			// cross-tenant data leaks (RepoFinding has no project_id column).
			if !strings.Contains(listArgs.Search, "intelligence_id") {
				return nil, errors.Validation("intelligence_id filter is required (findings must be scoped to an intelligence record)")
			}

			var items []RepoFinding
			paging, err := h.generic.List(ctx, "id", listArgs, &items)
			if err != nil {
				return nil, err
			}

			list := RepoFindingListAPI{
				Kind:  "RepoFindingList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []RepoFindingAPI{},
			}
			for _, item := range items {
				list.Items = append(list.Items, PresentRepoFinding(&item))
			}
			return list, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h repoFindingHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			rf, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentRepoFinding(rf), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h repoFindingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
