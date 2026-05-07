package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// pollInterval is the auto-refresh interval for resource tables.
const pollInterval = 5 * time.Second

// messagePollInterval is the polling interval for session messages when the
// messages view is active. Faster than the table poll to keep messages fresh.
const messagePollInterval = 1 * time.Second

// infoTimeout is how long ephemeral info messages are displayed.
const infoTimeout = 5 * time.Second

// staleThreshold marks data as stale in the header when exceeded.
const staleThreshold = 15 * time.Second

// numberKeyExcludedViews are views where digit keys do NOT trigger project
// switching (e.g. overlay views, the projects list itself).
var numberKeyExcludedViews = map[string]bool{
	"projects": true,
	"contexts": true,
	"messages": true,
	"detail":   true,
	"inbox":    true,
	"help":     true,
}

// projectShortcutHandledViews are the views explicitly handled in
// handleProjectShortcut's digit-1-9 switch. Every view reachable by number
// keys that is NOT in numberKeyExcludedViews must appear here, or it will
// silently fall through to the agents view.
var projectShortcutHandledViews = map[string]bool{
	"agents":            true,
	"sessions":          true,
	"scheduledsessions": true,
}

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

	scheduledSessionTable views.ResourceTable

	// Current view determines which table/view is active
	activeView string // "projects", "agents", "sessions", "messages", "inbox", "contexts", "scheduledsessions"

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
	cachedProjects          []sdktypes.Project
	cachedAgents            []sdktypes.Agent
	cachedSessions          []sdktypes.Session
	cachedInbox             []sdktypes.InboxMessage
	cachedScheduledSessions []sdktypes.ScheduledSession

	// Message polling state.
	messagePollActive bool // true when message poll tick is running

	// Errors
	lastError   string
	authExpired bool // set on 401 — renders logo red + "Session Expired" badge

	// Dialog overlay for confirm/delete prompts.
	dialog       *views.Dialog
	dialogAction func(value string) tea.Cmd // executed on DialogConfirmMsg{Confirmed: true}

	// Form overlay for multi-field creation dialogs (huh forms).
	formOverlay    *huh.Form
	formTitle      string         // title shown in the form border
	formOnComplete func() tea.Cmd // called when form reaches StateCompleted

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
	sst := views.NewScheduledSessionTable("all", views.DefaultTableStyle())
	// Scheduled session rows: SUSPENDED is column index 3
	// (NAME, SCHEDULE, PROJECT, SUSPENDED, ACTIVE, LAST RUN, AGE)
	// Dim (240) when suspended, orange (214) when active.
	sst.SetRowColorFunc(func(row table.Row) lipgloss.Color {
		if len(row) > 3 && row[3] == "Yes" {
			return lipgloss.Color("240") // dim for suspended
		}
		return lipgloss.Color("214") // orange for active
	})

	m := &AppModel{
		config: cfg,
		client: client,
		navStack: []NavEntry{
			{Kind: "projects", Scope: "all"},
		},
		activeView:            "projects",
		projectTable:          pt,
		agentTable:            at,
		sessionTable:          st,
		inboxTable:            it,
		contextTable:          ct,
		scheduledSessionTable: sst,
		commandInput:          ci,
		filterInput:           fi,
		promptInput:           pi,
	}

	return m, nil
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

// currentUser returns the authenticated username from the JWT token.
func (m *AppModel) currentUser() string {
	if m.config == nil {
		return "unknown"
	}
	ctx := m.config.Current()
	if ctx == nil {
		return "unknown"
	}
	return ctx.Username()
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
	if fetchCmd := m.fetchActiveView(); fetchCmd != nil {
		return fetchCmd
	}
	m.pollInFlight = false
	return nil
}

// popView pops the current navigation entry, switches back to the parent view,
// and returns a fetch command to refresh the parent data.
func (m *AppModel) popView() tea.Cmd {
	if len(m.navStack) <= 1 {
		return nil
	}
	// If we're leaving the messages view, stop polling.
	poppedKind := m.navStack[len(m.navStack)-1].Kind
	if poppedKind == "messages" {
		m.messagePollActive = false
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
	case "scheduledsessions":
		if m.currentProject != "" {
			return m.client.FetchScheduledSessions(m.currentProject)
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
	case "scheduledsessions":
		return &m.scheduledSessionTable
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
	// When a form overlay is active, forward ALL messages to it — huh emits
	// internal messages (nextFieldMsg, etc.) that must round-trip through
	// bubbletea's message loop. Only window-resize and ctrl-c are handled
	// here; everything else goes to the form.
	if m.formOverlay != nil {
		return m.updateFormOverlay(msg)
	}

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

	case ScheduledSessionsMsg:
		return m.handleScheduledSessionsMsg(msg)

	case CreateScheduledSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create scheduled session failed: " + msg.Err.Error())
		}
		name := ""
		if msg.ScheduledSession != nil {
			name = msg.ScheduledSession.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session created: "+name))

	case DeleteScheduledSessionMsg:
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "404") || strings.Contains(msg.Err.Error(), "not found") {
				m.pollInFlight = true
				return m, tea.Batch(m.fetchActiveView(), m.setInfo("Already deleted — refreshing"))
			}
			return m, m.setInfo("Delete scheduled session failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session deleted"))

	case SuspendScheduledSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Suspend failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session suspended"))

	case ResumeScheduledSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Resume failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session resumed"))

	case TriggerScheduledSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Trigger failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session triggered"))

	case InterruptSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Interrupt failed: " + msg.Err.Error())
		}
		return m, m.setInfo("Session interrupted")

	case views.DialogCancelMsg:
		m.dialog = nil
		m.dialogAction = nil
		return m, m.setInfo("Cancelled")

	case views.DialogConfirmMsg:
		return m.handleDialogConfirm(msg)

	case views.MsgStreamCopyMsg:
		// Clipboard copy result from the message stream sub-model.
		if msg.Err != nil {
			return m, m.setInfo("Copy failed: " + msg.Err.Error())
		}
		copied := msg.Text
		if len(copied) > 60 {
			copied = copied[:57] + "..."
		}
		return m, m.setInfo("Copied: " + copied)

	case views.MsgStreamBackMsg:
		// User pressed Esc in the message stream — pop back.
		cmd := m.popView()
		return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))

	case views.MsgStreamSendMsg:
		// User composed a message to send to a session.
		if msg.Body == "" {
			return m, nil
		}
		projectID := m.currentProject
		if projectID == "" {
			// Resolve from cached session data.
			if s := m.findSessionByShortID(m.currentSession); s != nil {
				projectID = s.ProjectID
			}
		}
		if projectID == "" {
			return m, m.setInfo("Cannot send: no project context")
		}
		return m, tea.Batch(
			m.client.SendSessionMessage(projectID, m.currentSession, msg.Body),
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
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo(info))

	case StopAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Stop agent failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent stopped"))

	case CreateAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create agent failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Agent != nil {
			name = msg.Agent.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent created: "+name))

	case DeleteAgentMsg:
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "404") || strings.Contains(msg.Err.Error(), "not found") {
				m.pollInFlight = true
				return m, tea.Batch(m.fetchActiveView(), m.setInfo("Already deleted — refreshing"))
			}
			return m, m.setInfo("Delete agent failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent deleted"))

	case CreateProjectMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create project failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Project != nil {
			name = msg.Project.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project created: "+name))

	case DeleteProjectMsg:
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "404") || strings.Contains(msg.Err.Error(), "not found") {
				m.pollInFlight = true
				return m, tea.Batch(m.fetchActiveView(), m.setInfo("Already deleted — refreshing"))
			}
			return m, m.setInfo("Delete project failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project deleted"))

	case CreateSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Create session failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Session != nil {
			name = msg.Session.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Session created: "+name))

	case DeleteSessionMsg:
		if msg.Err != nil {
			if strings.Contains(msg.Err.Error(), "404") || strings.Contains(msg.Err.Error(), "not found") {
				m.pollInFlight = true
				return m, tea.Batch(m.fetchActiveView(), m.setInfo("Already deleted — refreshing"))
			}
			return m, m.setInfo("Delete session failed: " + msg.Err.Error())
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Session deleted"))

	case UpdateAgentMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update agent failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Agent != nil {
			name = msg.Agent.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Agent updated: "+name))

	case UpdateProjectMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update project failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Project != nil {
			name = msg.Project.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Project updated: "+name))

	case UpdateSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update session failed: " + msg.Err.Error())
		}
		name := ""
		if msg.Session != nil {
			name = msg.Session.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Session updated: "+name))

	case UpdateScheduledSessionMsg:
		if msg.Err != nil {
			return m, m.setInfo("Update scheduled session failed: " + msg.Err.Error())
		}
		name := ""
		if msg.ScheduledSession != nil {
			name = msg.ScheduledSession.Name
		}
		m.pollInFlight = true
		return m, tea.Batch(m.fetchActiveView(), m.setInfo("Scheduled session updated: "+name))

	case editCompleteMsg:
		return m.handleEditComplete(msg)

	case SendMessageMsg:
		if msg.Err != nil {
			return m, m.setInfo("Send message failed: " + msg.Err.Error())
		}
		// Add the user message immediately so it's visible without
		// waiting for the next poll cycle.
		if msg.Message != nil {
			ts := time.Now()
			if msg.Message.CreatedAt != nil {
				ts = *msg.Message.CreatedAt
			}
			m.messageStream.AddMessage(views.MessageEntry{
				Seq:       msg.Message.Seq,
				EventType: "user",
				Payload:   msg.Message.Payload,
				Timestamp: ts,
			})
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

	case SessionMessagesMsg:
		// Polling: batch of messages from REST ListMessages.
		if msg.Err != nil {
			// Non-fatal — polling will retry on next tick, but inform user.
			return m, m.setInfo("Message poll error: " + msg.Err.Error())
		}
		if m.activeView != "messages" {
			return m, nil
		}
		for _, sm := range msg.Messages {
			// Simple seq-based dedup.
			if sm.Seq <= m.messageStream.LastSeq() {
				continue
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
		}
		m.lastFetch = time.Now()
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
		if m.currentSession != "" {
			projectID := m.currentProject
			if projectID == "" {
				if s := m.findSessionByShortID(m.currentSession); s != nil {
					projectID = s.ProjectID
				}
			}
			if projectID != "" {
				cmds = append(cmds, m.client.FetchSessionMessages(
					projectID, m.currentSession, m.messageStream.LastSeq(),
				))
			}
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
	//   header block: 6 lines (5-line grid + server row)
	//   command/filter bar: 1 line (when visible) — accounted for dynamically
	//   title bar: 1 line
	//   breadcrumb: 1 line
	//   info line: 1 line
	// Total chrome: ~9 lines, leaving the rest for the table.
	tableHeight := m.height - 9
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
	m.scheduledSessionTable.SetHeight(tableHeight)
	m.scheduledSessionTable.SetWidth(m.width)

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
		m.authExpired = true
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
	m.authExpired = false
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
			age = views.FormatAge(time.Since(*p.CreatedAt))
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

	if len(msg.Projects) >= 200 {
		cmds = append(cmds, m.setInfo("Showing first 200 projects"))
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
			age = views.FormatAge(now.Sub(*p.CreatedAt))
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
	m.authExpired = false
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

	if len(msg.Agents) >= 200 {
		cmds = append(cmds, m.setInfo("Showing first 200 agents"))
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
	m.authExpired = false
	m.cachedSessions = msg.Sessions
	now := time.Now()

	// If agent-scoped, filter sessions to only those belonging to this agent.
	sessions := msg.Sessions
	if m.currentAgentID != "" {
		rows := make([]table.Row, 0)
		for _, s := range sessions {
			if s.AgentID == m.currentAgentID {
				row := views.SessionRow(s, m.currentAgent, now)
				// Sanitize all cells except PHASE (index 4): [ID(0), NAME(1), AGENT(2), PROJECT(3), PHASE(4), STARTED(5), DURATION(6)].
				for i := range row {
					if i != 4 {
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
			// Sanitize all cells except PHASE (index 4): [ID(0), NAME(1), AGENT(2), PROJECT(3), PHASE(4), STARTED(5), DURATION(6)].
			for i := range row {
				if i != 4 {
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

	if len(msg.Sessions) >= 200 {
		return m, m.setInfo("Showing first 200 sessions")
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
	m.authExpired = false
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

	if len(msg.Messages) >= 200 {
		return m, m.setInfo("Showing first 200 inbox messages")
	}

	return m, nil
}

// handleScheduledSessionsMsg populates the scheduled session table from a fetch result.
func (m *AppModel) handleScheduledSessionsMsg(msg ScheduledSessionsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		errMsg, skipPoll := m.classifyAPIError(msg.Err, "scheduled sessions")
		m.lastError = errMsg
		m.skipNextPoll = m.skipNextPoll || skipPoll
		return m, nil
	}

	m.lastError = ""
	m.authExpired = false
	m.cachedScheduledSessions = msg.ScheduledSessions
	now := time.Now()

	rows := make([]table.Row, 0, len(msg.ScheduledSessions))
	for _, ss := range msg.ScheduledSessions {
		row := views.ScheduledSessionRow(ss, now)
		for i := range row {
			row[i] = Sanitize(row[i])
		}
		rows = append(rows, row)
	}
	m.scheduledSessionTable.SetRows(rows)

	// Re-apply active filter if present and we're on scheduled sessions view.
	if m.activeView == "scheduledsessions" && m.activeFilter != nil {
		f := m.activeFilter
		m.scheduledSessionTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	if len(msg.ScheduledSessions) >= 200 {
		return m, m.setInfo("Showing first 200 scheduled sessions")
	}

	return m, nil
}

// findScheduledSessionByName returns the cached ScheduledSession with the given
// name, or nil.
func (m *AppModel) findScheduledSessionByName(name string) *sdktypes.ScheduledSession {
	for i := range m.cachedScheduledSessions {
		if m.cachedScheduledSessions[i].Name == name {
			return &m.cachedScheduledSessions[i]
		}
	}
	return nil
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
		m.messagePollActive = false
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
// returns the resulting command to the bubbletea runtime for dispatch.
func (m *AppModel) handleDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dlg, cmd := m.dialog.Update(msg)
	m.dialog = &dlg
	return m, cmd
}

// handleDialogResult processes DialogConfirmMsg / DialogCancelMsg delivered
// by the bubbletea runtime (rather than being called inline).
func (m *AppModel) handleDialogConfirm(confirm views.DialogConfirmMsg) (tea.Model, tea.Cmd) {
	if confirm.Confirmed {
		fn := m.dialogAction
		m.dialog = nil
		m.dialogAction = nil
		if fn != nil {
			return m, tea.Batch(fn(confirm.Value), m.setInfo("Processing..."))
		}
	} else {
		m.dialog = nil
		infoText := "Cancelled"
		if m.dialogAction == nil {
			infoText = "Dismissed"
		}
		m.dialogAction = nil
		return m, m.setInfo(infoText)
	}
	m.dialog = nil
	m.dialogAction = nil
	return m, nil
}

// updateFormOverlay forwards all messages to the active huh form and detects
// completion or abort. Called from the top of Update() before the type switch
// so that huh's internal messages (nextFieldMsg, etc.) are properly routed.
func (m *AppModel) updateFormOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle resize even while form is active.
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.resizeTable()
	}

	// Don't swallow tick messages — they keep the poll and UI refresh chains alive.
	if _, ok := msg.(appTickMsg); ok {
		return m.handleTick()
	}
	// Don't swallow data-fetch responses — they clear pollInFlight and update caches.
	switch typedMsg := msg.(type) {
	case ProjectsMsg:
		return m.handleProjectsMsg(typedMsg)
	case AgentsMsg:
		return m.handleAgentsMsg(typedMsg)
	case SessionsMsg:
		return m.handleSessionsMsg(typedMsg)
	case InboxMsg:
		return m.handleInboxMsg(typedMsg)
	case ProjectCountsMsg:
		return m.handleProjectCountsMsg(typedMsg)
	case AgentCountsMsg:
		return m.handleAgentCountsMsg(typedMsg)
	case ScheduledSessionsMsg:
		return m.handleScheduledSessionsMsg(typedMsg)
	}

	// Esc dismisses the form (huh uses ctrl+c for its own abort).
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.Type == tea.KeyEsc {
			m.formOverlay = nil
			m.formTitle = ""
			m.formOnComplete = nil
			return m, m.setInfo("Cancelled")
		}
		if key.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	// Forward everything to the form.
	model, cmd := m.formOverlay.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.formOverlay = f
	}

	// Check terminal states.
	switch m.formOverlay.State {
	case huh.StateCompleted:
		fn := m.formOnComplete
		m.formOverlay = nil
		m.formTitle = ""
		m.formOnComplete = nil
		if fn != nil {
			return m, tea.Batch(fn(), m.setInfo("Processing..."))
		}
		return m, nil
	case huh.StateAborted:
		m.formOverlay = nil
		m.formTitle = ""
		m.formOnComplete = nil
		return m, m.setInfo("Cancelled")
	}

	return m, cmd
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
				return m, m.setInfo("Error: " + err.Error())
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
				m.setInfo("Viewing messages for session " + shortID),
			}

			// Resolve project ID — may be empty if reached from global sessions.
			projectID := m.currentProject
			if projectID == "" && session != nil {
				projectID = session.ProjectID
			}

			if projectID != "" {
				// Fetch initial messages and start 1-second polling.
				cmds = append(cmds, m.client.FetchSessionMessages(projectID, fullSessionID, 0))
				m.messagePollActive = true
				cmds = append(cmds, m.messagePollTickCmd())
			}

			return m, tea.Batch(cmds...)
		}

	case "scheduledsessions":
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) > 0 {
			name := row[0]
			ss := m.findScheduledSessionByName(name)
			if ss == nil {
				return m, m.setInfo("Scheduled session not found in cache: " + name)
			}
			// Show detail view for the scheduled session.
			m.detailView = views.NewDetailView("Scheduled: "+name, views.ScheduledSessionDetail(*ss))
			m.detailView.SetSize(m.width, m.height-10)
			cmd := m.pushView("detail", name, ss.ID)
			return m, tea.Batch(cmd, m.setInfo("Scheduled session detail: "+name))
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
			m.messagePollActive = false
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
		// For sessions, resolve the full ID from cache (table shows truncated short IDs).
		if tbl := m.activeTable(); tbl != nil {
			row := tbl.SelectedRow()
			if len(row) > 0 {
				value := row[0]
				// Resolve full session ID from cache if we're in sessions view.
				if m.activeView == "sessions" {
					if s := m.findSessionByShortID(value); s != nil {
						value = s.ID
					}
				}
				if err := clipboard.WriteAll(value); err != nil {
					return m, m.setInfo("Copy failed: " + err.Error())
				}
				return m, m.setInfo("Copied: " + value)
			}
		}
		return m, nil
	}

	// Number-key project shortcuts (0-9) — only active on table views below project level.
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' &&
		!numberKeyExcludedViews[m.activeView] {
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
	case "scheduledsessions":
		return m.handleScheduledSessionsRune(key)
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
		var name, description string
		form := views.NewProjectForm(&name, &description)
		form.WithWidth(60)
		m.formOverlay = form
		m.formTitle = "New Project"
		m.formOnComplete = func() tea.Cmd {
			return tea.Batch(
				m.client.CreateProject(name, description),
				m.setInfo("Creating project "+name+"..."),
			)
		}
		return m, m.formOverlay.Init()
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
		m.messageStream = views.NewMessageStream(sessionID, agentName, "active")
		m.resizeTable()

		cmds := []tea.Cmd{
			m.pushView("messages", sessionID, sessionID),
			m.setInfo("Viewing messages for session " + sessionID),
		}

		if m.currentProject != "" {
			cmds = append(cmds, m.client.FetchSessionMessages(m.currentProject, sessionID, 0))
			m.messagePollActive = true
			cmds = append(cmds, m.messagePollTickCmd())
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
		if m.currentProject == "" {
			return m, m.setInfo("Navigate to a project first")
		}
		project := m.currentProject
		var name, prompt string
		form := views.NewAgentForm(&name, &prompt)
		form.WithWidth(60)
		m.formOverlay = form
		m.formTitle = "New Agent"
		m.formOnComplete = func() tea.Cmd {
			return tea.Batch(
				m.client.CreateAgent(project, name, prompt),
				m.setInfo("Creating agent "+name+"..."),
			)
		}
		return m, m.formOverlay.Init()
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
		m.detailView = views.NewDetailView("JSON: "+agentName, views.ResourceJSON(*agent))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", agentName, agent.ID)
		return m, tea.Batch(cmd, m.setInfo("JSON: "+agentName))
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
		var name, prompt, projectID, agentID string
		// Pre-select current project if set.
		projectID = m.currentProject
		// Pre-select current agent if set.
		agentID = m.currentAgentID
		// Build project options from cache.
		var projectOpts []huh.Option[string]
		for _, p := range m.cachedProjects {
			opt := huh.NewOption(p.Name, p.Name)
			if p.Name == projectID {
				opt = opt.Selected(true)
			}
			projectOpts = append(projectOpts, opt)
		}
		if len(projectOpts) == 0 {
			return m, m.setInfo("Navigate to projects view first to populate project list")
		}
		// Build agent options from cache, filtered to the selected project.
		agentOpts := []huh.Option[string]{
			huh.NewOption("(none — standalone)", ""),
		}
		for _, a := range m.cachedAgents {
			// Only show agents belonging to the pre-selected project.
			if projectID != "" && a.ProjectID != projectID {
				continue
			}
			agentOpts = append(agentOpts, huh.NewOption(a.Name, a.ID))
		}
		var repoURL string
		form := views.NewSessionForm(&name, &prompt, &repoURL, &projectID, projectOpts, &agentID, agentOpts)
		form.WithWidth(60)
		m.formOverlay = form
		m.formTitle = "New Session"
		m.formOnComplete = func() tea.Cmd {
			if projectID == "" {
				return m.setInfo("Project is required")
			}
			return tea.Batch(
				m.client.CreateSession(projectID, name, prompt, agentID, repoURL),
				m.setInfo("Creating session "+name+"..."),
			)
		}
		return m, m.formOverlay.Init()
	case "x":
		// Interrupt the selected session.
		row := m.sessionTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No session selected")
		}
		shortID := row[0]
		session := m.findSessionByShortID(shortID)
		if session == nil {
			return m, m.setInfo("Session not found in cache: " + shortID)
		}
		capturedSessionID := session.ID
		d := views.NewConfirmDialog("Interrupt", "Interrupt session "+session.Name+"?")
		m.dialog = &d
		m.dialogAction = func(_ string) tea.Cmd {
			return m.client.InterruptSession(capturedSessionID)
		}
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
		m.detailView = views.NewDetailView("JSON: "+shortID, views.ResourceJSON(*session))
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

// handleScheduledSessionsRune handles scheduled-session-view-specific hotkeys.
func (m *AppModel) handleScheduledSessionsRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "e":
		return m.openEditorForScheduledSession()
	case "d":
		// Show detail view for the selected scheduled session.
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		name := row[0]
		ss := m.findScheduledSessionByName(name)
		if ss == nil {
			return m, m.setInfo("Scheduled session not found in cache: " + name)
		}
		m.detailView = views.NewDetailView("Scheduled: "+name, views.ScheduledSessionDetail(*ss))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", name, ss.ID)
		return m, tea.Batch(cmd, m.setInfo("Scheduled session detail: "+name))

	case "n":
		// Create new scheduled session.
		if m.currentProject == "" {
			return m, m.setInfo("Navigate to a project first")
		}
		project := m.currentProject
		var agentOpts []huh.Option[string]
		for _, a := range m.cachedAgents {
			if a.ProjectID != project {
				continue
			}
			agentOpts = append(agentOpts, huh.NewOption(a.Name, a.ID))
		}
		var name, schedule, description, sessionPrompt, timezone, agentID string
		form := views.NewScheduledSessionForm(&name, &schedule, &description, &sessionPrompt, &timezone, &agentID, agentOpts)
		form.WithWidth(60)
		m.formOverlay = form
		m.formTitle = "New Scheduled Session"
		m.formOnComplete = func() tea.Cmd {
			return tea.Batch(
				m.client.CreateScheduledSession(project, name, agentID, schedule, timezone, sessionPrompt, description),
				m.setInfo("Creating scheduled session "+name+"..."),
			)
		}
		return m, m.formOverlay.Init()

	case "s":
		// Suspend/resume toggle.
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No scheduled session selected")
		}
		name := row[0]
		ss := m.findScheduledSessionByName(name)
		if ss == nil {
			return m, m.setInfo("Scheduled session not found in cache: " + name)
		}
		if !ss.Enabled {
			return m, tea.Batch(
				m.client.ResumeScheduledSession(m.currentProject, ss.ID),
				m.setInfo("Resuming "+name+"..."),
			)
		}
		return m, tea.Batch(
			m.client.SuspendScheduledSession(m.currentProject, ss.ID),
			m.setInfo("Suspending "+name+"..."),
		)

	case "t":
		// Trigger manual run with confirmation.
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) == 0 {
			return m, m.setInfo("No scheduled session selected")
		}
		name := row[0]
		ss := m.findScheduledSessionByName(name)
		if ss == nil {
			return m, m.setInfo("Scheduled session not found in cache: " + name)
		}
		ssID := ss.ID
		currentProject := m.currentProject
		d := views.NewConfirmDialog("Trigger", "Trigger manual run of "+name+"?")
		m.dialog = &d
		m.dialogAction = func(_ string) tea.Cmd {
			return m.client.TriggerScheduledSession(currentProject, ssID)
		}
		return m, nil

	case "y":
		// JSON view.
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		name := row[0]
		ss := m.findScheduledSessionByName(name)
		if ss == nil {
			return m, m.setInfo("Scheduled session not found in cache: " + name)
		}
		m.detailView = views.NewDetailView("JSON: "+name, views.ResourceJSON(*ss))
		m.detailView.SetSize(m.width, m.height-10)
		cmd := m.pushView("detail", name, ss.ID)
		return m, tea.Batch(cmd, m.setInfo("JSON: "+name))
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
			m.dialogAction = func(_ string) tea.Cmd {
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
			m.dialogAction = func(_ string) tea.Cmd {
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
			m.dialogAction = func(_ string) tea.Cmd {
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
			m.dialogAction = func(_ string) tea.Cmd {
				return m.client.DeleteInboxMessage(currentProject, currentAgentID, msgID)
			}
			return m, nil
		}
	case "scheduledsessions":
		row := m.scheduledSessionTable.SelectedRow()
		if len(row) > 0 {
			name := row[0]
			ss := m.findScheduledSessionByName(name)
			if ss == nil {
				return m, m.setInfo("Scheduled session not found in cache: " + name)
			}
			ssID := ss.ID
			currentProject := m.currentProject
			d := views.NewDeleteDialog("scheduled session", name)
			m.dialog = &d
			m.dialogAction = func(_ string) tea.Cmd {
				return m.client.DeleteScheduledSession(currentProject, ssID)
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
	// When compose mode is active, ALL keys go to the message stream —
	// don't intercept :, ?, q etc. as they're meant to be typed.
	if m.messageStream.IsComposeMode() {
		var cmd tea.Cmd
		m.messageStream, cmd = m.messageStream.Update(msg)
		return m, cmd
	}

	// Intercept global keys before delegating to the message stream.
	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case ":":
			m.commandMode = true
			m.commandInput.Focus()
			m.resizeTable()
			return m, nil
		case "/":
			m.promptMode = true
			m.promptInput.Prompt = "Search: "
			m.promptInput.Reset()
			m.promptInput.Focus()
			m.promptCallback = func(input string) (tea.Model, tea.Cmd) {
				if input == "" {
					m.messageStream.SetSearchPattern(nil)
					return m, m.setInfo("Search cleared")
				}
				pat, err := regexp.Compile("(?i)" + input)
				if err != nil {
					return m, m.setInfo("Invalid pattern: " + err.Error())
				}
				m.messageStream.SetSearchPattern(pat)
				return m, m.setInfo("Searching: " + input)
			}
			m.resizeTable()
			return m, nil
		case "?":
			return m.showHelp()
		case "q":
			return m, m.popView()
		case "x":
			// Interrupt the current session.
			if m.currentSession == "" {
				return m, m.setInfo("No session context for interrupt")
			}
			// Resolve session display name from cache for the dialog.
			sessionLabel := m.currentSession
			capturedSessionID := m.currentSession
			if s := m.findSessionByShortID(m.currentSession); s != nil {
				sessionLabel = s.Name
				capturedSessionID = s.ID
			}
			d := views.NewConfirmDialog("Interrupt", "Interrupt session "+sessionLabel+"?")
			m.dialog = &d
			m.dialogAction = func(_ string) tea.Cmd {
				return m.client.InterruptSession(capturedSessionID)
			}
			return m, nil
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
// Hints are pulled from the viewHintRegistry (hints.go) — the single source of truth.
func (m *AppModel) showHelp() (tea.Model, tea.Cmd) {
	viewName := m.activeView
	h := hintsForView(viewName)

	m.helpView = views.NewHelpView(viewName, h.Resource, h.General, h.Navigation)
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
	// If we're leaving the messages view via a command, stop polling.
	if m.activeView == "messages" {
		m.messagePollActive = false
	}

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

	case CmdScheduledSessions:
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
		m.scheduledSessionTable.SetScope(project)
		m.navStack = []NavEntry{
			{Kind: "projects", Scope: "all"},
			{Kind: "scheduledsessions", Scope: project},
		}
		m.activeView = "scheduledsessions"
		m.activeFilter = nil
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchScheduledSessions(project),
			m.setInfo("Viewing scheduled sessions in project "+project),
		)

	case CmdMessages:
		return m, m.setInfo("Use Enter from sessions view to open messages")

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
			return m, m.setInfo("Error: " + err.Error())
		}
		// Reset everything on context switch.
		m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
		m.activeView = "projects"
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.activeFilter = nil
		return m, m.setInfo("Switched to context " + cmd.Arg)

	case CmdProject:
		if cmd.Arg != "" {
			ctx := m.config.Current()
			if ctx != nil {
				ctx.Project = cmd.Arg
			}
			m.currentProject = cmd.Arg
			return m, m.setInfo("Switched to project " + cmd.Arg)
		}
		return m, nil

	case CmdAliases:
		entries := AliasTable()
		var detailLines []views.DetailLine
		for _, e := range entries {
			aliases := ""
			if len(e.Aliases) > 0 {
				aliases = " (" + strings.Join(e.Aliases, ", ") + ")"
			}
			detailLines = append(detailLines, views.DetailLine{
				Key:   e.Command + aliases,
				Value: e.Description,
			})
		}
		m.detailView = views.NewDetailView("Commands", detailLines)
		m.detailView.SetSize(m.width, m.height-10)
		cmdPush := m.pushView("detail", "aliases", "")
		return m, tea.Batch(cmdPush, m.setInfo(fmt.Sprintf("%d commands available", len(entries))))

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
			return m, m.setInfo("Invalid filter: " + err.Error())
		}

		m.activeFilter = f
		m.applyFilterToActiveTable(f)
		return m, m.setInfo("Filter applied: " + f.String())

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
		// Switch to "all" — clear project scope, navigate back to projects view.
		m.currentProject = ""
		m.currentAgent = ""
		m.currentAgentID = ""
		m.currentSession = ""
		m.activeFilter = nil
		m.pollInFlight = true

		switch m.activeView {
		case "agents":
			// Can't list all agents across projects — go back to projects view.
			m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
			m.activeView = "projects"
			return m, tea.Batch(m.client.FetchProjects(), m.setInfo("Back to projects"))
		case "sessions":
			m.sessionTable.SetScope("all")
			m.navStack = []NavEntry{{Kind: "sessions", Scope: "all"}}
			return m, tea.Batch(m.client.FetchAllSessions(), m.setInfo("Viewing all sessions"))
		case "inbox":
			m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
			m.activeView = "projects"
			return m, tea.Batch(m.client.FetchProjects(), m.setInfo("Viewing all projects"))
		case "scheduledsessions":
			m.navStack = []NavEntry{{Kind: "scheduledsessions", Scope: "all"}}
			m.scheduledSessionTable.SetScope("all")
			return m, tea.Batch(m.client.FetchScheduledSessions(""), m.setInfo("Viewing all scheduled sessions"))
		default:
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
	case "scheduledsessions":
		m.scheduledSessionTable.SetScope(projectName)
		m.navStack = []NavEntry{
			{Kind: "scheduledsessions", Scope: projectName},
		}
		m.activeView = "scheduledsessions"
		return m, tea.Batch(
			m.client.FetchScheduledSessions(projectName),
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

// openEditorForScheduledSession serializes the selected scheduled session as
// JSON, writes it to a temp file, and suspends the TUI to open the user's
// $EDITOR.
func (m *AppModel) openEditorForScheduledSession() (tea.Model, tea.Cmd) {
	row := m.scheduledSessionTable.SelectedRow()
	if len(row) == 0 {
		return m, m.setInfo("No scheduled session selected")
	}
	name := row[0]
	ss := m.findScheduledSessionByName(name)
	if ss == nil {
		return m, m.setInfo("Scheduled session not found in cache: " + name)
	}
	if m.currentProject == "" {
		return m, m.setInfo("No project context for edit")
	}

	return m.openEditorForResource("scheduledsession", ss.ID, m.currentProject, *ss)
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

	header := "// Please edit the object below. Lines beginning with '//' will be ignored,\n" +
		"// and an empty file will abort the edit. If an error occurs while saving,\n" +
		"// this file will be reopened with the relevant failures.\n" +
		"//\n"
	if _, err := tmpFile.WriteString(header); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return m, m.setInfo("Failed to write temp file: " + err.Error())
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
	if msg.Err != nil {
		os.Remove(msg.TempFile)
		return m, m.setInfo("Editor exited with error: " + msg.Err.Error())
	}

	// Read the edited file.
	editedJSON, err := os.ReadFile(msg.TempFile)
	if err != nil {
		return m, m.setInfo("Failed to read edited file: " + err.Error())
	}

	// Strip comment lines (// ...) before parsing.
	strippedJSON := stripJSONComments(string(editedJSON))

	// Empty file = abort.
	if strings.TrimSpace(strippedJSON) == "" {
		return m, m.setInfo("Edit aborted (empty file)")
	}

	// Parse both original and edited JSON into maps for diffing.
	var original map[string]any
	if err := json.Unmarshal(msg.OriginalJSON, &original); err != nil {
		return m, m.setInfo("Failed to parse original JSON: " + err.Error())
	}
	var edited map[string]any
	if err := json.Unmarshal([]byte(strippedJSON), &edited); err != nil {
		// Reopen the editor with the error as a comment at the top.
		errorHeader := fmt.Sprintf("// ERROR: %s\n// Fix the JSON below and save again. Empty file aborts.\n//\n", err.Error())
		_ = os.WriteFile(msg.TempFile, []byte(errorHeader+string(editedJSON)), 0600)
		editor := getEditor()
		c := exec.Command(editor, msg.TempFile) //nolint:gosec
		return m, tea.ExecProcess(c, func(editorErr error) tea.Msg {
			return editCompleteMsg{
				ResourceKind: msg.ResourceKind,
				ResourceID:   msg.ResourceID,
				ProjectID:    msg.ProjectID,
				TempFile:     msg.TempFile,
				OriginalJSON: msg.OriginalJSON,
				Err:          editorErr,
			}
		})
	}

	// Determine which fields are editable based on resource kind.
	var editableFields []string
	switch msg.ResourceKind {
	case "agent":
		editableFields = []string{
			"name", "prompt", "labels", "annotations",
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
	case "scheduledsession":
		editableFields = []string{
			"name", "description", "schedule", "timezone",
			"session_prompt", "agent_id", "enabled",
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
		os.Remove(msg.TempFile)
		return m, m.setInfo("No changes detected")
	}
	os.Remove(msg.TempFile)

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
	case "scheduledsession":
		return m, tea.Batch(
			m.client.UpdateScheduledSession(msg.ProjectID, msg.ResourceID, patch),
			m.setInfo("Updating scheduled session ("+summary+")..."),
		)
	default:
		return m, m.setInfo("Unknown resource kind: " + msg.ResourceKind)
	}
}

// stripJSONComments removes lines starting with // from the input.
func stripJSONComments(s string) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Contextual hotkey hints for the header
// ---------------------------------------------------------------------------

// contextualHints returns the hotkey hints for the current active view,
// derived from the viewHintRegistry (hints.go).
func (m *AppModel) contextualHints() []string {
	h := hintsForView(m.activeView)
	var out []string
	for _, e := range h.Resource {
		out = append(out, "<"+e.Key+"> "+e.Action)
	}
	return out
}
