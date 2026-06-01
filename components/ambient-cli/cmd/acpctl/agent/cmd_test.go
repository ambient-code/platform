package agent

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

func sampleAgent(id, name, projectID string) types.Agent {
	return types.Agent{
		ObjectReference: types.ObjectReference{ID: id, CreatedAt: &testTime, UpdatedAt: &testTime},
		Name:            name,
		ProjectID:       projectID,
	}
}

func handleAgentLookup(t *testing.T, srv *testhelper.Server, projectID string, agent types.Agent) {
	t.Helper()
	srv.Handle("/api/ambient/v1/projects/"+projectID+"/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
				ListMeta: types.ListMeta{Total: 1},
				Items:    []types.Agent{agent},
			})
			return
		}
		if r.Method == http.MethodPost {
			srv.RespondJSON(t, w, http.StatusCreated, agent)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}

func TestListAgents_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agents := []types.Agent{
		sampleAgent("a1", "lead", testhelper.TestProject),
		sampleAgent("a2", "worker", testhelper.TestProject),
	}
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			ListMeta: types.ListMeta{Total: 2},
			Items:    agents,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "lead") {
		t.Errorf("expected 'lead' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "worker") {
		t.Errorf("expected 'worker' in output, got: %s", result.Stdout)
	}
}

func TestListAgents_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			Items: []types.Agent{sampleAgent("a1", "json-agent", testhelper.TestProject)},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-agent"`) {
		t.Errorf("expected JSON with 'json-agent', got: %s", result.Stdout)
	}
}

func TestGetAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-get", "my-agent", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get", "my-agent")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "my-agent") {
		t.Errorf("expected 'my-agent' in output, got: %s", result.Stdout)
	}
}

func TestGetAgent_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-gj", "json-get", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get", "json-get", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-get"`) {
		t.Errorf("expected JSON with 'json-get', got: %s", result.Stdout)
	}
}

func TestGetAgent_MissingArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "get")
	if result.Err == nil {
		t.Fatal("expected error for missing agent argument")
	}
}

func TestCreateAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var agent types.Agent
		if err := json.Unmarshal(body, &agent); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if agent.Prompt != "You are the lead" {
			t.Errorf("expected prompt 'You are the lead', got %q", agent.Prompt)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.Agent{
			ObjectReference: types.ObjectReference{ID: "a-new"},
			Name:            "lead",
			ProjectID:       testhelper.TestProject,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--name", "lead", "--prompt", "You are the lead")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "agent/lead created") {
		t.Errorf("expected 'agent/lead created', got: %s", result.Stdout)
	}
}

func TestCreateAgent_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
	if !strings.Contains(result.Err.Error(), "--name is required") {
		t.Errorf("expected '--name is required', got: %v", result.Err)
	}
}

func TestCreateAgent_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.Agent{
			ObjectReference: types.ObjectReference{ID: "a-json"},
			Name:            "json-agent",
			ProjectID:       testhelper.TestProject,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "create", "--name", "json-agent", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-agent"`) {
		t.Errorf("expected JSON with 'json-agent', got: %s", result.Stdout)
	}
}

func TestUpdateAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-u1", "update-me", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-u1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var patch map[string]interface{}
		if err := json.Unmarshal(body, &patch); err != nil {
			t.Fatalf("unmarshal patch: %v", err)
		}
		if patch["prompt"] != "new instructions" {
			t.Errorf("expected prompt 'new instructions', got %v", patch["prompt"])
		}
		agent.Prompt = "new instructions"
		srv.RespondJSON(t, w, http.StatusOK, agent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "update", "update-me", "--prompt", "new instructions")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "agent/update-me updated") {
		t.Errorf("expected 'agent/update-me updated', got: %s", result.Stdout)
	}
}

func TestUpdateAgent_MissingArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "update")
	if result.Err == nil {
		t.Fatal("expected error for missing agent argument")
	}
}

func TestDeleteAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-d1", "delete-me", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-d1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "delete-me", "--confirm")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "agent/delete-me deleted") {
		t.Errorf("expected 'agent/delete-me deleted', got: %s", result.Stdout)
	}
}

func TestDeleteAgent_MissingConfirm(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "some-agent")
	if result.Err == nil {
		t.Fatal("expected error for missing --confirm")
	}
	if !strings.Contains(result.Err.Error(), "--confirm") {
		t.Errorf("expected '--confirm' in error, got: %v", result.Err)
	}
}

func TestDeleteAgent_MissingArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "delete", "--confirm")
	if result.Err == nil {
		t.Fatal("expected error for missing agent argument")
	}
}

func TestStartAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-s1", "start-me", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-s1/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "sess-1"},
				Phase:           "Pending",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start", "start-me")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "session/sess-1 started") {
		t.Errorf("expected 'session/sess-1 started', got: %s", result.Stdout)
	}
}

func TestStartAgent_WithPrompt(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-sp", "prompt-agent", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-sp/start", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req types.StartRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal start request: %v", err)
		}
		if req.Prompt != "fix the bug" {
			t.Errorf("expected prompt 'fix the bug', got %q", req.Prompt)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "sess-p"},
				Phase:           "Pending",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start", "prompt-agent", "--prompt", "fix the bug")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
}

func TestStartAgent_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-sj", "json-start", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-sj/start", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "sess-json"},
				Phase:           "Pending",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start", "json-start", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"sess-json"`) {
		t.Errorf("expected JSON with 'sess-json', got: %s", result.Stdout)
	}
}

func TestStartAgent_MissingArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start")
	if result.Err == nil {
		t.Fatal("expected error for missing agent argument")
	}
}

func TestStartAgent_AllFlag(t *testing.T) {
	srv := testhelper.NewServer(t)
	agents := []types.Agent{
		sampleAgent("a-all1", "agent-1", testhelper.TestProject),
		sampleAgent("a-all2", "agent-2", testhelper.TestProject),
	}
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			ListMeta: types.ListMeta{Total: 2},
			Items:    agents,
		})
	})
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-all1/start", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "s-1"},
				Phase:           "Pending",
			},
		})
	})
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-all2/start", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "s-2"},
				Phase:           "Pending",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start", "--all")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "s-1") || !strings.Contains(result.Stdout, "s-2") {
		t.Errorf("expected both sessions in output, got: %s", result.Stdout)
	}
}

func TestStartAgent_AllWithNameConflict(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start", "some-agent", "--all")
	if result.Err == nil {
		t.Fatal("expected error for --all with agent name")
	}
	if !strings.Contains(result.Err.Error(), "cannot specify agent name with --all") {
		t.Errorf("expected conflict error, got: %v", result.Err)
	}
}

func TestStopAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-stop", "stop-me", testhelper.TestProject)
	agent.CurrentSessionID = "sess-active"
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/sessions/sess-active", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "sess-active"},
			Phase:           "Running",
		})
	})
	srv.Handle("/api/ambient/v1/sessions/sess-active/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "sess-active"},
			Phase:           "Stopping",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "stop", "stop-me")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "agent/stop-me session/sess-active stopped") {
		t.Errorf("expected stop confirmation, got: %s", result.Stdout)
	}
}

func TestStopAgent_NoActiveSession(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-noss", "idle-agent", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "stop", "idle-agent")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "no active session") {
		t.Errorf("expected 'no active session' message, got: %s", result.Stdout)
	}
}

func TestStopAgent_MissingArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "stop")
	if result.Err == nil {
		t.Fatal("expected error for missing agent argument")
	}
}

func TestStopAgent_AllFlag(t *testing.T) {
	srv := testhelper.NewServer(t)
	agents := []types.Agent{
		{ObjectReference: types.ObjectReference{ID: "a-sa1", CreatedAt: &testTime}, Name: "agent-1", ProjectID: testhelper.TestProject, CurrentSessionID: "sess-a1"},
		{ObjectReference: types.ObjectReference{ID: "a-sa2", CreatedAt: &testTime}, Name: "agent-2", ProjectID: testhelper.TestProject},
	}
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			ListMeta: types.ListMeta{Total: 2},
			Items:    agents,
		})
	})
	srv.Handle("/api/ambient/v1/sessions/sess-a1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "sess-a1"},
			Phase:           "Running",
		})
	})
	srv.Handle("/api/ambient/v1/sessions/sess-a1/stop", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "sess-a1"},
			Phase:           "Stopping",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "stop", "--all")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "sess-a1 stopped") {
		t.Errorf("expected stop confirmation for sess-a1, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "no active session") {
		t.Errorf("expected 'no active session' for agent-2, got: %s", result.Stdout)
	}
}

func TestStartPreview_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-prev", "preview-agent", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-prev/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.StartResponse{
			StartingPrompt: "You are the lead agent. Your task is...",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "start-preview", "preview-agent")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "You are the lead agent") {
		t.Errorf("expected preview prompt, got: %s", result.Stdout)
	}
}

func TestSessions_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-ses", "session-agent", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-ses/sessions", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.SessionList{
			ListMeta: types.ListMeta{Total: 1},
			Items: []types.Session{
				{
					ObjectReference: types.ObjectReference{ID: "sess-hist", CreatedAt: &testTime},
					Name:            "run-1",
					Phase:           "Completed",
				},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "sessions", "session-agent")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "run-1") {
		t.Errorf("expected 'run-1' in output, got: %s", result.Stdout)
	}
}

func TestSessions_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	agent := sampleAgent("a-sj2", "json-ses", testhelper.TestProject)
	handleAgentLookup(t, srv, testhelper.TestProject, agent)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a-sj2/sessions", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.SessionList{
			Items: []types.Session{
				{
					ObjectReference: types.ObjectReference{ID: "sess-j"},
					Name:            "json-run",
					Phase:           "Running",
				},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "sessions", "json-ses", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-run"`) {
		t.Errorf("expected JSON with 'json-run', got: %s", result.Stdout)
	}
}
