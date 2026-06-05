package rbac

import (
	"net/http"
	"testing"
)

func TestPathToResource(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/ambient/v1/credentials", "credential"},
		{"/api/ambient/v1/credentials/abc123", "credential"},
		{"/api/ambient/v1/credentials/abc123/token", "credential"},
		{"/api/ambient/v1/projects/prtest/credentials/abc123/token", "credential"},
		{"/api/ambient/v1/projects/prtest/credentials", "credential"},
		{"/api/ambient/v1/projects", "project"},
		{"/api/ambient/v1/projects/prtest", "project"},
		{"/api/ambient/v1/sessions", "session"},
		{"/api/ambient/v1/role_bindings", "role_binding"},
		{"/api/ambient/v1/roles", "role"},
		{"/api/ambient/v1/projects/proj-1/scheduled-sessions", "session"},
		{"/api/ambient/v1/projects/proj-1/scheduled-sessions/ss-1", "session"},
		{"/foo/bar", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pathToResource(tt.path)
			if got != tt.want {
				t.Errorf("pathToResource(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPathToAction(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/ambient/v1/credentials/abc123/token", "fetch_token"},
		{http.MethodGet, "/api/ambient/v1/projects/prtest/credentials/abc123/token", "fetch_token"},
		{http.MethodGet, "/api/ambient/v1/credentials", "read"},
		{http.MethodPost, "/api/ambient/v1/credentials", "create"},
		{http.MethodPatch, "/api/ambient/v1/credentials/abc123", "update"},
		{http.MethodDelete, "/api/ambient/v1/credentials/abc123", "delete"},
		{http.MethodGet, "/api/ambient/v1/agents/abc123/start", "start"},
		{http.MethodGet, "/api/ambient/v1/agents/abc123/stop", "stop"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := pathToAction(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("pathToAction(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAuthExempt(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodPost, "/api/ambient/v1/projects", true},
		{http.MethodGet, "/api/ambient/v1/projects", false},
		{http.MethodPost, "/api/ambient/v1/credentials", true},
		{http.MethodGet, "/api/ambient/v1/credentials", false},
		{http.MethodGet, "/api/ambient/v1/roles", true},
		{http.MethodGet, "/api/ambient/v1/roles/abc123", true},
		{http.MethodPost, "/api/ambient/v1/roles", false},
		{http.MethodDelete, "/api/ambient/v1/roles/abc123", false},
		{http.MethodGet, "/api/ambient/v1/roles/abc123/something", false},
		{http.MethodPost, "/api/ambient/v1/role_bindings", true},
		{http.MethodPatch, "/api/ambient/v1/role_bindings/rb1", false},
		{http.MethodDelete, "/api/ambient/v1/role_bindings/rb1", false},
		{http.MethodGet, "/api/ambient/v1/role_bindings", false},
		{http.MethodGet, "/api/ambient/v1/sessions", false},
		{http.MethodPost, "/api/ambient/v1/projects/p1/agents", false},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := isAuthExempt(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("isAuthExempt(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsListEndpoint(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/ambient/v1/projects", true},
		{http.MethodGet, "/api/ambient/v1/sessions", true},
		{http.MethodGet, "/api/ambient/v1/credentials", true},
		{http.MethodGet, "/api/ambient/v1/projects/p1/agents", true},
		{http.MethodGet, "/api/ambient/v1/projects/p1", false},
		{http.MethodGet, "/api/ambient/v1/sessions/s1", false},
		{http.MethodPost, "/api/ambient/v1/projects", false},
		{http.MethodGet, "/api/ambient/v1/role_bindings", true},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := isListEndpoint(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("isListEndpoint(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsSingletonGet(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/ambient/v1/projects/p1", true},
		{http.MethodGet, "/api/ambient/v1/sessions/s1", true},
		{http.MethodGet, "/api/ambient/v1/projects", false},
		{http.MethodPost, "/api/ambient/v1/projects", false},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := isSingletonGet(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("isSingletonGet(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestBindingMatchesPermission(t *testing.T) {
	tests := []struct {
		name      string
		permsJSON string
		required  string
		want      bool
	}{
		{"wildcard matches all", `["*:*"]`, "project:read", true},
		{"exact match", `["project:read"]`, "project:read", true},
		{"resource wildcard", `["project:*"]`, "project:read", true},
		{"action wildcard", `["*:read"]`, "project:read", true},
		{"no match", `["agent:read"]`, "project:read", false},
		{"partial match resource", `["project:create"]`, "project:read", false},
		{"multiple perms", `["agent:read","project:read"]`, "project:read", true},
		{"invalid json", `invalid`, "project:read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bindingMatchesPermission(tt.permsJSON, tt.required)
			if got != tt.want {
				t.Errorf("bindingMatchesPermission(%q, %q) = %v, want %v", tt.permsJSON, tt.required, got, tt.want)
			}
		})
	}
}

func TestBindingCoversScope(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		binding bindingRow
		scope   RequestScope
		want    bool
	}{
		{
			name:    "global covers everything",
			binding: bindingRow{Scope: "global"},
			scope:   RequestScope{ProjectID: "proj-1"},
			want:    true,
		},
		{
			name:    "project matches same project",
			binding: bindingRow{Scope: "project", ProjectID: strPtr("proj-1")},
			scope:   RequestScope{ProjectID: "proj-1"},
			want:    true,
		},
		{
			name:    "project does not match different project",
			binding: bindingRow{Scope: "project", ProjectID: strPtr("proj-1")},
			scope:   RequestScope{ProjectID: "proj-2"},
			want:    false,
		},
		{
			name:    "agent matches same agent",
			binding: bindingRow{Scope: "agent", AgentID: strPtr("agent-1")},
			scope:   RequestScope{AgentID: "agent-1"},
			want:    true,
		},
		{
			name:    "agent does not match different agent",
			binding: bindingRow{Scope: "agent", AgentID: strPtr("agent-1")},
			scope:   RequestScope{AgentID: "agent-2"},
			want:    false,
		},
		{
			name:    "credential matches same credential",
			binding: bindingRow{Scope: "credential", CredentialID: strPtr("cred-1")},
			scope:   RequestScope{CredentialID: "cred-1"},
			want:    true,
		},
		{
			name:    "credential does not match different credential",
			binding: bindingRow{Scope: "credential", CredentialID: strPtr("cred-1")},
			scope:   RequestScope{CredentialID: "cred-2"},
			want:    false,
		},
		{
			name:    "session matches",
			binding: bindingRow{Scope: "session", SessionID: strPtr("s-1")},
			scope:   RequestScope{SessionID: "s-1"},
			want:    true,
		},
		{
			name:    "project binding with nil project_id",
			binding: bindingRow{Scope: "project", ProjectID: nil},
			scope:   RequestScope{ProjectID: "proj-1"},
			want:    false,
		},
		{
			name:    "unknown scope",
			binding: bindingRow{Scope: "unknown"},
			scope:   RequestScope{ProjectID: "proj-1"},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bindingCoversScope(tt.binding, tt.scope)
			if got != tt.want {
				t.Errorf("bindingCoversScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRequestScope(t *testing.T) {
	tests := []struct {
		name string
		path string
		want RequestScope
	}{
		{
			name: "project list",
			path: "/api/ambient/v1/projects",
			want: RequestScope{},
		},
		{
			name: "project get",
			path: "/api/ambient/v1/projects/proj-1",
			want: RequestScope{ProjectID: "proj-1"},
		},
		{
			name: "project agents list",
			path: "/api/ambient/v1/projects/proj-1/agents",
			want: RequestScope{ProjectID: "proj-1"},
		},
		{
			name: "project agent get",
			path: "/api/ambient/v1/projects/proj-1/agents/agent-1",
			want: RequestScope{ProjectID: "proj-1", AgentID: "agent-1"},
		},
		{
			name: "session get",
			path: "/api/ambient/v1/sessions/sess-1",
			want: RequestScope{SessionID: "sess-1"},
		},
		{
			name: "credential get",
			path: "/api/ambient/v1/credentials/cred-1",
			want: RequestScope{CredentialID: "cred-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			got := ExtractRequestScope(r)
			if got != tt.want {
				t.Errorf("ExtractRequestScope() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"/api/ambient/v1/projects", 4},
		{"/api/ambient/v1/projects/proj-1/agents/agent-1", 7},
		{"", 0},
		{"/", 0},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != tt.want {
				t.Errorf("splitPath(%q) len = %d, want %d; segments: %v", tt.path, len(got), tt.want, got)
			}
		})
	}
}
