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
