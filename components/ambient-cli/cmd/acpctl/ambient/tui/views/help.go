package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Local color constants for the help view. Defined here instead of importing
// from the parent tui package to avoid circular imports.
var (
	helpBorderColor = lipgloss.Color("240") // dim for borders
	helpTitleColor  = lipgloss.Color("214") // orange for title
	helpHeaderColor = lipgloss.Color("240") // dim for column headers
	helpKeyColor    = lipgloss.Color("240") // dim for key brackets
	helpActionColor = lipgloss.Color("255") // white for action text
	helpHintColor   = lipgloss.Color("240") // dim for close hint
)

// HelpEntry represents a single keyboard shortcut entry in the help overlay.
type HelpEntry struct {
	Key    string // e.g. "<s>", "<Ctrl-D>", "<Enter>"
	Action string // e.g. "Start", "Delete", "Drill into sessions"
}

// HelpView renders a full-screen help overlay showing keyboard shortcuts
// organized into three columns: Resource, General, and Navigation.
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

// View renders the help overlay as a bordered box with three columns.
func (h HelpView) View() string {
	borderStyle := lipgloss.NewStyle().Foreground(helpBorderColor)
	titleStyle := lipgloss.NewStyle().Foreground(helpTitleColor).Bold(true)
	headerStyle := lipgloss.NewStyle().Foreground(helpHeaderColor).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(helpKeyColor)
	actionStyle := lipgloss.NewStyle().Foreground(helpActionColor)
	hintStyle := lipgloss.NewStyle().Foreground(helpHintColor)

	contentWidth := h.width
	if contentWidth < 20 {
		contentWidth = 80
	}
	innerWidth := contentWidth - 4 // 2 for borders + 2 for padding

	// Render title bar: ┌──── Help(agents) ────┐
	titleText := " " + titleStyle.Render("Help("+h.title+")") + " "
	titleVisualWidth := lipgloss.Width(titleText)
	remaining := contentWidth - titleVisualWidth - 2
	if remaining < 2 {
		remaining = 2
	}
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes
	titleBar := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleText +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

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
	resKeyW := maxEntryKeyWidth(h.resource)
	genKeyW := maxEntryKeyWidth(h.general)
	navKeyW := maxEntryKeyWidth(h.navigation)

	// Find the tallest column to know how many rows we need.
	maxRows := len(h.resource)
	if len(h.general) > maxRows {
		maxRows = len(h.general)
	}
	if len(h.navigation) > maxRows {
		maxRows = len(h.navigation)
	}

	// Available content height: total height minus title (1), bottom border (1), blank lines (2), headers (2), hint (1).
	vpHeight := h.height - 7
	if vpHeight < 1 {
		vpHeight = 1
	}
	if maxRows > vpHeight {
		maxRows = vpHeight
	}

	var bodyLines []string

	// Empty line.
	bodyLines = append(bodyLines, h.emptyLine(borderStyle, innerWidth))

	// Column headers.
	hdr1 := headerStyle.Render(padRight("Resource", col1W))
	hdr2 := headerStyle.Render(padRight("General", col2W))
	hdr3 := headerStyle.Render(padRight("Navigation", col3W))
	headerLine := hdr1 + hdr2 + hdr3
	headerLineWidth := lipgloss.Width(headerLine)
	headerPad := innerWidth - headerLineWidth
	if headerPad < 0 {
		headerPad = 0
	}
	bodyLines = append(bodyLines,
		borderStyle.Render("│")+" "+headerLine+strings.Repeat(" ", headerPad)+" "+borderStyle.Render("│"))

	// Underlines for column headers.
	ul1 := headerStyle.Render(padRight(strings.Repeat("─", min(len("Resource"), col1W-2)), col1W))
	ul2 := headerStyle.Render(padRight(strings.Repeat("─", min(len("General"), col2W-2)), col2W))
	ul3 := headerStyle.Render(padRight(strings.Repeat("─", min(len("Navigation"), col3W-2)), col3W))
	underlineLine := ul1 + ul2 + ul3
	underlineWidth := lipgloss.Width(underlineLine)
	underlinePad := innerWidth - underlineWidth
	if underlinePad < 0 {
		underlinePad = 0
	}
	bodyLines = append(bodyLines,
		borderStyle.Render("│")+" "+underlineLine+strings.Repeat(" ", underlinePad)+" "+borderStyle.Render("│"))

	// Data rows.
	for i := range maxRows {
		c1 := renderHelpEntry(h.resource, i, resKeyW, col1W, keyStyle, actionStyle)
		c2 := renderHelpEntry(h.general, i, genKeyW, col2W, keyStyle, actionStyle)
		c3 := renderHelpEntry(h.navigation, i, navKeyW, col3W, keyStyle, actionStyle)

		rowText := c1 + c2 + c3
		rowWidth := lipgloss.Width(rowText)
		rowPad := innerWidth - rowWidth
		if rowPad < 0 {
			rowPad = 0
		}
		bodyLines = append(bodyLines,
			borderStyle.Render("│")+" "+rowText+strings.Repeat(" ", rowPad)+" "+borderStyle.Render("│"))
	}

	// Fill remaining viewport with empty lines.
	contentLines := len(bodyLines)
	// We want: blank + headers(2) + data rows + blank + hint = vpHeight + 5
	targetLines := vpHeight + 3 // blank + headers(2) + data + blank + hint - 2 already counted
	for i := contentLines; i < targetLines; i++ {
		bodyLines = append(bodyLines, h.emptyLine(borderStyle, innerWidth))
	}

	// Empty line before hint.
	bodyLines = append(bodyLines, h.emptyLine(borderStyle, innerWidth))

	// Hint line: "Press Esc or ? to close" centered.
	hint := hintStyle.Render("Press Esc or ? to close")
	hintWidth := lipgloss.Width(hint)
	hintLeftPad := (innerWidth - hintWidth) / 2
	if hintLeftPad < 0 {
		hintLeftPad = 0
	}
	hintRightPad := innerWidth - hintLeftPad - hintWidth
	if hintRightPad < 0 {
		hintRightPad = 0
	}
	bodyLines = append(bodyLines,
		borderStyle.Render("│")+" "+strings.Repeat(" ", hintLeftPad)+hint+strings.Repeat(" ", hintRightPad)+" "+borderStyle.Render("│"))

	// Bottom border.
	bottom := borderStyle.Render("└" + strings.Repeat("─", contentWidth-2) + "┘")

	return titleBar + "\n" + strings.Join(bodyLines, "\n") + "\n" + bottom
}

// emptyLine renders an empty bordered line.
func (h HelpView) emptyLine(borderStyle lipgloss.Style, innerWidth int) string {
	return borderStyle.Render("│") + " " + strings.Repeat(" ", innerWidth) + " " + borderStyle.Render("│")
}

// renderHelpEntry renders a single help entry cell for a column, or empty space
// if the index is out of range for that column's entries.
func renderHelpEntry(entries []HelpEntry, idx, maxKeyW, colW int, keyStyle, actionStyle lipgloss.Style) string {
	if idx >= len(entries) {
		return padRight("", colW)
	}
	e := entries[idx]
	keyRendered := keyStyle.Render(padRight(e.Key, maxKeyW))
	actionRendered := actionStyle.Render(e.Action)
	cell := keyRendered + " " + actionRendered
	cellWidth := lipgloss.Width(cell)
	if cellWidth < colW {
		cell += strings.Repeat(" ", colW-cellWidth)
	}
	return cell
}

// maxEntryKeyWidth returns the maximum key string length across entries.
func maxEntryKeyWidth(entries []HelpEntry) int {
	maxW := 0
	for _, e := range entries {
		if len(e.Key) > maxW {
			maxW = len(e.Key)
		}
	}
	return maxW
}

// padRight pads s with spaces to reach width w. If s is already wider, it is
// returned unmodified.
func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
