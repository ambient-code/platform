package agentMessages

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = agentMessageHandler{}

type agentMessageHandler struct {
	agentMessage AgentMessageService
	generic      services.GenericService
}

func NewAgentMessageHandler(agentMessage AgentMessageService, generic services.GenericService) *agentMessageHandler {
	return &agentMessageHandler{
		agentMessage: agentMessage,
		generic:      generic,
	}
}

func (h agentMessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var agentMessage openapi.AgentMessage
	cfg := &handlers.HandlerConfig{
		Body: &agentMessage,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&agentMessage, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			agentMessageModel := ConvertAgentMessage(agentMessage)
			agentMessageModel, err := h.agentMessage.Create(ctx, agentMessageModel)
			if err != nil {
				return nil, err
			}
			return PresentAgentMessage(agentMessageModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h agentMessageHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.AgentMessagePatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.agentMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.RecipientAgentId != nil {
				found.RecipientAgentId = *patch.RecipientAgentId
			}
			if patch.SenderAgentId != nil {
				found.SenderAgentId = patch.SenderAgentId
			}
			if patch.SenderUserId != nil {
				found.SenderUserId = patch.SenderUserId
			}
			if patch.SenderName != nil {
				found.SenderName = patch.SenderName
			}
			if patch.Body != nil {
				found.Body = patch.Body
			}
			if patch.Read != nil {
				found.Read = patch.Read
			}

			agentMessageModel, err := h.agentMessage.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentAgentMessage(agentMessageModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h agentMessageHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var agentMessages []AgentMessage
			paging, err := h.generic.List(ctx, "id", listArgs, &agentMessages)
			if err != nil {
				return nil, err
			}
			agentMessageList := openapi.AgentMessageList{
				Kind:  "AgentMessageList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.AgentMessage{},
			}

			for _, agentMessage := range agentMessages {
				converted := PresentAgentMessage(&agentMessage)
				agentMessageList.Items = append(agentMessageList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, agentMessageList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return agentMessageList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h agentMessageHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			agentMessage, err := h.agentMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentAgentMessage(agentMessage), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h agentMessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.agentMessage.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
