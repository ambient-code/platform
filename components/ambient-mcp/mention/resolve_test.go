package mention

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewResolver_TokenFunc(t *testing.T) {
	callCount := 0
	tokenFn := func() string {
		callCount++
		return "dynamic-token"
	}
	r, err := NewResolver("http://localhost:8080", tokenFn)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	if callCount != 0 {
		t.Errorf("tokenFn called %d times at construction, want 0", callCount)
	}
}

func TestNewResolver_NilTokenFunc(t *testing.T) {
	_, err := NewResolver("http://localhost:8080", nil)
	if err == nil {
		t.Fatal("expected error for nil tokenFn")
	}
}

func TestResolve_ByUUID_SendsCurrentToken(t *testing.T) {
	var tokenSeq atomic.Int32
	tokens := []string{"token-v1", "token-v2"}

	var receivedAuths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuths = append(receivedAuths, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "550e8400-e29b-41d4-a716-446655440000"})
	}))
	defer srv.Close()

	r, err := NewResolver(srv.URL, func() string {
		idx := tokenSeq.Load()
		if int(idx) < len(tokens) {
			return tokens[idx]
		}
		return tokens[len(tokens)-1]
	})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	ctx := context.Background()
	agentID, err := r.Resolve(ctx, "proj1", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if agentID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("agentID = %q, want UUID", agentID)
	}
	if receivedAuths[0] != "Bearer token-v1" {
		t.Errorf("first auth = %q, want %q", receivedAuths[0], "Bearer token-v1")
	}

	tokenSeq.Store(1)
	_, err = r.Resolve(ctx, "proj1", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Resolve (2nd): %v", err)
	}
	if receivedAuths[1] != "Bearer token-v2" {
		t.Errorf("second auth = %q, want %q", receivedAuths[1], "Bearer token-v2")
	}
}

func TestResolve_ByName_SendsCurrentToken(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agentSearchResult{
			Items: []struct {
				ID string `json:"id"`
			}{{ID: "resolved-agent-id"}},
			Total: 1,
		})
	}))
	defer srv.Close()

	r, err := NewResolver(srv.URL, func() string { return "name-lookup-token" })
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	agentID, err := r.Resolve(context.Background(), "proj1", "my-agent")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if agentID != "resolved-agent-id" {
		t.Errorf("agentID = %q, want %q", agentID, "resolved-agent-id")
	}
	if receivedAuth != "Bearer name-lookup-token" {
		t.Errorf("auth = %q, want %q", receivedAuth, "Bearer name-lookup-token")
	}
}

func TestResolve_ByUUID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	r, err := NewResolver(srv.URL, func() string { return "t" })
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	_, err = r.Resolve(context.Background(), "proj1", "550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestResolve_ByName_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agentSearchResult{Total: 0})
	}))
	defer srv.Close()

	r, err := NewResolver(srv.URL, func() string { return "t" })
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	_, err = r.Resolve(context.Background(), "proj1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestResolve_ByName_Ambiguous(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agentSearchResult{
			Items: []struct {
				ID string `json:"id"`
			}{{ID: "a"}, {ID: "b"}},
			Total: 2,
		})
	}))
	defer srv.Close()

	r, err := NewResolver(srv.URL, func() string { return "t" })
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	_, err = r.Resolve(context.Background(), "proj1", "ambiguous")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
}

func TestExtract(t *testing.T) {
	matches := Extract("Hello @alice and @bob, also @alice again")
	if len(matches) != 2 {
		t.Fatalf("len(matches) = %d, want 2", len(matches))
	}
	if matches[0].Identifier != "alice" {
		t.Errorf("matches[0].Identifier = %q, want %q", matches[0].Identifier, "alice")
	}
	if matches[1].Identifier != "bob" {
		t.Errorf("matches[1].Identifier = %q, want %q", matches[1].Identifier, "bob")
	}
}

func TestExtract_NoMentions(t *testing.T) {
	matches := Extract("no mentions here")
	if len(matches) != 0 {
		t.Errorf("len(matches) = %d, want 0", len(matches))
	}
}

func TestStripToken_RemovesMention(t *testing.T) {
	result := StripToken("hello @alice do this", "@alice")
	if result != "hello  do this" {
		t.Errorf("StripToken = %q, want %q", result, "hello  do this")
	}
}
