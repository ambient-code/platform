package views

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// MsgStreamBackMsg signals that the user pressed Esc to leave the message stream.
type MsgStreamBackMsg struct{}

// MsgStreamSendMsg carries a composed message to be sent by the parent.
type MsgStreamSendMsg struct {
	SessionID string
	Body      string
}

// MsgStreamCopyMsg carries the result of a clipboard copy attempt. The parent
// handles this to display success or failure via the info line.
type MsgStreamCopyMsg struct {
	Text string // the text that was (or was attempted to be) copied
	Err  error  // non-nil if the clipboard write failed
}

// ---------------------------------------------------------------------------
// Color palette (duplicated from parent tui package to avoid circular import)
// ---------------------------------------------------------------------------

var (
	msgColorWhite  = lipgloss.Color("255")
	msgColorGreen  = lipgloss.Color("28")
	msgColorDim    = lipgloss.Color("240")
	msgColorYellow = lipgloss.Color("33")
	msgColorRed    = lipgloss.Color("196")
	msgColorOrange = lipgloss.Color("214")
	msgColorCyan   = lipgloss.Color("36")
	msgColorBlue   = lipgloss.Color("69")
)

// Hoisted styles for the message stream View to avoid allocations on every frame.
var (
	msgBorderStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	msgKindStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	msgScopeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	msgCountStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	msgDimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	msgDimIndicator    = lipgloss.NewStyle().Foreground(msgColorDim)
	msgActiveIndicator = lipgloss.NewStyle().Foreground(msgColorBlue)
	msgSepStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
)

// eventColor returns the lipgloss color for a semantic event type.
// This duplicates the 6-entry mapping from the parent tui.EventColor to avoid
// a circular import.
func eventColor(eventType string) lipgloss.Color {
	switch eventType {
	case "user":
		return msgColorWhite // 255
	case "assistant":
		return msgColorBlue // 69 — complementary accent
	case "tool_use":
		return msgColorDim // 240
	case "tool_result":
		return msgColorDim // 240
	case "system":
		return msgColorYellow // 33
	case "error", "RUN_ERROR":
		return msgColorRed // 31
	case "TEXT_MESSAGE_START", "TEXT_MESSAGE_CONTENT", "TEXT_MESSAGE_END":
		return msgColorBlue
	case "TOOL_CALL_START", "TOOL_CALL_ARGS", "TOOL_CALL_END", "TOOL_CALL_RESULT":
		return msgColorCyan
	case "RUN_STARTED", "RUN_FINISHED":
		return msgColorGreen
	case "reasoning",
		"REASONING_START", "REASONING_MESSAGE_START",
		"REASONING_MESSAGE_CONTENT", "REASONING_MESSAGE_END",
		"REASONING_END":
		return msgColorDim
	case "STEP_STARTED", "STEP_FINISHED":
		return msgColorYellow
	default:
		return msgColorDim
	}
}

// phaseColor returns the display color for a session phase.
func phaseColor(phase string) lipgloss.Color {
	switch strings.ToLower(phase) {
	case "pending":
		return msgColorYellow
	case "running", "active":
		return msgColorOrange
	case "succeeded", "completed":
		return msgColorDim
	case "failed":
		return msgColorRed
	case "cancelled":
		return msgColorDim
	default:
		return msgColorDim
	}
}

// ---------------------------------------------------------------------------
// Local event summary renderer
// ---------------------------------------------------------------------------

// eventSummary produces a one-line display string for a message entry.
// This is a simplified version of the parent tui.EventSummary — enough for
// conversation-mode rendering without requiring a circular import.
func eventSummary(eventType, payload string) string {
	switch eventType {
	case "user":
		return truncatePayload(payload, 120)
	case "assistant":
		return truncatePayload(payload, 120)
	case "reasoning":
		return truncatePayload(payload, 120)
	case "tool_use":
		name := extractJSONField(payload, "name")
		if name == "" {
			return truncatePayload(payload, 120)
		}
		input := extractJSONField(payload, "input")
		if input != "" {
			return name + " " + truncatePayload(input, 80)
		}
		return name
	case "tool_result":
		content := extractJSONField(payload, "content")
		isError := extractJSONField(payload, "is_error")
		indicator := "✓" // checkmark
		if isError == "true" {
			indicator = "✗" // cross
		}
		return fmt.Sprintf("%s %d bytes", indicator, len(content))
	case "system":
		return truncatePayload(payload, 120)
	case "error":
		msg := extractJSONField(payload, "message")
		if msg != "" {
			return "✗ " + truncatePayload(msg, 120)
		}
		if payload != "" {
			return "✗ " + truncatePayload(payload, 120)
		}
		return "✗ unknown error"
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT":
		return extractJSONField(payload, "delta")
	case "TOOL_CALL_START":
		name := extractJSONField(payload, "tool_call_name")
		if name == "" {
			name = extractJSONField(payload, "tool_name")
		}
		if name == "" {
			name = extractJSONField(payload, "toolCallName")
		}
		if name != "" {
			return "⚙ " + name
		}
		return ""
	case "TOOL_CALL_RESULT":
		return extractJSONField(payload, "content")
	case "RUN_FINISHED":
		return "[done]"
	case "RUN_ERROR":
		msg := extractJSONField(payload, "message")
		if msg != "" {
			return "✗ " + msg
		}
		return "✗ error"
	case "TEXT_MESSAGE_START":
		return "…"
	case "TOOL_CALL_ARGS":
		delta := extractJSONField(payload, "delta")
		if delta != "" {
			return truncatePayload(delta, 120)
		}
		return ""
	case "TEXT_MESSAGE_END", "TOOL_CALL_END":
		return ""
	case "RUN_STARTED":
		threadID := extractJSONField(payload, "threadId")
		if threadID != "" {
			return "run started (thread " + truncatePayload(threadID, 40) + ")"
		}
		return "run started"
	case "REASONING_START", "REASONING_END",
		"REASONING_MESSAGE_START", "REASONING_MESSAGE_END":
		return ""
	case "MESSAGES_SNAPSHOT":
		return "[snapshot]"
	case "STATE_SNAPSHOT", "STATE_DELTA":
		return ""
	case "STEP_STARTED":
		name := extractJSONField(payload, "stepName")
		if name != "" {
			return "step: " + name
		}
		return ""
	case "STEP_FINISHED":
		return ""
	case "ACTIVITY_SNAPSHOT", "ACTIVITY_DELTA":
		return ""
	case "CUSTOM":
		name := extractJSONField(payload, "name")
		if name != "" {
			return "custom: " + name
		}
		return ""
	case "RAW":
		return ""
	}
	if payload != "" && len(payload) <= 120 {
		return payload
	}
	return ""
}

// eventFullText produces the full untruncated display string for a message entry.
// Used when wrapMode is enabled to show complete message payloads.
func eventFullText(eventType, payload string) string {
	switch eventType {
	case "user":
		return strings.TrimSpace(payload)
	case "reasoning":
		return strings.TrimSpace(payload)
	case "assistant":
		return strings.TrimSpace(payload)
	case "tool_use":
		name := extractJSONField(payload, "name")
		if name == "" {
			return strings.TrimSpace(payload)
		}
		input := extractJSONField(payload, "input")
		if input != "" {
			return name + " " + strings.TrimSpace(input)
		}
		return name
	case "tool_result":
		content := extractJSONField(payload, "content")
		isError := extractJSONField(payload, "is_error")
		indicator := "✓"
		if isError == "true" {
			indicator = "✗"
		}
		if content != "" {
			return fmt.Sprintf("%s %s", indicator, strings.TrimSpace(content))
		}
		return fmt.Sprintf("%s %d bytes", indicator, len(content))
	case "system":
		return strings.TrimSpace(payload)
	case "error":
		msg := extractJSONField(payload, "message")
		if msg != "" {
			return "✗ " + strings.TrimSpace(msg)
		}
		if payload != "" {
			return "✗ " + strings.TrimSpace(payload)
		}
		return "✗ unknown error"
	case "TOOL_CALL_ARGS":
		delta := extractJSONField(payload, "delta")
		if delta != "" {
			return strings.TrimSpace(delta)
		}
		return ""
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT":
		delta := extractJSONField(payload, "delta")
		if delta != "" {
			return strings.TrimSpace(delta)
		}
		return ""
	case "TOOL_CALL_START":
		name := extractJSONField(payload, "tool_call_name")
		if name == "" {
			name = extractJSONField(payload, "tool_name")
		}
		if name == "" {
			name = extractJSONField(payload, "toolCallName")
		}
		if name != "" {
			return "⚙ " + name
		}
		return ""
	case "TOOL_CALL_RESULT":
		content := extractJSONField(payload, "content")
		if content != "" {
			return strings.TrimSpace(content)
		}
		return ""
	case "RUN_FINISHED":
		return "[done]"
	case "RUN_ERROR":
		msg := extractJSONField(payload, "message")
		if msg != "" {
			return "✗ " + strings.TrimSpace(msg)
		}
		return "✗ error"
	case "RUN_STARTED":
		threadID := extractJSONField(payload, "threadId")
		if threadID != "" {
			return "run started (thread " + strings.TrimSpace(threadID) + ")"
		}
		return "run started"
	}
	// Fallback: same as eventSummary for other streaming event types.
	return eventSummary(eventType, payload)
}

// truncatePayload trims whitespace and truncates to max runes (not bytes) to
// avoid splitting multi-byte UTF-8 characters.
func truncatePayload(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

// extractJSONField extracts a string field from a JSON payload.
// Returns empty string on parse failure or missing key.
func extractJSONField(payload, key string) string {
	if payload == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return ""
	}
	v, ok := obj[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// ---------------------------------------------------------------------------
// MessageEntry
// ---------------------------------------------------------------------------

// MessageEntry represents a single message in the stream buffer.
type MessageEntry struct {
	Seq       int
	EventType string
	Payload   string
	Timestamp time.Time
}

// ---------------------------------------------------------------------------
// MessageStream — Bubbletea sub-model
// ---------------------------------------------------------------------------

// defaultMaxMessages is the ring buffer capacity per the TUI spec.
const defaultMaxMessages = 2000

// MessageStream is a Bubbletea sub-model for the live session message stream.
// It renders messages in conversation or raw mode, supports scrolling,
// autoscroll, compose input, and search.
//
// Messages arrive via 1-second REST polling of /messages.
type MessageStream struct {
	sessionID string
	agentName string
	phase     string

	// Single message buffer.
	messages    []MessageEntry
	maxMessages int // ring buffer capacity

	// Highest seq seen — used for polling dedup.
	lastSeq int

	// Display
	scrollOffset  int
	autoScroll    bool // default true — view follows new messages
	rawMode       bool // false=conversation, true=raw JSON
	wrapMode      bool // false=truncated (120 chars), true=full text with word wrap
	timestampMode int  // 0=off, 1=relative, 2=absolute

	// Glamour markdown renderer (created lazily on first use, cached).
	glamourRenderer *glamour.TermRenderer
	glamourWidth    int // width used to create the cached renderer

	// Cached display lines — rebuilt when mode/messages change, not every frame.
	cachedLines     []string
	cachedDirty     bool // true when lines need rebuilding
	cachedMsgCount  int
	cachedRawMode   bool
	cachedWrapMode  bool
	cachedTSMode    int
	cachedSearchPat string

	// Per-message glamour render cache (key = Seq).
	glamourCache map[int]string

	// Compose
	composeMode  bool
	composeInput textinput.Model

	// Search
	searchMode    bool
	searchInput   textinput.Model
	searchPattern *regexp.Regexp

	// Dimensions
	width, height int
}

// NewMessageStream creates a MessageStream sub-model for the given session.
func NewMessageStream(sessionID, agentName, phase string) MessageStream {
	ci := textinput.New()
	ci.Prompt = "> send message: "
	ci.CharLimit = 4096
	ci.Width = 80

	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 256
	si.Width = 40

	return MessageStream{
		sessionID:    sessionID,
		agentName:    agentName,
		phase:        phase,
		messages:     make([]MessageEntry, 0, 256),
		maxMessages:  defaultMaxMessages,
		autoScroll:   true,
		composeInput: ci,
		searchInput:  si,
	}
}

// ---------------------------------------------------------------------------
// Public methods
// ---------------------------------------------------------------------------

// AddMessage appends a message to the ring buffer. When the buffer exceeds
// maxMessages, the oldest message is evicted. If autoScroll is enabled the
// scroll offset is advanced to keep the newest message visible.
func (ms *MessageStream) AddMessage(entry MessageEntry) {
	ms.messages = append(ms.messages, entry)
	if len(ms.messages) > ms.maxMessages {
		// Evict oldest — shift the slice. For a 2000-entry buffer this is
		// acceptable; a true ring buffer optimisation can come later.
		excess := len(ms.messages) - ms.maxMessages
		// Clean up glamour cache entries for evicted messages.
		if ms.glamourCache != nil {
			for _, evicted := range ms.messages[:excess] {
				delete(ms.glamourCache, evicted.Seq)
			}
		}
		ms.messages = ms.messages[excess:]
		// Don't adjust scrollOffset here — it's a display-line offset, not a
		// message-array index. renderContent's clamp handles any overshoot.
	}
	// Track highest seq for polling dedup.
	if entry.Seq > ms.lastSeq {
		ms.lastSeq = entry.Seq
	}
	ms.cachedDirty = true
	if ms.autoScroll {
		ms.scrollToBottom()
	}
}

// LastSeq returns the highest seq in the buffer. Used by the polling path
// for dedup.
func (ms *MessageStream) LastSeq() int {
	return ms.lastSeq
}

// SetSize updates the viewport dimensions and invalidates caches that depend
// on width (glamour renderer and per-message glamour cache).
func (ms *MessageStream) SetSize(w, h int) {
	if w != ms.width {
		// Width changed — glamour output is width-dependent.
		ms.glamourRenderer = nil
		ms.glamourCache = nil
		ms.cachedDirty = true
	}
	ms.width = w
	ms.height = h
	ms.composeInput.Width = max(w-lipgloss.Width(ms.composeInput.Prompt)-4, 20)
	ms.searchInput.Width = max(w/3, 20)
}

// SetPhase updates the session phase (shown in the header and used to decide
// whether to render the streaming cursor).
func (ms *MessageStream) SetPhase(phase string) {
	ms.phase = phase
}

// IsComposeMode returns true when the compose input is active.
func (ms MessageStream) IsComposeMode() bool {
	return ms.composeMode
}

func (ms MessageStream) ComposeValue() string {
	return ms.composeInput.Value()
}

// IsAutoScroll returns true when auto-scroll is enabled.
func (ms MessageStream) IsAutoScroll() bool { return ms.autoScroll }
func (ms MessageStream) IsRawMode() bool    { return ms.rawMode }
func (ms MessageStream) IsWrapMode() bool   { return ms.wrapMode }
func (ms MessageStream) TimestampMode() int { return ms.timestampMode }

// SetSearchPattern sets or clears the message filter pattern.
func (ms *MessageStream) SetSearchPattern(pat *regexp.Regexp) {
	ms.searchPattern = pat
}

// ClearCompose resets the compose input and exits compose mode.
func (ms *MessageStream) ClearCompose() {
	ms.composeInput.Reset()
	ms.composeMode = false
	ms.composeInput.Blur()
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update handles input messages. It returns an updated MessageStream and any
// commands to execute.
//
// Key bindings (normal mode):
//
//	Esc       -> MsgStreamBackMsg (signal parent to pop navigation)
//	r         -> toggle raw/conversation mode
//	s         -> toggle autoscroll
//	m / Enter -> enter compose mode
//	G         -> jump to bottom, re-enable autoscroll
//	g         -> jump to top
//	j / Down  -> scroll down (disables autoscroll)
//	k / Up    -> scroll up (disables autoscroll)
//	/         -> enter search mode
//	scroll    -> mouse wheel scroll (disables autoscroll)
//
// Key bindings (compose mode):
//
//	Esc       -> exit compose mode
//	Enter     -> send message (MsgStreamSendMsg)
//
// Key bindings (search mode):
//
//	Esc       -> exit search mode, clear search
//	Enter     -> apply search pattern
func (ms *MessageStream) Update(msg tea.Msg) (MessageStream, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if ms.composeMode {
			return ms.updateCompose(msg)
		}
		if ms.searchMode {
			return ms.updateSearch(msg)
		}
		return ms.updateNormal(msg)

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			ms.scrollUp(3)
			return *ms, nil
		case tea.MouseButtonWheelDown:
			ms.scrollDown(3)
			return *ms, nil
		}
	}

	return *ms, nil
}

func (ms *MessageStream) updateNormal(msg tea.KeyMsg) (MessageStream, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// If search filter is active, clear it first instead of backing out.
		if ms.searchPattern != nil {
			ms.searchPattern = nil
			return *ms, nil
		}
		return *ms, func() tea.Msg { return MsgStreamBackMsg{} }

	case tea.KeyEnter:
		ms.enterComposeMode()
		return *ms, nil

	case tea.KeyUp:
		ms.scrollUp(1)
		return *ms, nil

	case tea.KeyDown:
		ms.scrollDown(1)
		return *ms, nil

	case tea.KeyPgUp:
		ms.scrollUp(ms.contentHeight())
		return *ms, nil

	case tea.KeyPgDown:
		ms.scrollDown(ms.contentHeight())
		return *ms, nil

	case tea.KeyRunes:
		switch msg.String() {
		case "r":
			ms.rawMode = !ms.rawMode
			if ms.autoScroll {
				ms.scrollToBottom()
			}
			return *ms, nil
		case "p":
			ms.wrapMode = !ms.wrapMode
			if ms.autoScroll {
				ms.scrollToBottom()
			}
			return *ms, nil
		case "t":
			ms.timestampMode = (ms.timestampMode + 1) % 3
			return *ms, nil
		case "s":
			ms.autoScroll = !ms.autoScroll
			if ms.autoScroll {
				ms.scrollToBottom()
			}
			return *ms, nil
		case "m":
			ms.enterComposeMode()
			return *ms, nil
		case "G":
			ms.scrollToBottom()
			ms.autoScroll = true
			return *ms, nil
		case "g":
			ms.scrollOffset = 0
			ms.autoScroll = false
			return *ms, nil
		case "j":
			ms.scrollDown(1)
			return *ms, nil
		case "k":
			ms.scrollUp(1)
			return *ms, nil
		case "c":
			// Copy the first visible message's payload to clipboard.
			// scrollOffset is a display-line offset, so we iterate all messages
			// and count display lines to find the right one.
			if len(ms.messages) > 0 {
				lineCount := 0
				for _, entry := range ms.messages {
					var entryLines []string
					if ms.rawMode {
						entryLines = ms.renderRawEntry(entry, max(ms.width-4, 20))
					} else {
						entryLines = ms.renderConversationEntry(entry, max(ms.width-4, 20))
					}
					if len(entryLines) == 0 {
						continue
					}
					lineCount += len(entryLines)
					if lineCount > ms.scrollOffset {
						text := eventSummary(entry.EventType, entry.Payload)
						if text == "" {
							text = entry.Payload
						}
						// Return a command so the parent can handle
						// clipboard write and display success/failure.
						return *ms, func() tea.Msg {
							err := clipboard.WriteAll(text)
							return MsgStreamCopyMsg{Text: text, Err: err}
						}
					}
				}
			}
			return *ms, nil
		}
	}

	return *ms, nil
}

func (ms *MessageStream) updateCompose(msg tea.KeyMsg) (MessageStream, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		ms.ClearCompose()
		return *ms, nil
	case tea.KeyEnter:
		value := strings.TrimSpace(ms.composeInput.Value())
		if value == "" {
			// Empty message — just exit compose mode.
			ms.ClearCompose()
			return *ms, nil
		}
		sid := ms.sessionID
		ms.ClearCompose()
		return *ms, func() tea.Msg {
			return MsgStreamSendMsg{SessionID: sid, Body: value}
		}
	}

	// Delegate to textinput for character entry.
	var cmd tea.Cmd
	ms.composeInput, cmd = ms.composeInput.Update(msg)
	return *ms, cmd
}

func (ms *MessageStream) updateSearch(msg tea.KeyMsg) (MessageStream, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		ms.searchMode = false
		ms.searchPattern = nil
		ms.searchInput.Reset()
		ms.searchInput.Blur()
		return *ms, nil
	case tea.KeyEnter:
		pattern := ms.searchInput.Value()
		if pattern == "" {
			ms.searchPattern = nil
		} else {
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				// Invalid regex — treat as literal.
				re = regexp.MustCompile(regexp.QuoteMeta(pattern))
			}
			ms.searchPattern = re
		}
		ms.searchMode = false
		ms.searchInput.Blur()
		return *ms, nil
	}

	var cmd tea.Cmd
	ms.searchInput, cmd = ms.searchInput.Update(msg)
	return *ms, cmd
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the message stream. Layout from top to bottom:
//  1. Header line: Session {id} -- Phase: {phase} -- Agent: {agentName}
//  2. Message content area (scrollable)
//  3. Streaming cursor ("streaming..." when phase is running)
//  4. Compose input (when composeMode is active)
//  5. Status bar (autoscroll indicator, search pattern, key hints)
func (ms *MessageStream) View() string {
	if ms.width == 0 {
		return "Loading…"
	}

	borderStyle := msgBorderStyle
	kindStyle := msgKindStyle
	scopeStyle := msgScopeStyle
	countStyle := msgCountStyle
	dimStyle := msgDimStyle

	// -- k9s-style title bar: messages(agent/session)[count] --
	shortID := ms.sessionID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	scope := ms.agentName + "/" + shortID
	titleRendered := " " +
		kindStyle.Render("messages") +
		dimStyle.Render("(") + scopeStyle.Render(scope) + dimStyle.Render(")") +
		dimStyle.Render("[") + countStyle.Render(fmt.Sprintf("%d", len(ms.messages))) + dimStyle.Render("]") +
		" "
	titleWidth := lipgloss.Width(titleRendered)
	remaining := max(ms.width-titleWidth-2, 2)
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes
	titleBar := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	// -- Status indicators line (below title, inside border) --
	autoScrollLabel := "Off"
	if ms.autoScroll {
		autoScrollLabel = "On"
	}
	rawLabel := "Off"
	if ms.rawMode {
		rawLabel = "On"
	}
	prettyLabel := "Off"
	if ms.wrapMode {
		prettyLabel = "On"
	}
	phaseStyle := lipgloss.NewStyle().Foreground(phaseColor(ms.phase))
	dimIndicator := msgDimIndicator
	tsLabel := "Off"
	switch ms.timestampMode {
	case 1:
		tsLabel = "Relative"
	case 2:
		tsLabel = "Absolute"
	}
	// Scroll position indicator.
	allLines := ms.buildDisplayLines()
	scrollPct := ""
	if len(allLines) > 0 {
		total := len(allLines)
		contentH := ms.contentHeight()
		if total <= contentH {
			scrollPct = "All"
		} else if ms.scrollOffset <= 0 {
			scrollPct = "Top"
		} else if ms.scrollOffset >= total-contentH {
			scrollPct = "Bot"
		} else {
			pct := ms.scrollOffset * 100 / (total - contentH)
			scrollPct = fmt.Sprintf("%d%%", pct)
		}
	}

	activeIndicator := msgActiveIndicator
	renderToggle := func(label, value string, on bool) string {
		s := dimIndicator
		if on {
			s = activeIndicator
		}
		return dimIndicator.Render(label+":") + s.Render(value)
	}
	indicators := fmt.Sprintf("%s     %s     %s     %s     Phase:%s     %s",
		renderToggle("Autoscroll", autoScrollLabel, ms.autoScroll),
		renderToggle("Raw", rawLabel, ms.rawMode),
		renderToggle("Pretty", prettyLabel, ms.wrapMode),
		renderToggle("Time", tsLabel, ms.timestampMode > 0),
		phaseStyle.Render(ms.phase),
		dimIndicator.Render(scrollPct),
	)
	// Center the indicators line.
	indWidth := lipgloss.Width(indicators)
	indPad := max((ms.width-2-indWidth)/2, 0)
	indicatorLine := borderStyle.Render("│") +
		padToWidth(strings.Repeat(" ", indPad)+indicators, ms.width-2) +
		borderStyle.Render("│")
	headerSep := borderStyle.Render("├" + strings.Repeat("─", max(ms.width-2, 0)) + "┤")

	// -- Compose / streaming cursor area (rendered bottom-up) --
	var bottomLines []string

	bottomBorder := borderStyle.Render("└" + strings.Repeat("─", max(ms.width-2, 0)) + "┘")
	bottomLines = append(bottomLines, bottomBorder)

	// Compose input (if active).
	if ms.composeMode {
		composeSep := borderStyle.Render("├" + strings.Repeat("─", max(ms.width-2, 0)) + "┤")
		composeView := ms.composeInput.View()
		composeLine := borderStyle.Render("│") +
			" " + padToWidth(composeView, ms.width-3) +
			borderStyle.Render("│")
		// Prepend compose above the status bar.
		bottomLines = append([]string{composeSep, composeLine}, bottomLines...)
	}

	// -- Content area --
	// 3 = header bar + header line + header separator
	topLines := 3
	contentH := max(ms.height-topLines-len(bottomLines), 1)

	contentLines := ms.renderContent(contentH)

	// Pad/truncate content to fill the viewport.
	rendered := make([]string, contentH)
	for i := range contentH {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		rendered[i] = borderStyle.Render("│") +
			padToWidth(" "+line, ms.width-2) +
			borderStyle.Render("│")
	}

	// Assemble.
	var sb strings.Builder
	sb.WriteString(titleBar)
	sb.WriteByte('\n')
	sb.WriteString(indicatorLine)
	sb.WriteByte('\n')
	sb.WriteString(headerSep)
	sb.WriteByte('\n')
	sb.WriteString(strings.Join(rendered, "\n"))
	sb.WriteByte('\n')
	sb.WriteString(strings.Join(bottomLines, "\n"))

	return sb.String()
}

// renderContent produces the visible message lines for the content area.
func (ms *MessageStream) renderContent(height int) []string {
	if len(ms.messages) == 0 {
		return []string{msgDimStyle.Render("No messages yet.")}
	}

	// Build all display lines from messages. Search filtering is already
	// applied inside buildDisplayLines at the message level.
	allLines := ms.buildDisplayLines()

	// Apply scroll offset.
	total := len(allLines)
	if ms.scrollOffset > total-height {
		ms.scrollOffset = total - height
	}
	if ms.scrollOffset < 0 {
		ms.scrollOffset = 0
	}

	start := ms.scrollOffset
	end := min(start+height, total)
	if start >= total {
		return nil
	}

	return allLines[start:end]
}

// buildDisplayLines converts messages into styled display lines.
// Results are cached and only rebuilt when mode/messages change.
func (ms *MessageStream) buildDisplayLines() []string {
	searchStr := ""
	if ms.searchPattern != nil {
		searchStr = ms.searchPattern.String()
	}
	totalCount := len(ms.messages)
	// Check if cache is still valid (timestamps always invalidate since relative times change).
	if !ms.cachedDirty &&
		ms.cachedMsgCount == totalCount &&
		ms.cachedRawMode == ms.rawMode &&
		ms.cachedWrapMode == ms.wrapMode &&
		ms.cachedTSMode == ms.timestampMode &&
		ms.cachedSearchPat == searchStr &&
		ms.timestampMode == 0 {
		return ms.cachedLines
	}

	maxLineWidth := max(ms.width-4, 20) // 2 for borders, 2 for padding

	lines := make([]string, 0, totalCount)

	const tagPad = 14
	turnSeparator := strings.Repeat(" ", tagPad) + msgSepStyle.Render(strings.Repeat("─", max(maxLineWidth-tagPad, 10)))

	now := time.Now()

	prevWasUserOrAssistant := false
	for _, entry := range ms.messages {
		entryLines := ms.renderEntry(entry, maxLineWidth, now)
		if len(entryLines) == 0 {
			continue
		}

		// Add dim separator between user/assistant messages in conversation mode.
		isUserOrAssistant := entry.EventType == "user" || entry.EventType == "assistant"
		if !ms.rawMode && isUserOrAssistant && prevWasUserOrAssistant {
			lines = append(lines, turnSeparator)
		}
		prevWasUserOrAssistant = isUserOrAssistant

		lines = append(lines, entryLines...)
	}

	ms.cachedLines = lines
	ms.cachedDirty = false
	ms.cachedMsgCount = totalCount
	ms.cachedRawMode = ms.rawMode
	ms.cachedWrapMode = ms.wrapMode
	ms.cachedTSMode = ms.timestampMode
	ms.cachedSearchPat = searchStr
	return lines
}

// renderEntry renders a single message entry into display lines, applying the
// search filter and optional timestamp prefix. Shared by history and overlay rendering.
func (ms *MessageStream) renderEntry(entry MessageEntry, maxLineWidth int, now time.Time) []string {
	// Apply search filter if active.
	if ms.searchPattern != nil {
		text := eventSummary(entry.EventType, entry.Payload)
		if !ms.searchPattern.MatchString(text) && !ms.searchPattern.MatchString(entry.Payload) {
			return nil
		}
	}

	var entryLines []string
	if ms.rawMode {
		entryLines = ms.renderRawEntry(entry, maxLineWidth)
	} else {
		entryLines = ms.renderConversationEntry(entry, maxLineWidth)
	}
	if len(entryLines) == 0 {
		return nil
	}

	// Prepend timestamp to the first line if timestamps are enabled.
	if ms.timestampMode > 0 && !entry.Timestamp.IsZero() {
		tsStyle := msgDimStyle
		var ts string
		if ms.timestampMode == 1 {
			d := now.Sub(entry.Timestamp)
			if d < time.Minute {
				ts = fmt.Sprintf("%ds", int(d.Seconds()))
			} else if d < time.Hour {
				ts = fmt.Sprintf("%dm", int(d.Minutes()))
			} else if d < 24*time.Hour {
				ts = fmt.Sprintf("%dh", int(d.Hours()))
			} else {
				ts = fmt.Sprintf("%dd", int(d.Hours()/24))
			}
		} else {
			ts = entry.Timestamp.Format("15:04:05")
		}
		entryLines[0] = tsStyle.Render(fmt.Sprintf("%-8s", ts)) + entryLines[0]
	}
	return entryLines
}

// getGlamourRenderer returns a cached glamour renderer, creating one lazily on
// first use. If the terminal width has changed, the renderer is recreated.
func (ms *MessageStream) getGlamourRenderer(wrapWidth int) *glamour.TermRenderer {
	if ms.glamourRenderer != nil && ms.glamourWidth == wrapWidth {
		return ms.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		return nil
	}
	ms.glamourRenderer = r
	ms.glamourWidth = wrapWidth
	return r
}

// renderConversationEntry renders a single message in conversation mode.
// Format: [event_type]  summary text (wrapped)
func (ms *MessageStream) renderConversationEntry(entry MessageEntry, maxWidth int) []string {
	color := eventColor(entry.EventType)
	typeStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(color)

	// Sanitize payload to strip ANSI escapes and control characters from agent output.
	sanitizedPayload := SanitizePayload(entry.Payload)

	// Choose full text or truncated summary based on wrapMode.
	var displayText string
	if ms.wrapMode {
		displayText = eventFullText(entry.EventType, sanitizedPayload)
	} else {
		displayText = eventSummary(entry.EventType, sanitizedPayload)
	}
	if displayText == "" {
		// Suppressed event types (TOOL_CALL_ARGS, etc.) — don't render.
		return nil
	}

	// Pad all tags to a fixed width so text always starts at the same column.
	const tagPadWidth = 14 // widest is [tool_result] = 13 chars + 1 padding
	rawTag := "[" + entry.EventType + "]"
	padded := rawTag + strings.Repeat(" ", max(tagPadWidth-len(rawTag), 1))
	tag := typeStyle.Render(padded)
	tagWidth := tagPadWidth

	indent := strings.Repeat(" ", tagWidth)

	availWidth := max(maxWidth-tagWidth, 10)

	// In pretty mode, render through glamour for markdown support.
	// Uses per-message cache to avoid re-rendering on every frame.
	// Glamour renders displayText (extracted content), not the raw payload
	// which may be a JSON envelope for AG-UI events.
	if ms.wrapMode {
		if ms.glamourCache == nil {
			ms.glamourCache = make(map[int]string)
		}
		var rendered string
		if cached, ok := ms.glamourCache[entry.Seq]; ok {
			rendered = cached
		} else {
			glamourWidth := max(ms.width-20, 20)
			if r := ms.getGlamourRenderer(glamourWidth); r != nil {
				out, err := r.Render(strings.TrimSpace(displayText))
				if err == nil {
					rendered = strings.TrimSpace(out)
					ms.glamourCache[entry.Seq] = rendered
				}
			}
		}
		if rendered != "" {
			glamourLines := strings.Split(rendered, "\n")
			result := make([]string, 0, len(glamourLines))
			for i, line := range glamourLines {
				if i == 0 {
					result = append(result, tag+line)
				} else {
					result = append(result, indent+line)
				}
			}
			return result
		}
	}

	wrapped := wrapText(displayText, availWidth)
	if len(wrapped) == 0 {
		return []string{tag}
	}

	result := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		if i == 0 {
			result = append(result, tag+"  "+textStyle.Render(line))
		} else {
			result = append(result, indent+textStyle.Render(line))
		}
	}

	return result
}

// renderRawEntry renders a single message as a JSON line in raw mode.
func (ms *MessageStream) renderRawEntry(entry MessageEntry, maxWidth int) []string {
	dimStyle := msgDimStyle

	// Sanitize payload to strip ANSI escapes and control characters from agent output.
	sanitizedPayload := SanitizePayload(entry.Payload)

	raw := struct {
		Seq       int    `json:"seq"`
		EventType string `json:"event_type"`
		Payload   string `json:"payload"`
		Timestamp string `json:"timestamp"`
	}{
		Seq:       entry.Seq,
		EventType: entry.EventType,
		Payload:   sanitizedPayload,
		Timestamp: entry.Timestamp.Format(time.RFC3339),
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return []string{dimStyle.Render("[marshal error]")}
	}

	line := string(b)
	wrapped := wrapText(line, maxWidth)
	result := make([]string, len(wrapped))
	for i, w := range wrapped {
		result[i] = dimStyle.Render(w)
	}
	return result
}

// renderStatusBar builds the bottom status line with mode indicators and key hints.
// ---------------------------------------------------------------------------
// Scroll helpers
// ---------------------------------------------------------------------------

func (ms *MessageStream) scrollUp(n int) {
	ms.autoScroll = false
	ms.scrollOffset -= n
	if ms.scrollOffset < 0 {
		ms.scrollOffset = 0
	}
}

func (ms *MessageStream) scrollDown(n int) {
	ms.autoScroll = false
	ms.scrollOffset += n
	// Clamping happens in renderContent.
}

func (ms *MessageStream) scrollToBottom() {
	// Set a large value; renderContent will clamp.
	ms.scrollOffset = len(ms.messages) * 10
}

// contentHeight returns the usable content height given the current dimensions.
// This must match the calculation in View() to avoid scroll/display mismatches.
func (ms *MessageStream) contentHeight() int {
	// Top: title bar + indicator line + header separator = 3 lines.
	topLines := 3
	// Bottom: bottom border = 1 line.
	bottomLines := 1
	if ms.composeMode {
		bottomLines += 2 // compose separator + compose line
	}
	h := ms.height - topLines - bottomLines
	if h < 1 {
		h = 1
	}
	return h
}

func (ms *MessageStream) enterComposeMode() {
	ms.composeMode = true
	ms.composeInput.Focus()
	ms.scrollToBottom()
	ms.autoScroll = true
}

// ---------------------------------------------------------------------------
// Text helpers
// ---------------------------------------------------------------------------

// wrapText breaks a string into lines of at most maxWidth visual characters.
// It splits on word boundaries where possible, falling back to hard breaks
// for very long tokens. Uses rune-aware operations and lipgloss.Width for
// visual width measurement to avoid splitting multi-byte UTF-8 characters.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 80
	}
	if s == "" {
		return nil
	}

	// Replace embedded newlines with spaces for single-line rendering,
	// then split into words.
	s = strings.ReplaceAll(s, "\n", " ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]

	for _, word := range words[1:] {
		if lipgloss.Width(current)+1+lipgloss.Width(word) <= maxWidth {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)

	// Hard-break any lines that still exceed maxWidth (long single tokens).
	var result []string
	for _, line := range lines {
		for lipgloss.Width(line) > maxWidth {
			// Slice by rune to avoid splitting multi-byte characters.
			runes := []rune(line)
			take := len(runes)
			// Binary-ish search: start from end and find the cut point.
			for take > 0 && lipgloss.Width(string(runes[:take])) > maxWidth {
				take--
			}
			if take == 0 {
				take = 1 // always make progress
			}
			result = append(result, string(runes[:take]))
			line = string(runes[take:])
		}
		result = append(result, line)
	}

	return result
}

// padToWidth pads a styled string to exactly w visual characters.
func padToWidth(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}
