package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

func newRequest(queryParams, headerProject string) *http.Request {
	reqURL := "/sessions"
	if queryParams != "" {
		reqURL += "?" + queryParams
	}
	r := httptest.NewRequest(http.MethodGet, reqURL, nil)
	if headerProject != "" {
		r.Header.Set("X-Ambient-Project", headerProject)
	}
	return r
}

func newRequestWithProjectParam(projectID, headerProject string) *http.Request {
	reqURL := "/sessions?project_id=" + url.QueryEscape(projectID)
	r := httptest.NewRequest(http.MethodGet, reqURL, nil)
	if headerProject != "" {
		r.Header.Set("X-Ambient-Project", headerProject)
	}
	return r
}

func TestApplyProjectScope_HeaderOnly(t *testing.T) {
	r := newRequest("", "my-project")
	listArgs := services.NewListArguments(r.URL.Query())

	err := ApplyProjectScope(r, listArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listArgs.Search != "project_id = 'my-project'" {
		t.Errorf("expected project filter in search, got %q", listArgs.Search)
	}
}

func TestApplyProjectScope_QueryParamOnly(t *testing.T) {
	r := newRequest("project_id=query-proj", "")
	listArgs := services.NewListArguments(r.URL.Query())

	err := ApplyProjectScope(r, listArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listArgs.Search != "project_id = 'query-proj'" {
		t.Errorf("expected project filter in search, got %q", listArgs.Search)
	}
}

func TestApplyProjectScope_QueryParamTakesPrecedence(t *testing.T) {
	r := newRequest("project_id=from-param", "from-header")
	listArgs := services.NewListArguments(r.URL.Query())

	err := ApplyProjectScope(r, listArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listArgs.Search != "project_id = 'from-param'" {
		t.Errorf("expected query param to take precedence, got %q", listArgs.Search)
	}
}

func TestApplyProjectScope_NoProjectReturnsNoFilter(t *testing.T) {
	r := newRequest("", "")
	listArgs := services.NewListArguments(r.URL.Query())

	err := ApplyProjectScope(r, listArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listArgs.Search != "" {
		t.Errorf("expected empty search, got %q", listArgs.Search)
	}
}

func TestApplyProjectScope_CombinesWithExistingSearch(t *testing.T) {
	r := newRequest("search=name+%3D+%27test%27", "my-project")
	listArgs := services.NewListArguments(r.URL.Query())

	err := ApplyProjectScope(r, listArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listArgs.Search != "project_id = 'my-project' and (name = 'test')" {
		t.Errorf("expected combined search, got %q", listArgs.Search)
	}
}

func TestApplyProjectScope_RejectsInjection(t *testing.T) {
	payloads := []struct {
		name  string
		value string
	}{
		{"SQL injection single quote", "x' OR 1=1--"},
		{"SQL injection drop", "x'; DROP TABLE sessions;--"},
		{"space", "test project"},
		{"quote", "test'quote"},
		{"semicolon", "proj;evil"},
		{"percent", "proj%20evil"},
	}

	for _, tt := range payloads {
		t.Run(tt.name+" via header", func(t *testing.T) {
			r := newRequest("", tt.value)
			listArgs := services.NewListArguments(r.URL.Query())
			err := ApplyProjectScope(r, listArgs)
			if err == nil {
				t.Errorf("expected validation error for %q, got nil", tt.value)
			}
		})

		t.Run(tt.name+" via query param", func(t *testing.T) {
			r := newRequestWithProjectParam(tt.value, "")
			listArgs := services.NewListArguments(r.URL.Query())
			err := ApplyProjectScope(r, listArgs)
			if err == nil {
				t.Errorf("expected validation error for %q, got nil", tt.value)
			}
		})
	}
}

func TestApplyProjectScope_AcceptsValidPatterns(t *testing.T) {
	valid := []string{
		"my-project",
		"project_123",
		"ABC-DEF",
		"a",
		"test-cp-verify-2",
	}

	for _, v := range valid {
		t.Run(v, func(t *testing.T) {
			r := newRequest("", v)
			listArgs := services.NewListArguments(r.URL.Query())
			err := ApplyProjectScope(r, listArgs)
			if err != nil {
				t.Errorf("expected no error for %q, got %v", v, err)
			}
		})
	}
}
