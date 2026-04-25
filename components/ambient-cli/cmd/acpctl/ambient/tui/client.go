package tui

import (
	"context"
	"fmt"
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

// ProjectCounts holds agent and session counts for a single project.
type ProjectCounts struct {
	AgentCount   int
	SessionCount int
}

// ProjectCountsMsg carries per-project agent and session counts keyed by
// project name. Sent after a background fan-out fetch completes.
type ProjectCountsMsg struct {
	Counts map[string]ProjectCounts
	Err    error
}

// AgentCounts holds the session count for a single agent.
type AgentCounts struct {
	SessionCount int
}

// AgentCountsMsg carries per-agent session counts keyed by agent ID.
// Sent after a background fan-out fetch completes.
type AgentCountsMsg struct {
	Counts map[string]AgentCounts
	Err    error
}

// ---------------------------------------------------------------------------
// CRUD message types for mutating operations.
// ---------------------------------------------------------------------------

// StartAgentMsg carries the result of starting an agent.
type StartAgentMsg struct {
	Response *sdktypes.StartResponse
	Err      error
}

// StopAgentMsg carries the result of stopping an agent's current session.
// The SDK has no AgentAPI.Stop — stopping an agent means stopping its current
// session via SessionAPI.Stop. The caller must resolve the agent's
// current_session_id before calling StopAgent.
type StopAgentMsg struct {
	Session *sdktypes.Session
	Err     error
}

// CreateAgentMsg carries the result of creating an agent.
type CreateAgentMsg struct {
	Agent *sdktypes.Agent
	Err   error
}

// UpdateAgentMsg carries the result of patching an agent.
type UpdateAgentMsg struct {
	Agent *sdktypes.Agent
	Err   error
}

// DeleteAgentMsg carries the result of deleting an agent.
type DeleteAgentMsg struct {
	Err error
}

// CreateProjectMsg carries the result of creating a project.
type CreateProjectMsg struct {
	Project *sdktypes.Project
	Err     error
}

// UpdateProjectMsg carries the result of patching a project.
type UpdateProjectMsg struct {
	Project *sdktypes.Project
	Err     error
}

// DeleteProjectMsg carries the result of deleting a project.
type DeleteProjectMsg struct {
	Err error
}

// UpdateSessionMsg carries the result of patching a session.
type UpdateSessionMsg struct {
	Session *sdktypes.Session
	Err     error
}

// DeleteSessionMsg carries the result of deleting a session.
type DeleteSessionMsg struct {
	Err error
}

// SendMessageMsg carries the result of sending a message to a session.
type SendMessageMsg struct {
	Message *sdktypes.SessionMessage
	Err     error
}

// SendInboxMsg carries the result of sending an inbox message to an agent.
type SendInboxMsg struct {
	Message *sdktypes.InboxMessage
	Err     error
}

// MarkInboxReadMsg carries the result of marking an inbox message as read.
type MarkInboxReadMsg struct {
	Err error
}

// DeleteInboxMsg carries the result of deleting an inbox message.
type DeleteInboxMsg struct {
	Err error
}

// SessionMessageEvent carries a single session message received from an SSE
// stream. Sent to the Bubbletea program via program.Send().
type SessionMessageEvent struct {
	Message *sdktypes.SessionMessage
	Err     error
}

// SessionMessagesMsg carries a batch of messages fetched via polling
// (ListMessages). Used as a fallback when SSE is unavailable or stalled.
type SessionMessagesMsg struct {
	Messages []sdktypes.SessionMessage
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

	// watchMu protects watchCancel.
	watchMu     sync.Mutex
	watchCancel context.CancelFunc
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

// FetchProjectCounts returns a tea.Cmd that fans out per-project agent and
// session list fetches and returns a ProjectCountsMsg with the counts. Partial
// failures are tolerated — failed projects get count -1 for both fields.
func (tc *TUIClient) FetchProjectCounts(projects []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		var (
			mu     sync.Mutex
			counts = make(map[string]ProjectCounts, len(projects))
			wg     sync.WaitGroup
		)

		for _, proj := range projects {
			wg.Add(1)
			go func() {
				defer wg.Done()

				client, err := tc.factory.ForProject(proj)
				if err != nil {
					mu.Lock()
					counts[proj] = ProjectCounts{AgentCount: -1, SessionCount: -1}
					mu.Unlock()
					return
				}

				var ac, sc int

				agentList, err := client.Agents().List(ctx, defaultListOpts())
				if err != nil {
					ac = -1
				} else {
					ac = len(agentList.Items)
				}

				sessionList, err := client.Sessions().List(ctx, defaultListOpts())
				if err != nil {
					sc = -1
				} else {
					sc = len(sessionList.Items)
				}

				mu.Lock()
				counts[proj] = ProjectCounts{AgentCount: ac, SessionCount: sc}
				mu.Unlock()
			}()
		}

		wg.Wait()
		return ProjectCountsMsg{Counts: counts}
	}
}

// FetchAgentCounts returns a tea.Cmd that fans out per-agent session list
// fetches and returns an AgentCountsMsg with the counts. Uses the
// AgentAPI.Sessions() endpoint to count sessions per agent. Partial failures
// are tolerated — failed agents get count -1.
func (tc *TUIClient) FetchAgentCounts(projectID string, agentIDs []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		var (
			mu     sync.Mutex
			counts = make(map[string]AgentCounts, len(agentIDs))
			wg     sync.WaitGroup
		)

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return AgentCountsMsg{Err: err}
		}

		for _, agentID := range agentIDs {
			wg.Add(1)
			go func() {
				defer wg.Done()

				sessionList, err := client.Agents().Sessions(ctx, projectID, agentID, defaultListOpts())
				sc := -1
				if err == nil {
					sc = len(sessionList.Items)
				}

				mu.Lock()
				counts[agentID] = AgentCounts{SessionCount: sc}
				mu.Unlock()
			}()
		}

		wg.Wait()
		return AgentCountsMsg{Counts: counts}
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

// ---------------------------------------------------------------------------
// Agent CRUD
// ---------------------------------------------------------------------------

// StartAgent returns a tea.Cmd that starts an agent by calling
// POST /projects/{projectID}/agents/{agentID}/start with the given prompt.
func (tc *TUIClient) StartAgent(projectID, agentID, prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return StartAgentMsg{Err: err}
		}

		resp, err := client.Agents().Start(ctx, projectID, agentID, prompt)
		if err != nil {
			return StartAgentMsg{Err: err}
		}
		return StartAgentMsg{Response: resp}
	}
}

// StopAgent returns a tea.Cmd that stops an agent's current session.
// The SDK has no AgentAPI.Stop method. Stopping an agent is done by stopping
// its current session via SessionAPI.Stop. The caller must provide the
// session ID (from agent.CurrentSessionID).
func (tc *TUIClient) StopAgent(projectID, sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return StopAgentMsg{Err: err}
		}

		session, err := client.Sessions().Stop(ctx, sessionID)
		if err != nil {
			return StopAgentMsg{Err: err}
		}
		return StopAgentMsg{Session: session}
	}
}

// CreateAgent returns a tea.Cmd that creates a new agent in the given project.
func (tc *TUIClient) CreateAgent(projectID, name, prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return CreateAgentMsg{Err: err}
		}

		agent := &sdktypes.Agent{
			Name:      name,
			ProjectID: projectID,
			Prompt:    prompt,
		}

		result, err := client.Agents().CreateInProject(ctx, projectID, agent)
		if err != nil {
			return CreateAgentMsg{Err: err}
		}
		return CreateAgentMsg{Agent: result}
	}
}

// UpdateAgent returns a tea.Cmd that patches an agent with the given fields.
func (tc *TUIClient) UpdateAgent(projectID, agentID string, patch map[string]any) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return UpdateAgentMsg{Err: err}
		}

		result, err := client.Agents().UpdateInProject(ctx, projectID, agentID, patch)
		if err != nil {
			return UpdateAgentMsg{Err: err}
		}
		return UpdateAgentMsg{Agent: result}
	}
}

// DeleteAgent returns a tea.Cmd that deletes an agent from the given project.
func (tc *TUIClient) DeleteAgent(projectID, agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return DeleteAgentMsg{Err: err}
		}

		err = client.Agents().DeleteInProject(ctx, projectID, agentID)
		return DeleteAgentMsg{Err: err}
	}
}

// ---------------------------------------------------------------------------
// Project CRUD
// ---------------------------------------------------------------------------

// CreateProject returns a tea.Cmd that creates a new project.
func (tc *TUIClient) CreateProject(name, description string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		// Projects are a global resource; any project-scoped client can
		// create them. Use a minimal project name for the SDK constructor.
		client, err := tc.factory.ForProject("_")
		if err != nil {
			return CreateProjectMsg{Err: err}
		}

		proj := &sdktypes.Project{
			Name:        name,
			Description: description,
		}

		result, err := client.Projects().Create(ctx, proj)
		if err != nil {
			return CreateProjectMsg{Err: err}
		}
		return CreateProjectMsg{Project: result}
	}
}

// UpdateProject returns a tea.Cmd that patches a project with the given fields.
func (tc *TUIClient) UpdateProject(projectID string, patch map[string]any) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject("_")
		if err != nil {
			return UpdateProjectMsg{Err: err}
		}

		result, err := client.Projects().Update(ctx, projectID, patch)
		if err != nil {
			return UpdateProjectMsg{Err: err}
		}
		return UpdateProjectMsg{Project: result}
	}
}

// DeleteProject returns a tea.Cmd that deletes a project by ID.
func (tc *TUIClient) DeleteProject(projectID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject("_")
		if err != nil {
			return DeleteProjectMsg{Err: err}
		}

		err = client.Projects().Delete(ctx, projectID)
		return DeleteProjectMsg{Err: err}
	}
}

// ---------------------------------------------------------------------------
// Session operations
// ---------------------------------------------------------------------------

// UpdateSession returns a tea.Cmd that patches a session with the given fields.
func (tc *TUIClient) UpdateSession(projectID, sessionID string, patch map[string]any) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return UpdateSessionMsg{Err: err}
		}

		result, err := client.Sessions().Update(ctx, sessionID, patch)
		if err != nil {
			return UpdateSessionMsg{Err: err}
		}
		return UpdateSessionMsg{Session: result}
	}
}

// DeleteSession returns a tea.Cmd that deletes a session by ID.
func (tc *TUIClient) DeleteSession(projectID, sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return DeleteSessionMsg{Err: err}
		}

		err = client.Sessions().Delete(ctx, sessionID)
		return DeleteSessionMsg{Err: err}
	}
}

// SendSessionMessage returns a tea.Cmd that sends a user message to a
// session. This supports the "Send-While-Streaming" pattern: the call is
// non-blocking and the message appears in the SSE stream when the server
// echoes it back.
func (tc *TUIClient) SendSessionMessage(projectID, sessionID, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return SendMessageMsg{Err: err}
		}

		msg, err := client.Sessions().PushMessage(ctx, sessionID, body)
		if err != nil {
			return SendMessageMsg{Err: err}
		}
		return SendMessageMsg{Message: msg}
	}
}

// ---------------------------------------------------------------------------
// Inbox operations
// ---------------------------------------------------------------------------

// SendInboxMessage returns a tea.Cmd that sends an inbox message to an agent.
func (tc *TUIClient) SendInboxMessage(projectID, agentID, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return SendInboxMsg{Err: err}
		}

		msg := &sdktypes.InboxMessage{
			AgentID: agentID,
			Body:    body,
		}

		result, err := client.InboxMessages().Send(ctx, projectID, agentID, msg)
		if err != nil {
			return SendInboxMsg{Err: err}
		}
		return SendInboxMsg{Message: result}
	}
}

// MarkInboxRead returns a tea.Cmd that marks an inbox message as read.
func (tc *TUIClient) MarkInboxRead(projectID, agentID, msgID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return MarkInboxReadMsg{Err: err}
		}

		err = client.InboxMessages().MarkRead(ctx, projectID, agentID, msgID)
		return MarkInboxReadMsg{Err: err}
	}
}

// DeleteInboxMessage returns a tea.Cmd that deletes an inbox message.
func (tc *TUIClient) DeleteInboxMessage(projectID, agentID, msgID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return DeleteInboxMsg{Err: err}
		}

		err = client.InboxMessages().DeleteMessage(ctx, projectID, agentID, msgID)
		return DeleteInboxMsg{Err: err}
	}
}

// ---------------------------------------------------------------------------
// SSE streaming
// ---------------------------------------------------------------------------

// WatchSessionMessages returns a tea.Cmd that starts an SSE stream for
// session messages. Messages are delivered to the Bubbletea program via
// program.Send(SessionMessageEvent{...}).
//
// The SSE goroutine:
//   - Connects to GET /sessions/{id}/messages via the SDK's WatchMessages.
//   - Forwards each message as a SessionMessageEvent to the program.
//   - Handles reconnection with exponential backoff (1s, 2s, 4s, max 30s)
//     internally via the SDK's WatchMessages implementation.
//   - Is cancellable via StopWatching().
//   - Sends an error event if the channel closes without context cancellation,
//     signalling a silent SSE failure so the TUI can fall back to polling.
//
// Only one watch can be active at a time. Calling WatchSessionMessages while
// a previous watch is running cancels the old one first.
func (tc *TUIClient) WatchSessionMessages(projectID, sessionID string, afterSeq int, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		// Cancel any previously active watch.
		tc.StopWatching()

		ctx, cancel := context.WithCancel(context.Background())

		tc.watchMu.Lock()
		tc.watchCancel = cancel
		tc.watchMu.Unlock()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			cancel()
			program.Send(SessionMessageEvent{Err: err})
			return nil
		}

		// The SDK's WatchMessages handles SSE connection, parsing, and
		// reconnection with exponential backoff (1s, 2s, 4s, max 30s).
		// It returns a channel of *SessionMessage and a stop function.
		msgs, _, sseErr := client.Sessions().WatchMessages(ctx, sessionID, afterSeq)
		if sseErr != nil {
			cancel()
			program.Send(SessionMessageEvent{Err: sseErr})
			return nil
		}

		// Forward messages from the SDK channel to the Bubbletea program.
		// This goroutine exits when the channel closes (on context
		// cancellation or stream end). If the channel closes without
		// cancellation, it means the SSE stream died silently -- notify
		// the TUI so it can fall back to polling.
		go func() {
			defer cancel()
			receivedAny := false
			for msg := range msgs {
				receivedAny = true
				program.Send(SessionMessageEvent{Message: msg})
			}
			// Channel closed. If the context was not cancelled by us
			// (StopWatching or view change), this is an unexpected close.
			if ctx.Err() == nil {
				errMsg := "SSE stream closed"
				if !receivedAny {
					errMsg = "SSE connection failed (no messages received)"
				}
				program.Send(SessionMessageEvent{
					Err: fmt.Errorf("%s — falling back to polling", errMsg),
				})
			}
		}()

		return nil
	}
}

// FetchSessionMessages returns a tea.Cmd that polls session messages via the
// REST ListMessages endpoint. This is used as a fallback when SSE streaming is
// unavailable or stalled. Only messages with seq > afterSeq are returned.
func (tc *TUIClient) FetchSessionMessages(projectID, sessionID string, afterSeq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		client, err := tc.factory.ForProject(projectID)
		if err != nil {
			return SessionMessagesMsg{Err: err}
		}

		msgs, err := client.Sessions().ListMessages(ctx, sessionID, afterSeq)
		if err != nil {
			return SessionMessagesMsg{Err: err}
		}
		return SessionMessagesMsg{Messages: msgs}
	}
}

// StopWatching cancels any active SSE watch goroutine started by
// WatchSessionMessages.
func (tc *TUIClient) StopWatching() {
	tc.watchMu.Lock()
	defer tc.watchMu.Unlock()

	if tc.watchCancel != nil {
		tc.watchCancel()
		tc.watchCancel = nil
	}
}
