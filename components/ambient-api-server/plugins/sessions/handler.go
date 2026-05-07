package sessions

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
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

// RepoEntry represents a single repository attached to a session.
type RepoEntry struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	Name   string `json:"name,omitempty"`
}

// SetWorkflowRequest is the body for POST /{id}/workflow.
type SetWorkflowRequest struct {
	GitURL string `json:"git_url"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// SetModelRequest is the body for POST /{id}/model.
type SetModelRequest struct {
	Model string `json:"model"`
}

// AddRepoRequest is the body for POST /{id}/repos.
type AddRepoRequest struct {
	URL      string `json:"url"`
	Branch   string `json:"branch,omitempty"`
	AutoPush bool   `json:"auto_push,omitempty"`
}

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

// Clone creates a new session that is a copy of an existing one.
func (h sessionHandler) Clone(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			src, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			clone := &Session{
				Name:                 src.Name + "-clone",
				RepoUrl:              src.RepoUrl,
				AssignedUserId:       src.AssignedUserId,
				WorkflowId:           src.WorkflowId,
				Repos:                src.Repos,
				Timeout:              src.Timeout,
				LlmModel:             src.LlmModel,
				LlmTemperature:       src.LlmTemperature,
				LlmMaxTokens:         src.LlmMaxTokens,
				BotAccountName:       src.BotAccountName,
				ResourceOverrides:    src.ResourceOverrides,
				EnvironmentVariables: src.EnvironmentVariables,
				SessionLabels:        src.SessionLabels,
				SessionAnnotations:   src.SessionAnnotations,
				ProjectId:            src.ProjectId,
				AgentId:              src.AgentId,
			}
			cloneID := src.ID
			clone.ParentSessionId = &cloneID

			created, err := h.session.Create(ctx, clone)
			if err != nil {
				return nil, err
			}
			return PresentSession(created), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.HandleDelete(w, r, cfg, http.StatusCreated)
}

// AddRepo appends a repository to the session's repos list.
func (h sessionHandler) AddRepo(w http.ResponseWriter, r *http.Request) {
	var body AddRepoRequest
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			func() *errors.ServiceError {
				if body.URL == "" {
					return errors.Validation("url is required")
				}
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			branch := body.Branch
			if branch == "" {
				branch = "main"
			}
			repoName := path.Base(strings.TrimSuffix(body.URL, ".git"))
			entry := RepoEntry{URL: body.URL, Branch: branch, Name: repoName}

			var repos []RepoEntry
			if session.Repos != nil && *session.Repos != "" {
				if jsonErr := json.Unmarshal([]byte(*session.Repos), &repos); jsonErr != nil {
					repos = nil
				}
			}
			repos = append(repos, entry)

			raw, jsonErr := json.Marshal(repos)
			if jsonErr != nil {
				return nil, errors.GeneralError("failed to serialize repos: %v", jsonErr)
			}
			reposStr := string(raw)
			session.Repos = &reposStr

			updated, err := h.session.Replace(ctx, session)
			if err != nil {
				return nil, err
			}
			return PresentSession(updated), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

// RemoveRepo removes a repository by name from the session's repos list.
func (h sessionHandler) RemoveRepo(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			vars := mux.Vars(r)
			id := vars["id"]
			repoName := vars["repoName"]

			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			var repos []RepoEntry
			if session.Repos != nil && *session.Repos != "" {
				_ = json.Unmarshal([]byte(*session.Repos), &repos)
			}

			filtered := repos[:0]
			for _, repo := range repos {
				if repo.Name != repoName {
					filtered = append(filtered, repo)
				}
			}

			raw, jsonErr := json.Marshal(filtered)
			if jsonErr != nil {
				return nil, errors.GeneralError("failed to serialize repos: %v", jsonErr)
			}
			reposStr := string(raw)
			session.Repos = &reposStr

			updated, err := h.session.Replace(ctx, session)
			if err != nil {
				return nil, err
			}
			return PresentSession(updated), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.HandleDelete(w, r, cfg, http.StatusOK)
}

// SetWorkflow updates the active workflow configuration for the session.
func (h sessionHandler) SetWorkflow(w http.ResponseWriter, r *http.Request) {
	var body SetWorkflowRequest
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			func() *errors.ServiceError {
				if body.GitURL == "" {
					return errors.Validation("git_url is required")
				}
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if body.Branch == "" {
				body.Branch = "main"
			}

			raw, jsonErr := json.Marshal(body)
			if jsonErr != nil {
				return nil, errors.GeneralError("failed to serialize workflow: %v", jsonErr)
			}
			workflowStr := string(raw)
			session.WorkflowId = &workflowStr

			updated, err := h.session.Replace(ctx, session)
			if err != nil {
				return nil, err
			}
			return PresentSession(updated), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

// SetModel switches the LLM model for the session.
func (h sessionHandler) SetModel(w http.ResponseWriter, r *http.Request) {
	var body SetModelRequest
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			func() *errors.ServiceError {
				if body.Model == "" {
					return errors.Validation("model is required")
				}
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			session, err := h.session.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			session.LlmModel = &body.Model

			updated, err := h.session.Replace(ctx, session)
			if err != nil {
				return nil, err
			}
			return PresentSession(updated), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
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

// runnerBaseURL returns the base URL of the runner pod for a session, or "".
func runnerBaseURL(session *Session) string {
	if session.KubeCrName == nil || session.KubeNamespace == nil {
		return ""
	}
	return fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001",
		strings.ToLower(*session.KubeCrName), *session.KubeNamespace)
}

// proxyToRunner proxies an HTTP request to the runner and writes the response.
// Returns false if the runner is unreachable (caller should write a stub response).
func proxyToRunner(w http.ResponseWriter, r *http.Request, runnerURL string) bool {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, runnerURL, r.Body)
	if err != nil {
		glog.Errorf("proxyToRunner: build request to %s: %v", runnerURL, err)
		http.Error(w, "failed to build runner request", http.StatusInternalServerError)
		return true
	}
	for k, vals := range r.Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	resp, doErr := EventsHTTPClient.Do(req)
	if doErr != nil {
		glog.V(4).Infof("proxyToRunner: runner unreachable at %s: %v", runnerURL, doErr)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	return true
}

// AGUIEvents proxies the AG-UI SSE event stream from the runner.
// Falls back to an empty SSE stream if no runner is available.
func (h sessionHandler) AGUIEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, err := h.session.Get(ctx, id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	base := runnerBaseURL(session)
	if base == "" {
		// No runner: emit an empty SSE stream that closes immediately.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		return
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/agui/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, doErr := EventsHTTPClient.Do(req)
	if doErr != nil {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		return
	}
	defer func() { _ = resp.Body.Close() }()

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
			_, _ = w.Write(buf[:n])
			_ = rc.Flush()
		}
		if readErr != nil {
			return
		}
	}
}

// AGUIRun proxies an AG-UI run request to the runner.
func (h sessionHandler) AGUIRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/agui/run")
}

// AGUIInterrupt proxies an AG-UI interrupt to the runner.
func (h sessionHandler) AGUIInterrupt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/agui/interrupt")
}

// AGUIFeedback proxies AG-UI feedback to the runner.
func (h sessionHandler) AGUIFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/agui/feedback")
}

// AGUITasks lists background tasks from the runner, or returns an empty list.
func (h sessionHandler) AGUITasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tasks":[],"total":0}`))
		return
	}
	if !proxyToRunner(w, r, base+"/agui/tasks") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tasks":[],"total":0}`))
	}
}

// AGUITaskStop proxies a task stop request to the runner.
func (h sessionHandler) AGUITaskStop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	taskID := vars["taskId"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/agui/tasks/"+taskID+"/stop")
}

// AGUITaskOutput proxies a task output request to the runner.
func (h sessionHandler) AGUITaskOutput(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	taskID := vars["taskId"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/agui/tasks/"+taskID+"/output")
}

// AGUICapabilities returns the runner's capabilities, or a stub if unavailable.
func (h sessionHandler) AGUICapabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/agui/capabilities") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"framework":"unknown"}`))
	}
}

// MCPStatus returns the runner's MCP server status, or a stub if unavailable.
func (h sessionHandler) MCPStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/mcp/status") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"servers":[],"totalCount":0}`))
	}
}

// ---------------------------------------------------------------------------
// Workspace file proxy (Part 1 — runner-proxy sub-resources)
// ---------------------------------------------------------------------------

// WorkspaceList lists workspace files from the runner, or returns an empty stub.
func (h sessionHandler) WorkspaceList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/workspace") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"files":[]}`))
	}
}

// WorkspaceFile proxies workspace file GET/PUT/DELETE to the runner.
func (h sessionHandler) WorkspaceFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	filePath := vars["path"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/workspace/"+filePath)
}

// ---------------------------------------------------------------------------
// Pre-upload file proxy (Part 1 — runner-proxy sub-resources)
// ---------------------------------------------------------------------------

// FilesList lists staged files from the runner, or returns an empty stub.
func (h sessionHandler) FilesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/files") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"files":[]}`))
	}
}

// FilesFile proxies staged file PUT/DELETE to the runner.
func (h sessionHandler) FilesFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	filePath := vars["path"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/files/"+filePath)
}

// ---------------------------------------------------------------------------
// Git proxy (Part 1 — runner-proxy sub-resources)
// ---------------------------------------------------------------------------

// GitStatus proxies git status from the runner, or returns an empty stub.
func (h sessionHandler) GitStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/git/status") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"modified":[],"staged":[],"untracked":[]}`))
	}
}

// GitConfigureRemote proxies a git configure-remote request to the runner.
func (h sessionHandler) GitConfigureRemote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" {
		http.Error(w, "session runner not available", http.StatusServiceUnavailable)
		return
	}
	proxyToRunner(w, r, base+"/git/configure-remote")
}

// GitBranches proxies git branch listing from the runner, or returns an empty stub.
func (h sessionHandler) GitBranches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/git/branches") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}
}

// ---------------------------------------------------------------------------
// Repos status + pod-events (Part 1 — runner-proxy sub-resources)
// ---------------------------------------------------------------------------

// ReposStatus proxies repo sync status from the runner, or returns an empty stub.
func (h sessionHandler) ReposStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	session, svcErr := h.session.Get(ctx, id)
	if svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	base := runnerBaseURL(session)
	if base == "" || !proxyToRunner(w, r, base+"/repos/status") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}
}

// PodEvents returns Kubernetes pod events for a session.
// This is a K8s-native endpoint; the runner does not serve it.
// Returns an empty list stub until the control plane implements native event streaming.
func (h sessionHandler) PodEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	if _, svcErr := h.session.Get(ctx, id); svcErr != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`[]`))
}

// ---------------------------------------------------------------------------
// Operational sub-resources (Part 2)
// ---------------------------------------------------------------------------

// PatchDisplayName updates the display name (Name field) of a session.
func (h sessionHandler) PatchDisplayName(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body struct {
		Name string `json:"name"`
	}
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			func() *errors.ServiceError {
				if body.Name == "" {
					return errors.Validation("name is required")
				}
				return nil
			},
		},
		Action: func() (interface{}, *errors.ServiceError) {
			sess, svcErr := h.session.Get(r.Context(), id)
			if svcErr != nil {
				return nil, svcErr
			}
			sess.Name = body.Name
			updated, svcErr := h.session.Replace(r.Context(), sess)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentReference(updated.ID, updated), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

// WorkflowMetadata returns workflow metadata parsed from the session's WorkflowId field.
func (h sessionHandler) WorkflowMetadata(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			sess, svcErr := h.session.Get(r.Context(), id)
			if svcErr != nil {
				return nil, svcErr
			}
			if sess.WorkflowId == nil || *sess.WorkflowId == "" {
				return map[string]interface{}{"workflow": nil}, nil
			}
			var wf map[string]interface{}
			if err := json.Unmarshal([]byte(*sess.WorkflowId), &wf); err != nil {
				return map[string]interface{}{"workflow": nil, "raw": *sess.WorkflowId}, nil
			}
			return map[string]interface{}{"workflow": wf}, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// OAuthProviderURL returns an OAuth redirect URL for a session provider.
// This is a K8s-backed endpoint requiring secrets access; returning 501 until natively implemented.
func (h sessionHandler) OAuthProviderURL(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "oauth provider URL generation not yet implemented natively", http.StatusNotImplemented)
}

// ExportSession returns the session data as an exportable JSON envelope.
func (h sessionHandler) ExportSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			sess, svcErr := h.session.Get(r.Context(), id)
			if svcErr != nil {
				return nil, svcErr
			}
			return map[string]interface{}{
				"session":   presenters.PresentReference(sess.ID, sess),
				"export_at": time.Now().UTC(),
				"version":   "1",
			}, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
