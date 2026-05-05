package handlerunit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
)

// ---------------------------------------------------------------------------
// PatchDisplayName
// ---------------------------------------------------------------------------

func TestPatchDisplayName_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedSession(t, svc)

	body := map[string]string{"name": "new-display-name"}
	req := jsonReq(t, http.MethodPatch,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/displayname", sess.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestPatchDisplayName_EmptyName_Returns400(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedSession(t, svc)

	body := map[string]string{"name": ""}
	req := jsonReq(t, http.MethodPatch,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/displayname", sess.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPatchDisplayName_NotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	body := map[string]string{"name": "anything"}
	req := jsonReq(t, http.MethodPatch, "/api/ambient/v1/sessions/bad-id/displayname", body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// WorkflowMetadata
// ---------------------------------------------------------------------------

func TestWorkflowMetadata_NoWorkflow_ReturnsNullWorkflow(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedSession(t, svc) // no workflow set

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow/metadata", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m["workflow"] != nil {
		t.Errorf("expected null workflow, got %v", m["workflow"])
	}
}

func TestWorkflowMetadata_WithWorkflow_ReturnsMetadata(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	// Create session then set workflow via SetWorkflow handler
	src := seedSession(t, svc)
	wfBody := SetWorkflowRequest{GitURL: "https://github.com/org/workflow.git", Branch: "main"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow", src.ID), wfBody)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("setup set-workflow: expected 200, got %d: %s", rr.Code, rr.Body)
	}

	// Now fetch metadata
	req2 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow/metadata", src.ID), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m["workflow"] == nil {
		t.Error("expected non-null workflow in metadata")
	}
}

func TestWorkflowMetadata_NotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions/bad-id/workflow/metadata", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// OAuthProviderURL
// ---------------------------------------------------------------------------

func TestOAuthProviderURL_Returns501(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/oauth/github/url", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// ExportSession
// ---------------------------------------------------------------------------

func TestExportSession_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/export", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if m["session"] == nil {
		t.Error("expected session field in export")
	}
	if m["version"] == nil {
		t.Error("expected version field in export")
	}
}

func TestExportSession_NotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions/bad-id/export", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
