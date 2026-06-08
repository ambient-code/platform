package credential

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

var testTime = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

func sampleTokenResponse(id, provider, tok string) *types.CredentialTokenResponse {
	resp := &types.CredentialTokenResponse{CredentialID: id, Provider: provider}
	resp.Token = tok
	return resp
}

func sampleCredential(id, name, provider string) types.Credential {
	return types.Credential{
		ObjectReference: types.ObjectReference{ID: id, CreatedAt: &testTime, UpdatedAt: &testTime},
		Name:            name,
		Provider:        provider,
		Description:     "test credential",
	}
}

func TestCreateCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, sampleCredential("cred-1", "github-main", "github"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--name", "github-main", "--provider", "github", "--token", "ghp_xxx")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "credential/github-main created") {
		t.Errorf("expected 'credential/github-main created', got: %s", result.Stdout)
	}
}

func TestCreateCredential_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--provider", "github")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
	if !strings.Contains(result.Err.Error(), "--name is required") {
		t.Errorf("expected '--name is required', got: %v", result.Err)
	}
}

func TestCreateCredential_MissingProvider(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--name", "my-cred")
	if result.Err == nil {
		t.Fatal("expected error for missing --provider")
	}
	if !strings.Contains(result.Err.Error(), "--provider is required") {
		t.Errorf("expected '--provider is required', got: %v", result.Err)
	}
}

func TestCreateCredential_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, sampleCredential("cred-json", "json-cred", "gitlab"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--name", "json-cred", "--provider", "gitlab", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-cred"`) {
		t.Errorf("expected JSON with 'json-cred', got: %s", result.Stdout)
	}
}

func TestCreateCredential_AllFlags(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var cred types.Credential
		if err := json.Unmarshal(body, &cred); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if cred.Token != "tok123" {
			t.Errorf("expected token 'tok123', got %q", cred.Token)
		}
		if cred.Description != "my desc" {
			t.Errorf("expected description 'my desc', got %q", cred.Description)
		}
		if cred.URL != "https://jira.example.com" {
			t.Errorf("expected url 'https://jira.example.com', got %q", cred.URL)
		}
		if cred.Email != "test@example.com" {
			t.Errorf("expected email 'test@example.com', got %q", cred.Email)
		}
		srv.RespondJSON(t, w, http.StatusCreated, sampleCredential("cred-all", "full-cred", "jira"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create",
		"--name", "full-cred",
		"--provider", "jira",
		"--token", "tok123",
		"--description", "my desc",
		"--url", "https://jira.example.com",
		"--email", "test@example.com",
		"--labels", `{"env":"test"}`,
		"--annotations", `{"note":"demo"}`,
	)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
}

func TestListCredentials_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{
			Items: []types.Credential{
				sampleCredential("c1", "github-pat", "github"),
				sampleCredential("c2", "jira-token", "jira"),
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "github-pat") {
		t.Errorf("expected 'github-pat' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "jira-token") {
		t.Errorf("expected 'jira-token' in output, got: %s", result.Stdout)
	}
}

func TestListCredentials_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{
			Items: []types.Credential{sampleCredential("c1", "gh-pat", "github")},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"gh-pat"`) {
		t.Errorf("expected JSON with 'gh-pat', got: %s", result.Stdout)
	}
}

func TestListCredentials_Empty(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{Items: []types.Credential{}})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
}

func TestListCredentials_ProviderFilter(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("search")
		if !strings.Contains(search, "provider='github'") {
			t.Errorf("expected search to contain provider='github', got: %s", search)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{
			Items: []types.Credential{sampleCredential("c1", "gh-only", "github")},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list", "--provider", "github")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
}

func TestGetCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, sampleCredential("cred-42", "my-github", "github"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get", "cred-42")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "my-github") {
		t.Errorf("expected 'my-github' in output, got: %s", result.Stdout)
	}
}

func TestGetCredential_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-j", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, sampleCredential("cred-j", "json-get", "gitlab"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get", "cred-j", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-get"`) {
		t.Errorf("expected JSON with 'json-get', got: %s", result.Stdout)
	}
}

func TestGetCredential_MissingID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get")
	if result.Err == nil {
		t.Fatal("expected error for missing credential ID argument")
	}
}

func TestUpdateCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var patch map[string]interface{}
		if err := json.Unmarshal(body, &patch); err != nil {
			t.Fatalf("unmarshal patch body: %v", err)
		}
		if patch["description"] != "updated desc" {
			t.Errorf("expected description 'updated desc', got %v", patch["description"])
		}
		srv.RespondJSON(t, w, http.StatusOK, sampleCredential("cred-u1", "updated-cred", "github"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "update", "cred-u1", "--description", "updated desc")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "credential/updated-cred updated") {
		t.Errorf("expected 'credential/updated-cred updated', got: %s", result.Stdout)
	}
}

func TestUpdateCredential_MissingID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "update")
	if result.Err == nil {
		t.Fatal("expected error for missing credential ID argument")
	}
}

func TestDeleteCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-d1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "cred-d1", "--confirm")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "credential/cred-d1 deleted") {
		t.Errorf("expected 'credential/cred-d1 deleted', got: %s", result.Stdout)
	}
}

func TestDeleteCredential_MissingConfirm(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "cred-d1")
	if result.Err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(result.Err.Error(), "--confirm") {
		t.Errorf("expected '--confirm' in error, got: %v", result.Err)
	}
}

func TestDeleteCredential_MissingID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "--confirm")
	if result.Err == nil {
		t.Fatal("expected error for missing credential ID argument")
	}
}

func TestTokenCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-t1/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, sampleTokenResponse("cred-t1", "github", "test-value-gh"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "token", "cred-t1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "test-value-gh") {
		t.Errorf("expected raw token in output, got: %s", result.Stdout)
	}
}

func TestTokenCredential_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials/cred-tj/token", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, sampleTokenResponse("cred-tj", "gitlab", "test-value-gl"))
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "token", "cred-tj", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"cred-tj"`) {
		t.Errorf("expected JSON with 'cred-tj', got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"test-value-gl"`) {
		t.Errorf("expected JSON with token, got: %s", result.Stdout)
	}
}

func TestTokenCredential_MissingID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "token")
	if result.Err == nil {
		t.Fatal("expected error for missing credential ID argument")
	}
}

func TestBindCredential_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{
			ListMeta: types.ListMeta{Total: 1},
			Items:    []types.Credential{sampleCredential("cred-bind-1", "github-pat", "github")},
		})
	})
	srv.Handle("/api/ambient/v1/role_bindings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var rb map[string]interface{}
		if err := json.Unmarshal(body, &rb); err != nil {
			t.Fatalf("unmarshal role binding body: %v", err)
		}
		if rb["scope"] != "credential" {
			t.Errorf("expected scope 'credential', got %v", rb["scope"])
		}
		if rb["role_id"] != "credential:viewer" {
			t.Errorf("expected role_id 'credential:viewer', got %v", rb["role_id"])
		}
		credID := "cred-bind-1"
		projectID := "my-project"
		srv.RespondJSON(t, w, http.StatusCreated, &types.RoleBinding{
			ObjectReference: types.ObjectReference{ID: "rb-new"},
			RoleID:          "credential:viewer",
			Scope:           "credential",
			CredentialID:    &credID,
			ProjectID:       &projectID,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "bind", "github-pat", "--project", "my-project")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "credential/github-pat bound to project/my-project") {
		t.Errorf("expected bind confirmation, got: %s", result.Stdout)
	}
}

func TestBindCredential_MissingProject(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "bind", "github-pat")
	if result.Err == nil {
		t.Fatal("expected error for missing --project")
	}
	if !strings.Contains(result.Err.Error(), "--project is required") {
		t.Errorf("expected '--project is required', got: %v", result.Err)
	}
}

func TestBindCredential_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "bind")
	if result.Err == nil {
		t.Fatal("expected error for missing credential name argument")
	}
}

func TestBindCredential_NotFound(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/credentials", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.CredentialList{
			ListMeta: types.ListMeta{Total: 0},
			Items:    []types.Credential{},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "bind", "nonexistent-cred", "--project", "my-project")
	if result.Err == nil {
		t.Fatal("expected error for credential not found")
	}
	if !strings.Contains(result.Err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", result.Err)
	}
}
