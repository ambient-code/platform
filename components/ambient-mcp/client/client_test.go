package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	c := New("http://localhost:8080/", "my-token")
	if c.BaseURL() != "http://localhost:8080" {
		t.Errorf("BaseURL() = %q, want trailing slash stripped", c.BaseURL())
	}
	if c.Token() != "my-token" {
		t.Errorf("Token() = %q, want %q", c.Token(), "my-token")
	}
}

func TestSetToken(t *testing.T) {
	c := New("http://localhost:8080", "initial")
	c.SetToken("refreshed")
	if c.Token() != "refreshed" {
		t.Errorf("Token() after SetToken = %q, want %q", c.Token(), "refreshed")
	}
}

func TestSetToken_ConcurrentAccess(t *testing.T) {
	c := New("http://localhost:8080", "initial")
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			c.SetToken("token-" + string(rune('A'+n%26)))
		}(i)
		go func() {
			defer wg.Done()
			_ = c.Token()
		}()
	}
	wg.Wait()

	got := c.Token()
	if got == "" {
		t.Error("Token() is empty after concurrent access")
	}
}

func TestGet_SendsBearerToken(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-bearer")
	var result map[string]string
	err := c.Get(context.Background(), "/healthz", &result)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if receivedAuth != "Bearer test-bearer" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-bearer")
	}
}

func TestGet_UsesRefreshedToken(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, "old-token")
	c.SetToken("new-token")

	var result map[string]string
	err := c.Get(context.Background(), "/healthz", &result)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if receivedAuth != "Bearer new-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer new-token")
	}
}

func TestGet_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	err := c.Get(context.Background(), "/missing", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestPost_SendsBody(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	body := map[string]string{"name": "test"}
	var result map[string]string
	err := c.Post(context.Background(), "/items", body, &result, http.StatusCreated)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if receivedBody["name"] != "test" {
		t.Errorf("received body name = %q, want %q", receivedBody["name"], "test")
	}
	if result["id"] != "123" {
		t.Errorf("result id = %q, want %q", result["id"], "123")
	}
}
