package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/inbox"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	pkgerrors "github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
)

type StartResponse struct {
	Session        openapi.Session `json:"session"`
	StartingPrompt string          `json:"starting_prompt"`
}

type ProjectPromptFetcher interface {
	GetPrompt(ctx context.Context, projectID string) (*string, error)
}

type startHandler struct {
	agent   AgentService
	inbox   inbox.InboxMessageService
	session sessions.SessionService
	msg     sessions.MessageService
	project ProjectPromptFetcher
	locks   sync.Map
}

func NewStartHandler(agent AgentService, inboxSvc inbox.InboxMessageService, session sessions.SessionService, msg sessions.MessageService, project ProjectPromptFetcher) *startHandler {
	return &startHandler{
		agent:   agent,
		inbox:   inboxSvc,
		session: session,
		msg:     msg,
		project: project,
	}
}

func (h *startHandler) Start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := mux.Vars(r)["id"]
	agentID := mux.Vars(r)["agent_id"]

	mu := &sync.Mutex{}
	if existing, loaded := h.locks.LoadOrStore(agentID, mu); loaded {
		mu = existing.(*sync.Mutex)
	}
	mu.Lock()
	defer mu.Unlock()

	agent, err := h.agent.Get(ctx, agentID)
	if err != nil {
		handlers.HandleError(ctx, w, err)
		return
	}

	if agent.ProjectId != projectID {
		handlers.HandleError(ctx, w, pkgerrors.Forbidden("agent does not belong to this project"))
		return
	}

	existing, activeErr := h.session.ActiveByAgentID(ctx, agentID)
	if activeErr != nil {
		handlers.HandleError(ctx, w, activeErr)
		return
	}
	if existing != nil {
		resp := &StartResponse{
			Session: sessions.PresentSession(existing),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			unread, inboxErr := h.inbox.UnreadByAgentID(ctx, agentID)
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

			sess := &sessions.Session{
				Name:      fmt.Sprintf("%s-%d", agent.Name, time.Now().Unix()),
				Prompt:    agent.Prompt,
				ProjectId: &agent.ProjectId,
				AgentId:   &agentID,
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
					glog.Warningf("Start agent %s: mark inbox message %s read: %v", agentID, msg.ID, replErr)
				}
			}

			peers, peersErr := h.agent.AllByProjectID(ctx, agent.ProjectId)
			if peersErr != nil {
				return nil, peersErr
			}

			var projectPrompt *string
			if h.project != nil {
				if pp, ppErr := h.project.GetPrompt(ctx, agent.ProjectId); ppErr == nil {
					projectPrompt = pp
				}
			}

			prompt := buildStartPrompt(agent, peers, unread, projectPrompt, requestPrompt)

			if prompt != "" {
				if _, pushErr := h.msg.Push(ctx, created.ID, "user", prompt); pushErr != nil {
					glog.Errorf("Start agent %s: store start prompt for session %s: %v", agentID, created.ID, pushErr)
				}
			}

			agentCopy := *agent
			agentCopy.CurrentSessionId = &created.ID
			if _, replErr := h.agent.Replace(ctx, &agentCopy); replErr != nil {
				return nil, replErr
			}

			if _, startErr := h.session.Start(ctx, created.ID); startErr != nil {
				return nil, startErr
			}

			return &StartResponse{
				Session:        sessions.PresentSession(created),
				StartingPrompt: prompt,
			}, nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h *startHandler) StartPreview(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *pkgerrors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			agentID := mux.Vars(r)["agent_id"]

			agent, err := h.agent.Get(ctx, agentID)
			if err != nil {
				return nil, err
			}

			if agent.ProjectId != projectID {
				return nil, pkgerrors.Forbidden("agent does not belong to this project")
			}

			unread, inboxErr := h.inbox.UnreadByAgentID(ctx, agentID)
			if inboxErr != nil {
				return nil, inboxErr
			}

			peers, peersErr := h.agent.AllByProjectID(ctx, agent.ProjectId)
			if peersErr != nil {
				return nil, peersErr
			}

			var projectPrompt *string
			if h.project != nil {
				if pp, ppErr := h.project.GetPrompt(ctx, agent.ProjectId); ppErr == nil {
					projectPrompt = pp
				}
			}

			prompt := buildStartPrompt(agent, peers, unread, projectPrompt, nil)

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

func buildStartPrompt(agent *Agent, peers AgentList, unread inbox.InboxMessageList, projectPrompt *string, requestPrompt *string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Agent Start: %s\n\n", agent.Name)
	fmt.Fprintf(&sb, "You are **%s**, working in project **%s**.\n\n", agent.Name, agent.ProjectId)

	if projectPrompt != nil && *projectPrompt != "" {
		fmt.Fprintf(&sb, "## Workspace Context\n\n%s\n\n", *projectPrompt)
	}

	if agent.Prompt != nil && *agent.Prompt != "" {
		fmt.Fprintf(&sb, "## Standing Instructions\n\n%s\n\n", *agent.Prompt)
	}

	var peerAgents AgentList
	for _, p := range peers {
		if p.ID != agent.ID {
			peerAgents = append(peerAgents, p)
		}
	}

	if len(peerAgents) > 0 {
		sb.WriteString("## Peer Agents in this Project\n\n")
		sb.WriteString("| Agent | Description |\n")
		sb.WriteString("| ----- | ----------- |\n")
		for _, p := range peerAgents {
			fmt.Fprintf(&sb, "| %s | — |\n", p.Name)
		}
		sb.WriteString("\n")
	}

	if len(unread) > 0 {
		sb.WriteString("## Inbox Messages (unread at start)\n\n")
		for _, m := range unread {
			from := "system"
			if m.FromName != nil && *m.FromName != "" {
				from = *m.FromName
			} else if m.FromAgentId != nil && *m.FromAgentId != "" {
				from = *m.FromAgentId
			}
			fmt.Fprintf(&sb, "**From %s:** %s\n\n", from, m.Body)
		}
	}

	if requestPrompt != nil && *requestPrompt != "" {
		fmt.Fprintf(&sb, "## Task for this Run\n\n%s\n\n", *requestPrompt)
	}

	return sb.String()
}
