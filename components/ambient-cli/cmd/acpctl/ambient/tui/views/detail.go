package views

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// Local color constants for the detail view. Defined here instead of importing
// from the parent tui package to avoid circular imports.
var (
	detailBorderColor = lipgloss.Color("240") // dim for borders
	detailKeyColor    = lipgloss.Color("240") // dim for field names
	detailValueColor  = lipgloss.Color("255") // white for values
	detailTitleColor  = lipgloss.Color("36")  // cyan for title
	detailHintColor   = lipgloss.Color("240") // dim for hints
)

// DetailBackMsg is sent when the user presses Esc or q to navigate back from
// the detail view to the parent list view.
type DetailBackMsg struct{}

// DetailLine represents a single key-value line in the detail view.
type DetailLine struct {
	Key   string         // field name (e.g. "Name", "Prompt", "Phase")
	Value string         // field value
	Color lipgloss.Color // optional color override for the value; empty string uses default
}

// DetailView is a Bubbletea sub-model that renders a scrollable key-value
// detail pane for a single resource. It handles Esc (back), j/k/arrow/scroll
// for scrolling, and c to copy the selected value.
type DetailView struct {
	title  string
	lines  []DetailLine
	scroll int
	cursor int
	width  int
	height int
}

// NewDetailView creates a DetailView with the given title and detail lines.
// The title is shown in the bordered header (e.g. "Project: my-project").
func NewDetailView(title string, lines []DetailLine) DetailView {
	return DetailView{
		title:  title,
		lines:  lines,
		scroll: 0,
		cursor: 0,
		width:  80,
		height: 24,
	}
}

// SetSize updates the available width and height for rendering.
func (dv *DetailView) SetSize(w, h int) {
	dv.width = w
	dv.height = h
}

// Update handles key and mouse messages for the detail view. It returns the
// updated DetailView and an optional tea.Cmd (DetailBackMsg for navigation).
func (dv *DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return *dv, func() tea.Msg { return DetailBackMsg{} }
		case "j", "down":
			dv.moveCursor(1)
		case "k", "up":
			dv.moveCursor(-1)
		case "g", "home":
			dv.cursor = 0
			dv.scroll = 0
		case "G", "end":
			rendered := dv.renderedLines()
			if len(rendered) > 0 {
				dv.cursor = len(rendered) - 1
			}
			dv.ensureCursorVisible()
		case "pgdown":
			dv.moveCursor(dv.viewportHeight())
		case "pgup":
			dv.moveCursor(-dv.viewportHeight())
		case "c":
			// Copy value of the current rendered line to clipboard.
			// The cursor indexes rendered (wrapped) lines, so we map back to
			// the source line's value via renderedLines.
			rendered := dv.renderedLines()
			if dv.cursor >= 0 && dv.cursor < len(rendered) {
				line := rendered[dv.cursor]
				// If this is a continuation line (empty Key), walk backwards
				// to find the source key-value pair and copy its full value.
				if line.Key == "" {
					for j := dv.cursor - 1; j >= 0; j-- {
						if rendered[j].Key != "" {
							// Find the original source line by Key.
							for _, src := range dv.lines {
								if src.Key == rendered[j].Key {
									return *dv, copyToClipboard(src.Value)
								}
							}
							break
						}
					}
					// Fallback: copy the continuation line's value.
					return *dv, copyToClipboard(line.Value)
				}
				// Key-value line — find the full source value.
				for _, src := range dv.lines {
					if src.Key == line.Key {
						return *dv, copyToClipboard(src.Value)
					}
				}
				return *dv, copyToClipboard(line.Value)
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			dv.moveCursor(-3)
		case tea.MouseButtonWheelDown:
			dv.moveCursor(3)
		}
	}

	return *dv, nil
}

// View renders the detail view as a bordered box with a title, scrollable
// key-value pairs, and a hint line at the bottom.
func (dv *DetailView) View() string {
	borderStyle := lipgloss.NewStyle().Foreground(detailBorderColor)
	titleStyle := lipgloss.NewStyle().Foreground(detailTitleColor).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(detailHintColor)

	contentWidth := dv.width
	if contentWidth < 20 {
		contentWidth = 80
	}
	innerWidth := contentWidth - 4 // 2 for borders + 2 for padding

	// Render title bar.
	titleText := " " + titleStyle.Render(dv.title) + " "
	titleVisualWidth := lipgloss.Width(titleText)
	remaining := contentWidth - titleVisualWidth - 2 // 2 for corner chars
	if remaining < 2 {
		remaining = 2
	}
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes

	titleBar := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleText +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	// Compute the maximum key width for right-aligned key column.
	rendered := dv.renderedLines()
	maxKeyWidth := dv.maxKeyWidth()
	if maxKeyWidth > innerWidth/3 {
		maxKeyWidth = innerWidth / 3
	}

	// Determine visible window.
	vpHeight := dv.viewportHeight()
	start := dv.scroll
	end := start + vpHeight
	if end > len(rendered) {
		end = len(rendered)
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(detailKeyColor).
		Width(maxKeyWidth).
		Align(lipgloss.Right)
	defaultValueStyle := lipgloss.NewStyle().Foreground(detailValueColor)

	// Render visible lines.
	var bodyLines []string
	for i := start; i < end; i++ {
		line := rendered[i]
		var lineStr string
		if line.Key == "" {
			// Continuation line (wrapped value) — indent to match value column.
			pad := strings.Repeat(" ", maxKeyWidth+3) // key width + "  " separator + 1
			valStyle := defaultValueStyle
			if line.Color != "" {
				valStyle = lipgloss.NewStyle().Foreground(line.Color)
			}
			valText := line.Value
			if lipgloss.Width(valText) > innerWidth-maxKeyWidth-3 {
				valText = TruncateString(valText, innerWidth-maxKeyWidth-3)
			}
			lineStr = pad + valStyle.Render(valText)
		} else {
			// Key-value line.
			valStyle := defaultValueStyle
			if line.Color != "" {
				valStyle = lipgloss.NewStyle().Foreground(line.Color)
			}
			keyText := keyStyle.Render(line.Key)
			valText := line.Value
			if lipgloss.Width(valText) > innerWidth-maxKeyWidth-3 {
				valText = TruncateString(valText, innerWidth-maxKeyWidth-3)
			}
			lineStr = keyText + "  " + valStyle.Render(valText)
		}

		// Highlight selected line.
		if i == dv.cursor {
			lineStr = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Render(lineStr)
		}

		lineVisualWidth := lipgloss.Width(lineStr)
		pad := ""
		if lineVisualWidth < innerWidth {
			pad = strings.Repeat(" ", innerWidth-lineVisualWidth)
		}
		bodyLines = append(bodyLines,
			borderStyle.Render("│")+" "+lineStr+pad+" "+borderStyle.Render("│"))
	}

	// Fill remaining viewport with empty lines.
	for i := len(bodyLines); i < vpHeight; i++ {
		empty := strings.Repeat(" ", innerWidth+2)
		bodyLines = append(bodyLines,
			borderStyle.Render("│")+empty+borderStyle.Render("│"))
	}

	// Scroll indicator.
	scrollInfo := ""
	if len(rendered) > vpHeight {
		pct := 0
		if len(rendered)-vpHeight > 0 {
			pct = (dv.scroll * 100) / (len(rendered) - vpHeight)
		}
		scrollInfo = fmt.Sprintf(" %d%% ", pct)
	}

	// Bottom border with hints.
	hint := hintStyle.Render(" Esc:back  j/k:scroll  c:copy ")
	hintWidth := lipgloss.Width(hint)
	scrollWidth := lipgloss.Width(scrollInfo)
	bottomDashes := contentWidth - 2 - hintWidth - scrollWidth
	if bottomDashes < 2 {
		bottomDashes = 2
	}
	bottom := borderStyle.Render("└") +
		hint +
		borderStyle.Render(strings.Repeat("─", bottomDashes)) +
		hintStyle.Render(scrollInfo) +
		borderStyle.Render("┘")

	return titleBar + "\n" + strings.Join(bodyLines, "\n") + "\n" + bottom
}

// renderedLines returns the detail lines after wrapping long values to fit the
// available width. Continuation lines have an empty Key.
func (dv *DetailView) renderedLines() []DetailLine {
	innerWidth := dv.width - 4
	maxKeyWidth := dv.maxKeyWidth()
	if maxKeyWidth > innerWidth/3 {
		maxKeyWidth = innerWidth / 3
	}
	valueWidth := innerWidth - maxKeyWidth - 3 // 3 for "  " separator + margin
	if valueWidth < 20 {
		valueWidth = 20
	}

	var result []DetailLine
	for _, line := range dv.lines {
		wrapped := detailWrapText(line.Value, valueWidth)
		for i, segment := range wrapped {
			if i == 0 {
				result = append(result, DetailLine{
					Key:   line.Key,
					Value: segment,
					Color: line.Color,
				})
			} else {
				result = append(result, DetailLine{
					Key:   "",
					Value: segment,
					Color: line.Color,
				})
			}
		}
	}
	return result
}

// maxKeyWidth computes the width of the longest key across all detail lines.
func (dv *DetailView) maxKeyWidth() int {
	maxW := 0
	for _, line := range dv.lines {
		if len(line.Key) > maxW {
			maxW = len(line.Key)
		}
	}
	return maxW
}

// viewportHeight returns the number of content lines visible in the viewport.
// Reserves space for the title bar (1), bottom border (1).
func (dv *DetailView) viewportHeight() int {
	h := dv.height - 2
	if h < 1 {
		h = 1
	}
	return h
}

// moveCursor moves the cursor by delta lines, clamping to valid bounds and
// adjusting scroll to keep the cursor visible.
func (dv *DetailView) moveCursor(delta int) {
	rendered := dv.renderedLines()
	if len(rendered) == 0 {
		return
	}

	dv.cursor += delta
	if dv.cursor < 0 {
		dv.cursor = 0
	}
	if dv.cursor >= len(rendered) {
		dv.cursor = len(rendered) - 1
	}
	dv.ensureCursorVisible()
}

// ensureCursorVisible adjusts the scroll offset so the cursor is within the
// visible viewport.
func (dv *DetailView) ensureCursorVisible() {
	vpHeight := dv.viewportHeight()
	rendered := dv.renderedLines()

	if dv.cursor < dv.scroll {
		dv.scroll = dv.cursor
	}
	if dv.cursor >= dv.scroll+vpHeight {
		dv.scroll = dv.cursor - vpHeight + 1
	}
	maxScroll := len(rendered) - vpHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if dv.scroll > maxScroll {
		dv.scroll = maxScroll
	}
	if dv.scroll < 0 {
		dv.scroll = 0
	}
}

// detailWrapText splits text into lines of at most width runes, preserving
// existing newlines. It wraps on word boundaries when possible, falling back to
// hard wraps for long unbroken tokens. Empty input returns a single-element
// slice with an empty string. This variant preserves newlines (unlike the
// conversation-mode wrapText in messages.go which collapses them).
func detailWrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	if text == "" {
		return []string{""}
	}

	// Split on existing newlines first.
	rawLines := strings.Split(text, "\n")
	var result []string
	for _, raw := range rawLines {
		if len([]rune(raw)) <= width {
			result = append(result, raw)
			continue
		}
		wrapped := detailWrapLine(raw, width)
		result = append(result, wrapped...)
	}
	return result
}

// detailWrapLine wraps a single line of text at word boundaries to fit within width.
func detailWrapLine(line string, width int) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := ""
	for _, word := range words {
		wordRunes := []rune(word)
		// If the word itself is too long, hard-wrap it.
		if len(wordRunes) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			for len(wordRunes) > 0 {
				take := width
				if take > len(wordRunes) {
					take = len(wordRunes)
				}
				lines = append(lines, string(wordRunes[:take]))
				wordRunes = wordRunes[take:]
			}
			continue
		}

		if current == "" {
			current = word
		} else if len([]rune(current))+1+len(wordRunes) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// copyToClipboard returns a tea.Cmd that writes the value to the system
// clipboard using the atotto/clipboard package (already a dependency).
func copyToClipboard(value string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.WriteAll(value)
		return nil
	}
}

// formatTimePtr formats a *time.Time as a human-readable string. Returns an
// empty string for nil pointers.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// formatJSON attempts to pretty-print a JSON string. If the input is not valid
// JSON (or is empty), it is returned as-is.
func formatJSON(s string) string {
	if s == "" {
		return ""
	}
	// Try to parse as a JSON object or array.
	var obj interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return s
	}
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return s
	}
	return string(formatted)
}

// ResourceJSON converts any resource to DetailLines showing pretty-printed JSON.
// Used by the `y` (YAML) hotkey to show the raw resource data.
func ResourceJSON(resource any) []DetailLine {
	data, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return []DetailLine{{Key: "error", Value: err.Error()}}
	}
	var lines []DetailLine
	for _, line := range strings.Split(string(data), "\n") {
		lines = append(lines, DetailLine{Value: line})
	}
	return lines
}

// --- Resource-specific detail constructors ---

// ProjectDetail returns detail lines for all fields of a Project resource.
func ProjectDetail(p sdktypes.Project) []DetailLine {
	lines := []DetailLine{
		{Key: "ID", Value: p.ID},
		{Key: "Name", Value: p.Name},
		{Key: "Display Name", Value: p.DisplayName},
		{Key: "Description", Value: p.Description},
		{Key: "Status", Value: p.Status},
		{Key: "Prompt", Value: p.Prompt},
		{Key: "Labels", Value: formatJSON(p.Labels)},
		{Key: "Annotations", Value: formatJSON(p.Annotations)},
		{Key: "Kind", Value: p.Kind},
		{Key: "Href", Value: p.Href},
		{Key: "Created At", Value: formatTimePtr(p.CreatedAt)},
		{Key: "Updated At", Value: formatTimePtr(p.UpdatedAt)},
	}
	return lines
}

// AgentDetail returns detail lines for all fields of an Agent resource.
func AgentDetail(a sdktypes.Agent) []DetailLine {
	lines := []DetailLine{
		{Key: "ID", Value: a.ID},
		{Key: "Name", Value: a.Name},
		{Key: "Display Name", Value: a.DisplayName},
		{Key: "Description", Value: a.Description},
		{Key: "Project ID", Value: a.ProjectID},
		{Key: "Prompt", Value: a.Prompt},
		{Key: "Current Session", Value: a.CurrentSessionID},
		{Key: "Owner User ID", Value: a.OwnerUserID},
		{Key: "Parent Agent ID", Value: a.ParentAgentID},
		{Key: "Bot Account", Value: a.BotAccountName},
		{Key: "LLM Model", Value: a.LlmModel},
		{Key: "LLM Max Tokens", Value: formatInt32(a.LlmMaxTokens)},
		{Key: "LLM Temperature", Value: formatFloat64(a.LlmTemperature)},
		{Key: "Repo URL", Value: a.RepoURL},
		{Key: "Workflow ID", Value: a.WorkflowID},
		{Key: "Resource Overrides", Value: formatJSON(a.ResourceOverrides)},
		{Key: "Env Variables", Value: formatJSON(a.EnvironmentVariables)},
		{Key: "Labels", Value: formatJSON(a.Labels)},
		{Key: "Annotations", Value: formatJSON(a.Annotations)},
		{Key: "Kind", Value: a.Kind},
		{Key: "Href", Value: a.Href},
		{Key: "Created At", Value: formatTimePtr(a.CreatedAt)},
		{Key: "Updated At", Value: formatTimePtr(a.UpdatedAt)},
	}
	return lines
}

// SessionDetail returns detail lines for all fields of a Session resource.
func SessionDetail(s sdktypes.Session) []DetailLine {
	lines := []DetailLine{
		{Key: "ID", Value: s.ID},
		{Key: "Name", Value: s.Name},
		{Key: "Phase", Value: s.Phase, Color: phaseColor(s.Phase)},
		{Key: "Project ID", Value: s.ProjectID},
		{Key: "Agent ID", Value: s.AgentID},
		{Key: "Prompt", Value: s.Prompt},
		{Key: "Triggered By", Value: s.TriggeredByUserID},
		{Key: "Assigned User", Value: s.AssignedUserID},
		{Key: "Created By", Value: s.CreatedByUserID},
		{Key: "Bot Account", Value: s.BotAccountName},
		{Key: "Parent Session", Value: s.ParentSessionID},
		{Key: "Start Time", Value: formatTimePtr(s.StartTime)},
		{Key: "Completion Time", Value: formatTimePtr(s.CompletionTime)},
		{Key: "Duration", Value: formatDuration(s.StartTime, s.CompletionTime)},
		{Key: "Timeout", Value: formatInt(s.Timeout)},
		{Key: "LLM Model", Value: s.LlmModel},
		{Key: "LLM Max Tokens", Value: formatInt(s.LlmMaxTokens)},
		{Key: "LLM Temperature", Value: formatFloat64(s.LlmTemperature)},
		{Key: "Repo URL", Value: s.RepoURL},
		{Key: "Repos", Value: formatJSON(s.Repos)},
		{Key: "Reconciled Repos", Value: formatJSON(s.ReconciledRepos)},
		{Key: "Workflow ID", Value: s.WorkflowID},
		{Key: "Reconciled Workflow", Value: formatJSON(s.ReconciledWorkflow)},
		{Key: "Resource Overrides", Value: formatJSON(s.ResourceOverrides)},
		{Key: "Env Variables", Value: formatJSON(s.EnvironmentVariables)},
		{Key: "SDK Session ID", Value: s.SdkSessionID},
		{Key: "SDK Restart Count", Value: formatInt(s.SdkRestartCount)},
		{Key: "Conditions", Value: formatJSON(s.Conditions)},
		{Key: "Labels", Value: formatJSON(s.Labels)},
		{Key: "Annotations", Value: formatJSON(s.Annotations)},
		{Key: "Kube CR Name", Value: s.KubeCrName},
		{Key: "Kube CR UID", Value: s.KubeCrUid},
		{Key: "Kube Namespace", Value: s.KubeNamespace},
		{Key: "Kind", Value: s.Kind},
		{Key: "Href", Value: s.Href},
		{Key: "Created At", Value: formatTimePtr(s.CreatedAt)},
		{Key: "Updated At", Value: formatTimePtr(s.UpdatedAt)},
	}
	return lines
}

// InboxDetail returns detail lines for all fields of an InboxMessage resource.
func InboxDetail(msg sdktypes.InboxMessage) []DetailLine {
	from := msg.FromName
	if from == "" {
		from = "(human)"
	}

	readStr := "No"
	if msg.Read {
		readStr = "Yes"
	}

	lines := []DetailLine{
		{Key: "ID", Value: msg.ID},
		{Key: "Agent ID", Value: msg.AgentID},
		{Key: "From", Value: from},
		{Key: "From Agent ID", Value: msg.FromAgentID},
		{Key: "Read", Value: readStr},
		{Key: "Body", Value: msg.Body},
		{Key: "Kind", Value: msg.Kind},
		{Key: "Href", Value: msg.Href},
		{Key: "Created At", Value: formatTimePtr(msg.CreatedAt)},
		{Key: "Updated At", Value: formatTimePtr(msg.UpdatedAt)},
	}
	return lines
}

// --- Numeric formatting helpers ---

// formatInt formats an int as a string. Returns empty for zero values to keep
// the detail view clean (zero typically means "not set").
func formatInt(v int) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

// formatInt32 formats an int32 as a string. Returns empty for zero values.
func formatInt32(v int32) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

// formatFloat64 formats a float64 as a string. Returns empty for zero values.
func formatFloat64(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", v)
}

// formatDuration computes and formats the duration between two time pointers.
// Returns empty if either is nil.
func formatDuration(start, end *time.Time) string {
	if start == nil || end == nil {
		return ""
	}
	d := end.Sub(*start)
	if d < 0 {
		return "0s"
	}
	return FormatAge(d)
}
