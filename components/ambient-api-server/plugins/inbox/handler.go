package inbox

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = inboxMessageHandler{}

type inboxMessageHandler struct {
	inboxMessage InboxMessageService
	generic      services.GenericService
}

func NewInboxMessageHandler(inboxMessage InboxMessageService, generic services.GenericService) *inboxMessageHandler {
	return &inboxMessageHandler{
		inboxMessage: inboxMessage,
		generic:      generic,
	}
}

func (h inboxMessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var inboxMessage openapi.InboxMessage
	cfg := &handlers.HandlerConfig{
		Body: &inboxMessage,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&inboxMessage, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			inboxMessage.AgentId = mux.Vars(r)["agent_id"]
			inboxMessageModel := ConvertInboxMessage(inboxMessage)
			inboxMessageModel, err := h.inboxMessage.Create(ctx, inboxMessageModel)
			if err != nil {
				return nil, err
			}
			return PresentInboxMessage(inboxMessageModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h inboxMessageHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.InboxMessagePatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["msg_id"]
			found, err := h.inboxMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Read != nil {
				found.Read = patch.Read
			}

			inboxMessageModel, err := h.inboxMessage.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentInboxMessage(inboxMessageModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h inboxMessageHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			agentID := mux.Vars(r)["agent_id"]

			listArgs := services.NewListArguments(r.URL.Query())
			if agentID != "" {
				if listArgs.Search == "" {
					listArgs.Search = "agent_id = '" + agentID + "'"
				} else {
					listArgs.Search = listArgs.Search + " and agent_id = '" + agentID + "'"
				}
			}
			var inboxMessages []InboxMessage
			paging, err := h.generic.List(ctx, "id", listArgs, &inboxMessages)
			if err != nil {
				return nil, err
			}
			inboxMessageList := openapi.InboxMessageList{
				Kind:  "InboxMessageList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.InboxMessage{},
			}

			for _, inboxMessage := range inboxMessages {
				converted := PresentInboxMessage(&inboxMessage)
				inboxMessageList.Items = append(inboxMessageList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, inboxMessageList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return inboxMessageList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h inboxMessageHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["msg_id"]
			ctx := r.Context()
			inboxMessage, err := h.inboxMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentInboxMessage(inboxMessage), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h inboxMessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["msg_id"]
			ctx := r.Context()
			err := h.inboxMessage.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
