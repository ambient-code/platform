package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// pollInterval is the auto-refresh interval for resource tables.
const pollInterval = 5 * time.Second

// messagePollInterval is the polling interval for session messages when the
// messages view is active. Faster than the table poll to keep messages fresh.
const messagePollInterval = 2 * time.Second

// infoTimeout is how long ephemeral info messages are displayed.
const infoTimeout = 5 * time.Second

// staleThreshold marks data as stale in the header when exceeded.
const staleThreshold = 15 * time.Second

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

// NavEntry represents a single level in the navigation stack.
type NavEntry struct {
	Kind  string // "projects", "agents", "sessions", "messages", "inbox"
	Scope string // project name, agent name, etc.
	ID    string // resource ID if applicable
}

// ---------------------------------------------------------------------------
// Message types (prefixed with "app" to avoid collision with model.go types)
// ---------------------------------------------------------------------------

// appTickMsg fires every pollInterval to trigger data refresh.
type appTickMsg struct{ t time.Time }

// messagePollTickMsg fires every messagePollInterval when the messages view is
// active, triggering a REST poll for new session messages.
type messagePollTickMsg struct{ t time.Time }

// infoExpiredMsg signals the ephemeral info line should be cleared.
type infoExpiredMsg struct{}

// editCompleteMsg is sent when the user's $EDITOR exits after editing a
// resource as JSON. The handler reads the temp file, diffs against the
// original, and PATCHes any changed fields.
type editCompleteMsg struct {
	ResourceKind string // "agent", "project", "session"
	ResourceID   string // ID of the resource being edited
	ProjectID    string // project scope (for agents/sessions)
	TempFile     string // path to the temp file containing edited JSON
	OriginalJSON []byte // original JSON before editing (for diffing)
	Err          error  // non-nil if the editor process failed
}

// getEditor returns the user's preferred editor command by checking $EDITOR,
// then $VISUAL, falling back to "vi".
func getEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "vi"
}

// ---------------------------------------------------------------------------
// AppModel — the TUI model with full navigation hierarchy
// ---------------------------------------------------------------------------

// AppModel is the top-level Bubbletea model for the rewritten TUI.
// It coexists with the legacy Model type in model.go until migration is
// complete.
type AppModel struct {
	// Config
	config *TUIConfig
	client *TUIClient

	// Navigation
	navStack []NavEntry // stack of views; rightmost is current

	// Tables for each resource view
	projectTable  views.ResourceTable
	agentTable    views.ResourceTable
	sessionTable  views.ResourceTable
	inboxTable    views.ResourceTable
	contextTable  views.ResourceTable
	messageStream views.MessageStream

	// Current view determines which table/view is active
	activeView string // "projects", "agents", "sessions", "messages", "inbox", "contexts"

	// Context for scoped views
	currentProject string // set when drilling into a project
	currentAgent   string // set when drilling into an agent (name)
	currentAgentID string // agent ID for API calls
	currentSession string // set when drilling into a session

	// Command mode
	commandMode  bool
	commandInput textinput.Model

	// Filter mode
	filterMode   bool
	filterInput  textinput.Model
	activeFilter *Filter

	// Polling
	pollInFlight bool
	lastFetch    time.Time

	// Info line (ephemeral toast)
	infoMessage string
	infoExpiry  time.Time

	// Detail view
	detailView views.DetailView

	// Help overlay
	helpView views.HelpView

	// Cached resource data for CRUD lookups (maps name/ID -> full resource).
	cachedProjects []sdktypes.Project
	cachedAgents   []sdktypes.Agent
	cachedSessions []sdktypes.Session
	cachedInbox    []sdktypes.InboxMessage

	// SSE program reference (set via SetProgram after tea.NewProgram).
	program *tea.Program

	// Message polling state (fallback when SSE is unavailable).
	lastMessageSeq   int  // highest seq seen — poll for messages after this
	messagePollActive bool // true when message poll tick is running

	// Errors
	lastError string

	// Dialog overlay (replaces inline delete confirmation and prompt mode for new resources).
	dialog       *views.Dialog
	dialogAction func() tea.Cmd // executed on DialogConfirmMsg{Confirmed: true}

	// Rate-limit backoff: skip the next poll cycle when a 429 is received.
	skipNextPoll bool

	// Project shortcuts for number-key switching (like k9s namespace shortcuts).
	// Holds project names in alphabetical order, refreshed on ProjectsMsg.
	projectShortcuts []string

	// Prompt mode for inline text input (e.g. new session prompt).
	promptMode     bool
	promptInput    textinput.Model
	promptCallback func(string) (tea.Model, tea.Cmd) // called on Enter

	// Terminal size
	width, height int
}

// NewAppModel creates a new AppModel. It loads config, creates the API client,
// and initialises sub-components. The caller (cmd.go) passes the ClientFactory
// obtained from connection.NewClientFactory().
func NewAppModel(factory *connection.ClientFactory) (*AppModel, error) {
	cfg, err := LoadTUIConfig()
	if err != nil {
		return nil, fmt.Errorf("load TUI config: %w", err)
	}

	client := NewTUIClient(factory)

	// Command bar input.
	ci := textinput.New()
	ci.Prompt = ":"
	ci.CharLimit = 256
	ci.ShowSuggestions = true

	// Filter bar input.
	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 256

	// Prompt bar input (for inline prompts like new session).
	pi := textinput.New()
	pi.Prompt = "Session prompt: "
	pi.CharLimit = 1024

	pt := views.NewProjectTable(views.DefaultTableStyle())
	// Project rows: STATUS is column index 2 (NAME, DESCRIPTION, STATUS, AGENTS, SESSIONS, AGE)
	pt.SetRowColorFunc(func(row table.Row) lipgloss.Color {
		if len(row) > 2 {
			return views.PhaseColor(row[2])
		}
		return lipgloss.Color("240")
	})
	at := views.NewAgentTable("all", views.DefaultTableStyle())
	// Agent rows: PHASE is column index 3 (NAME, PROMPT, SESSIONS, PHASE, AGE)
	at.SetRowColorFunc(func(row table.Row) lipgloss.Color {
		if len(row) > 3 {
			return views.PhaseColor(row[3])
		}
		return lipgloss.Color("240")
	})
	st := views.NewSessionTable("all", views.DefaultTableStyle())
	// Session rows: PHASE is column index 4 (ID, NAME, AGENT, PROJECT, PHASE, ...)
	st.SetRowColorFunc(func(row table.Row) lipgloss.Color {
		if len(row) > 4 {
			return views.PhaseColor(row[4])
		}
		return lipgloss.Color("240")
	})
	it := views.NewInboxTable("all", views.DefaultTableStyle())
	ct := views.NewContextTable(views.DefaultTableStyle())

	m := &AppModel{
		config: cfg,
		client: client,
		navStack: []NavEntry{
			{Kind: "projects", Scope: "all"},
		},
		activeView:   "projects",
		projectTable: pt,
		agentTable:   at,
		sessionTable: st,
		inboxTable:   it,
		contextTable: ct,
		commandInput: ci,
		filterInput:  fi,
		promptInput:  pi,
	}

	return m, nil
}

// SetProgram stores a reference to the tea.Program so the model can pass it to
// WatchSessionMessages for SSE delivery. Call this after tea.NewProgram returns.
func (m *AppModel) SetProgram(p *tea.Program) {
	m.program = p
}

// findAgentByName returns the cached Agent with the given name, or nil.
func (m *AppModel) findAgentByName(name string) *sdktypes.Agent {
	for i := range m.cachedAgents {
		if m.cachedAgents[i].Name == name {
			return &m.cachedAgents[i]
		}
	}
	return nil
}

// findProjectByName returns the cached Project with the given name, or nil.
func (m *AppModel) findProjectByName(name string) *sdktypes.Project {
	for i := range m.cachedProjects {
		if m.cachedProjects[i].Name == name {
			return &m.cachedProjects[i]
		}
	}
	return nil
}

// findSessionByShortID returns the cached Session whose ID starts with the given
// short ID prefix, or nil.
func (m *AppModel) findSessionByShortID(shortID string) *sdktypes.Session {
	for i := range m.cachedSessions {
		if m.cachedSessions[i].ID == shortID || (len(m.cachedSessions[i].ID) >= len(shortID) && m.cachedSessions[i].ID[:len(shortID)] == shortID) {
			return &m.cachedSessions[i]
		}
	}
	return nil
}

// findInboxByID returns the cached InboxMessage with the given ID, or nil.
func (m *AppModel) findInboxByID(id string) *sdktypes.InboxMessage {
	for i := range m.cachedInbox {
		if m.cachedInbox[i].ID == id {
			return &m.cachedInbox[i]
		}
	}
	return nil
}

// Init implements tea.Model. It returns a batch of initial commands:
// window size query, first data fetch, and the periodic tick.
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		m.client.FetchProjects(),
		m.tickCmd(),
	)
}

// tickCmd returns a tea.Cmd that sends an appTickMsg after pollInterval.
func (m *AppModel) tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return appTickMsg{t: t}
	})
}

// messagePollTickCmd returns a tea.Cmd that sends a messagePollTickMsg after
// messagePollInterval. Used to drive the REST polling fallback.
func (m *AppModel) messagePollTickCmd() tea.Cmd {
	return tea.Tick(messagePollInterval, func(t time.Time) tea.Msg {
		return messagePollTickMsg{t: t}
	})
}

// infoExpireCmd returns a tea.Cmd that clears the info line after infoTimeout.
func (m *AppModel) infoExpireCmd() tea.Cmd {
	return tea.Tick(infoTimeout, func(_ time.Time) tea.Msg {
		return infoExpiredMsg{}
	})
}

// setInfo sets an ephemeral info message and returns the expiry command.
func (m *AppModel) setInfo(msg string) tea.Cmd {
	m.infoMessage = msg
	m.infoExpiry = time.Now().Add(infoTimeout)
	return m.infoExpireCmd()
}

// currentNav returns the current (topmost) navigation entry.
func (m *AppModel) currentNav() NavEntry {
	if len(m.navStack) == 0 {
		return NavEntry{Kind: "projects", Scope: "all"}
	}
	return m.navStack[len(m.navStack)-1]
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

// pushView pushes a new navigation entry, switches to the target view, and
// returns a fetch command for the new view's data.
func (m *AppModel) pushView(kind, scope, id string) tea.Cmd {
	m.navStack = append(m.navStack, NavEntry{Kind: kind, Scope: scope, ID: id})
	m.activeView = kind
	m.activeFilter = nil
	m.pollInFlight = true
	return m.fetchActiveView()
}

// popView pops the current navigation entry, switches back to the parent view,
// and returns a fetch command to refresh the parent data.
func (m *AppModel) popView() tea.Cmd {
	if len(m.navStack) <= 1 {
		return nil
	}
	m.navStack = m.navStack[:len(m.navStack)-1]
	nav := m.currentNav()
	m.activeView = nav.Kind
	m.activeFilter = nil

	// Restore context based on what we popped back to.
	switch nav.Kind {
	case "projects":
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
	case "agents":
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
	case "sessions":
		m.currentSession = ""
	}

	m.pollInFlight = true
	return m.fetchActiveView()
}

// fetchActiveView returns a tea.Cmd to fetch data for the currently active view.
func (m *AppModel) fetchActiveView() tea.Cmd {
	switch m.activeView {
	case "projects":
		return m.client.FetchProjects()
	case "agents":
		if m.currentProject != "" {
			return m.client.FetchAgents(m.currentProject)
		}
		// Fall back to config project if no drill-down context.
		if ctx := m.config.Current(); ctx != nil && ctx.Project != "" {
			return m.client.FetchAgents(ctx.Project)
		}
		return nil
	case "sessions":
		if m.currentAgentID != "" && m.currentProject != "" {
			// Agent-scoped sessions — fetch project sessions and filter client-side
			// in the handler.
			return m.client.FetchSessions(m.currentProject)
		}
		// Global sessions view.
		return m.client.FetchAllSessions()
	case "inbox":
		if m.currentAgentID != "" && m.currentProject != "" {
			return m.client.FetchInbox(m.currentProject, m.currentAgentID)
		}
		return nil
	case "messages":
		// Message stream uses SSE, not polling. No fetch command needed yet.
		return nil
	default:
		return nil
	}
}

// activeTable returns a pointer to the currently active ResourceTable, or nil
// for the message stream and detail views.
func (m *AppModel) activeTable() *views.ResourceTable {
	switch m.activeView {
	case "projects":
		return &m.projectTable
	case "agents":
		return &m.agentTable
	case "sessions":
		return &m.sessionTable
	case "inbox":
		return &m.inboxTable
	case "contexts":
		return &m.contextTable
	default:
		return nil
	}
}

// populateContextTable fills the context table from config.
func (m *AppModel) populateContextTable() {
	names := m.config.ContextNames()
	rows := make([]table.Row, 0, len(names))
	for _, name := range names {
		ctx := m.config.Contexts[name]
		if ctx == nil {
			continue
		}
		active := name == m.config.CurrentContext
		rows = append(rows, views.ContextRow(name, ctx.Server, ctx.Project, active))
	}
	m.contextTable.SetRows(rows)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update implements tea.Model. It dispatches messages to the appropriate
// handler based on the current mode and message type.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTable()
		return m, nil

	case tea.MouseMsg:
		// Delegate scroll events to the active table, message stream, or detail view.
		if m.activeView == "messages" {
			var cmd tea.Cmd
			m.messageStream, cmd = m.messageStream.Update(msg)
			return m, cmd
		}
		if m.activeView == "detail" {
			var cmd tea.Cmd
			m.detailView, cmd = m.detailView.Update(msg)
			return m, cmd
		}
		if tbl := m.activeTable(); tbl != nil {
			var cmd tea.Cmd
			*tbl, cmd = tbl.Update(msg)
			return m, cmd
		}
		return m, nil

	case ProjectsMsg:
		return m.handleProjectsMsg(msg)

	case ProjectCountsMsg:
		return m.handleProjectCountsMsg(msg)

	case AgentsMsg:
		return m.handleAgentsMsg(msg)

	case AgentCountsMsg:
		return m.handleAgentCountsMsg(msg)

	case SessionsMsg:
		return m.handleSessionsMsg(msg)

	case InboxMsg:
		return m.handleInboxMsg(msg)

	case views.MsgStreamBackMsg:
		// User pressed Esc in the message stream — pop back.
		m.client.StopWatching()
		cmd := m.popView()
		return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))

	case views.MsgStreamSendMsg:
		// User composed a message to send to a session.
		if msg.Body == "" {
			return m, nil
		}
		return m, tea.Batch(
			m.client.SendSessionMessage(m.currentProject, m.currentSession, msg.Body),
			m.setInfo("Sending message..."),
		)

	case views.DetailBackMsg:
		// User pressed Esc/q in the detail view — pop back.
		cmd := m.popView()
		return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))

	case StartAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Start agent failed: " + msg.Err.Error())
		}
		sessionID := ""
		if msg.Response != nil && msg.Response.Session != nil {
			sessionID = msg.Response.Session.ID
		}
		info := "Agent started"
		if sessionID != "" {
			info += " (session " + sessionID + ")"
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo(info))

	case StopAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Stop agent failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent stopped"))

	case CreateAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create agent failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Agent != nil {
			name = msg.Agent.Name
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent created: "+name))

	case DeleteAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Delete agent failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent deleted"))

	case CreateProjectMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create project failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Project != nil {
			name = msg.Project.Name
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project created: "+name))

	case DeleteProjectMsg:
		if msg.Err != nil {
			return m, m.setInfo("Delete project failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project deleted"))

	case DeleteSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Delete session failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Session deleted"))

	case UpdateAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update agent failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Agent != nil {
			name = msg.Agent.Name
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent updated: "+name))

	case UpdateProjectMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update project failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Project != nil {
			name = msg.Project.Name
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project updated: "+name))

	case UpdateSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update session failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Session != nil {
			name = msg.Session.Name
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Session updated: "+name))

	case editCompleteMsg:
		return m.handleEditComplete(msg)

	case SendMessageMsg:
		if msg.Err != nil {
			return m, m.setInfo("Send message failed: " + msg.Err.Error())
		}
		return m, m.setInfo("Message sent")

	case SendInboxMsg:
		if msg.Err != nil {
			return m, m.setInfo("Send inbox message failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Inbox message sent"))

	case MarkInboxReadMsg:
		if msg.Err != nil {
			return m, m.setInfo("Mark inbox read failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Marked as read"))

	case DeleteInboxMsg:
		if msg.Err != nil {
			return m, m.setInfo("Delete inbox message failed: " + msg.Err.Error())
		}
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Inbox message deleted"))

	case SessionMessageEvent:
		// SSE message received — add to the message stream.
		if msg.Err != nil {
			m.messageStream.SetSSEStatus("reconnecting")
			m.messageStream.AddMessage(views.MessageEntry{
				EventType: "error",
				Payload:   msg.Err.Error(),
				Timestamp: time.Now(),
			})
			// SSE failed — ensure polling fallback is running.
			if m.activeView == "messages" && !m.messagePollActive {
				m.messagePollActive = true
				return m, m.messagePollTickCmd()
			}
			return m, nil
		}
		if msg.Message != nil && m.activeView == "messages" {
			m.messageStream.SetSSEStatus("connected")
			ts := time.Now()
			if msg.Message.CreatedAt != nil {
				ts = *msg.Message.CreatedAt
			}
			m.messageStream.AddMessage(views.MessageEntry{
				Seq:       msg.Message.Seq,
				EventType: msg.Message.EventType,
				Payload:   msg.Message.Payload,
				Timestamp: ts,
			})
			// Track highest seq for polling.
			if msg.Message.Seq > m.lastMessageSeq {
				m.lastMessageSeq = msg.Message.Seq
			}
		}
		return m, nil

	case SessionMessagesMsg:
		// Polling fallback: batch of messages from REST ListMessages.
		if msg.Err != nil {
			// Non-fatal — polling will retry on next tick.
			return m, nil
		}
		if m.activeView != "messages" {
			return m, nil
		}
		for _, sm := range msg.Messages {
			if sm.Seq <= m.lastMessageSeq {
				continue // already seen via SSE or previous poll
			}
			ts := time.Now()
			if sm.CreatedAt != nil {
				ts = *sm.CreatedAt
			}
			m.messageStream.AddMessage(views.MessageEntry{
				Seq:       sm.Seq,
				EventType: sm.EventType,
				Payload:   sm.Payload,
				Timestamp: ts,
			})
			if sm.Seq > m.lastMessageSeq {
				m.lastMessageSeq = sm.Seq
			}
		}
		m.lastFetch = time.Now()
		if len(msg.Messages) > 0 {
			m.messageStream.SetSSEStatus("polling")
		}
		return m, nil

	case messagePollTickMsg:
		// Periodic poll for session messages — only active in messages view.
		if m.activeView != "messages" {
			m.messagePollActive = false
			return m, nil
		}
		// Schedule next poll tick and fetch messages.
		var cmds []tea.Cmd
		cmds = append(cmds, m.messagePollTickCmd())
		if m.currentProject != "" && m.currentSession != "" {
			cmds = append(cmds, m.client.FetchSessionMessages(
				m.currentProject, m.currentSession, m.lastMessageSeq,
			))
		}
		return m, tea.Batch(cmds...)

	case appTickMsg:
		return m.handleTick()

	case infoExpiredMsg:
		// Only clear if the expiry time has actually passed (guards against
		// stale expire messages from a previously superseded info).
		if !m.infoExpiry.IsZero() && time.Now().After(m.infoExpiry) {
			m.infoMessage = ""
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// resizeTable adjusts all table dimensions and the message stream to fill
// available space.
func (m *AppModel) resizeTable() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Layout budget:
	//   header block: 5 lines
	//   command/filter bar: 1 line (when visible) — accounted for dynamically
	//   title bar: 1 line
	//   breadcrumb: 1 line
	//   info line: 1 line
	// Total chrome: ~8 lines, leaving the rest for the table.
	tableHeight := m.height - 8
	if m.commandMode || m.filterMode || m.promptMode {
		tableHeight -= 3 // bordered command bar: top border + content + bottom border
	}
	if tableHeight < 1 {
		tableHeight = 1
	}

	// Resize all tables so they're ready when switched to.
	m.projectTable.SetHeight(tableHeight)
	m.projectTable.SetWidth(m.width)
	m.agentTable.SetHeight(tableHeight)
	m.agentTable.SetWidth(m.width)
	m.sessionTable.SetHeight(tableHeight)
	m.sessionTable.SetWidth(m.width)
	m.inboxTable.SetHeight(tableHeight)
	m.inboxTable.SetWidth(m.width)
	m.contextTable.SetHeight(tableHeight)
	m.contextTable.SetWidth(m.width)

	// Message stream and detail view get the full table area.
	m.messageStream.SetSize(m.width, tableHeight+2)
	m.detailView.SetSize(m.width, tableHeight+2)
}

// classifyAPIError inspects the error string and returns a user-friendly message
// plus a flag indicating whether the caller should skip the next poll cycle (429).
func (m *AppModel) classifyAPIError(err error, resourceKind string) (string, bool) {
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "401") || strings.Contains(errStr, "Unauthorized"):
		return "Session expired — run 'acpctl login' in another terminal", false
	case strings.Contains(errStr, "403") || strings.Contains(errStr, "Forbidden"):
		return "Insufficient permissions to list " + resourceKind, false
	case strings.Contains(errStr, "429"):
		return "Rate limited — backing off", true
	default:
		return errStr, false
	}
}

// handleProjectsMsg populates the project table from a fetch result.
func (m *AppModel) handleProjectsMsg(msg ProjectsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		errMsg, skipPoll := m.classifyAPIError(msg.Err, "projects")
		m.lastError = errMsg
		m.skipNextPoll = m.skipNextPoll || skipPoll
		// Preserve stale data — don't clear table rows.
		return m, nil
	}

	m.lastError = ""
	m.cachedProjects = msg.Projects

	// Refresh project shortcuts (alphabetically sorted names for number-key switching).
	names := make([]string, 0, len(msg.Projects))
	for _, p := range msg.Projects {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	m.projectShortcuts = names

	rows := make([]table.Row, 0, len(msg.Projects))
	for _, p := range msg.Projects {
		age := ""
		if p.CreatedAt != nil {
			age = fmtAge(time.Since(*p.CreatedAt))
		}
		desc := p.Description
		if len(desc) > 60 {
			desc = desc[:59] + "..."
		}
		status := p.Status
		if status == "" {
			status = "active"
		}
		rows = append(rows, table.Row{
			Sanitize(p.Name),
			Sanitize(desc),
			Sanitize(status),
			"-", // AGENTS — placeholder until ProjectCountsMsg arrives
			"-", // SESSIONS — placeholder until ProjectCountsMsg arrives
			age,
		})
	}
	m.projectTable.SetRows(rows)

	// Re-apply active filter if present and we're on projects view.
	if m.activeView == "projects" && m.activeFilter != nil {
		f := m.activeFilter
		m.projectTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	// Trigger background fetch of agent/session counts per project.
	var cmds []tea.Cmd
	if len(names) > 0 {
		cmds = append(cmds, m.client.FetchProjectCounts(names))
	}

	return m, tea.Batch(cmds...)
}

// handleProjectCountsMsg rebuilds the project table rows with real agent and
// session counts returned from the background FetchProjectCounts fan-out.
func (m *AppModel) handleProjectCountsMsg(msg ProjectCountsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		// Non-fatal — just keep the "-" placeholders.
		return m, nil
	}

	now := time.Now()
	rows := make([]table.Row, 0, len(m.cachedProjects))
	for _, p := range m.cachedProjects {
		age := ""
		if p.CreatedAt != nil {
			age = fmtAge(now.Sub(*p.CreatedAt))
		}
		desc := p.Description
		if len(desc) > 60 {
			desc = desc[:59] + "..."
		}
		status := p.Status
		if status == "" {
			status = "active"
		}

		agentCount := -1
		sessionCount := -1
		if counts, ok := msg.Counts[p.Name]; ok {
			agentCount = counts.AgentCount
			sessionCount = counts.SessionCount
		}

		agents := "-"
		if agentCount >= 0 {
			agents = fmt.Sprintf("%d", agentCount)
		}
		sessions := "-"
		if sessionCount >= 0 {
			sessions = fmt.Sprintf("%d", sessionCount)
		}

		rows = append(rows, table.Row{
			Sanitize(p.Name),
			Sanitize(desc),
			Sanitize(status),
			agents,
			sessions,
			age,
		})
	}
	m.projectTable.SetRows(rows)

	// Re-apply active filter if present and we're on projects view.
	if m.activeView == "projects" && m.activeFilter != nil {
		f := m.activeFilter
		m.projectTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	return m, nil
}

// handleAgentsMsg populates the agent table from a fetch result.
// Session counts are initially shown as "-" until AgentCountsMsg arrives.
func (m *AppModel) handleAgentsMsg(msg AgentsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		errMsg, skipPoll := m.classifyAPIError(msg.Err, "agents")
		m.lastError = errMsg
		m.skipNextPoll = m.skipNextPoll || skipPoll
		// Preserve stale data — don't clear table rows.
		return m, nil
	}

	m.lastError = ""
	m.cachedAgents = msg.Agents
	now := time.Now()

	rows := make([]table.Row, 0, len(msg.Agents))
	for _, a := range msg.Agents {
		// Pass -1 for session count — placeholder until AgentCountsMsg arrives.
		row := views.AgentRow(a, -1, now)
		// Sanitize all cells except PHASE (index 3) which contains embedded ANSI color.
		for i := range row {
			if i != 3 {
				row[i] = Sanitize(row[i])
			}
		}
		rows = append(rows, row)
	}
	m.agentTable.SetRows(rows)

	// Re-apply active filter if present and we're on agents view.
	if m.activeView == "agents" && m.activeFilter != nil {
		f := m.activeFilter
		m.agentTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	// Trigger background fetch of session counts per agent.
	var cmds []tea.Cmd
	if len(msg.Agents) > 0 && m.currentProject != "" {
		agentIDs := make([]string, 0, len(msg.Agents))
		for _, a := range msg.Agents {
			agentIDs = append(agentIDs, a.ID)
		}
		cmds = append(cmds, m.client.FetchAgentCounts(m.currentProject, agentIDs))
	}

	return m, tea.Batch(cmds...)
}

// handleAgentCountsMsg rebuilds agent table rows with real session counts
// returned from the background FetchAgentCounts fan-out.
func (m *AppModel) handleAgentCountsMsg(msg AgentCountsMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		// Non-fatal — just keep the "-" placeholders.
		return m, nil
	}

	now := time.Now()
	rows := make([]table.Row, 0, len(m.cachedAgents))
	for _, a := range m.cachedAgents {
		sc := -1
		if counts, ok := msg.Counts[a.ID]; ok {
			sc = counts.SessionCount
		}
		row := views.AgentRow(a, sc, now)
		// Sanitize all cells except PHASE (index 3) which contains embedded ANSI color.
		for i := range row {
			if i != 3 {
				row[i] = Sanitize(row[i])
			}
		}
		rows = append(rows, row)
	}
	m.agentTable.SetRows(rows)

	// Re-apply active filter if present and we're on agents view.
	if m.activeView == "agents" && m.activeFilter != nil {
		f := m.activeFilter
		m.agentTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	return m, nil
}

// handleSessionsMsg populates the session table from a fetch result.
func (m *AppModel) handleSessionsMsg(msg SessionsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		errMsg, skipPoll := m.classifyAPIError(msg.Err, "sessions")
		m.lastError = errMsg
		m.skipNextPoll = m.skipNextPoll || skipPoll
		// Preserve stale data — don't clear table rows.
		return m, nil
	}

	m.lastError = ""
	m.cachedSessions = msg.Sessions
	now := time.Now()

	// If agent-scoped, filter sessions to only those belonging to this agent.
	sessions := msg.Sessions
	if m.currentAgentID != "" {
		rows := make([]table.Row, 0)
		for _, s := range sessions {
			if s.AgentID == m.currentAgentID {
				row := views.SessionRow(s, m.currentAgent, now)
				// Sanitize all cells except PHASE (index 3) which contains embedded ANSI color.
				for i := range row {
					if i != 3 {
						row[i] = Sanitize(row[i])
					}
				}
				rows = append(rows, row)
			}
		}
		m.sessionTable.SetRows(rows)
	} else {
		// Global view — agent name is not resolved (would need N+1 fetch).
		rows := make([]table.Row, 0, len(sessions))
		for _, s := range sessions {
			agentName := s.AgentID
			if len(agentName) > 12 {
				agentName = agentName[:12]
			}
			row := views.SessionRow(s, agentName, now)
			// Sanitize all cells except PHASE (index 3) which contains embedded ANSI color.
			for i := range row {
				if i != 3 {
					row[i] = Sanitize(row[i])
				}
			}
			rows = append(rows, row)
		}
		m.sessionTable.SetRows(rows)
	}

	// Re-apply active filter if present and we're on sessions view.
	if m.activeView == "sessions" && m.activeFilter != nil {
		f := m.activeFilter
		m.sessionTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	return m, nil
}

// handleInboxMsg populates the inbox table from a fetch result.
func (m *AppModel) handleInboxMsg(msg InboxMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		errMsg, skipPoll := m.classifyAPIError(msg.Err, "inbox messages")
		m.lastError = errMsg
		m.skipNextPoll = m.skipNextPoll || skipPoll
		// Preserve stale data — don't clear table rows.
		return m, nil
	}

	m.lastError = ""
	m.cachedInbox = msg.Messages
	now := time.Now()

	rows := make([]table.Row, 0, len(msg.Messages))
	for _, im := range msg.Messages {
		row := views.InboxRow(im, now)
		for i := range row {
			row[i] = Sanitize(row[i])
		}
		rows = append(rows, row)
	}
	m.inboxTable.SetRows(rows)

	// Re-apply active filter if present and we're on inbox view.
	if m.activeView == "inbox" && m.activeFilter != nil {
		f := m.activeFilter
		m.inboxTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	return m, nil
}

// handleTick manages periodic polling. Skips if a fetch is already in flight
// or if skipNextPoll is set (e.g. after a 429 rate-limit response).
func (m *AppModel) handleTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{m.tickCmd()} // always schedule next tick

	// If rate-limited, skip this cycle and reset the flag for the next one.
	if m.skipNextPoll {
		m.skipNextPoll = false
		return m, tea.Batch(cmds...)
	}

	if !m.pollInFlight && m.activeView != "messages" {
		m.pollInFlight = true
		if fetchCmd := m.fetchActiveView(); fetchCmd != nil {
			cmds = append(cmds, fetchCmd)
		} else {
			m.pollInFlight = false
		}
	}

	return m, tea.Batch(cmds...)
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// handleKey dispatches key events based on the current mode.
func (m *AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl-C always quits.
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Dialog overlay takes priority over all other modes.
	if m.dialog != nil {
		return m.handleDialogKey(msg)
	}

	// Prompt mode (inline text input for new session, etc.).
	if m.promptMode {
		return m.handlePromptKey(msg)
	}

	if m.commandMode {
		return m.handleCommandKey(msg)
	}
	if m.filterMode {
		return m.handleFilterKey(msg)
	}

	// Help overlay handles its own keys.
	if m.activeView == "help" {
		return m.handleHelpKey(msg)
	}

	// Message stream handles its own keys.
	if m.activeView == "messages" {
		return m.handleMessagesKey(msg)
	}

	// Detail view handles its own keys.
	if m.activeView == "detail" {
		return m.handleDetailKey(msg)
	}

	return m.handleNormalKey(msg)
}

// handleDialogKey delegates key events to the active dialog overlay and
// processes the resulting DialogConfirmMsg / DialogCancelMsg.
func (m *AppModel) handleDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dlg, cmd := m.dialog.Update(msg)
	m.dialog = &dlg

	if cmd == nil {
		return m, nil
	}

	// Execute the command to get the message, then dispatch it.
	resultMsg := cmd()
	switch resultMsg.(type) {
	case views.DialogCancelMsg:
		m.dialog = nil
		m.dialogAction = nil
		return m, m.setInfo("Cancelled")
	case views.DialogConfirmMsg:
		confirm := resultMsg.(views.DialogConfirmMsg)
		if confirm.Confirmed {
			fn := m.dialogAction
			m.dialog = nil
			m.dialogAction = nil
			if fn != nil {
				return m, tea.Batch(fn(), m.setInfo("Processing..."))
			}
		} else {
			m.dialog = nil
			m.dialogAction = nil
			return m, m.setInfo("Cancelled")
		}
	}

	return m, nil
}

// handleNormalKey processes keys when neither command nor filter mode is active.
// Dispatches based on activeView for view-specific hotkeys.
func (m *AppModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keybindings first.
	switch msg.Type {
	case tea.KeyEsc:
		// If a filter is active, clear it first instead of popping the view.
		if m.activeFilter != nil {
			m.activeFilter = nil
			if tbl := m.activeTable(); tbl != nil {
				tbl.ClearFilter()
			}
			return m, m.setInfo("Filter cleared")
		}
		cmd := m.popView()
		if cmd != nil {
			return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))
		}
		return m, nil

	case tea.KeyCtrlD:
		return m.handleCtrlD()

	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		// Delegate to active table for row navigation.
		if tbl := m.activeTable(); tbl != nil {
			var cmd tea.Cmd
			*tbl, cmd = tbl.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyEnter:
		return m.handleEnter()

	case tea.KeyRunes:
		return m.handleRuneKey(msg)
	}

	return m, nil
}

// handleEnter processes the Enter key based on the active view.
func (m *AppModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case "contexts":
		row := m.contextTable.SelectedRow()
		if len(row) > 1 {
			contextName := row[1] // NAME column (index 1, after ACTIVE)
			if err := m.config.SwitchContext(contextName); err != nil {
				return m, m.setInfo("Error: "+err.Error())
			}
			m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
			m.activeView = "projects"
			m.currentProject = ""
			m.currentAgent = ""
			m.currentAgentID = ""
			m.currentSession = ""
			m.activeFilter = nil
			m.pollInFlight = true
			return m, tea.Batch(m.client.FetchProjects(), m.setInfo("Switched to context "+contextName))
		}

	case "projects":
		row := m.projectTable.SelectedRow()
		if len(row) > 0 {
			projectName := row[0]
			m.currentProject = projectName
			m.agentTable.SetScope(projectName)
			cmd := m.pushView("agents", projectName, "")
			return m, tea.Batch(cmd, m.setInfo("Viewing agents in project "+projectName))
		}

	case "agents":
		row := m.agentTable.SelectedRow()
		if len(row) > 0 {
			agentName := row[0]
			m.currentAgent = agentName
			// Look up the real agent ID from cache.
			agent := m.findAgentByName(agentName)
			if agent != nil {
				m.currentAgentID = agent.ID
			} else {
				m.currentAgentID = agentName // fallback
			}
			m.sessionTable.SetScope(agentName)
			cmd := m.pushView("sessions", agentName, "")
			return m, tea.Batch(cmd, m.setInfo("Viewing sessions for agent "+agentName))
		}

	case "sessions":
		row := m.sessionTable.SelectedRow()
		if len(row) > 0 {
			shortID := row[0] // Short ID is in first column
			// Resolve the full session ID from cache.
			session := m.findSessionByShortID(shortID)
			fullSessionID := shortID
			if session != nil {
				fullSessionID = session.ID
			}
			m.currentSession = fullSessionID
			m.lastMessageSeq = 0

			// Create a new message stream for this session.
			agentName := m.currentAgent
			if agentName == "" && len(row) > 1 {
				agentName = row[2] // AGENT column
			}
			phase := ""
			if len(row) > 4 {
				phase = row[4] // PHASE column
			}
			m.messageStream = views.NewMessageStream(fullSessionID, agentName, phase)
			m.resizeTable() // set message stream dimensions

			cmds := []tea.Cmd{
				m.pushView("messages", fullSessionID, fullSessionID),
				m.setInfo("Streaming messages for session " + shortID),
			}

			// Start SSE watcher if we have a program reference and project context.
			if m.program != nil && m.currentProject != "" {
				cmds = append(cmds, m.client.WatchSessionMessages(m.currentProject, fullSessionID, 0, m.program))
			}

			// Always start polling fallback alongside SSE. Polling is
			// idempotent (deduplicates by seq) and ensures messages appear
			// even if SSE fails silently.
			if m.currentProject != "" {
				m.messagePollActive = true
				cmds = append(cmds, m.messagePollTickCmd())
				// Immediately fetch existing messages so the view is not empty.
				cmds = append(cmds, m.client.FetchSessionMessages(m.currentProject, fullSessionID, 0))
			}

			return m, tea.Batch(cmds...)
		}

	case "inbox":
		row := m.inboxTable.SelectedRow()
		if len(row) > 0 {
			msgID := row[0]
			inboxMsg := m.findInboxByID(msgID)
			if inboxMsg == nil {
				return m, m.setInfo("Inbox message not found in cache: " + msgID)
			}
			m.detailView = views.NewDetailView("Inbox: "+msgID, views.InboxDetail(*inboxMsg))
			m.detailView.SetSize(m.width, m.height-10)
			cmd := m.pushView("detail", msgID, msgID)
			return m, tea.Batch(cmd, m.setInfo("Inbox message detail"))
		}
	}

	return m, nil
}

// handleRuneKey processes single-character keys in normal mode.
func (m *AppModel) handleRuneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global rune keybindings.
	switch key {
	case ":":
		m.commandMode = true
		m.commandInput.Reset()
		m.commandInput.Focus()
		m.resizeTable()
		return m, nil

	case "/":
		m.filterMode = true
		m.filterInput.Reset()
		m.filterInput.Focus()
		m.resizeTable()
		return m, nil

	case "?":
		return m.showHelp()

	case "q":
		if len(m.navStack) <= 1 {
			return m, tea.Quit
		}
		cmd := m.popView()
		return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))

	case "j":
		if tbl := m.activeTable(); tbl != nil {
			var cmd tea.Cmd
			*tbl, cmd = tbl.Update(tea.KeyMsg{Type: tea.KeyDown})
			return m, cmd
		}
		return m, nil

	case "k":
		if tbl := m.activeTable(); tbl != nil {
			var cmd tea.Cmd
			*tbl, cmd = tbl.Update(tea.KeyMsg{Type: tea.KeyUp})
			return m, cmd
		}
		return m, nil

	case "N":
		// Sort by NAME column (index 0) — works for all table views.
		if tbl := m.activeTable(); tbl != nil {
			tbl.SortByColumn(0)
		}
		return m, nil

	case "A":
		// Sort by AGE column — last column in all views.
		if tbl := m.activeTable(); tbl != nil {
			cols := tbl.Columns()
			// AGE is the last column in all table views.
			tbl.SortByColumn(len(cols) - 1)
		}
		return m, nil

	case "c":
		// Copy the first column value (resource name/ID) of the selected row to clipboard.
		if tbl := m.activeTable(); tbl != nil {
			row := tbl.SelectedRow()
			if len(row) > 0 {
				value := row[0]
				_ = clipboard.WriteAll(value)
				return m, m.setInfo("Copied: " + value)
			}
		}
		return m, nil
	}

	// Number-key project shortcuts (0-9) — only active on table views below project level.
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' &&
		m.activeView != "projects" && m.activeView != "contexts" &&
		m.activeView != "messages" && m.activeView != "detail" {
		return m.handleProjectShortcut(key[0] - '0')
	}

	// View-specific rune keybindings.
	switch m.activeView {
	case "projects":
		return m.handleProjectsRune(key)
	case "agents":
		return m.handleAgentsRune(key)
	case "sessions":
		return m.handleSessionsRune(key)
	case "inbox":
		return m.handleInboxRune(key)
	}

	return m, nil
}

// handleProjectsRune handles project-view-specific hotkeys.
func (m *AppModel) handleProjectsRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "d":
		// Show detail view for the selected project.
		row := m.projectTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		projectName := row[0]
		project := m.findProjectByName(projectName)
		if project == nil {
			return m, m.setInfo("Project not found in cache: " + projectName)
		}
		m.detailView = views.NewDetailView("Project: "+projectName, views.ProjectDetail(*project))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", projectName, project.ID)
		return m, tea.Batch(cmd, m.setInfo("Project detail: "+projectName))
	case "e":
		return m.openEditorForProject()
	case "n":
		return m, m.setInfo("Use acpctl project create")
	}
	return m, nil
}

// handleAgentsRune handles agent-view-specific hotkeys.
func (m *AppModel) handleAgentsRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "i":
		// Drill into inbox for selected agent.
		row := m.agentTable.SelectedRow()
		if len(row) > 0 {
			agentName := row[0]
			m.currentAgent = agentName
			agent := m.findAgentByName(agentName)
			if agent != nil {
				m.currentAgentID = agent.ID
			} else {
				m.currentAgentID = agentName // fallback
			}
			m.inboxTable.SetScope(agentName)
			cmd := m.pushView("inbox", agentName, "")
			return m, tea.Batch(cmd, m.setInfo("Viewing inbox for agent "+agentName))
		}
	case "s":
		// Start the selected agent.
		row := m.agentTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No agent selected")
		}
		agentName := row[0]
		agent := m.findAgentByName(agentName)
		if agent == nil {
			return m, m.setInfo("Agent not found in cache: " + agentName)
		}
		return m, tea.Batch(
			m.client.StartAgent(m.currentProject, agent.ID, ""),
			m.setInfo("Starting agent "+agentName+"..."),
		)
	case "x":
		// Stop the selected agent's current session.
		row := m.agentTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No agent selected")
		}
		agentName := row[0]
		agent := m.findAgentByName(agentName)
		if agent == nil {
			return m, m.setInfo("Agent not found in cache: " + agentName)
		}
		if agent.CurrentSessionID == "" {
			return m, m.setInfo("Agent " + agentName + " has no active session")
		}
		return m, tea.Batch(
			m.client.StopAgent(m.currentProject, agent.CurrentSessionID),
			m.setInfo("Stopping agent "+agentName+"..."),
		)
	case "e":
		return m.openEditorForAgent()
	case "l":
		// Logs — if agent has an active session, jump to message stream.
		row := m.agentTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No agent selected")
		}
		agentName := row[0]
		agent := m.findAgentByName(agentName)
		if agent == nil {
			return m, m.setInfo("Agent not found in cache: " + agentName)
		}
		if agent.CurrentSessionID == "" {
			return m, m.setInfo("No active session for this agent")
		}
		sessionID := agent.CurrentSessionID
		m.currentAgent = agentName
		m.currentAgentID = agent.ID
		m.currentSession = sessionID
		m.lastMessageSeq = 0
		m.messageStream = views.NewMessageStream(sessionID, agentName, "active")
		m.resizeTable()

		cmds := []tea.Cmd{
			m.pushView("messages", sessionID, sessionID),
			m.setInfo("Streaming messages for session " + sessionID),
		}

		// Start SSE watcher if we have a program reference and project context.
		if m.program != nil && m.currentProject != "" {
			cmds = append(cmds, m.client.WatchSessionMessages(m.currentProject, sessionID, 0, m.program))
		}

		// Always start polling fallback alongside SSE.
		if m.currentProject != "" {
			m.messagePollActive = true
			cmds = append(cmds, m.messagePollTickCmd())
			// Immediately fetch existing messages so the view is not empty.
			cmds = append(cmds, m.client.FetchSessionMessages(m.currentProject, sessionID, 0))
		}

		return m, tea.Batch(cmds...)
	case "d":
		// Show detail view for the selected agent.
		row := m.agentTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		agentName := row[0]
		agent := m.findAgentByName(agentName)
		if agent == nil {
			return m, m.setInfo("Agent not found in cache: " + agentName)
		}
		m.detailView = views.NewDetailView("Agent: "+agentName, views.AgentDetail(*agent))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", agentName, agent.ID)
		return m, tea.Batch(cmd, m.setInfo("Agent detail: "+agentName))
	case "m":
		return m, m.setInfo("Use :inbox or acpctl inbox send")
	case "n":
		return m, m.setInfo("Use acpctl agent create")
	case "y":
		row := m.agentTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		agentName := row[0]
		agent := m.findAgentByName(agentName)
		if agent == nil {
			return m, m.setInfo("Agent not found in cache: " + agentName)
		}
		m.detailView = views.NewDetailView("YAML: "+agentName, views.ResourceJSON(*agent))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", agentName, agent.ID)
		return m, tea.Batch(cmd, m.setInfo("YAML: "+agentName))
	}
	return m, nil
}

// handleSessionsRune handles session-view-specific hotkeys.
func (m *AppModel) handleSessionsRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "e":
		return m.openEditorForSession()
	case "d":
		// Show detail view for the selected session.
		row := m.sessionTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		shortID := row[0]
		session := m.findSessionByShortID(shortID)
		if session == nil {
			return m, m.setInfo("Session not found in cache: " + shortID)
		}
		m.detailView = views.NewDetailView("Session: "+shortID, views.SessionDetail(*session))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", shortID, session.ID)
		return m, tea.Batch(cmd, m.setInfo("Session detail: "+shortID))
	case "l":
		// Same as Enter — drill into message stream.
		return m.handleEnter()
	case "m":
		return m, m.setInfo("Use Enter to view messages, then m to compose")
	case "n":
		// Start a new session for the current agent.
		if m.currentAgentID == "" || m.currentProject == "" {
			return m, m.setInfo("Navigate to an agent first to start a session")
		}
		// Open prompt input for session prompt text.
		agentID := m.currentAgentID
		project := m.currentProject
		m.promptMode = true
		m.promptInput.Prompt = "Session prompt: "
		m.promptInput.Reset()
		m.promptInput.Focus()
		m.promptCallback = func(text string) (tea.Model, tea.Cmd) {
			return m, tea.Batch(
				m.client.StartAgent(project, agentID, text),
				m.setInfo("Starting session..."),
			)
		}
		m.resizeTable()
		return m, nil
	case "y":
		row := m.sessionTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		shortID := row[0]
		session := m.findSessionByShortID(shortID)
		if session == nil {
			return m, m.setInfo("Session not found in cache: " + shortID)
		}
		m.detailView = views.NewDetailView("YAML: "+shortID, views.ResourceJSON(*session))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", shortID, session.ID)
		return m, tea.Batch(cmd, m.setInfo("Session detail: "+shortID))
	}
	return m, nil
}

// handleInboxRune handles inbox-view-specific hotkeys.
func (m *AppModel) handleInboxRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "m":
		return m, m.setInfo("Use acpctl inbox send")
	case "r":
		// Mark selected inbox message as read.
		row := m.inboxTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No inbox message selected")
		}
		msgID := row[0] // ID column
		if m.currentProject == "" || m.currentAgentID == "" {
			return m, m.setInfo("No agent context for inbox")
		}
		return m, tea.Batch(
			m.client.MarkInboxRead(m.currentProject, m.currentAgentID, msgID),
			m.setInfo("Marking as read..."),
		)
	}
	return m, nil
}

// handleCtrlD handles the delete/cancel keybinding across all views.
// Instead of deleting immediately, it sets up a confirmation prompt.
func (m *AppModel) handleCtrlD() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case "projects":
		row := m.projectTable.SelectedRow()
		if len(row) > 0 {
			projectName := row[0]
			project := m.findProjectByName(projectName)
			if project == nil {
				return m, m.setInfo("Project not found in cache: " + projectName)
			}
			projectID := project.ID
			d := views.NewDeleteDialog("project", projectName)
			m.dialog = &d
			m.dialogAction = func() tea.Cmd {
				return m.client.DeleteProject(projectID)
			}
			return m, nil
		}
	case "agents":
		row := m.agentTable.SelectedRow()
		if len(row) > 0 {
			agentName := row[0]
			agent := m.findAgentByName(agentName)
			if agent == nil {
				return m, m.setInfo("Agent not found in cache: " + agentName)
			}
			agentID := agent.ID
			currentProject := m.currentProject
			d := views.NewDeleteDialog("agent", agentName)
			m.dialog = &d
			m.dialogAction = func() tea.Cmd {
				return m.client.DeleteAgent(currentProject, agentID)
			}
			return m, nil
		}
	case "sessions":
		row := m.sessionTable.SelectedRow()
		if len(row) > 0 {
			shortID := row[0]
			session := m.findSessionByShortID(shortID)
			if session == nil {
				return m, m.setInfo("Session not found in cache: " + shortID)
			}
			project := m.currentProject
			if project == "" {
				project = session.ProjectID
			}
			sessionID := session.ID
			d := views.NewDeleteDialog("session", shortID)
			m.dialog = &d
			m.dialogAction = func() tea.Cmd {
				return m.client.DeleteSession(project, sessionID)
			}
			return m, nil
		}
	case "inbox":
		row := m.inboxTable.SelectedRow()
		if len(row) > 0 {
			msgID := row[0]
			if m.currentProject == "" || m.currentAgentID == "" {
				return m, m.setInfo("No agent context for inbox")
			}
			currentProject := m.currentProject
			currentAgentID := m.currentAgentID
			d := views.NewDeleteDialog("inbox message", msgID)
			m.dialog = &d
			m.dialogAction = func() tea.Cmd {
				return m.client.DeleteInboxMessage(currentProject, currentAgentID, msgID)
			}
			return m, nil
		}
	}
	return m, nil
}

// handleDetailKey delegates key events to the detail view sub-model.
func (m *AppModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.detailView, cmd = m.detailView.Update(msg)
	return m, cmd
}

// handleMessagesKey delegates key events to the message stream sub-model.
func (m *AppModel) handleMessagesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Intercept global keys before delegating to the message stream.
	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case ":":
			m.commandMode = true
			m.commandInput.Focus()
			m.resizeTable()
			return m, nil
		case "?":
			return m.showHelp()
		case "q":
			return m, m.popView()
		}
	}
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.messageStream, cmd = m.messageStream.Update(msg)
	return m, cmd
}

// showHelp creates a HelpView for the current view and pushes it onto the nav stack.
func (m *AppModel) showHelp() (tea.Model, tea.Cmd) {
	// Fix 3: Capture the view name BEFORE changing activeView.
	viewName := m.activeView

	general := []views.HelpEntry{
		{":", "Command"},
		{"/", "Filter"},
		{"?", "Help"},
		{"c", "Copy ID"},
		{"shift-n", "Sort Name"},
		{"shift-a", "Sort Age"},
	}

	var resource, navigation []views.HelpEntry

	switch viewName {
	case "projects":
		resource = []views.HelpEntry{
			{"d", "Describe"}, {"e", "Edit"}, {"n", "New"}, {"ctrl-d", "Delete"},
		}
		navigation = []views.HelpEntry{
			{"Enter", "Drill into agents"}, {"q", "Quit"},
		}
	case "agents":
		resource = []views.HelpEntry{
			{"s", "Start"}, {"x", "Stop"}, {"e", "Edit"}, {"i", "Inbox"},
			{"l", "Logs"}, {"d", "Describe"}, {"n", "New"}, {"ctrl-d", "Delete"},
		}
		navigation = []views.HelpEntry{
			{"Enter", "Drill into sessions"}, {"Esc", "Back to projects"},
			{"q", "Back"}, {"0-9", "Switch project"},
		}
	case "sessions":
		resource = []views.HelpEntry{
			{"d", "Describe"}, {"e", "Edit"}, {"l", "Logs"}, {"m", "Send"}, {"n", "New"},
			{"y", "JSON"}, {"ctrl-d", "Delete"},
		}
		navigation = []views.HelpEntry{
			{"Enter", "Drill into messages"}, {"Esc", "Back to agents"},
			{"q", "Back"}, {"0-9", "Switch project"},
		}
	case "inbox":
		resource = []views.HelpEntry{
			{"m", "Compose"}, {"r", "Mark Read"}, {"ctrl-d", "Delete"},
		}
		navigation = []views.HelpEntry{
			{"Enter", "View body"}, {"Esc", "Back to agents"}, {"q", "Back"},
		}
	case "messages":
		resource = []views.HelpEntry{
			{"s", "Autoscroll"}, {"r", "Raw Mode"}, {"m", "Send"},
			{"c", "Copy"}, {"shift-g", "Bottom"}, {"g", "Top"},
		}
		general = []views.HelpEntry{
			{":", "Command"}, {"?", "Help"},
		}
		navigation = []views.HelpEntry{
			{"Esc", "Back to sessions"}, {"q", "Back"},
		}
	case "contexts":
		resource = []views.HelpEntry{}
		navigation = []views.HelpEntry{
			{"Enter", "Switch context"}, {"Esc", "Back"}, {"q", "Back"},
		}
	case "detail":
		resource = []views.HelpEntry{
			{"c", "Copy value"}, {"j/k", "Scroll"},
		}
		general = []views.HelpEntry{
			{"?", "Help"},
		}
		navigation = []views.HelpEntry{
			{"Esc", "Back"}, {"q", "Back"},
		}
	}

	title := viewName
	m.helpView = views.NewHelpView(title, resource, general, navigation)
	m.helpView.SetSize(m.width, m.height-10)
	m.navStack = append(m.navStack, NavEntry{Kind: "help", Scope: viewName})
	m.activeView = "help"
	return m, nil
}

// handleHelpKey processes keys while the help overlay is shown.
func (m *AppModel) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "?") ||
		(msg.Type == tea.KeyRunes && string(msg.Runes) == "q") {
		return m, m.popView()
	}
	return m, nil
}

// handleCommandKey processes keys while in command mode.
func (m *AppModel) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.commandMode = false
		m.commandInput.SetSuggestions(nil)
		m.commandInput.Reset()
		m.commandInput.Blur()
		m.resizeTable()
		return m, nil

	case tea.KeyEnter:
		input := m.commandInput.Value()
		m.commandMode = false
		m.commandInput.SetSuggestions(nil)
		m.commandInput.Reset()
		m.commandInput.Blur()
		m.resizeTable()
		return m.executeCommand(input)

	case tea.KeyTab:
		// Accept the inline suggestion.
		// bubbles/textinput handles Tab natively when ShowSuggestions is on,
		// but we also update suggestions after acceptance.
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		m.updateCommandHint()
		return m, cmd

	default:
		// Delegate to textinput for character entry.
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		// Update hint as user types.
		m.updateCommandHint()
		return m, cmd
	}
}

// executeCommand parses and dispatches a command-mode input.
func (m *AppModel) executeCommand(input string) (tea.Model, tea.Cmd) {
	cmd := ParseCommand(input)

	switch cmd.Kind {
	case CmdQuit:
		return m, tea.Quit

	case CmdProjects:
		// Reset nav stack to projects root.
		m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
		m.activeView = "projects"
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.activeFilter = nil
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchProjects(),
			m.setInfo("Viewing projects"),
		)

	case CmdAgents:
		// Use current project from nav stack or config.
		project := m.currentProject
		if project == "" {
			if ctx := m.config.Current(); ctx != nil {
				project = ctx.Project
			}
		}
		if project == "" {
			return m, m.setInfo("No project context — drill into a project first or set one with :project <name>")
		}
		m.currentProject = project
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.agentTable.SetScope(project)
		// Reset nav stack to project > agents.
		m.navStack = []NavEntry{
			{Kind: "projects", Scope: "all"},
			{Kind: "agents", Scope: project},
		}
		m.activeView = "agents"
		m.activeFilter = nil
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchAgents(project),
			m.setInfo("Viewing agents in project "+project),
		)

	case CmdSessions:
		// Global if no agent context, scoped if we have one.
		m.currentSession = ""
		m.activeFilter = nil

		if m.currentAgentID != "" && m.currentProject != "" {
			// Agent-scoped sessions.
			m.sessionTable.SetScope(m.currentAgent)
			m.navStack = append(m.navStack[:0],
				NavEntry{Kind: "projects", Scope: "all"},
				NavEntry{Kind: "agents", Scope: m.currentProject},
				NavEntry{Kind: "sessions", Scope: m.currentAgent},
			)
			m.activeView = "sessions"
			m.pollInFlight = true
			return m, tea.Batch(
				m.client.FetchSessions(m.currentProject),
				m.setInfo("Viewing sessions for agent "+m.currentAgent),
			)
		}

		// Global sessions view.
		m.sessionTable.SetScope("all")
		m.navStack = []NavEntry{
			{Kind: "projects", Scope: "all"},
			{Kind: "sessions", Scope: "all"},
		}
		m.activeView = "sessions"
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchAllSessions(),
			m.setInfo("Viewing all sessions"),
		)

	case CmdInbox:
		if m.currentAgentID == "" || m.currentProject == "" {
			return m, m.setInfo("No agent context — drill into an agent first or use :agents then i")
		}
		m.inboxTable.SetScope(m.currentAgent)
		m.activeView = "inbox"
		m.activeFilter = nil
		// Rebuild nav to include inbox.
		m.navStack = append(m.navStack[:0],
			NavEntry{Kind: "projects", Scope: "all"},
			NavEntry{Kind: "agents", Scope: m.currentProject},
			NavEntry{Kind: "inbox", Scope: m.currentAgent},
		)
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchInbox(m.currentProject, m.currentAgentID),
			m.setInfo("Viewing inbox for agent "+m.currentAgent),
		)

	case CmdMessages:
		if m.currentSession == "" {
			return m, m.setInfo("No session context — drill into a session first")
		}
		m.activeView = "messages"
		m.activeFilter = nil
		return m, m.setInfo("Streaming messages for session "+m.currentSession)

	case CmdContext:
		if cmd.Arg == "" {
			// Show contexts in a table view.
			m.populateContextTable()
			m.navStack = []NavEntry{{Kind: "contexts", Scope: "all"}}
			m.activeView = "contexts"
			m.resizeTable()
			return m, m.setInfo("Viewing contexts")
		}
		// Switch context.
		if err := m.config.SwitchContext(cmd.Arg); err != nil {
			return m, m.setInfo("Error: "+err.Error())
		}
		// Reset everything on context switch.
		m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
		m.activeView = "projects"
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.activeFilter = nil
		return m, m.setInfo("Switched to context "+cmd.Arg)

	case CmdProject:
		if cmd.Arg != "" {
			ctx := m.config.Current()
			if ctx != nil {
				ctx.Project = cmd.Arg
			}
			m.currentProject = cmd.Arg
			return m, m.setInfo("Switched to project "+cmd.Arg)
		}
		return m, nil

	case CmdAliases:
		entries := AliasTable()
		var lines []string
		for _, e := range entries {
			aliases := ""
			if len(e.Aliases) > 0 {
				aliases = " (" + fmt.Sprintf("%v", e.Aliases) + ")"
			}
			lines = append(lines, e.Command+aliases+" - "+e.Description)
		}
		return m, m.setInfo("Commands: " + fmt.Sprintf("%d available", len(entries)))

	default:
		ascii := "" +
			"            __\n" +
			"           / _)\n" +
			"    .-^^^-/ /\n" +
			"  __/       /\n" +
			" <__.|_|-|_|"
		msg := "< Ruroh? '" + input + "' not found >"
		d := views.NewErrorDialog("error", msg, ascii)
		m.dialog = &d
		m.dialogAction = nil // single-button dismiss
		return m, nil
	}
}

// updateCommandHint refreshes inline tab-completion suggestions.
func (m *AppModel) updateCommandHint() {
	partial := m.commandInput.Value()
	if partial == "" {
		m.commandInput.SetSuggestions(nil)
		return
	}
	contextNames := m.config.ContextNames()
	var projectNames []string
	for _, row := range m.projectTable.Rows() {
		if len(row) > 0 {
			projectNames = append(projectNames, row[0])
		}
	}
	suggestions := TabComplete(partial, contextNames, projectNames)
	m.commandInput.SetSuggestions(suggestions)
}

// handleFilterKey processes keys while in filter mode.
func (m *AppModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterMode = false
		m.filterInput.Reset()
		m.filterInput.Blur()
		m.activeFilter = nil
		m.clearActiveTableFilter()
		m.resizeTable()
		return m, m.setInfo("Filter cleared")

	case tea.KeyEnter:
		input := m.filterInput.Value()
		m.filterMode = false
		m.filterInput.Blur()
		m.resizeTable()

		if input == "" {
			m.activeFilter = nil
			m.clearActiveTableFilter()
			return m, m.setInfo("Filter cleared")
		}

		f, err := ParseFilter(input)
		if err != nil {
			return m, m.setInfo("Invalid filter: "+err.Error())
		}

		m.activeFilter = f
		m.applyFilterToActiveTable(f)
		return m, m.setInfo("Filter applied: "+f.String())

	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		// Apply filter live as user types.
		m.applyLiveFilter()
		return m, cmd
	}
}

// applyLiveFilter updates the active table filter on every keystroke.
func (m *AppModel) applyLiveFilter() {
	input := m.filterInput.Value()
	if input == "" {
		m.activeFilter = nil
		m.clearActiveTableFilter()
		return
	}
	f, err := ParseFilter(input)
	if err != nil {
		return // don't apply invalid regex while typing
	}
	m.activeFilter = f
	m.applyFilterToActiveTable(f)
}

// applyFilterToActiveTable applies a filter to whichever table is currently active.
func (m *AppModel) applyFilterToActiveTable(f *Filter) {
	if tbl := m.activeTable(); tbl != nil {
		tbl.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
		tbl.SetFilterText(f.Raw)
	}
}

// clearActiveTableFilter removes the filter from the currently active table.
func (m *AppModel) clearActiveTableFilter() {
	if tbl := m.activeTable(); tbl != nil {
		tbl.ClearFilter()
		tbl.SetFilterText("")
	}
}

// ---------------------------------------------------------------------------
// Prompt mode (inline text input for new session, etc.)
// ---------------------------------------------------------------------------

// handlePromptKey processes keys while in prompt mode.
func (m *AppModel) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.promptMode = false
		m.promptCallback = nil
		m.promptInput.Reset()
		m.promptInput.Blur()
		m.resizeTable()
		return m, m.setInfo("Cancelled")

	case tea.KeyEnter:
		input := m.promptInput.Value()
		cb := m.promptCallback
		m.promptMode = false
		m.promptCallback = nil
		m.promptInput.Reset()
		m.promptInput.Blur()
		m.resizeTable()
		if cb != nil {
			return cb(input)
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.promptInput, cmd = m.promptInput.Update(msg)
		return m, cmd
	}
}

// ---------------------------------------------------------------------------
// Project number-key shortcuts
// ---------------------------------------------------------------------------

// handleProjectShortcut switches the project scope when a digit 0-9 is pressed.
// 0 = "all" (clear project scope), 1-9 = projectShortcuts[digit-1].
func (m *AppModel) handleProjectShortcut(digit byte) (tea.Model, tea.Cmd) {
	if digit == 0 {
		// Switch to "all" — clear project scope, stay in current view type.
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.activeFilter = nil
		m.pollInFlight = true

		switch m.activeView {
		case "sessions":
			m.sessionTable.SetScope("all")
			m.navStack = []NavEntry{{Kind: "sessions", Scope: "all"}}
			return m, tea.Batch(m.client.FetchAllSessions(), m.setInfo("Viewing all sessions"))
		default:
			m.agentTable.SetScope("all")
			m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
			m.activeView = "projects"
			return m, tea.Batch(m.client.FetchProjects(), m.setInfo("Viewing all projects"))
		}
	}

	idx := int(digit) - 1
	if idx >= len(m.projectShortcuts) {
		return m, m.setInfo(fmt.Sprintf("No project at index %d", digit))
	}

	projectName := m.projectShortcuts[idx]
	m.currentProject = projectName
	m.currentAgent = ""
	m.currentAgentID = ""
	m.currentSession = ""
	m.activeFilter = nil
	m.pollInFlight = true

	// Stay in the same view type when switching projects.
	targetView := m.activeView
	switch targetView {
	case "sessions":
		m.sessionTable.SetScope(projectName)
		m.navStack = []NavEntry{
			{Kind: "projects", Scope: "all"},
			{Kind: "agents", Scope: projectName},
			{Kind: "sessions", Scope: projectName},
		}
		m.activeView = "sessions"
		return m, tea.Batch(
			m.client.FetchSessions(projectName),
			m.setInfo("Switched to project "+projectName),
		)
	default:
		m.agentTable.SetScope(projectName)
		m.navStack = []NavEntry{
			{Kind: "projects", Scope: "all"},
			{Kind: "agents", Scope: projectName},
		}
		m.activeView = "agents"
		return m, tea.Batch(
			m.client.FetchAgents(projectName),
			m.setInfo("Switched to project "+projectName),
		)
	}
}

// ---------------------------------------------------------------------------
// $EDITOR integration
// ---------------------------------------------------------------------------

// openEditorForAgent serializes the selected agent as JSON, writes it to a
// temp file, and suspends the TUI to open the user's $EDITOR. On return the
// editCompleteMsg handler diffs and PATCHes any changes.
func (m *AppModel) openEditorForAgent() (tea.Model, tea.Cmd) {
	row := m.agentTable.SelectedRow()
	if len(row) == 0 {
		return m, m.setInfo("No agent selected")
	}
	agentName := row[0]
	agent := m.findAgentByName(agentName)
	if agent == nil {
		return m, m.setInfo("Agent not found in cache: " + agentName)
	}
	if m.currentProject == "" {
		return m, m.setInfo("No project context for edit")
	}

	return m.openEditorForResource("agent", agent.ID, m.currentProject, *agent)
}

// openEditorForProject serializes the selected project as JSON, writes it to a
// temp file, and suspends the TUI to open the user's $EDITOR.
func (m *AppModel) openEditorForProject() (tea.Model, tea.Cmd) {
	row := m.projectTable.SelectedRow()
	if len(row) == 0 {
		return m, m.setInfo("No project selected")
	}
	projectName := row[0]
	project := m.findProjectByName(projectName)
	if project == nil {
		return m, m.setInfo("Project not found in cache: " + projectName)
	}

	return m.openEditorForResource("project", project.ID, "", *project)
}

// openEditorForSession serializes the selected session as JSON, writes it to a
// temp file, and suspends the TUI to open the user's $EDITOR.
func (m *AppModel) openEditorForSession() (tea.Model, tea.Cmd) {
	row := m.sessionTable.SelectedRow()
	if len(row) == 0 {
		return m, m.setInfo("No session selected")
	}
	shortID := row[0]
	session := m.findSessionByShortID(shortID)
	if session == nil {
		return m, m.setInfo("Session not found in cache: " + shortID)
	}

	projectID := m.currentProject
	if projectID == "" {
		projectID = session.ProjectID
	}
	if projectID == "" {
		return m, m.setInfo("No project context for edit")
	}

	return m.openEditorForResource("session", session.ID, projectID, *session)
}

// openEditorForResource is the shared implementation that writes JSON to a temp
// file, opens $EDITOR via tea.ExecProcess, and returns an editCompleteMsg when
// the editor exits.
func (m *AppModel) openEditorForResource(kind, resourceID, projectID string, resource any) (tea.Model, tea.Cmd) {
	originalJSON, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return m, m.setInfo("Failed to serialize " + kind + ": " + err.Error())
	}

	tmpFile, err := os.CreateTemp("", "acpctl-edit-*.json")
	if err != nil {
		return m, m.setInfo("Failed to create temp file: " + err.Error())
	}

	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		os.Remove(tmpFile.Name())
		return m, m.setInfo("Failed to set temp file permissions: " + err.Error())
	}

	if _, err := tmpFile.Write(originalJSON); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return m, m.setInfo("Failed to write temp file: " + err.Error())
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return m, m.setInfo("Failed to close temp file: " + err.Error())
	}

	editor := getEditor()
	tmpPath := tmpFile.Name()
	origCopy := make([]byte, len(originalJSON))
	copy(origCopy, originalJSON)

	c := exec.Command(editor, tmpPath) //nolint:gosec // editor is from user's env
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return editCompleteMsg{
			ResourceKind: kind,
			ResourceID:   resourceID,
			ProjectID:    projectID,
			TempFile:     tmpPath,
			OriginalJSON: origCopy,
			Err:          err,
		}
	})
}

// handleEditComplete processes the editCompleteMsg after the editor exits.
// It reads the edited JSON, diffs against the original, builds a patch map
// with only changed fields, and calls the appropriate update method.
func (m *AppModel) handleEditComplete(msg editCompleteMsg) (tea.Model, tea.Cmd) {
	// Always clean up the temp file.
	defer os.Remove(msg.TempFile)

	if msg.Err != nil {
		return m, m.setInfo("Editor exited with error: " + msg.Err.Error())
	}

	// Read the edited file.
	editedJSON, err := os.ReadFile(msg.TempFile)
	if err != nil {
		return m, m.setInfo("Failed to read edited file: " + err.Error())
	}

	// Parse both original and edited JSON into maps for diffing.
	var original map[string]any
	if err := json.Unmarshal(msg.OriginalJSON, &original); err != nil {
		return m, m.setInfo("Failed to parse original JSON: " + err.Error())
	}
	var edited map[string]any
	if err := json.Unmarshal(editedJSON, &edited); err != nil {
		return m, m.setInfo("Failed to parse edited JSON: " + err.Error())
	}

	// Determine which fields are editable based on resource kind.
	var editableFields []string
	switch msg.ResourceKind {
	case "agent":
		editableFields = []string{
			"name", "prompt", "labels", "annotations", "description",
			"display_name", "llm_model", "llm_max_tokens", "llm_temperature",
			"repo_url", "environment_variables", "resource_overrides",
		}
	case "project":
		editableFields = []string{
			"name", "description", "display_name", "labels", "annotations",
			"prompt", "status",
		}
	case "session":
		editableFields = []string{
			"name", "prompt", "labels", "annotations",
			"llm_model", "llm_max_tokens", "llm_temperature",
			"repo_url", "repos", "resource_overrides", "timeout",
			"environment_variables",
		}
	}

	// Build patch with only changed editable fields.
	patch := make(map[string]any)
	for _, field := range editableFields {
		origVal, origOK := original[field]
		editVal, editOK := edited[field]

		// Field was added in the edit.
		if !origOK && editOK {
			patch[field] = editVal
			continue
		}
		// Field was removed in the edit.
		if origOK && !editOK {
			// Send zero value to clear the field.
			patch[field] = nil
			continue
		}
		// Both present — compare serialized forms for robustness.
		if origOK && editOK {
			origSer, _ := json.Marshal(origVal)
			editSer, _ := json.Marshal(editVal)
			if string(origSer) != string(editSer) {
				patch[field] = editVal
			}
		}
	}

	if len(patch) == 0 {
		return m, m.setInfo("No changes detected")
	}

	// Build a summary of changed fields.
	var changedFields []string
	for k := range patch {
		changedFields = append(changedFields, k)
	}
	sort.Strings(changedFields)
	summary := strings.Join(changedFields, ", ")

	switch msg.ResourceKind {
	case "agent":
		return m, tea.Batch(
			m.client.UpdateAgent(msg.ProjectID, msg.ResourceID, patch),
			m.setInfo("Updating agent ("+summary+")..."),
		)
	case "project":
		return m, tea.Batch(
			m.client.UpdateProject(msg.ResourceID, patch),
			m.setInfo("Updating project ("+summary+")..."),
		)
	case "session":
		return m, tea.Batch(
			m.client.UpdateSession(msg.ProjectID, msg.ResourceID, patch),
			m.setInfo("Updating session ("+summary+")..."),
		)
	default:
		return m, m.setInfo("Unknown resource kind: " + msg.ResourceKind)
	}
}

// ---------------------------------------------------------------------------
// Contextual hotkey hints for the header
// ---------------------------------------------------------------------------

// contextualHints returns the hotkey hints for the current active view.
func (m *AppModel) contextualHints() []string {
	switch m.activeView {
	case "projects":
		return []string{
			"<d> Describe",
			"<e> Edit",
			"<n> New",
			"<Ctrl-D> Delete",
		}
	case "agents":
		return []string{
			"<s> Start",
			"<x> Stop",
			"<i> Inbox",
			"<d> Describe",
			"<e> Edit",
			"<l> Logs",
			"<n> New",
			"<Ctrl-D> Delete",
		}
	case "sessions":
		return []string{
			"<d> Describe",
			"<e> Edit",
			"<l> Logs",
			"<m> Send",
			"<n> New",
			"<y> YAML",
			"<Ctrl-D> Delete",
		}
	case "inbox":
		return []string{
			"<m> Compose",
			"<r> Mark Read",
			"<Ctrl-D> Delete",
		}
	case "messages":
		return []string{
			"<s> Autoscroll",
			"<r> Raw",
			"<m> Send",
			"<c> Copy",
		}
	case "contexts":
		return []string{
			"(Enter to switch)",
		}
	case "detail":
		return []string{
			"<c> Copy",
			"<Esc> Back",
		}
	default:
		return nil
	}
}
