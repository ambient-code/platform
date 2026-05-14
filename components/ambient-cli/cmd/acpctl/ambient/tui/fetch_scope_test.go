package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestFetchActiveView_SessionScopingByProject verifies that the sessions view
// respects project scope on poll refresh, not just on initial navigation.
// Regression test for: project number-key switch correctly fetched project-scoped
// sessions initially, but fetchActiveView() fell through to FetchAllSessions()
// on every subsequent tick because it required currentAgentID to be set.
func TestFetchActiveView_SessionScopingByProject(t *testing.T) {
	tests := []struct {
		name           string
		currentProject string
		currentAgentID string
		wantScoped     bool
	}{
		{
			name:           "project set via number key (no agent)",
			currentProject: "hyperloop",
			currentAgentID: "",
			wantScoped:     true,
		},
		{
			name:           "project and agent set (drill-down)",
			currentProject: "hyperloop",
			currentAgentID: "agent-123",
			wantScoped:     true,
		},
		{
			name:           "no project (global view)",
			currentProject: "",
			currentAgentID: "",
			wantScoped:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &scopeTrackingClient{}
			m := &AppModel{
				activeView:     "sessions",
				currentProject: tt.currentProject,
				currentAgentID: tt.currentAgentID,
				fetcher:        fake,
			}

			cmd := m.fetchActiveView()
			if cmd == nil {
				t.Fatal("fetchActiveView() returned nil")
			}
			cmd()

			if tt.wantScoped {
				if fake.lastFetchAll {
					t.Errorf("expected project-scoped fetch for %q, got FetchAllSessions", tt.currentProject)
				}
				if fake.lastFetchProject != tt.currentProject {
					t.Errorf("expected fetch for project %q, got %q", tt.currentProject, fake.lastFetchProject)
				}
			} else {
				if !fake.lastFetchAll {
					t.Errorf("expected FetchAllSessions, got project-scoped fetch for %q", fake.lastFetchProject)
				}
			}
		})
	}
}

// TestFetchActiveView_ScheduledSessionScopingByProject verifies scheduled
// sessions respect project scope on refresh.
func TestFetchActiveView_ScheduledSessionScopingByProject(t *testing.T) {
	t.Run("with project", func(t *testing.T) {
		fake := &scopeTrackingClient{}
		m := &AppModel{
			activeView:     "scheduledsessions",
			currentProject: "hyperloop",
			fetcher:        fake,
		}
		cmd := m.fetchActiveView()
		if cmd == nil {
			t.Fatal("fetchActiveView() returned nil")
		}
		cmd()
		if fake.lastFetchProject != "hyperloop" {
			t.Errorf("expected fetch for project %q, got %q", "hyperloop", fake.lastFetchProject)
		}
	})

	t.Run("no project returns nil", func(t *testing.T) {
		fake := &scopeTrackingClient{}
		m := &AppModel{
			activeView:     "scheduledsessions",
			currentProject: "",
			fetcher:        fake,
		}
		cmd := m.fetchActiveView()
		if cmd != nil {
			t.Error("expected nil command for scheduledsessions with no project")
		}
	})
}

// TestFetchActiveView_AgentsScopingByProject verifies agents view respects
// project scope on refresh.
func TestFetchActiveView_AgentsScopingByProject(t *testing.T) {
	fake := &scopeTrackingClient{}
	m := &AppModel{
		activeView:     "agents",
		currentProject: "hyperloop",
		fetcher:        fake,
	}
	cmd := m.fetchActiveView()
	if cmd == nil {
		t.Fatal("fetchActiveView() returned nil")
	}
	cmd()
	if fake.lastFetchProject != "hyperloop" {
		t.Errorf("expected agents fetch for project %q, got %q", "hyperloop", fake.lastFetchProject)
	}
}

// scopeTrackingClient records which fetch method was called and with what scope.
type scopeTrackingClient struct {
	lastFetchProject string
	lastFetchAll     bool
}

var _ dataFetcher = (*scopeTrackingClient)(nil)

func (c *scopeTrackingClient) FetchProjects() tea.Cmd {
	return func() tea.Msg { return ProjectsMsg{} }
}

func (c *scopeTrackingClient) FetchAgents(projectID string) tea.Cmd {
	c.lastFetchProject = projectID
	return func() tea.Msg { return AgentsMsg{} }
}

func (c *scopeTrackingClient) FetchSessions(projectID string) tea.Cmd {
	c.lastFetchProject = projectID
	c.lastFetchAll = false
	return func() tea.Msg { return SessionsMsg{} }
}

func (c *scopeTrackingClient) FetchAllSessions() tea.Cmd {
	c.lastFetchAll = true
	c.lastFetchProject = ""
	return func() tea.Msg { return SessionsMsg{} }
}

func (c *scopeTrackingClient) FetchScheduledSessions(projectID string) tea.Cmd {
	c.lastFetchProject = projectID
	return func() tea.Msg { return ScheduledSessionsMsg{} }
}

func (c *scopeTrackingClient) FetchInbox(projectID, agentID string) tea.Cmd {
	c.lastFetchProject = projectID
	return func() tea.Msg { return InboxMsg{} }
}
