package projectAgents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/inbox"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	pkgerrors "github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
)

type IgniteResponse struct {
	Session        openapi.Session `json:"session"`
	IgnitionPrompt string          `json:"ignition_prompt"`
}

type paIgniteHandler struct {
	projectAgent ProjectAgentService
	agent        agents.AgentService
	inbox        inbox.InboxMessageService
	session      sessions.SessionService
	msg          sessions.MessageService
}

func NewPAIgniteHandler(
	pa ProjectAgentService,
	agent agents.AgentService,
	inboxSvc inbox.InboxMessageService,
	session sessions.SessionService,
	msg sessions.MessageService,
) *paIgniteHandler {
	return &paIgniteHandler{
		projectAgent: pa,
		agent:        agent,
		inbox:        inboxSvc,
		session:      session,
		msg:          msg,
	}
}

func (h *paIgniteHandler) Ignite(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			paID := mux.Vars(r)["pa_id"]

			pa, paErr := h.projectAgent.Get(ctx, paID)
			if paErr != nil {
				return nil, paErr
			}

			agent, agentErr := h.agent.Get(ctx, pa.AgentId)
			if agentErr != nil {
				return nil, agentErr
			}

			unread, inboxErr := h.inbox.UnreadByProjectAgentID(ctx, paID)
			if inboxErr != nil {
				return nil, inboxErr
			}

			var requestPrompt *string
			var body struct {
				Prompt string `json:"prompt"`
			}
			if r.ContentLength > 0 {
				if decErr := json.NewDecoder(r.Body).Decode(&body); decErr == nil && body.Prompt != "" {
					requestPrompt = &body.Prompt
				}
			}

			llmModel := agent.LlmModel
			llmTemp := agent.LlmTemperature
			llmTokens := agent.LlmMaxTokens

			sessName := fmt.Sprintf("%s-%d", agent.Name, time.Now().Unix())
			sess := &sessions.Session{
				Name:                 sessName,
				Prompt:               agent.Prompt,
				RepoUrl:              agent.RepoUrl,
				WorkflowId:           agent.WorkflowId,
				LlmModel:             &llmModel,
				LlmTemperature:       &llmTemp,
				LlmMaxTokens:         &llmTokens,
				BotAccountName:       agent.BotAccountName,
				ResourceOverrides:    agent.ResourceOverrides,
				EnvironmentVariables: agent.EnvironmentVariables,
				ProjectId:            &pa.ProjectId,
			}

			username := auth.GetUsernameFromContext(ctx)
			if username != "" {
				sess.CreatedByUserId = &username
			}

			created, sessErr := h.session.Create(ctx, sess)
			if sessErr != nil {
				return nil, sessErr
			}

			for _, msg := range unread {
				read := true
				msgCopy := *msg
				msgCopy.Read = &read
				if _, replErr := h.inbox.Replace(ctx, &msgCopy); replErr != nil {
					glog.Warningf("Ignite PA %s: mark inbox message %s read: %v", paID, msg.ID, replErr)
				}
			}

			peers, peersErr := h.projectAgent.AllByProjectID(ctx, pa.ProjectId)
			if peersErr != nil {
				return nil, peersErr
			}

			prompt := buildPAIgnitionPrompt(agent, pa, peers, unread, requestPrompt)

			if prompt != "" {
				if _, pushErr := h.msg.Push(ctx, created.ID, "user", prompt); pushErr != nil {
					glog.Errorf("Ignite PA %s: store ignition prompt for session %s: %v", paID, created.ID, pushErr)
				}
			}

			paCopy := *pa
			paCopy.CurrentSessionId = &created.ID
			if _, replErr := h.projectAgent.Replace(ctx, &paCopy); replErr != nil {
				return nil, replErr
			}

			if _, startErr := h.session.Start(ctx, created.ID); startErr != nil {
				return nil, startErr
			}

			return &IgniteResponse{
				Session:        sessions.PresentSession(created),
				IgnitionPrompt: prompt,
			}, nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func buildPAIgnitionPrompt(
	agent *agents.Agent,
	pa *ProjectAgent,
	peers ProjectAgentList,
	unread inbox.InboxMessageList,
	requestPrompt *string,
) string {
	var sb strings.Builder

	displayName := agent.Name
	if agent.DisplayName != nil && *agent.DisplayName != "" {
		displayName = *agent.DisplayName
	}

	fmt.Fprintf(&sb, "# Agent Ignition: %s\n\n", displayName)
	fmt.Fprintf(&sb, "You are **%s**, working in project **%s**.\n\n", displayName, pa.ProjectId)

	if agent.Description != nil && *agent.Description != "" {
		fmt.Fprintf(&sb, "## Role\n\n%s\n\n", *agent.Description)
	}

	if agent.Prompt != nil && *agent.Prompt != "" {
		fmt.Fprintf(&sb, "## Standing Instructions\n\n%s\n\n", *agent.Prompt)
	}

	if len(peers) > 1 {
		sb.WriteString("## Peer Agents in this Project\n\n")
		sb.WriteString("| Agent ID | Agent Version |\n")
		sb.WriteString("| -------- | ------------- |\n")
		for _, p := range peers {
			if p.ID == pa.ID {
				continue
			}
			ver := 0
			if p.AgentVersion != nil {
				ver = *p.AgentVersion
			}
			fmt.Fprintf(&sb, "| %s | v%d |\n", p.AgentId, ver)
		}
		sb.WriteString("\n")
	}

	if len(unread) > 0 {
		sb.WriteString("## Inbox Messages (unread at ignition)\n\n")
		for _, m := range unread {
			from := "system"
			if m.FromName != nil && *m.FromName != "" {
				from = *m.FromName
			} else if m.FromProjectAgentId != nil && *m.FromProjectAgentId != "" {
				from = *m.FromProjectAgentId
			}
			fmt.Fprintf(&sb, "**From %s:** %s\n\n", from, m.Body)
		}
	}

	if requestPrompt != nil && *requestPrompt != "" {
		fmt.Fprintf(&sb, "## Task for this Run\n\n%s\n\n", *requestPrompt)
	}

	return sb.String()
}
