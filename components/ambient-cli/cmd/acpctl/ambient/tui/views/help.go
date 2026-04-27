package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Local color constants for the help view. Defined here instead of importing
// from the parent tui package to avoid circular imports.
var (
	helpHeaderColor = lipgloss.Color("214") // orange/cyan for column headers (k9s style)
	helpKeyColor    = lipgloss.Color("240") // dim for key brackets
	helpActionColor = lipgloss.Color("255") // white for action text
	helpHintColor   = lipgloss.Color("240") // dim for close hint
)

// HelpEntry represents a single keyboard shortcut entry in the help overlay.
type HelpEntry struct {
	Key    string // e.g. "s", "ctrl-d", "Enter"
	Action string // e.g. "Start", "Delete", "Drill into sessions"
}

// HelpView renders a full-screen help overlay showing keyboard shortcuts
// organized into three columns: Resource, General, and Navigation.
// Renders without borders, filling the table area like k9s does.
type HelpView struct {
	title      string
	resource   []HelpEntry
	general    []HelpEntry
	navigation []HelpEntry
	width      int
	height     int
}

// NewHelpView creates a HelpView with the given title and shortcut entries.
func NewHelpView(title string, resource, general, navigation []HelpEntry) HelpView {
	return HelpView{
		title:      title,
		resource:   resource,
		general:    general,
		navigation: navigation,
		width:      80,
		height:     24,
	}
}

// SetSize updates the available width and height for rendering.
func (h *HelpView) SetSize(w, ht int) {
	h.width = w
	h.height = ht
}

// View renders the help view as borderless columns filling the table area.
func (h HelpView) View() string {
	headerStyle := lipgloss.NewStyle().Foreground(helpHeaderColor).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(helpKeyColor)
	actionStyle := lipgloss.NewStyle().Foreground(helpActionColor)
	hintStyle := lipgloss.NewStyle().Foreground(helpHintColor)

	contentWidth := h.width
	if contentWidth < 20 {
		contentWidth = 80
	}
	innerWidth := contentWidth - 4 // padding on each side

	// Compute column widths. Split inner width roughly into thirds.
	colWidth := innerWidth / 3
	if colWidth < 15 {
		colWidth = 15
	}
	col1W := colWidth
	col2W := colWidth
	col3W := innerWidth - col1W - col2W
	if col3W < 10 {
		col3W = 10
	}

	// Compute the max key width per column for alignment.
	// Account for the <> brackets that renderHelpKey adds.
	resKeyW := maxFormattedKeyWidth(h.resource)
	genKeyW := maxFormattedKeyWidth(h.general)
	navKeyW := maxFormattedKeyWidth(h.navigation)

	// Find the tallest column to know how many rows we need (Fix 2: no blank rows).
	maxRows := len(h.resource)
	if len(h.general) > maxRows {
		maxRows = len(h.general)
	}
	if len(h.navigation) > maxRows {
		maxRows = len(h.navigation)
	}

	// Available content height.
	vpHeight := h.height - 5 // headers(2) + blank + hint + padding
	if vpHeight < 1 {
		vpHeight = 1
	}
	if maxRows > vpHeight {
		maxRows = vpHeight
	}

	var bodyLines []string

	// Blank line before headers.
	bodyLines = append(bodyLines, "")

	// Column headers (colored like k9s — orange).
	hdr1 := headerStyle.Render(padRight("RESOURCE", col1W))
	hdr2 := headerStyle.Render(padRight("GENERAL", col2W))
	hdr3 := headerStyle.Render(padRight("NAVIGATION", col3W))
	bodyLines = append(bodyLines, "  "+hdr1+hdr2+hdr3)

	// Underlines for column headers.
	ul1 := headerStyle.Render(padRight(strings.Repeat("─", min(len("RESOURCE"), col1W-2)), col1W))
	ul2 := headerStyle.Render(padRight(strings.Repeat("─", min(len("GENERAL"), col2W-2)), col2W))
	ul3 := headerStyle.Render(padRight(strings.Repeat("─", min(len("NAVIGATION"), col3W-2)), col3W))
	bodyLines = append(bodyLines, "  "+ul1+ul2+ul3)

	// Data rows (Fix 2: only render up to maxRows, empty cells are blank space).
	for i := range maxRows {
		c1 := renderHelpEntry(h.resource, i, resKeyW, col1W, keyStyle, actionStyle)
		c2 := renderHelpEntry(h.general, i, genKeyW, col2W, keyStyle, actionStyle)
		c3 := renderHelpEntry(h.navigation, i, navKeyW, col3W, keyStyle, actionStyle)
		bodyLines = append(bodyLines, "  "+c1+c2+c3)
	}

	// Fill remaining space.
	targetLines := vpHeight + 3
	for i := len(bodyLines); i < targetLines; i++ {
		bodyLines = append(bodyLines, "")
	}

	// Hint line: "Press Esc or ? to close" centered.
	hint := hintStyle.Render("Press Esc or ? to close")
	hintWidth := lipgloss.Width(hint)
	hintLeftPad := (innerWidth - hintWidth) / 2
	if hintLeftPad < 0 {
		hintLeftPad = 0
	}
	bodyLines = append(bodyLines, strings.Repeat(" ", hintLeftPad)+hint)

	return strings.Join(bodyLines, "\n")
}

// renderHelpEntry renders a single help entry cell for a column, or empty space
// if the index is out of range for that column's entries.
// Keys are rendered with dim brackets like the header hints: <key>.
func renderHelpEntry(entries []HelpEntry, idx, maxKeyW, colW int, keyStyle, actionStyle lipgloss.Style) string {
	if idx >= len(entries) {
		return padRight("", colW)
	}
	e := entries[idx]
	// Render key with dim brackets: <key>
	keyText := "<" + e.Key + ">"
	keyRendered := keyStyle.Render(padRight(keyText, maxKeyW))
	actionRendered := actionStyle.Render(e.Action)
	cell := keyRendered + " " + actionRendered
	cellWidth := lipgloss.Width(cell)
	if cellWidth < colW {
		cell += strings.Repeat(" ", colW-cellWidth)
	}
	return cell
}

// maxFormattedKeyWidth returns the maximum formatted key width (with <> brackets).
func maxFormattedKeyWidth(entries []HelpEntry) int {
	maxW := 0
	for _, e := range entries {
		w := len("<" + e.Key + ">")
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

// padRight pads s with spaces to reach visual width w. Uses lipgloss.Width to
// correctly handle multi-byte Unicode characters and ANSI escape sequences.
func padRight(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}
