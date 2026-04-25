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

// ---------------------------------------------------------------------------
// Color palette (duplicated from parent tui package to avoid circular import)
// ---------------------------------------------------------------------------

var (
	msgColorWhite  = lipgloss.Color("255")
	msgColorGreen  = lipgloss.Color("28")
	msgColorDim    = lipgloss.Color("240")
	msgColorYellow = lipgloss.Color("33")
	msgColorRed    = lipgloss.Color("31")
	msgColorOrange = lipgloss.Color("214")
	msgColorCyan   = lipgloss.Color("36")
	msgColorBlue   = lipgloss.Color("69")
)

// eventColor returns the lipgloss color for a semantic event type.
// This duplicates the 6-entry mapping from the parent tui.EventColor to avoid
// a circular import.
func eventColor(eventType string) lipgloss.Color {
	switch eventType {
	case "user":
		return msgColorWhite
	case "assistant":
		return msgColorWhite
	case "tool_use":
		return msgColorDim
	case "tool_result":
		return msgColorDim
	case "system":
		return msgColorYellow
	case "error":
		return msgColorRed
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
	case "TEXT_MESSAGE_END", "TOOL_CALL_ARGS", "TOOL_CALL_END":
		return ""
	}
	if payload != "" && len(payload) <= 120 {
		return payload
	}
	return ""
}

// truncatePayload trims whitespace and truncates to max length.
func truncatePayload(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
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
type MessageStream struct {
	sessionID string
	agentName string
	phase     string

	// SSE connection status: "", "connected", "reconnecting", "disconnected".
	sseStatus string

	// Message buffer (ring buffer, 2000 max).
	messages    []MessageEntry
	maxMessages int

	// Display
	scrollOffset int
	autoScroll   bool // default true — view follows new messages
	rawMode      bool // false=conversation, true=raw JSON

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
		ms.messages = ms.messages[excess:]
		ms.scrollOffset -= excess
		if ms.scrollOffset < 0 {
			ms.scrollOffset = 0
		}
	}
	if ms.autoScroll {
		ms.scrollToBottom()
	}
}

// SetSize updates the viewport dimensions.
func (ms *MessageStream) SetSize(w, h int) {
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

// SetSSEStatus updates the SSE connection status indicator shown in the header.
// Valid values: "", "connected", "reconnecting", "disconnected".
func (ms *MessageStream) SetSSEStatus(status string) {
	ms.sseStatus = status
}

// ComposeValue returns the current text in the compose input.
func (ms MessageStream) ComposeValue() string {
	return ms.composeInput.Value()
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
		case "/":
			ms.searchMode = true
			ms.searchInput.Reset()
			ms.searchInput.Focus()
			return *ms, nil
		case "c":
			// Copy the selected message text to clipboard.
			if len(ms.messages) > 0 {
				idx := ms.scrollOffset
				if idx >= len(ms.messages) {
					idx = len(ms.messages) - 1
				}
				if idx >= 0 {
					text := eventSummary(ms.messages[idx].EventType, ms.messages[idx].Payload)
					if text == "" {
						text = ms.messages[idx].Payload
					}
					_ = clipboard.WriteAll(text)
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

	borderStyle := lipgloss.NewStyle().Foreground(msgColorDim)
	kindStyle := lipgloss.NewStyle().Foreground(msgColorOrange).Bold(true)
	scopeStyle := lipgloss.NewStyle().Foreground(msgColorDim).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(msgColorDim).Bold(true)

	// -- k9s-style title bar: messages(agent/session)[count] --
	shortID := ms.sessionID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}
	scope := ms.agentName + "/" + shortID
	titleRendered := " " +
		kindStyle.Render("messages") +
		scopeStyle.Render("("+scope+")") +
		countStyle.Render(fmt.Sprintf("[%d]", len(ms.messages))) +
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
	modeLabel := "Conversation"
	if ms.rawMode {
		modeLabel = "Raw"
	}
	phaseStyle := lipgloss.NewStyle().Foreground(phaseColor(ms.phase))
	dimIndicator := lipgloss.NewStyle().Foreground(msgColorDim)
	indicators := fmt.Sprintf("Autoscroll:%s     Mode:%s     Phase:%s",
		dimIndicator.Render(autoScrollLabel),
		dimIndicator.Render(modeLabel),
		phaseStyle.Render(ms.phase),
	)
	if ms.sseStatus != "" && ms.sseStatus != "connected" {
		var sseColor lipgloss.Color
		switch ms.sseStatus {
		case "reconnecting":
			sseColor = msgColorYellow
		default:
			sseColor = msgColorRed
		}
		indicators += fmt.Sprintf("     SSE:%s",
			lipgloss.NewStyle().Foreground(sseColor).Render(ms.sseStatus))
	}
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

	// Streaming cursor (when phase is running).
	if strings.ToLower(ms.phase) == "running" {
		cursorStyle := lipgloss.NewStyle().Foreground(msgColorOrange)
		cursor := cursorStyle.Render(" ▌ streaming…")
		cursorLine := borderStyle.Render("│") +
			padToWidth(cursor, ms.width-2) +
			borderStyle.Render("│")
		// Prepend cursor above compose/status.
		bottomLines = append([]string{cursorLine}, bottomLines...)
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
		dimStyle := lipgloss.NewStyle().Foreground(msgColorDim)
		return []string{dimStyle.Render("No messages yet.")}
	}

	// Build all display lines from messages.
	allLines := ms.buildDisplayLines()

	// Apply search filter — highlight matches.
	if ms.searchPattern != nil {
		filtered := make([]string, 0, len(allLines))
		for _, line := range allLines {
			if ms.searchPattern.MatchString(stripANSI(line)) {
				filtered = append(filtered, line)
			}
		}
		allLines = filtered
	}

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

// buildDisplayLines converts the message buffer into styled display lines.
func (ms *MessageStream) buildDisplayLines() []string {
	maxLineWidth := max(ms.width-4, 20) // 2 for borders, 2 for padding

	lines := make([]string, 0, len(ms.messages))

	for _, entry := range ms.messages {
		if ms.rawMode {
			lines = append(lines, ms.renderRawEntry(entry, maxLineWidth)...)
		} else {
			lines = append(lines, ms.renderConversationEntry(entry, maxLineWidth)...)
		}
	}

	return lines
}

// renderConversationEntry renders a single message in conversation mode.
// Format: [event_type]  summary text (wrapped)
func (ms *MessageStream) renderConversationEntry(entry MessageEntry, maxWidth int) []string {
	color := eventColor(entry.EventType)
	typeStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(color)

	summary := eventSummary(entry.EventType, entry.Payload)
	if summary == "" {
		// Suppressed event types (TOOL_CALL_ARGS, etc.) — don't render.
		return nil
	}

	tag := typeStyle.Render("[" + entry.EventType + "]")
	tagWidth := lipgloss.Width(tag)

	// Indent continuation lines to align with the text after the tag.
	indent := strings.Repeat(" ", tagWidth+2)

	// Wrap the summary text.
	availWidth := max(maxWidth-tagWidth-2, 10) // 2 for spacing between tag and text

	wrapped := wrapText(summary, availWidth)
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
	dimStyle := lipgloss.NewStyle().Foreground(msgColorDim)

	raw := struct {
		Seq       int    `json:"seq"`
		EventType string `json:"event_type"`
		Payload   string `json:"payload"`
		Timestamp string `json:"timestamp"`
	}{
		Seq:       entry.Seq,
		EventType: entry.EventType,
		Payload:   entry.Payload,
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
func (ms *MessageStream) contentHeight() int {
	// Approximate: total height minus header (3 lines) minus status/compose/cursor.
	h := ms.height - 5
	if ms.composeMode {
		h -= 2
	}
	if strings.ToLower(ms.phase) == "running" {
		h--
	}
	if ms.searchMode {
		h -= 2
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (ms *MessageStream) enterComposeMode() {
	ms.composeMode = true
	ms.composeInput.Focus()
}

// ---------------------------------------------------------------------------
// Text helpers
// ---------------------------------------------------------------------------

// wrapText breaks a string into lines of at most maxWidth characters.
// It splits on word boundaries where possible, falling back to hard breaks
// for very long tokens.
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
		if len(current)+1+len(word) <= maxWidth {
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
		for len(line) > maxWidth {
			result = append(result, line[:maxWidth])
			line = line[maxWidth:]
		}
		result = append(result, line)
	}

	return result
}

// ansiRe matches ANSI CSI escape sequences for stripping before search.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// stripANSI removes ANSI escape sequences from a string so that search
// matching operates on visible text only.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// padToWidth pads a styled string to exactly w visual characters.
func padToWidth(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}
