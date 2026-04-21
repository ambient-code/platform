package tui

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
)

type NavSection int

const (
	NavDashboard  NavSection = iota // system controls, port-forwards, login
	NavCluster                      // system pods in ambient-code namespace
	NavNamespaces                   // fleet-* namespaces
	NavProjects                     // SDK projects list
	NavSessions                     // SDK sessions list
	NavAgents                       // SDK agents list
)

var navLabels = []string{
	"Dashboard",
	"Cluster Pods",
	"Namespaces",
	"Projects",
	"Sessions",
	"Agents",
}

type PodRow struct {
	Namespace string
	Name      string
	Ready     string
	Status    string
	Restarts  string
	Age       string
}

type NamespaceRow struct {
	Name   string
	Status string
	Age    string
}

type PortForwardEntry struct {
	Label     string
	SvcName   string
	LocalPort int
	SvcPort   int
	PID       int
	Running   bool
}

type LoginStatus struct {
	LoggedIn  bool
	User      string
	Server    string
	Namespace string
}

type DashData struct {
	Pods       []PodRow
	Namespaces []NamespaceRow
	Projects   []sdktypes.Project
	Sessions   []sdktypes.Session
	Agents     []sdktypes.Agent
	FetchedAt  time.Time
	Err        string
}

type cmdInputModel struct {
	value  string
	cursor int
}

func (c *cmdInputModel) insert(ch rune) {
	s := []rune(c.value)
	s = append(s[:c.cursor], append([]rune{ch}, s[c.cursor:]...)...)
	c.value = string(s)
	c.cursor++
}

func (c *cmdInputModel) backspace() {
	if c.cursor > 0 {
		s := []rune(c.value)
		s = append(s[:c.cursor-1], s[c.cursor:]...)
		c.value = string(s)
		c.cursor--
	}
}

func (c *cmdInputModel) deleteForward() {
	s := []rune(c.value)
	if c.cursor < len(s) {
		s = append(s[:c.cursor], s[c.cursor+1:]...)
		c.value = string(s)
	}
}

func (c *cmdInputModel) moveLeft() {
	if c.cursor > 0 {
		c.cursor--
	}
}
func (c *cmdInputModel) moveRight() {
	if c.cursor < len([]rune(c.value)) {
		c.cursor++
	}
}
func (c *cmdInputModel) moveHome() { c.cursor = 0 }
func (c *cmdInputModel) moveEnd()  { c.cursor = len([]rune(c.value)) }
func (c *cmdInputModel) clear()    { c.value = ""; c.cursor = 0 }

func (c *cmdInputModel) render() string {
	runes := []rune(c.value)
	cur := c.cursor
	if cur >= len(runes) {
		return styleBlue.Render("$ ") + string(runes) + styleBold.Render("█")
	}
	before := string(runes[:cur])
	cursorChar := string(runes[cur : cur+1])
	after := string(runes[cur+1:])
	return styleBlue.Render("$ ") + before + styleBold.Render(cursorChar) + after
}

type Model struct {
	client             *sdkclient.Client
	clientFactory      *connection.ClientFactory
	width              int
	height             int
	nav                NavSection
	data               DashData
	mainLines          []string
	mainScroll         int
	input              cmdInputModel
	history            []string
	histIdx            int
	cmdRunning         bool
	refreshing         bool
	lastFetch          time.Time
	msgCh              chan tea.Msg
	cmdFocus           bool
	sessionMsgs        map[string][]sdktypes.SessionMessage
	sessionWatching    map[string]context.CancelFunc
	sessionTileContent map[string][2]int

	portForwards []PortForwardEntry
	loginStatus  LoginStatus

	panelFocus        bool
	panelRow          int
	detailMode        bool
	detailLines       []string
	detailTitle       string
	detailScroll      int
	detailSelectable  bool
	detailRow         int
	detailItems       []detailItem
	detailHeaderLines int

	detailSplit        bool
	detailTopLines     []string
	detailBottomLines  []string
	detailTopScroll    int
	detailBottomScroll int
	detailSplitFocus   int

	composeMode      bool
	composeSessionID string
	composeInput     cmdInputModel
	composeStatus    string

	agentEditMode    bool
	agentEditAgent   sdktypes.Agent
	agentEditPrompt  string
	agentEditDirty   bool
	agentEditCursor  int
	agentEditStatus  string
	agentEditEscOnce bool

	agentConfirmDelete bool
	agentDeleteID      string
	agentDeleteName    string
}

func NewModel(client *sdkclient.Client, factory *connection.ClientFactory) *Model {
	return &Model{
		client:        client,
		clientFactory: factory,
		msgCh:         make(chan tea.Msg, 256),
		histIdx:       -1,
		nav:           NavDashboard,
		portForwards: []PortForwardEntry{
			{Label: "REST API", SvcName: "ambient-api-server", LocalPort: 18000, SvcPort: 8000},
			{Label: "gRPC", SvcName: "ambient-api-server", LocalPort: 19000, SvcPort: 9000},
			{Label: "Frontend", SvcName: "frontend-service", LocalPort: 18080, SvcPort: 3000},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		m.listenForMsgs(),
		func() tea.Msg { return refreshMsg{} },
		m.tickCmd(),
		m.checkPortForwards(),
	)
}

func (m *Model) listenForMsgs() tea.Cmd {
	return func() tea.Msg { return <-m.msgCh }
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{t}
	})
}

type refreshMsg struct{}
type tickMsg struct{ t time.Time }
type dataMsg struct{ data DashData }
type sessionMsgsMsg struct {
	sessionID string
	msg       sdktypes.SessionMessage
}
type cmdOutputMsg struct {
	text string
	kind lineKind
}
type cmdDoneMsg struct{}

type lineKind int

const (
	lkNormal lineKind = iota
	lkDim
	lkGreen
	lkRed
	lkYellow
	lkCyan
	lkOrange
	lkBold
	lkHeader
)

type detailItem struct {
	namespace string
	name      string
	id        string
	kind      string
}

type detailReadyMsg struct {
	title       string
	lines       []string
	selectable  bool
	items       []detailItem
	headerLines int
}

type composeSentMsg struct{ sessionID string }
type composeErrMsg struct{ err string }

type splitDetailReadyMsg struct {
	title       string
	topLines    []string
	bottomLines []string
}

type agentSavedMsg struct{ agent sdktypes.Agent }
type agentSaveErrMsg struct{ err string }
type agentDeletedMsg struct{ id string }
type agentDeleteErrMsg struct{ err string }

type pfStatusMsg struct {
	idx     int
	running bool
	pid     int
}
type loginStatusMsg struct{ status LoginStatus }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildMain()

	case splitDetailReadyMsg:
		m.detailTitle = msg.title
		m.detailSplit = true
		m.detailTopLines = msg.topLines
		m.detailBottomLines = msg.bottomLines
		m.detailTopScroll = 0
		m.detailBottomScroll = 0
		m.detailSplitFocus = 0
		m.detailMode = true
		m.detailSelectable = false
		m.detailLines = nil
		return m, m.listenForMsgs()

	case detailReadyMsg:
		m.detailTitle = msg.title
		m.detailLines = msg.lines
		m.detailScroll = 0
		m.detailRow = 0
		m.detailSelectable = msg.selectable
		m.detailItems = msg.items
		m.detailHeaderLines = msg.headerLines
		m.detailMode = true
		if msg.selectable {
			m.applyDetailCursor(msg.headerLines)
		}
		return m, m.listenForMsgs()

	case agentSavedMsg:
		m.agentEditStatus = styleGreen.Render("✓ saved")
		m.agentEditDirty = false
		for i, a := range m.data.Agents {
			if a.ID == msg.agent.ID {
				m.data.Agents[i] = msg.agent
				m.agentEditAgent = msg.agent
			}
		}
		m.rebuildMain()
		return m, m.listenForMsgs()

	case agentSaveErrMsg:
		m.agentEditStatus = styleRed.Render("✗ " + msg.err)
		return m, m.listenForMsgs()

	case agentDeletedMsg:
		m.agentConfirmDelete = false
		m.agentDeleteID = ""
		m.agentDeleteName = ""
		var updated []sdktypes.Agent
		for _, a := range m.data.Agents {
			if a.ID != msg.id {
				updated = append(updated, a)
			}
		}
		m.data.Agents = updated
		m.panelFocus = false
		m.panelRow = 0
		m.rebuildMain()
		return m, m.listenForMsgs()

	case agentDeleteErrMsg:
		m.agentConfirmDelete = false
		m.agentEditStatus = styleRed.Render("✗ delete failed: " + msg.err)
		return m, m.listenForMsgs()

	case tea.KeyMsg:
		if m.agentEditMode {
			return m.updateAgentEdit(msg)
		}
		if m.agentConfirmDelete {
			return m.updateAgentDeleteConfirm(msg)
		}
		if m.composeMode {
			return m.updateComposeFocused(msg)
		}
		if m.cmdFocus {
			return m.updateInputFocused(msg)
		}
		if m.detailMode {
			return m.updateDetailMode(msg)
		}
		if m.panelFocus {
			return m.updatePanelFocused(msg)
		}
		return m.updateNavFocused(msg)

	case composeSentMsg:
		m.composeStatus = styleGreen.Render("✓ sent")
		m.rebuildMain()
		return m, m.listenForMsgs()

	case composeErrMsg:
		m.composeStatus = styleRed.Render("✗ " + msg.err)
		m.rebuildMain()
		return m, m.listenForMsgs()

	case refreshMsg:
		m.refreshing = true
		return m, tea.Batch(m.listenForMsgs(), fetchAll(m.client, m.clientFactory, m.msgCh))

	case tickMsg:
		m.refreshing = true
		return m, tea.Batch(m.listenForMsgs(), fetchAll(m.client, m.clientFactory, m.msgCh), m.tickCmd())

	case pfStatusMsg:
		if msg.idx >= 0 && msg.idx < len(m.portForwards) {
			m.portForwards[msg.idx].Running = msg.running
			m.portForwards[msg.idx].PID = msg.pid
		}
		if m.nav == NavDashboard {
			m.rebuildMain()
		}
		return m, m.listenForMsgs()

	case loginStatusMsg:
		m.loginStatus = msg.status
		if m.nav == NavDashboard {
			m.rebuildMain()
		}
		return m, m.listenForMsgs()

	case dataMsg:
		m.data = msg.data
		m.lastFetch = msg.data.FetchedAt
		m.refreshing = false
		m.restartSessionPoll()
		m.rebuildMain()
		return m, m.listenForMsgs()

	case sessionMsgsMsg:
		if m.sessionMsgs == nil {
			m.sessionMsgs = make(map[string][]sdktypes.SessionMessage)
		}
		m.sessionMsgs[msg.sessionID] = append(m.sessionMsgs[msg.sessionID], msg.msg)
		const maxPerSession = 300
		if n := len(m.sessionMsgs[msg.sessionID]); n > maxPerSession {
			m.sessionMsgs[msg.sessionID] = m.sessionMsgs[msg.sessionID][n-maxPerSession:]
		}
		if m.nav == NavSessions {
			m.rebuildMain()
		}
		return m, m.listenForMsgs()

	case cmdOutputMsg:
		m.mainLines = append(m.mainLines, renderLine(msg.text, msg.kind))
		m.mainScroll = max(0, len(m.mainLines)-m.mainContentH())
		return m, m.listenForMsgs()

	case cmdDoneMsg:
		m.cmdRunning = false
		return m, m.listenForMsgs()
	}

	return m, nil
}

func (m *Model) updateNavFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		if m.nav > 0 {
			m.nav--
			m.mainScroll = 0
			m.rebuildMain()
		}
	case tea.KeyDown:
		if int(m.nav) < len(navLabels)-1 {
			m.nav++
			m.mainScroll = 0
			m.rebuildMain()
		}
	case tea.KeyRight, tea.KeyEnter:
		m.panelFocus = true
		m.panelRow = 0
		m.rebuildMain()
	case tea.KeyPgUp:
		m.mainScroll = max(0, m.mainScroll-m.mainContentH())
	case tea.KeyPgDown:
		m.mainScroll = min(max(0, len(m.mainLines)-m.mainContentH()), m.mainScroll+m.mainContentH())
	case tea.KeyTab:
		m.cmdFocus = true
	case tea.KeyRunes:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "r":
			return m, func() tea.Msg { return refreshMsg{} }
		case "j":
			if int(m.nav) < len(navLabels)-1 {
				m.nav++
				m.mainScroll = 0
				m.rebuildMain()
			}
		case "k":
			if m.nav > 0 {
				m.nav--
				m.mainScroll = 0
				m.rebuildMain()
			}
		}
	}
	return m, nil
}

func (m *Model) updateInputFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.cmdFocus = false
		m.input.clear()
	case tea.KeyTab:
		m.cmdFocus = false
	case tea.KeyEnter:
		cmd := strings.TrimSpace(m.input.value)
		if cmd != "" {
			m.history = append(m.history, cmd)
			m.histIdx = len(m.history)
			m.mainLines = append(m.mainLines, renderLine("$ "+cmd, lkGreen))
			m.mainScroll = max(0, len(m.mainLines)-m.mainContentH())
			m.input.clear()
			if !m.cmdRunning {
				m.cmdRunning = true
				return m, tea.Batch(m.listenForMsgs(), m.execCommand(cmd))
			}
		} else {
			m.input.clear()
		}
	case tea.KeyBackspace:
		m.input.backspace()
	case tea.KeyDelete:
		m.input.deleteForward()
	case tea.KeyLeft:
		m.input.moveLeft()
	case tea.KeyRight:
		m.input.moveRight()
	case tea.KeyHome, tea.KeyCtrlA:
		m.input.moveHome()
	case tea.KeyEnd, tea.KeyCtrlE:
		m.input.moveEnd()
	case tea.KeyCtrlK:
		m.input.value = string([]rune(m.input.value)[:m.input.cursor])
	case tea.KeyCtrlU:
		m.input.value = string([]rune(m.input.value)[m.input.cursor:])
		m.input.cursor = 0
	case tea.KeyUp:
		if len(m.history) > 0 && m.histIdx > 0 {
			m.histIdx--
			m.input.value = m.history[m.histIdx]
			m.input.moveEnd()
		}
	case tea.KeyDown:
		if m.histIdx < len(m.history)-1 {
			m.histIdx++
			m.input.value = m.history[m.histIdx]
			m.input.moveEnd()
		} else {
			m.histIdx = len(m.history)
			m.input.clear()
		}
	case tea.KeySpace:
		m.input.insert(' ')
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.input.insert(r)
		}
	}
	return m, nil
}

func (m *Model) panelRowCount() int {
	switch m.nav {
	case NavDashboard:
		return len(m.portForwards) + 1
	case NavCluster:
		return len(m.data.Pods)
	case NavNamespaces:
		return len(m.data.Namespaces)
	case NavProjects:
		return len(m.data.Projects)
	case NavSessions:
		return len(m.data.Sessions)
	case NavAgents:
		return len(m.data.Agents)
	}
	return 0
}

func (m *Model) updatePanelFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := m.panelRowCount()
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyLeft:
		m.panelFocus = false
		m.panelRow = 0
		m.rebuildMain()
		return m, nil
	case tea.KeyUp:
		if m.panelRow > 0 {
			m.panelRow--
			m.rebuildMain()
		}
	case tea.KeyDown:
		if n > 0 && m.panelRow < n-1 {
			m.panelRow++
			m.rebuildMain()
		}
	case tea.KeyPgUp:
		m.panelRow = max(0, m.panelRow-m.mainContentH())
		m.rebuildMain()
	case tea.KeyPgDown:
		if n > 0 {
			m.panelRow = min(n-1, m.panelRow+m.mainContentH())
		}
		m.rebuildMain()
	case tea.KeyEnter, tea.KeyRight:
		if m.nav == NavDashboard {
			return m, m.dashboardActivate(m.panelRow)
		}
		if m.nav == NavSessions && m.panelRow < len(m.data.Sessions) {
			sess := m.data.Sessions[m.panelRow]
			m.composeMode = true
			m.composeSessionID = sess.ID
			m.composeStatus = ""
			m.composeInput.clear()
			m.rebuildMain()
			return m, nil
		}
		if m.nav == NavAgents && m.panelRow < len(m.data.Agents) {
			agent := m.data.Agents[m.panelRow]
			m.agentEditMode = true
			m.agentEditAgent = agent
			m.agentEditPrompt = agent.Prompt
			m.agentEditDirty = false
			m.agentEditCursor = len([]rune(agent.Prompt))
			m.agentEditStatus = ""
			return m, nil
		}
		return m, m.drillIntoSelected()
	case tea.KeyRunes:
		switch msg.String() {
		case "d", "D":
			if m.nav == NavAgents && m.panelRow < len(m.data.Agents) {
				agent := m.data.Agents[m.panelRow]
				m.agentConfirmDelete = true
				m.agentDeleteID = agent.ID
				m.agentDeleteName = agent.Name
				return m, nil
			}
		case "j":
			if n > 0 && m.panelRow < n-1 {
				m.panelRow++
				m.rebuildMain()
			}
		case "k":
			if m.panelRow > 0 {
				m.panelRow--
				m.rebuildMain()
			}
		case "r":
			return m, func() tea.Msg { return refreshMsg{} }
		}
	}
	return m, nil
}

func (m *Model) updateAgentDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.agentConfirmDelete = false
		return m, nil
	case tea.KeyRunes:
		switch msg.String() {
		case "y", "Y":
			id := m.agentDeleteID
			client := m.client
			m.agentConfirmDelete = false
			return m, tea.Batch(m.listenForMsgs(), func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := client.Agents().Delete(ctx, id); err != nil {
					return agentDeleteErrMsg{err: err.Error()}
				}
				return agentDeletedMsg{id: id}
			})
		case "n", "N":
			m.agentConfirmDelete = false
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) updateAgentEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	runes := []rune(m.agentEditPrompt)
	cur := m.agentEditCursor

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.agentEditDirty {
			m.agentConfirmDelete = false
			m.agentEditEscOnce = true
			m.agentEditStatus = styleYellow.Render("unsaved changes — y abandon  ·  n keep editing")
		} else {
			m.agentEditMode = false
			m.agentEditEscOnce = false
			m.agentEditStatus = ""
		}
		return m, nil
	case tea.KeyEnter:
		if !m.agentEditDirty {
			m.agentEditMode = false
			m.rebuildMain()
			return m, nil
		}
		prompt := m.agentEditPrompt
		agent := m.agentEditAgent
		client := m.client
		m.agentEditStatus = styleDim.Render("saving…")
		m.rebuildMain()
		return m, tea.Batch(m.listenForMsgs(), func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			updated, err := client.Agents().Update(ctx, agent.ID, map[string]any{"prompt": prompt})
			if err != nil {
				return agentSaveErrMsg{err: err.Error()}
			}
			return agentSavedMsg{agent: *updated}
		})
	case tea.KeyBackspace:
		m.agentEditEscOnce = false
		if cur > 0 {
			runes = append(runes[:cur-1], runes[cur:]...)
			m.agentEditPrompt = string(runes)
			m.agentEditCursor--
			m.agentEditDirty = true
		}
	case tea.KeyDelete:
		if cur < len(runes) {
			runes = append(runes[:cur], runes[cur+1:]...)
			m.agentEditPrompt = string(runes)
			m.agentEditDirty = true
		}
	case tea.KeyLeft:
		if cur > 0 {
			m.agentEditCursor--
		}
	case tea.KeyRight:
		if cur < len(runes) {
			m.agentEditCursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		m.agentEditCursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		m.agentEditCursor = len(runes)
	case tea.KeyCtrlK:
		m.agentEditPrompt = string(runes[:cur])
		m.agentEditDirty = true
	case tea.KeyCtrlU:
		m.agentEditPrompt = string(runes[cur:])
		m.agentEditCursor = 0
		m.agentEditDirty = true
	case tea.KeySpace:
		runes = append(runes[:cur], append([]rune{' '}, runes[cur:]...)...)
		m.agentEditPrompt = string(runes)
		m.agentEditCursor++
		m.agentEditDirty = true
	case tea.KeyRunes:
		if m.agentEditEscOnce {
			switch msg.String() {
			case "y", "Y":
				m.agentEditPrompt = m.agentEditAgent.Prompt
				m.agentEditDirty = false
				m.agentEditMode = false
				m.agentEditEscOnce = false
				m.agentEditStatus = ""
			case "n", "N":
				m.agentEditEscOnce = false
				m.agentEditStatus = ""
			}
			m.rebuildMain()
			return m, nil
		}
		for _, r := range msg.Runes {
			runes = append(runes[:cur], append([]rune{r}, runes[cur:]...)...)
			cur++
		}
		m.agentEditPrompt = string(runes)
		m.agentEditCursor = cur
		m.agentEditDirty = true
	}
	m.rebuildMain()
	return m, nil
}

func (m *Model) updateComposeFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.composeMode = false
		m.composeInput.clear()
		m.composeStatus = ""
		m.rebuildMain()
		return m, nil
	case tea.KeyEnter:
		text := strings.TrimSpace(m.composeInput.value)
		if text == "" {
			return m, nil
		}
		sessID := m.composeSessionID
		client := m.client
		factory := m.clientFactory
		sessions := m.data.Sessions
		m.composeInput.clear()
		m.composeStatus = styleDim.Render("sending…")
		m.rebuildMain()
		return m, tea.Batch(m.listenForMsgs(), func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c := client
			if factory != nil {
				for _, sess := range sessions {
					if sess.ID == sessID && sess.ProjectID != "" {
						if pc, err := factory.ForProject(sess.ProjectID); err == nil {
							c = pc
						}
					}
				}
			}
			_, err := c.Sessions().PushMessage(ctx, sessID, text)
			if err != nil {
				return composeErrMsg{err: err.Error()}
			}
			return composeSentMsg{sessionID: sessID}
		})
	case tea.KeyBackspace:
		m.composeInput.backspace()
	case tea.KeyDelete:
		m.composeInput.deleteForward()
	case tea.KeyLeft:
		m.composeInput.moveLeft()
	case tea.KeyRight:
		m.composeInput.moveRight()
	case tea.KeyHome, tea.KeyCtrlA:
		m.composeInput.moveHome()
	case tea.KeyEnd, tea.KeyCtrlE:
		m.composeInput.moveEnd()
	case tea.KeyCtrlK:
		m.composeInput.value = string([]rune(m.composeInput.value)[:m.composeInput.cursor])
	case tea.KeyCtrlU:
		m.composeInput.value = string([]rune(m.composeInput.value)[m.composeInput.cursor:])
		m.composeInput.cursor = 0
	case tea.KeySpace:
		m.composeInput.insert(' ')
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.composeInput.insert(r)
		}
	}
	m.rebuildMain()
	return m, nil
}

func (m *Model) popDetailMode() {
	m.detailMode = false
	m.detailLines = nil
	m.detailTitle = ""
	m.detailScroll = 0
	m.detailRow = 0
	m.detailSelectable = false
	m.detailItems = nil
	m.detailHeaderLines = 0
	m.detailSplit = false
	m.detailTopLines = nil
	m.detailBottomLines = nil
	m.detailTopScroll = 0
	m.detailBottomScroll = 0
	m.detailSplitFocus = 0
	m.rebuildMain()
}

func (m *Model) updateDetailMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	contentH := m.mainContentH()
	n := len(m.detailItems)
	headerLines := m.detailHeaderLines

	if m.detailSplit {
		halfH := contentH/2 - 1
		if halfH < 1 {
			halfH = 1
		}
		active := &m.detailTopScroll
		activeLines := m.detailTopLines
		if m.detailSplitFocus == 1 {
			active = &m.detailBottomScroll
			activeLines = m.detailBottomLines
		}
		maxScroll := len(activeLines) - halfH
		if maxScroll < 0 {
			maxScroll = 0
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			m.popDetailMode()
			return m, nil
		case tea.KeyLeft:
			if m.detailSplitFocus == 1 {
				m.detailSplitFocus = 0
			} else {
				m.popDetailMode()
				return m, nil
			}
		case tea.KeyRight:
			m.detailSplitFocus = 1
		case tea.KeyUp:
			if *active > 0 {
				*active--
			}
		case tea.KeyDown:
			if *active < maxScroll {
				*active++
			}
		case tea.KeyPgUp:
			*active = max(0, *active-halfH)
		case tea.KeyPgDown:
			*active = min(maxScroll, *active+halfH)
		case tea.KeyTab:
			m.detailSplitFocus = 1 - m.detailSplitFocus
		case tea.KeyRunes:
			switch msg.String() {
			case "j":
				if *active < maxScroll {
					*active++
				}
			case "k":
				if *active > 0 {
					*active--
				}
			}
		}
		return m, nil
	}

	if m.detailSelectable {
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyLeft:
			m.popDetailMode()
			return m, nil
		case tea.KeyEnter, tea.KeyRight:
			if m.detailRow < n {
				item := m.detailItems[m.detailRow]
				sessionMsgs := m.sessionMsgs
				return m, func() tea.Msg {
					switch item.kind {
					case "session":
						var msgs []sdktypes.SessionMessage
						if sessionMsgs != nil {
							msgs = sessionMsgs[item.id]
						}
						sess := sdktypes.Session{}
						sess.Name = item.name
						sess.ProjectID = item.namespace
						sess.ID = item.id
						return fetchSessionSplitDetail(sess, msgs)
					default:
						lines := fetchPodLogs(item.namespace, item.name)
						return detailReadyMsg{title: "Pod Logs: " + item.namespace + "/" + item.name, lines: lines}
					}
				}
			}
		case tea.KeyUp:
			if m.detailRow > 0 {
				m.detailRow--
				m.applyDetailCursor(headerLines)
			}
		case tea.KeyDown:
			if n > 0 && m.detailRow < n-1 {
				m.detailRow++
				m.applyDetailCursor(headerLines)
			}
		case tea.KeyRunes:
			switch msg.String() {
			case "j":
				if n > 0 && m.detailRow < n-1 {
					m.detailRow++
					m.applyDetailCursor(headerLines)
				}
			case "k":
				if m.detailRow > 0 {
					m.detailRow--
					m.applyDetailCursor(headerLines)
				}
			}
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc, tea.KeyLeft:
		m.popDetailMode()
		return m, nil
	case tea.KeyUp:
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case tea.KeyDown:
		if m.detailScroll < len(m.detailLines)-contentH {
			m.detailScroll++
		}
	case tea.KeyPgUp:
		m.detailScroll = max(0, m.detailScroll-contentH)
	case tea.KeyPgDown:
		m.detailScroll = min(max(0, len(m.detailLines)-contentH), m.detailScroll+contentH)
	case tea.KeyRunes:
		switch msg.String() {
		case "j":
			if m.detailScroll < len(m.detailLines)-contentH {
				m.detailScroll++
			}
		case "k":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		}
	}
	return m, nil
}

func (m *Model) dashboardActivate(row int) tea.Cmd {
	pfCount := len(m.portForwards)
	if row < pfCount {
		idx := row
		entry := m.portForwards[idx]
		return func() tea.Msg {
			if entry.Running && entry.PID > 0 {
				_ = exec.Command("kill", strconv.Itoa(entry.PID)).Run()
				return pfStatusMsg{idx: idx, running: false, pid: 0}
			}
			namespace := "ambient-code"
			cmd := exec.Command("kubectl", "port-forward",
				fmt.Sprintf("svc/%s", entry.SvcName),
				fmt.Sprintf("%d:%d", entry.LocalPort, entry.SvcPort),
				"-n", namespace)
			if err := cmd.Start(); err != nil {
				return pfStatusMsg{idx: idx, running: false, pid: 0}
			}
			time.Sleep(500 * time.Millisecond)
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", entry.LocalPort), 2*time.Second)
			if err == nil {
				conn.Close()
				return pfStatusMsg{idx: idx, running: true, pid: cmd.Process.Pid}
			}
			return pfStatusMsg{idx: idx, running: false, pid: 0}
		}
	}
	return func() tea.Msg {
		out, err := exec.Command("kubectl", "config", "current-context").Output()
		if err != nil {
			return loginStatusMsg{status: LoginStatus{}}
		}
		ctx := strings.TrimSpace(string(out))
		whoami, _ := exec.Command("kubectl", "auth", "whoami", "-o", "jsonpath={.status.userInfo.username}").Output()
		user := strings.TrimSpace(string(whoami))
		if user == "" {
			user = "unknown"
		}
		ns, _ := exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath={..namespace}").Output()
		namespace := strings.TrimSpace(string(ns))
		if namespace == "" {
			namespace = "default"
		}
		return loginStatusMsg{status: LoginStatus{
			LoggedIn:  true,
			User:      user,
			Server:    ctx,
			Namespace: namespace,
		}}
	}
}

func (m *Model) checkPortForwards() tea.Cmd {
	entries := make([]PortForwardEntry, len(m.portForwards))
	copy(entries, m.portForwards)
	return func() tea.Msg {
		for i, entry := range entries {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", entry.LocalPort), 300*time.Millisecond)
			if err == nil {
				conn.Close()
				if !entry.Running {
					m.msgCh <- pfStatusMsg{idx: i, running: true, pid: entry.PID}
				}
			} else {
				if entry.Running {
					m.msgCh <- pfStatusMsg{idx: i, running: false, pid: 0}
				}
			}
		}
		return nil
	}
}

func (m *Model) drillIntoSelected() tea.Cmd {
	row := m.panelRow
	client := m.client
	factory := m.clientFactory
	data := m.data
	sessionMsgs := m.sessionMsgs
	return func() tea.Msg {
		switch m.nav {
		case NavCluster:
			if row >= len(data.Pods) {
				break
			}
			pod := data.Pods[row]
			lines := fetchPodLogs(pod.Namespace, pod.Name)
			return detailReadyMsg{title: "Pod Logs: " + pod.Namespace + "/" + pod.Name, lines: lines}

		case NavNamespaces:
			if row >= len(data.Namespaces) {
				break
			}
			ns := data.Namespaces[row]
			return fetchNamespacePodsDetail(ns.Name)

		case NavSessions:
			if row >= len(data.Sessions) {
				break
			}
			sess := data.Sessions[row]
			msgs := sessionMsgs[sess.ID]
			return fetchSessionSplitDetail(sess, msgs)

		case NavProjects:
			if row >= len(data.Projects) {
				break
			}
			proj := data.Projects[row]
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			projClient := client
			if factory != nil {
				if pc, err := factory.ForProject(proj.Name); err == nil {
					projClient = pc
				}
			}
			return fetchProjectSessionsDetail(ctx, projClient, proj)

		case NavAgents:
			if row >= len(data.Agents) {
				break
			}
			agent := data.Agents[row]
			lines := renderAgentDetail(agent)
			return detailReadyMsg{title: "Agent: " + agent.Name, lines: lines}
		}
		return nil
	}
}

func (m *Model) mainContentH() int {
	h := m.height - 4
	if h < 1 {
		return 1
	}
	return h
}

func renderLine(text string, kind lineKind) string {
	switch kind {
	case lkDim:
		return styleDim.Render(text)
	case lkGreen:
		return styleGreen.Render(text)
	case lkRed:
		return styleRed.Render(text)
	case lkYellow:
		return styleYellow.Render(text)
	case lkCyan:
		return styleCyan.Render(text)
	case lkOrange:
		return styleOrange.Render(text)
	case lkBold:
		return styleBold.Render(text)
	case lkHeader:
		return styleBold.Render(styleWhite.Render(text))
	default:
		return text
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Model) restartSessionPoll() {
	if m.sessionMsgs == nil {
		m.sessionMsgs = make(map[string][]sdktypes.SessionMessage)
	}
	if m.sessionWatching == nil {
		m.sessionWatching = make(map[string]context.CancelFunc)
	}

	active := make(map[string]bool, len(m.data.Sessions))
	for _, sess := range m.data.Sessions {
		active[sess.ID] = true
	}

	for id, cancel := range m.sessionWatching {
		if !active[id] {
			cancel()
			delete(m.sessionWatching, id)
		}
	}

	defaultClient := m.client
	factory := m.clientFactory
	msgCh := m.msgCh

	for _, sess := range m.data.Sessions {
		if _, already := m.sessionWatching[sess.ID]; already {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.sessionWatching[sess.ID] = cancel
		sessID := sess.ID
		projID := sess.ProjectID

		watchClient := defaultClient
		if factory != nil && projID != "" {
			if pc, err := factory.ForProject(projID); err == nil {
				watchClient = pc
			}
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				msgs, stop, err := watchClient.Sessions().WatchMessages(ctx, sessID, 0)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case <-time.After(3 * time.Second):
						continue
					}
				}
				done := false
				for !done {
					select {
					case <-ctx.Done():
						stop()
						return
					case msg, ok := <-msgs:
						if !ok {
							done = true
							break
						}
						msgCh <- sessionMsgsMsg{sessionID: sessID, msg: *msg}
					}
				}
				stop()
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
			}
		}()
	}
}
