package sessions

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/common"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = sessionHandler{}

// EventsHTTPClient is used to proxy SSE streams from runner pods.
// Replaceable in tests to simulate runner behavior without a live cluster.
// ResponseHeaderTimeout times out only the header phase; body streaming is unlimited.
var EventsHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		ResponseHeaderTimeout: 5 * time.Second,
	},
}

type sessionHandler struct {
	session SessionService
	msg     MessageService
	generic services.GenericService
}

func NewSessionHandler(session SessionService, msg MessageService, generic services.GenericService) *sessionHandler {
	return &sessionHandler{
		session: session,
		msg:     msg,
		generic: generic,
	}
}

func (h sessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var session openapi.Session
	cfg := &handlers.HandlerConfig{
		Body: &session,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&session, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			sessionModel := ConvertSession(session)
			if username := auth.GetUsernameFromContext(ctx); username != "" {
				sessionModel.CreatedByUserId = &username
			}
			if sessionModel.ProjectId == nil {
				if hdr := r.Header.Get("X-Ambient-Project"); hdr != "" {
					sessionModel.ProjectId = &hdr
				}
			}
			sessionModel, err := h.session.Create(ctx, sessionModel)
			if err != nil {
				return nil, err
			}
			if sessionModel.Prompt != nil && *sessionModel.Prompt != "" {
				if _, pushErr := h.msg.Push(ctx, sessionModel.ID, "user", *sessionModel.Prompt); pushErr != nil {
					glog.Errorf("Create: push prompt for session %s: %v", sessionModel.ID, pushErr)
				}
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h sessionHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.SessionPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.RepoUrl != nil {
				found.RepoUrl = patch.RepoUrl
			}
			if patch.Prompt != nil {
				found.Prompt = patch.Prompt
			}
			if patch.AssignedUserId != nil {
				found.AssignedUserId = patch.AssignedUserId
			}
			if patch.WorkflowId != nil {
				found.WorkflowId = patch.WorkflowId
			}
			if patch.Repos != nil {
				found.Repos = patch.Repos
			}
			if patch.Timeout != nil {
				found.Timeout = patch.Timeout
			}
			if patch.LlmModel != nil {
				found.LlmModel = patch.LlmModel
			}
			if patch.LlmTemperature != nil {
				found.LlmTemperature = patch.LlmTemperature
			}
			if patch.LlmMaxTokens != nil {
				found.LlmMaxTokens = patch.LlmMaxTokens
			}
			if patch.ParentSessionId != nil {
				found.ParentSessionId = patch.ParentSessionId
			}
			if patch.BotAccountName != nil {
				found.BotAccountName = patch.BotAccountName
			}
			if patch.ResourceOverrides != nil {
				found.ResourceOverrides = patch.ResourceOverrides
			}
			if patch.EnvironmentVariables != nil {
				found.EnvironmentVariables = patch.EnvironmentVariables
			}
			if patch.Labels != nil {
				found.SessionLabels = patch.Labels
			}
			if patch.Annotations != nil {
				found.SessionAnnotations = patch.Annotations
			}

			sessionModel, err := h.session.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h sessionHandler) PatchStatus(w http.ResponseWriter, r *http.Request) {
	var patch SessionStatusPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.UpdateStatus(ctx, id, &patch)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h sessionHandler) Start(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.Start(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) Stop(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			sessionModel, err := h.session.Stop(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentSession(sessionModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			if err := common.ApplyProjectScope(r, listArgs); err != nil {
				return nil, err
			}
			var sessions []Session
			paging, err := h.generic.List(ctx, "id", listArgs, &sessions)
			if err != nil {
				return nil, err
			}
			sessionList := openapi.SessionList{
				Kind:  "SessionList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Session{},
			}

			for _, session := range sessions {
				converted := PresentSession(&session)
				sessionList.Items = append(sessionList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, sessionList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return sessionList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h sessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentSession(session), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h sessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.session.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h sessionHandler) StreamRunnerEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	session, err := h.session.Get(ctx, id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if session.KubeCrName == nil || session.KubeNamespace == nil {
		http.Error(w, "session has no associated runner pod", http.StatusNotFound)
		return
	}

	runnerURL := fmt.Sprintf(
		"http://session-%s.%s.svc.cluster.local:8001/events/%s",
		strings.ToLower(*session.KubeCrName), *session.KubeNamespace, *session.KubeCrName,
	)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, runnerURL, nil)
	if reqErr != nil {
		glog.Errorf("StreamRunnerEvents: build request for session %s: %v", id, reqErr)
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, doErr := EventsHTTPClient.Do(req)
	if doErr != nil {
		glog.Warningf("StreamRunnerEvents: upstream unreachable for session %s: %v", id, doErr)
		http.Error(w, "runner not reachable", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		glog.Warningf("StreamRunnerEvents: upstream returned %d for session %s", resp.StatusCode, id)
		http.Error(w, "runner returned non-OK status", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.Flush()

	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				glog.V(4).Infof("StreamRunnerEvents: write error for session %s: %v", id, writeErr)
				return
			}
			_ = rc.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				glog.V(4).Infof("StreamRunnerEvents: read error for session %s: %v", id, readErr)
			}
			return
		}
	}
}
