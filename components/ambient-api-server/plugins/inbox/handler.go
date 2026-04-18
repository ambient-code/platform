package inbox

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

type AgentOwnershipChecker interface {
	VerifyAgentProject(ctx context.Context, agentID, projectID string) *errors.ServiceError
}

var _ handlers.RestHandler = inboxMessageHandler{}

type inboxMessageHandler struct {
	inboxMessage   InboxMessageService
	generic        services.GenericService
	agentOwnership AgentOwnershipChecker
}

func NewInboxMessageHandler(inboxMessage InboxMessageService, generic services.GenericService, agentOwnership AgentOwnershipChecker) *inboxMessageHandler {
	return &inboxMessageHandler{
		inboxMessage:   inboxMessage,
		generic:        generic,
		agentOwnership: agentOwnership,
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
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]
			if svcErr := h.agentOwnership.VerifyAgentProject(ctx, agentID, projectID); svcErr != nil {
				return nil, svcErr
			}
			inboxMessage.AgentId = agentID
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
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]
			if svcErr := h.agentOwnership.VerifyAgentProject(ctx, agentID, projectID); svcErr != nil {
				return nil, svcErr
			}
			id := mux.Vars(r)["msg_id"]
			found, err := h.inboxMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if found.AgentId != agentID {
				return nil, errors.Forbidden("message does not belong to this agent")
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
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]

			if !validIDPattern.MatchString(agentID) {
				return nil, errors.Validation("invalid agent id")
			}

			if svcErr := h.agentOwnership.VerifyAgentProject(ctx, agentID, projectID); svcErr != nil {
				return nil, svcErr
			}

			listArgs := services.NewListArguments(r.URL.Query())
			if agentID != "" {
				agentFilter := "agent_id = '" + agentID + "'"
				if listArgs.Search == "" {
					listArgs.Search = agentFilter
				} else {
					listArgs.Search = agentFilter + " and (" + listArgs.Search + ")"
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
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]
			if svcErr := h.agentOwnership.VerifyAgentProject(ctx, agentID, projectID); svcErr != nil {
				return nil, svcErr
			}
			id := mux.Vars(r)["msg_id"]
			inboxMessage, err := h.inboxMessage.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if inboxMessage.AgentId != agentID {
				return nil, errors.Forbidden("message does not belong to this agent")
			}

			return PresentInboxMessage(inboxMessage), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h inboxMessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]
			if svcErr := h.agentOwnership.VerifyAgentProject(ctx, agentID, projectID); svcErr != nil {
				return nil, svcErr
			}
			id := mux.Vars(r)["msg_id"]
			found, getErr := h.inboxMessage.Get(ctx, id)
			if getErr != nil {
				return nil, getErr
			}
			if found.AgentId != agentID {
				return nil, errors.Forbidden("message does not belong to this agent")
			}
			err := h.inboxMessage.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
