package agents

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	pkgerrors "github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
)

type IgniteResponse struct {
	Session        openapi.Session `json:"session"`
	IgnitionPrompt string          `json:"ignition_prompt"`
}

type igniteHandler struct {
	agent   AgentService
	session sessions.SessionService
	msg     sessions.MessageService
}

func NewIgniteHandler(agent AgentService, session sessions.SessionService, msg sessions.MessageService) *igniteHandler {
	return &igniteHandler{
		agent:   agent,
		session: session,
		msg:     msg,
	}
}

func (h *igniteHandler) Ignite(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			agentID := mux.Vars(r)["id"]

			agent, err := h.agent.Get(ctx, agentID)
			if err != nil {
				return nil, err
			}

			username := auth.GetUsernameFromContext(ctx)

			llmModel := agent.LlmModel
			llmTemp := agent.LlmTemperature
			llmTokens := agent.LlmMaxTokens

			sess := &sessions.Session{
				Name:                 fmt.Sprintf("%s-%d", agent.Name, time.Now().Unix()),
				Prompt:               agent.Prompt,
				RepoUrl:              agent.RepoUrl,
				WorkflowId:           agent.WorkflowId,
				LlmModel:             &llmModel,
				LlmTemperature:       &llmTemp,
				LlmMaxTokens:         &llmTokens,
				BotAccountName:       agent.BotAccountName,
				ResourceOverrides:    agent.ResourceOverrides,
				EnvironmentVariables: agent.EnvironmentVariables,
				ProjectId:            &agent.ProjectId,
			}
			if username != "" {
				sess.CreatedByUserId = &username
			}

			created, serr := h.session.Create(ctx, sess)
			if serr != nil {
				return nil, serr
			}

			agentCopy := *agent
			agentCopy.CurrentSessionId = &created.ID
			if _, rerr := h.agent.Replace(ctx, &agentCopy); rerr != nil {
				return nil, rerr
			}

			peers, perr := h.agent.AllByProjectID(ctx, agent.ProjectId)
			if perr != nil {
				return nil, perr
			}

			prompt := buildIgnitionPrompt(agent, peers)

			if prompt != "" {
				if _, merr := h.msg.Push(ctx, created.ID, "user", prompt); merr != nil {
					glog.Errorf("Ignite: store ignition prompt for session %s: %v", created.ID, merr)
				}
			}

			if _, serr2 := h.session.Start(ctx, created.ID); serr2 != nil {
				return nil, serr2
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

func (h *igniteHandler) IgnitionPreview(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			agentID := mux.Vars(r)["id"]

			agent, err := h.agent.Get(ctx, agentID)
			if err != nil {
				return nil, err
			}

			peers, perr := h.agent.AllByProjectID(ctx, agent.ProjectId)
			if perr != nil {
				return nil, perr
			}

			prompt := buildIgnitionPrompt(agent, peers)

			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if _, werr := w.Write([]byte(prompt)); werr != nil {
				return nil, pkgerrors.GeneralError("failed to write response: %s", werr)
			}
			return nil, nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func buildIgnitionPrompt(agent *Agent, peers AgentList) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Agent Ignition: %s\n\n", agent.Name)
	if agent.DisplayName != nil {
		fmt.Fprintf(&sb, "You are **%s**", *agent.DisplayName)
	} else {
		fmt.Fprintf(&sb, "You are **%s**", agent.Name)
	}
	fmt.Fprintf(&sb, ", working in project **%s**.\n\n", agent.ProjectId)

	if agent.Description != nil {
		fmt.Fprintf(&sb, "## Role\n\n%s\n\n", *agent.Description)
	}

	if agent.Prompt != nil {
		fmt.Fprintf(&sb, "## Instructions\n\n%s\n\n", *agent.Prompt)
	}

	var peerAgents AgentList
	for _, p := range peers {
		if p.ID != agent.ID {
			peerAgents = append(peerAgents, p)
		}
	}

	if len(peerAgents) > 0 {
		sb.WriteString("## Peer Agents\n\n")
		sb.WriteString("| Agent | Description |\n")
		sb.WriteString("| ----- | ----------- |\n")
		for _, p := range peerAgents {
			desc := "—"
			if p.Description != nil {
				desc = *p.Description
			}
			fmt.Fprintf(&sb, "| %s | %s |\n", p.Name, desc)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
