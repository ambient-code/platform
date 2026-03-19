package tui

import (
	"context"
	"strings"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
)

type NavSection int

const (
	NavCluster    NavSection = iota // system pods in ambient-code namespace
	NavNamespaces                   // fleet-* namespaces
	NavProjects                     // SDK projects list
	NavSessions                     // SDK sessions list
	NavAgents                       // SDK agents list
	NavStats                        // summary counts
)

var navLabels = []string{
	"Cluster Pods",
	"Namespaces",
	"Projects",
	"Sessions",
	"Agents",
	"Stats",
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
		return styleGreen.Render("$ ") + string(runes) + styleBold.Render("█")
	}
	before := string(runes[:cur])
	cursorChar := string(runes[cur : cur+1])
	after := string(runes[cur+1:])
	return styleGreen.Render("$ ") + before + styleBold.Render(cursorChar) + after
}

type Model struct {
	client          *sdkclient.Client
	width           int
	height          int
	nav             NavSection
	data            DashData
	mainLines       []string
	mainScroll      int
	input           cmdInputModel
	history         []string
	histIdx         int
	cmdRunning      bool
	refreshing      bool
	lastFetch       time.Time
	msgCh           chan tea.Msg
	cmdFocus        bool
	sessionMsgs     map[string][]sdktypes.SessionMessage
	sessionWatching map[string]context.CancelFunc
}

func NewModel(client *sdkclient.Client) *Model {
	return &Model{
		client:  client,
		msgCh:   make(chan tea.Msg, 256),
		histIdx: -1,
		nav:     NavCluster,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		m.listenForMsgs(),
		func() tea.Msg { return refreshMsg{} },
		m.tickCmd(),
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildMain()

	case tea.KeyMsg:
		if m.cmdFocus {
			return m.updateInputFocused(msg)
		}
		return m.updateNavFocused(msg)

	case refreshMsg:
		m.refreshing = true
		return m, tea.Batch(m.listenForMsgs(), fetchAll(m.client, m.msgCh))

	case tickMsg:
		m.refreshing = true
		return m, tea.Batch(m.listenForMsgs(), fetchAll(m.client, m.msgCh), m.tickCmd())

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
	case tea.KeyPgUp:
		m.mainScroll = max(0, m.mainScroll-m.mainContentH())
	case tea.KeyPgDown:
		m.mainScroll = min(max(0, len(m.mainLines)-m.mainContentH()), m.mainScroll+m.mainContentH())
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
	case tea.KeyTab:
		m.cmdFocus = true
	case tea.KeyEnter:
		m.cmdFocus = true
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

	client := m.client
	msgCh := m.msgCh

	for _, sess := range m.data.Sessions {
		if _, already := m.sessionWatching[sess.ID]; already {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.sessionWatching[sess.ID] = cancel
		sessID := sess.ID
		go func() {
			msgs, stop, err := client.Sessions().WatchMessages(ctx, sessID, 0)
			if err != nil {
				return
			}
			defer stop()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgs:
					if !ok {
						return
					}
					msgCh <- sessionMsgsMsg{sessionID: sessID, msg: *msg}
				}
			}
		}()
	}
}
