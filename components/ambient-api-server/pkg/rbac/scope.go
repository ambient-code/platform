package rbac

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type ScopeLevel string

const (
	ScopeGlobal     ScopeLevel = "global"
	ScopeProject    ScopeLevel = "project"
	ScopeAgent      ScopeLevel = "agent"
	ScopeSession    ScopeLevel = "session"
	ScopeCredential ScopeLevel = "credential"
)

type RequestScope struct {
	ProjectID    string
	AgentID      string
	SessionID    string
	CredentialID string
}

func ExtractRequestScope(r *http.Request) RequestScope {
	vars := mux.Vars(r)
	scope := RequestScope{}

	path := r.URL.Path
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")

	for i, seg := range segments {
		if seg == "v1" && i+2 < len(segments) {
			resource := segments[i+1]
			switch resource {
			case "projects":
				if i+2 < len(segments) {
					scope.ProjectID = segments[i+2]
				}
				if i+4 < len(segments) {
					switch segments[i+3] {
					case "agents":
						scope.AgentID = segments[i+4]
					case "scheduled-sessions":
						// project-scoped, no further nesting
					}
				}
			case "sessions":
				if i+2 < len(segments) {
					scope.SessionID = segments[i+2]
				}
			case "credentials":
				if i+2 < len(segments) {
					scope.CredentialID = segments[i+2]
				}
			}
			break
		}
	}

	if r.Method == http.MethodPost && scope.ProjectID == "" && scope.CredentialID == "" && r.Body != nil {
		scope = extractScopeFromBody(r, scope)
	}

	if id := vars["id"]; id != "" && scope.ProjectID == "" && scope.SessionID == "" && scope.CredentialID == "" {
		resource := pathToResource(r.URL.Path)
		switch resource {
		case "project":
			scope.ProjectID = id
		case "session":
			scope.SessionID = id
		case "credential":
			scope.CredentialID = id
		}
	}
	if agentID := vars["agent_id"]; agentID != "" {
		scope.AgentID = agentID
	}

	return scope
}

func extractScopeFromBody(r *http.Request, scope RequestScope) RequestScope {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return scope
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		ProjectID    string `json:"project_id"`
		CredentialID string `json:"credential_id"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return scope
	}
	if payload.ProjectID != "" && scope.ProjectID == "" {
		scope.ProjectID = payload.ProjectID
	}
	if payload.CredentialID != "" && scope.CredentialID == "" {
		scope.CredentialID = payload.CredentialID
	}
	return scope
}

func isAuthExempt(method, path string) bool {
	normalized := strings.TrimSuffix(path, "/")
	switch {
	case method == http.MethodPost && normalized == "/api/ambient/v1/projects":
		return true
	case method == http.MethodPost && normalized == "/api/ambient/v1/credentials":
		return true
	case method == http.MethodPost && normalized == "/api/ambient/v1/role_bindings":
		return true
	case method == http.MethodGet && normalized == "/api/ambient/v1/roles":
		return true
	case method == http.MethodGet && strings.HasPrefix(normalized, "/api/ambient/v1/roles/"):
		remaining := strings.TrimPrefix(normalized, "/api/ambient/v1/roles/")
		return !strings.Contains(remaining, "/")
	default:
		return false
	}
}

func isListEndpoint(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	segments := strings.Split(strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/"), "/")
	if len(segments) < 3 {
		return false
	}
	last := segments[len(segments)-1]
	switch last {
	case "projects", "agents", "sessions", "credentials", "roles", "role_bindings",
		"users", "inbox", "session_messages", "scheduled-sessions", "messages":
		return true
	}
	return false
}

func isSingletonGet(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return !isListEndpoint(method, path)
}
