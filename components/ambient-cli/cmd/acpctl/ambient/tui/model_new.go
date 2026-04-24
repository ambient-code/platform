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
	Kind  string // "projects", "agents", "sessions", etc.
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
// AppModel — the Wave 0 TUI model
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

	// View state
	projectTable views.ResourceTable

	// Command mode
	commandMode  bool
	commandInput textinput.Model

	// Filter mode
	filterMode  bool
	filterInput textinput.Model
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

	// Filter bar input.
	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 256

	pt := views.NewProjectTable(views.DefaultTableStyle())

	m := &AppModel{
		config: cfg,
		client: client,
		navStack: []NavEntry{
			{Kind: "projects", Scope: "all"},
		},
		projectTable: pt,
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
		// Delegate scroll events to the project table.
		var cmd tea.Cmd
		m.projectTable, cmd = m.projectTable.Update(msg)
		return m, cmd

	case ProjectsMsg:
		return m.handleProjectsMsg(msg)

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

// resizeTable adjusts the project table dimensions to fill available space.
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
	m.projectTable.SetHeight(tableHeight)
	m.projectTable.SetWidth(m.width)
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

	// Re-apply active filter if present.
	if m.activeFilter != nil {
		f := m.activeFilter
		m.projectTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
	}

	return m, nil
}

// handleTick manages periodic polling. Skips if a fetch is already in flight.
func (m *AppModel) handleTick() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{m.tickCmd()} // always schedule next tick

	if !m.pollInFlight {
		m.pollInFlight = true
		cmds = append(cmds, m.client.FetchProjects())
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
	return m.handleNormalKey(msg)
}

// handleNormalKey processes keys when neither command nor filter mode is active.
func (m *AppModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// Pop navigation stack (if deeper than root).
		if len(m.navStack) > 1 {
			m.navStack = m.navStack[:len(m.navStack)-1]
			return m, m.setInfo("Back to "+m.currentNav().Kind)
		}
		return m, nil

	case tea.KeyEnter:
		// Drill into selected project (Wave 0: just set info — no child views yet).
		row := m.projectTable.SelectedRow()
		if row != nil && len(row) > 0 {
			return m, m.setInfo("Selected project: "+row[0])
		}
		return m, nil

	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		// Delegate to table for row navigation.
		var cmd tea.Cmd
		m.projectTable, cmd = m.projectTable.Update(msg)
		return m, cmd

	case tea.KeyRunes:
		switch msg.String() {
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
			return m, m.setInfo("Help: q quit | : command | / filter | Enter drill-in | Esc back | N sort name | A sort age")

		case "q":
			if len(m.navStack) <= 1 {
				return m, tea.Quit
			}
			// Pop nav stack (same as Esc from child view).
			m.navStack = m.navStack[:len(m.navStack)-1]
			return m, m.setInfo("Back to "+m.currentNav().Kind)

		case "j":
			var cmd tea.Cmd
			m.projectTable, cmd = m.projectTable.Update(tea.KeyMsg{Type: tea.KeyDown})
			return m, cmd

		case "k":
			var cmd tea.Cmd
			m.projectTable, cmd = m.projectTable.Update(tea.KeyMsg{Type: tea.KeyUp})
			return m, cmd

		case "N":
			// Sort by NAME column (index 0).
			m.projectTable.SortByColumn(0)
			return m, nil

		case "A":
			// Sort by AGE column (index 3).
			m.projectTable.SortByColumn(3)
			return m, nil
		}
	}

	return m, nil
}

// handleCommandKey processes keys while in command mode.
func (m *AppModel) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.commandMode = false
		m.commandInput.Reset()
		m.commandInput.Blur()
		m.resizeTable()
		return m, nil

	case tea.KeyEnter:
		input := m.commandInput.Value()
		m.commandMode = false
		m.commandInput.Reset()
		m.commandInput.Blur()
		m.resizeTable()
		return m.executeCommand(input)

	case tea.KeyTab:
		// Tab completion.
		partial := m.commandInput.Value()
		contextNames := m.config.ContextNames()
		// Collect project names from table rows.
		var projectNames []string
		for _, row := range m.projectTable.Rows() {
			if len(row) > 0 {
				projectNames = append(projectNames, row[0])
			}
		}
		suggestions := TabComplete(partial, contextNames, projectNames)
		if len(suggestions) == 1 {
			m.commandInput.SetValue(suggestions[0])
			m.commandInput.CursorEnd()
		}
		return m, nil

	default:
		// Delegate to textinput for character entry.
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
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
		m.pollInFlight = true
		return m, tea.Batch(
			m.client.FetchProjects(),
			m.setInfo("Viewing projects"),
		)

	case CmdContext:
		if cmd.Arg == "" {
			// List contexts.
			names := m.config.ContextNames()
			return m, m.setInfo("Contexts: "+fmt.Sprintf("%v", names))
		}
		// Switch context.
		if err := m.config.SwitchContext(cmd.Arg); err != nil {
			return m, m.setInfo("Error: "+err.Error())
		}
		m.navStack = []NavEntry{{Kind: "projects", Scope: "all"}}
		return m, m.setInfo("Switched to context "+cmd.Arg)

	case CmdProject:
		if cmd.Arg != "" {
			ctx := m.config.Current()
			if ctx != nil {
				ctx.Project = cmd.Arg
			}
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

	case CmdAgents, CmdSessions, CmdInbox, CmdMessages:
		// Not implemented in Wave 0.
		return m, m.setInfo(fmt.Sprintf(":%s not yet implemented (Wave 1+)", input))

	default:
		return m, m.setInfo("Unknown command: "+input)
	}
}

// handleFilterKey processes keys while in filter mode.
func (m *AppModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterMode = false
		m.filterInput.Reset()
		m.filterInput.Blur()
		m.activeFilter = nil
		m.projectTable.ClearFilter()
		m.resizeTable()
		return m, m.setInfo("Filter cleared")

	case tea.KeyEnter:
		input := m.filterInput.Value()
		m.filterMode = false
		m.filterInput.Blur()
		m.resizeTable()

		if input == "" {
			m.activeFilter = nil
			m.projectTable.ClearFilter()
			return m, m.setInfo("Filter cleared")
		}

		f, err := ParseFilter(input)
		if err != nil {
			return m, m.setInfo("Invalid filter: "+err.Error())
		}

		m.activeFilter = f
		m.projectTable.SetFilter(func(cols []string) bool {
			return f.MatchRow(cols)
		})
		return m, m.setInfo("Filter applied: "+f.String())

	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}
}
