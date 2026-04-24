package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
)

// pollInterval is the auto-refresh interval for resource tables.
const pollInterval = 5 * time.Second

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

// infoExpiredMsg signals the ephemeral info line should be cleared.
type infoExpiredMsg struct{}

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

	// Errors
	lastError string

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

	pt := views.NewProjectTable(views.DefaultTableStyle())
	at := views.NewAgentTable("all", views.DefaultTableStyle())
	st := views.NewSessionTable("all", views.DefaultTableStyle())
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
	}

	return m, nil
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
// for the message stream view.
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
		// Delegate scroll events to the active table or message stream.
		if m.activeView == "messages" {
			var cmd tea.Cmd
			m.messageStream, cmd = m.messageStream.Update(msg)
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

	case AgentsMsg:
		return m.handleAgentsMsg(msg)

	case SessionsMsg:
		return m.handleSessionsMsg(msg)

	case InboxMsg:
		return m.handleInboxMsg(msg)

	case views.MsgStreamBackMsg:
		// User pressed Esc in the message stream — pop back.
		cmd := m.popView()
		return m, tea.Batch(cmd, m.setInfo("Back to "+m.currentNav().Kind))

	case views.MsgStreamSendMsg:
		// User composed a message to send to a session.
		return m, m.setInfo("Send message: not yet implemented")

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
	//   separator lines: 2
	// Total chrome: ~10 lines, leaving the rest for the table.
	tableHeight := m.height - 10
	if m.commandMode || m.filterMode {
		tableHeight-- // command bar takes a line
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

	// Message stream gets the full table area.
	m.messageStream.SetSize(m.width, tableHeight+2) // +2 to account for title bar space
}

// handleProjectsMsg populates the project table from a fetch result.
func (m *AppModel) handleProjectsMsg(msg ProjectsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		m.lastError = msg.Err.Error()
		return m, nil
	}

	m.lastError = ""

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
func (m *AppModel) handleAgentsMsg(msg AgentsMsg) (tea.Model, tea.Cmd) {
	m.pollInFlight = false
	m.lastFetch = time.Now()

	if msg.Err != nil {
		m.lastError = msg.Err.Error()
		return m, nil
	}

	m.lastError = ""
	now := time.Now()

	rows := make([]table.Row, 0, len(msg.Agents))
	for _, a := range msg.Agents {
		row := views.AgentRow(a, now)
		// Sanitize all cells.
		for i := range row {
			row[i] = Sanitize(row[i])
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
		m.lastError = msg.Err.Error()
		return m, nil
	}

	m.lastError = ""
	now := time.Now()

	// If agent-scoped, filter sessions to only those belonging to this agent.
	sessions := msg.Sessions
	if m.currentAgentID != "" {
		rows := make([]table.Row, 0)
		for _, s := range sessions {
			if s.AgentID == m.currentAgentID {
				row := views.SessionRow(s, m.currentAgent, now)
				for i := range row {
					row[i] = Sanitize(row[i])
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
			for i := range row {
				row[i] = Sanitize(row[i])
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
		m.lastError = msg.Err.Error()
		return m, nil
	}

	m.lastError = ""
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

// handleTick manages periodic polling. Skips if a fetch is already in flight.
// Fetches data for the active view rather than always fetching projects.
func (m *AppModel) handleTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{m.tickCmd()} // always schedule next tick

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

	if m.commandMode {
		return m.handleCommandKey(msg)
	}
	if m.filterMode {
		return m.handleFilterKey(msg)
	}

	// Message stream handles its own keys.
	if m.activeView == "messages" {
		return m.handleMessagesKey(msg)
	}

	return m.handleNormalKey(msg)
}

// handleNormalKey processes keys when neither command nor filter mode is active.
// Dispatches based on activeView for view-specific hotkeys.
func (m *AppModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keybindings first.
	switch msg.Type {
	case tea.KeyEsc:
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
			// Agent ID comes from the SESSION column (index 2) — but we need the
			// actual ID. For now, we use the agent name and fetch sessions by project.
			// The agent table stores: NAME, PROMPT, SESSION, PHASE, AGE
			m.currentAgent = agentName
			// We don't have the agent ID directly in the table. Use name as a
			// best-effort identifier until the API provides it in the list response.
			m.currentAgentID = agentName
			m.sessionTable.SetScope(agentName)
			cmd := m.pushView("sessions", agentName, "")
			return m, tea.Batch(cmd, m.setInfo("Viewing sessions for agent "+agentName))
		}

	case "sessions":
		row := m.sessionTable.SelectedRow()
		if len(row) > 0 {
			sessionID := row[0] // Short ID is in first column
			m.currentSession = sessionID

			// Create a new message stream for this session.
			agentName := m.currentAgent
			if agentName == "" && len(row) > 1 {
				agentName = row[1] // AGENT column
			}
			phase := ""
			if len(row) > 3 {
				phase = row[3] // PHASE column
			}
			m.messageStream = views.NewMessageStream(sessionID, agentName, phase)
			m.resizeTable() // set message stream dimensions

			// Add placeholder messages since SSE is not wired yet.
			m.messageStream.AddMessage(views.MessageEntry{
				Seq:       1,
				EventType: "system",
				Payload:   "Connected to session " + sessionID + " (SSE not yet wired)",
				Timestamp: time.Now(),
			})

			cmd := m.pushView("messages", sessionID, sessionID)
			return m, tea.Batch(cmd, m.setInfo("Streaming messages for session "+sessionID))
		}

	case "inbox":
		row := m.inboxTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("View full message body: not yet implemented")
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
		return m, m.viewSpecificHelp()

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
		row := m.projectTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Describe project: not yet implemented")
		}
	case "n":
		return m, m.setInfo("New project: not yet implemented")
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
			m.currentAgentID = agentName
			m.inboxTable.SetScope(agentName)
			cmd := m.pushView("inbox", agentName, "")
			return m, tea.Batch(cmd, m.setInfo("Viewing inbox for agent "+agentName))
		}
	case "s":
		return m, m.setInfo("Start agent: not yet implemented")
	case "x":
		return m, m.setInfo("Stop agent: not yet implemented")
	case "e":
		return m, m.setInfo("Edit agent: not yet implemented")
	case "l":
		// Logs — if agent has an active session, jump to message stream.
		row := m.agentTable.SelectedRow()
		if len(row) > 2 && row[2] != "<none>" && row[2] != "" {
			agentName := row[0]
			sessionID := row[2]
			m.currentAgent = agentName
			m.currentAgentID = agentName
			m.currentSession = sessionID
			phase := ""
			if len(row) > 3 {
				phase = row[3]
			}
			m.messageStream = views.NewMessageStream(sessionID, agentName, phase)
			m.resizeTable()
			m.messageStream.AddMessage(views.MessageEntry{
				Seq:       1,
				EventType: "system",
				Payload:   "Connected to session " + sessionID + " (SSE not yet wired)",
				Timestamp: time.Now(),
			})
			cmd := m.pushView("messages", sessionID, sessionID)
			return m, tea.Batch(cmd, m.setInfo("Streaming messages for session "+sessionID))
		}
		return m, m.setInfo("No active session for this agent")
	case "d":
		row := m.agentTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Describe agent: not yet implemented")
		}
	case "m":
		return m, m.setInfo("Send inbox message: not yet implemented")
	case "n":
		return m, m.setInfo("New agent: not yet implemented")
	case "y":
		return m, m.setInfo("YAML dump: not yet implemented")
	}
	return m, nil
}

// handleSessionsRune handles session-view-specific hotkeys.
func (m *AppModel) handleSessionsRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "d":
		row := m.sessionTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Describe session: not yet implemented")
		}
	case "l":
		// Same as Enter — drill into message stream.
		return m.handleEnter()
	case "m":
		return m, m.setInfo("Send message to session: not yet implemented")
	case "y":
		return m, m.setInfo("YAML dump: not yet implemented")
	}
	return m, nil
}

// handleInboxRune handles inbox-view-specific hotkeys.
func (m *AppModel) handleInboxRune(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "m":
		return m, m.setInfo("Compose inbox message: not yet implemented")
	case "r":
		return m, m.setInfo("Mark as read: not yet implemented")
	}
	return m, nil
}

// handleCtrlD handles the delete/cancel keybinding across all views.
func (m *AppModel) handleCtrlD() (tea.Model, tea.Cmd) {
	switch m.activeView {
	case "projects":
		row := m.projectTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Delete project: not yet implemented")
		}
	case "agents":
		row := m.agentTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Delete agent: not yet implemented")
		}
	case "sessions":
		row := m.sessionTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Delete/cancel session: not yet implemented")
		}
	case "inbox":
		row := m.inboxTable.SelectedRow()
		if len(row) > 0 {
			return m, m.setInfo("Delete inbox message: not yet implemented")
		}
	}
	return m, nil
}

// handleMessagesKey delegates key events to the message stream sub-model.
func (m *AppModel) handleMessagesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.messageStream, cmd = m.messageStream.Update(msg)
	return m, cmd
}

// viewSpecificHelp returns a help info message based on the active view.
func (m *AppModel) viewSpecificHelp() tea.Cmd {
	switch m.activeView {
	case "projects":
		return m.setInfo("Help: Enter drill | d describe | n new | Ctrl-D delete | : cmd | / filter | q quit")
	case "agents":
		return m.setInfo("Help: Enter sessions | i inbox | s start | x stop | e edit | l logs | d describe | m send | n new | Ctrl-D delete")
	case "sessions":
		return m.setInfo("Help: Enter/l messages | d describe | m send | y YAML | Ctrl-D delete | q back")
	case "inbox":
		return m.setInfo("Help: Enter view | m compose | r mark read | Ctrl-D delete | q back")
	case "messages":
		return m.setInfo("Help: Esc back | r raw | s scroll | m send | G bottom | g top | / search")
	default:
		return m.setInfo("Help: q quit | : command | / filter | Enter drill-in | Esc back")
	}
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
		return m, m.setInfo("Unknown command: "+input)
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
	}
}

// clearActiveTableFilter removes the filter from the currently active table.
func (m *AppModel) clearActiveTableFilter() {
	if tbl := m.activeTable(); tbl != nil {
		tbl.ClearFilter()
	}
}
