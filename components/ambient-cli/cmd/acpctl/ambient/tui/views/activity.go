package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const activityMaxMessages = 2000

var (
	activityTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
)

type ActivityPane struct {
	messages    []MessageEntry
	maxMessages int

	scrollOffset int
	autoScroll   bool

	cachedLines    []string
	cachedDirty    bool
	cachedMsgCount int

	focused bool

	textBuf      strings.Builder
	reasoningBuf strings.Builder
	toolArgsBuf  strings.Builder
	accumType    string
	accumSeq     int

	width, height int
}

func NewActivityPane() ActivityPane {
	return ActivityPane{
		messages:    make([]MessageEntry, 0, 256),
		maxMessages: activityMaxMessages,
		autoScroll:  true,
	}
}

func IsActivityEvent(eventType string) bool {
	switch eventType {
	case "REASONING_START", "REASONING_MESSAGE_START",
		"REASONING_MESSAGE_CONTENT", "REASONING_MESSAGE_END",
		"REASONING_END",
		"reasoning":
		return true
	case "TOOL_CALL_START", "TOOL_CALL_ARGS", "TOOL_CALL_END", "TOOL_CALL_RESULT":
		return true
	case "tool_use", "tool_result":
		return true
	case "TEXT_MESSAGE_START", "TEXT_MESSAGE_CONTENT", "TEXT_MESSAGE_END":
		return true
	case "STEP_STARTED", "STEP_FINISHED":
		return true
	case "RUN_STARTED", "RUN_FINISHED", "RUN_ERROR":
		return true
	case "STATE_SNAPSHOT", "STATE_DELTA",
		"ACTIVITY_SNAPSHOT", "ACTIVITY_DELTA":
		return true
	default:
		return false
	}
}

func (ap *ActivityPane) AddMessage(entry MessageEntry) {
	switch entry.EventType {
	case "TEXT_MESSAGE_CONTENT":
		delta := extractJSONField(entry.Payload, "delta")
		if delta != "" {
			ap.textBuf.WriteString(delta)
			ap.accumType = "TEXT_MESSAGE_CONTENT"
			ap.accumSeq = entry.Seq
		}
		return

	case "TEXT_MESSAGE_END":
		ap.flushTextBuf(entry.Seq)
		return

	case "TEXT_MESSAGE_START":
		ap.flushTextBuf(entry.Seq)
		return

	case "REASONING_MESSAGE_CONTENT":
		delta := extractJSONField(entry.Payload, "delta")
		if delta != "" {
			ap.reasoningBuf.WriteString(delta)
			ap.accumType = "REASONING_MESSAGE_CONTENT"
			ap.accumSeq = entry.Seq
		}
		return

	case "REASONING_END":
		ap.flushReasoningBuf(entry.Seq)
		return

	case "REASONING_START", "REASONING_MESSAGE_START", "REASONING_MESSAGE_END":
		return

	case "TOOL_CALL_ARGS":
		delta := extractJSONField(entry.Payload, "delta")
		if delta != "" {
			ap.toolArgsBuf.WriteString(delta)
			ap.accumType = "TOOL_CALL_ARGS"
			ap.accumSeq = entry.Seq
		}
		return

	case "TOOL_CALL_END":
		ap.flushToolArgsBuf(entry.Seq)
		return
	}

	ap.flushAll(entry.Seq)
	ap.addRaw(entry)
}

func (ap *ActivityPane) flushTextBuf(seq int) {
	if ap.textBuf.Len() == 0 {
		return
	}
	text := strings.TrimSpace(ap.textBuf.String())
	ap.textBuf.Reset()
	if text == "" {
		return
	}
	ap.addRaw(MessageEntry{
		Seq:       seq,
		EventType: "text",
		Payload:   text,
		Timestamp: time.Now(),
	})
}

func (ap *ActivityPane) flushReasoningBuf(seq int) {
	if ap.reasoningBuf.Len() == 0 {
		return
	}
	text := strings.TrimSpace(ap.reasoningBuf.String())
	ap.reasoningBuf.Reset()
	if text == "" {
		return
	}
	ap.addRaw(MessageEntry{
		Seq:       seq,
		EventType: "reasoning",
		Payload:   text,
		Timestamp: time.Now(),
	})
}

func (ap *ActivityPane) flushToolArgsBuf(seq int) {
	if ap.toolArgsBuf.Len() == 0 {
		return
	}
	text := strings.TrimSpace(ap.toolArgsBuf.String())
	ap.toolArgsBuf.Reset()
	if text == "" {
		return
	}
	ap.addRaw(MessageEntry{
		Seq:       seq,
		EventType: "tool_args",
		Payload:   text,
		Timestamp: time.Now(),
	})
}

func (ap *ActivityPane) flushAll(seq int) {
	ap.flushTextBuf(seq)
	ap.flushReasoningBuf(seq)
	ap.flushToolArgsBuf(seq)
}

func (ap *ActivityPane) addRaw(entry MessageEntry) {
	ap.messages = append(ap.messages, entry)
	if len(ap.messages) > ap.maxMessages {
		excess := len(ap.messages) - ap.maxMessages
		ap.messages = ap.messages[excess:]
	}
	ap.cachedDirty = true
	if ap.autoScroll {
		ap.scrollToBottom()
	}
}

func (ap *ActivityPane) SetSize(w, h int) {
	if w != ap.width {
		ap.cachedDirty = true
	}
	ap.width = w
	ap.height = h
}

func (ap *ActivityPane) SetFocused(f bool) {
	ap.focused = f
}

func (ap *ActivityPane) IsFocused() bool {
	return ap.focused
}

func (ap *ActivityPane) ScrollUp(n int) {
	ap.autoScroll = false
	ap.scrollOffset -= n
	if ap.scrollOffset < 0 {
		ap.scrollOffset = 0
	}
}

func (ap *ActivityPane) ScrollDown(n int) {
	ap.autoScroll = false
	ap.scrollOffset += n
}

func (ap *ActivityPane) ScrollToBottom() {
	ap.scrollToBottom()
	ap.autoScroll = true
}

func (ap *ActivityPane) scrollToBottom() {
	ap.scrollOffset = len(ap.messages) * 10
}

func (ap *ActivityPane) ContentHeight() int {
	h := ap.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (ap *ActivityPane) View() string {
	if ap.width == 0 {
		return ""
	}

	borderColor := lipgloss.Color("240")
	if ap.focused {
		borderColor = lipgloss.Color("36")
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	titleRendered := " " +
		activityTitleStyle.Render("activity") +
		msgDimStyle.Render("[") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Render(fmt.Sprintf("%d", len(ap.messages))) +
		msgDimStyle.Render("]") +
		" "
	titleWidth := lipgloss.Width(titleRendered)
	remaining := max(ap.width-titleWidth-2, 2)
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes
	titleBar := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleRendered +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	contentH := ap.ContentHeight()
	contentLines := ap.renderContent(contentH)

	rendered := make([]string, contentH)
	for i := range contentH {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		rendered[i] = borderStyle.Render("│") +
			padToWidth(" "+line, ap.width-2) +
			borderStyle.Render("│")
	}

	bottomBorder := borderStyle.Render("└" + strings.Repeat("─", max(ap.width-2, 0)) + "┘")

	var sb strings.Builder
	sb.WriteString(titleBar)
	sb.WriteByte('\n')
	sb.WriteString(strings.Join(rendered, "\n"))
	sb.WriteByte('\n')
	sb.WriteString(bottomBorder)

	return sb.String()
}

func (ap *ActivityPane) renderContent(height int) []string {
	if len(ap.messages) == 0 {
		return []string{msgDimStyle.Render("Waiting for agent activity…")}
	}

	allLines := ap.buildDisplayLines()

	total := len(allLines)
	if ap.scrollOffset > total-height {
		ap.scrollOffset = total - height
	}
	if ap.scrollOffset < 0 {
		ap.scrollOffset = 0
	}

	start := ap.scrollOffset
	end := min(start+height, total)
	if start >= total {
		return nil
	}

	return allLines[start:end]
}

func (ap *ActivityPane) buildDisplayLines() []string {
	totalCount := len(ap.messages)
	if !ap.cachedDirty && ap.cachedMsgCount == totalCount {
		return ap.cachedLines
	}

	maxLineWidth := max(ap.width-4, 20)
	lines := make([]string, 0, totalCount)

	for _, entry := range ap.messages {
		entryLines := ap.renderActivityEntry(entry, maxLineWidth)
		lines = append(lines, entryLines...)
	}

	ap.cachedLines = lines
	ap.cachedDirty = false
	ap.cachedMsgCount = totalCount
	return lines
}

func (ap *ActivityPane) renderActivityEntry(entry MessageEntry, maxWidth int) []string {
	color := eventColor(entry.EventType)
	typeStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(color)

	var displayText string
	switch entry.EventType {
	case "text", "reasoning", "tool_args":
		displayText = entry.Payload
	default:
		sanitizedPayload := SanitizePayload(entry.Payload)
		displayText = eventSummary(entry.EventType, sanitizedPayload)
	}
	if displayText == "" {
		return nil
	}

	const tagPadWidth = 14
	rawTag := "[" + entry.EventType + "]"
	padded := rawTag + strings.Repeat(" ", max(tagPadWidth-len(rawTag), 1))
	tag := typeStyle.Render(padded)
	tagWidth := tagPadWidth

	availWidth := max(maxWidth-tagWidth, 10)

	wrapped := wrapText(displayText, availWidth)
	if len(wrapped) == 0 {
		return []string{tag}
	}

	indent := strings.Repeat(" ", tagWidth)
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

func (ap *ActivityPane) MessageCount() int {
	return len(ap.messages)
}
