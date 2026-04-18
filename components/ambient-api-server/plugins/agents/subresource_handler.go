package agents

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/roleBindings"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
	pkgerrors "github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type agentSubresourceHandler struct {
	agent       AgentService
	session     sessions.SessionService
	genericSvc  services.GenericService
	roleBinding roleBindings.RoleBindingService
}

func NewAgentSubresourceHandler(
	agent AgentService,
	session sessions.SessionService,
	generic services.GenericService,
	roleBinding roleBindings.RoleBindingService,
) *agentSubresourceHandler {
	return &agentSubresourceHandler{
		agent:       agent,
		session:     session,
		genericSvc:  generic,
		roleBinding: roleBinding,
	}
}

func (h *agentSubresourceHandler) ListRoleBindings(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]

			if !validIDPattern.MatchString(agentID) {
				return nil, pkgerrors.Validation("invalid agent id")
			}

			agent, err := h.agent.Get(ctx, agentID)
			if err != nil {
				return nil, err
			}

			if agent.ProjectId != projectID {
				return nil, pkgerrors.Forbidden("agent does not belong to this project")
			}

			listArgs := services.NewListArguments(r.URL.Query())
			scopeFilter := fmt.Sprintf("scope_id = '%s'", agentID)
			if listArgs.Search != "" {
				listArgs.Search = scopeFilter + " and (" + listArgs.Search + ")"
			} else {
				listArgs.Search = scopeFilter
			}

			var rbList []roleBindings.RoleBinding
			paging, err := h.genericSvc.List(ctx, "id", listArgs, &rbList)
			if err != nil {
				return nil, err
			}

			result := openapi.RoleBindingList{
				Kind:  "RoleBindingList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.RoleBinding{},
			}
			for i := range rbList {
				result.Items = append(result.Items, roleBindings.PresentRoleBinding(&rbList[i]))
			}
			return result, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h *agentSubresourceHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]

			if !validIDPattern.MatchString(agentID) {
				return nil, pkgerrors.Validation("invalid agent id")
			}

			agent, err := h.agent.Get(ctx, agentID)
			if err != nil {
				return nil, err
			}

			if agent.ProjectId != projectID {
				return nil, pkgerrors.Forbidden("agent does not belong to this project")
			}

			listArgs := services.NewListArguments(r.URL.Query())
			agentFilter := fmt.Sprintf("agent_id = '%s'", agentID)
			if listArgs.Search != "" {
				listArgs.Search = agentFilter + " and (" + listArgs.Search + ")"
			} else {
				listArgs.Search = agentFilter
			}

			var sessList []sessions.Session
			paging, err := h.genericSvc.List(ctx, "id", listArgs, &sessList)
			if err != nil {
				return nil, err
			}

			result := openapi.SessionList{
				Kind:  "SessionList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Session{},
			}
			for i := range sessList {
				result.Items = append(result.Items, sessions.PresentSession(&sessList[i]))
			}
			return result, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}
