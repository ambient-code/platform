package repoEvents

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/common"
)

type repoEventHandler struct {
	service RepoEventService
	generic services.GenericService
}

func NewRepoEventHandler(svc RepoEventService, generic services.GenericService) *repoEventHandler {
	return &repoEventHandler{
		service: svc,
		generic: generic,
	}
}

func (h repoEventHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())

			if serr := common.ApplyProjectScope(r, listArgs); serr != nil {
				return nil, serr
			}

			var items []RepoEvent
			paging, err := h.generic.List(ctx, "id", listArgs, &items)
			if err != nil {
				return nil, err
			}

			list := RepoEventListAPI{
				Kind:  "RepoEventList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []RepoEventAPI{},
			}
			for _, item := range items {
				list.Items = append(list.Items, PresentRepoEvent(&item))
			}
			return list, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h repoEventHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			re, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentRepoEvent(re), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
