package tui

import (
	"context"
	"sync"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
)

// fetchTimeout is the per-request context deadline for all API fetches.
const fetchTimeout = 15 * time.Second

// defaultListOpts returns the standard list options for TUI fetches.
func defaultListOpts() *sdktypes.ListOptions {
	return &sdktypes.ListOptions{Page: 1, Size: 200}
}

// ---------------------------------------------------------------------------
// Message types returned by TUIClient methods. Each carries the fetched data
// and any error encountered. The TUI's Update loop dispatches on these.
// ---------------------------------------------------------------------------

// ProjectsMsg carries the result of a project list fetch.
type ProjectsMsg struct {
	Projects []sdktypes.Project
	Err      error
}

// AgentsMsg carries the result of an agent list fetch.
type AgentsMsg struct {
	Agents []sdktypes.Agent
	Err    error
}

// SessionsMsg carries the result of a session list fetch (single- or
// multi-project).
type SessionsMsg struct {
	Sessions []sdktypes.Session
	Err      error
}

// InboxMsg carries the result of an inbox message list fetch.
type InboxMsg struct {
	Messages []sdktypes.InboxMessage
	Err      error
}

// ---------------------------------------------------------------------------
// TUIClient wraps connection.ClientFactory and provides clean data-fetching
// methods that return tea.Cmd functions for asynchronous execution inside the
// Bubbletea runtime. Every method creates its own context with fetchTimeout
// so the Update loop is never blocked.
//
// All data flows through the Ambient API Server -- no kubectl, no direct K8s
// API calls.
// ---------------------------------------------------------------------------

// TUIClient is the API client layer for the TUI. It creates per-project SDK
// clients via a ClientFactory and returns bubbletea Cmds that fetch data
// asynchronously.
type TUIClient struct {
	factory *connection.ClientFactory
}

// NewTUIClient creates a TUIClient from the given ClientFactory.
func NewTUIClient(factory *connection.ClientFactory) *TUIClient {
	return &TUIClient{factory: factory}
}

// FetchProjects returns a tea.Cmd that lists all projects visible to the
// authenticated user.
func (tc *TUIClient) FetchProjects() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		// Projects are a global resource; any project-scoped client can list
		// them. Use a minimal project name to satisfy the SDK constructor.
		client, err := tc.factory.ForProject("_")
		if err != nil {
			return ProjectsMsg{Err: err}
		}

		list, err := client.Projects().List(ctx, defaultListOpts())
		if err != nil {
			return ProjectsMsg{Err: err}
		}
		return ProjectsMsg{Projects: list.Items}
	}
}

// FetchAgents returns a tea.Cmd that lists agents in the given project.
func (tc *TUIClient) FetchAgents(projectID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return AgentsMsg{Err: err}
		}

		list, err := client.Agents().List(ctx, defaultListOpts())
		if err != nil {
			return AgentsMsg{Err: err}
		}
		return AgentsMsg{Agents: list.Items}
	}
}

// FetchSessions returns a tea.Cmd that lists sessions scoped to a single
// project. Use FetchAllSessions for the cross-project fan-out pattern.
func (tc *TUIClient) FetchSessions(projectID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return SessionsMsg{Err: err}
		}

		list, err := client.Sessions().List(ctx, defaultListOpts())
		if err != nil {
			return SessionsMsg{Err: err}
		}
		return SessionsMsg{Sessions: list.Items}
	}
}

// FetchAllSessions returns a tea.Cmd that lists sessions across all projects.
// It first fetches the project list, then fans out one goroutine per project
// to fetch sessions concurrently -- the same pattern used in fetchAll in
// fetch.go. Partial failures are collected; the first error is reported while
// successfully-fetched sessions are still returned.
func (tc *TUIClient) FetchAllSessions() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		// Step 1: list all projects.
		anyClient, err := tc.factory.ForProject("_")
		if err != nil {
			return SessionsMsg{Err: err}
		}

		projList, err := anyClient.Projects().List(ctx, defaultListOpts())
		if err != nil {
			return SessionsMsg{Err: err}
		}

		// Step 2: fan out per-project session fetches.
		var (
			mu          sync.Mutex
			allSessions []sdktypes.Session
			firstErr    error
			wg          sync.WaitGroup
		)

		for _, proj := range projList.Items {
			wg.Add(1)
			go func() {
				defer wg.Done()

				projClient, err := tc.factory.ForProject(proj.Name)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}

				list, err := projClient.Sessions().List(ctx, defaultListOpts())
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}

				mu.Lock()
				allSessions = append(allSessions, list.Items...)
				mu.Unlock()
			}()
		}

		wg.Wait()
		return SessionsMsg{Sessions: allSessions, Err: firstErr}
	}
}

// FetchInbox returns a tea.Cmd that lists inbox messages for a specific agent
// within a project. The SDK's InboxMessageAPI.ListByAgent is used to hit
// the /projects/{projectID}/agents/{agentID}/inbox endpoint.
func (tc *TUIClient) FetchInbox(projectID, agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return InboxMsg{Err: err}
		}

		list, err := client.InboxMessages().ListByAgent(ctx, projectID, agentID, defaultListOpts())
		if err != nil {
			return InboxMsg{Err: err}
		}
		return InboxMsg{Messages: list.Items}
	}
}
