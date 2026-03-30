package projects

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agents"
	pkgerrors "github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
)

type homeHandler struct {
	agentSvc agents.AgentService
}

func NewHomeHandler(agentSvc agents.AgentService) *homeHandler {
	return &homeHandler{agentSvc: agentSvc}
}

func (h *homeHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]

			agentList, err := h.agentSvc.AllByProjectID(ctx, projectID)
			if err != nil {
				return nil, err
			}

			result := openapi.AgentList{
				Kind:  "AgentList",
				Page:  1,
				Size:  int32(len(agentList)),
				Total: int32(len(agentList)),
				Items: []openapi.Agent{},
			}
			for _, a := range agentList {
				result.Items = append(result.Items, agents.PresentAgent(a))
			}
			return result, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}
