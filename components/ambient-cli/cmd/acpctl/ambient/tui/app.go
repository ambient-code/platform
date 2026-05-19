package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
)

// Hoisted command bar border style to avoid allocations on every frame.
var commandBarBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))

// ASCII art branding rendered in the header (Fix 9: extra left padding).
var brandLines = []string{
	`                  `,
	`    _    ___ ___  `,
	`   /_\  / __| _ \ `,
	`  / _ \| (__|  _/ `,
	` /_/ \_\\___|_|   `,
}

// View implements tea.Model. It renders the k9s-style full-screen layout.
func (m *AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// 1. Header block.
	sections = append(sections, m.viewHeader())

	// 2. Command/filter/prompt bar (only when active).
	if m.commandMode || m.filterMode || m.promptMode {
		sections = append(sections, m.viewCommandBar())
	}

	// 3. Resource table with title bar (+ dialog/form overlay if active).
	tableOutput := m.viewResourceTable()
	if m.formOverlay != nil {
		tableH := m.height - 10
		if tableH < 1 {
			tableH = 1
		}
		tableOutput = views.OverlayForm(tableOutput, m.formOverlay.View(), m.formTitle, m.width, tableH)
	} else if m.dialog != nil {
		tableH := m.height - 10
		if tableH < 1 {
			tableH = 1
		}
		tableOutput = views.OverlayDialog(tableOutput, *m.dialog, m.width, tableH)
	}
	sections = append(sections, tableOutput)

	// 4. Breadcrumb trail.
	sections = append(sections, m.viewBreadcrumb())

	// 5. Info line.
	sections = append(sections, m.viewInfoLine())

	return strings.Join(sections, "\n")
}

// viewHeader renders the header with 4 columns like k9s:
//
//	Col1: Metadata    Col2: Project shortcuts    Col3: Hotkey hints    Col4: Logo+refresh
func (m *AppModel) viewHeader() string {
	serverURL, project := "unknown", "none"
	if m.config != nil {
		if ctx := m.config.Current(); ctx != nil {
			if ctx.Server != "" {
				serverURL = ctx.Server
			}
			if ctx.Project != "" {
				project = ctx.Project
			}
		}
	}
	// Col 1: metadata (context URL on its own row below the grid).
	col1 := [5]string{
		fmt.Sprintf(" %s %s", styleDim.Render("User:   "), styleWhite.Render(m.currentUser())),
		fmt.Sprintf(" %s %s", styleDim.Render("Project:"), styleOrange.Render(project)),
	}

	// Col 2: project shortcuts (stacked, padded to fixed width).
	var col2 [5]string
	showShortcuts := m.activeView != "projects" && m.activeView != "contexts" &&
		m.activeView != "messages" && m.activeView != "detail" && len(m.projectShortcuts) > 0
	if showShortcuts {
		col2[0] = styleBlue.Render("<0>") + " " + styleWhite.Render("all")
		for i := range min(len(m.projectShortcuts), 4) {
			name := m.projectShortcuts[i]
			if len(name) > 16 {
				name = name[:13] + "..."
			}
			col2[i+1] = styleBlue.Render(fmt.Sprintf("<%d>", i+1)) + " " + styleWhite.Render(name)
		}
	}

	// Col 3: contextual hotkey hints (up to 4 rows, column-aligned).
	var col3 [5]string
	hints := m.contextualHints()
	perRow := 4
	if len(hints) <= 8 {
		perRow = (len(hints) + 3) / 4
		if perRow < 2 {
			perRow = 2
		}
	}

	colKeyWidths := make([]int, perRow)
	for i, h := range hints {
		if idx := strings.Index(h, ">"); idx >= 0 {
			if w := lipgloss.Width(h[:idx+1]); w > colKeyWidths[i%perRow] {
				colKeyWidths[i%perRow] = w
			}
		}
	}

	rendered := make([]string, len(hints))
	for i, h := range hints {
		rendered[i] = m.renderHint(h, colKeyWidths[i%perRow])
	}

	colWidths := make([]int, perRow)
	for i, r := range rendered {
		if w := lipgloss.Width(r); w > colWidths[i%perRow] {
			colWidths[i%perRow] = w
		}
	}

	rowIdx := 0
	var currentRow []string
	for i, r := range rendered {
		pad := colWidths[i%perRow] - lipgloss.Width(r)
		currentRow = append(currentRow, r+strings.Repeat(" ", pad))
		if (i+1)%perRow == 0 || i == len(rendered)-1 {
			if rowIdx < 5 {
				col3[rowIdx] = strings.Join(currentRow, "  ")
			}
			currentRow = nil
			rowIdx++
		}
	}

	// Col 4: static hints + logo + refresh.
	var col4 [5]string
	col4[0] = styleDim.Render("<?>") + " " + styleWhite.Render("Help   ")
	col4[1] = styleDim.Render("<:>") + " " + styleWhite.Render("Command")
	col4[2] = styleDim.Render("</>") + " " + styleWhite.Render("Filter ")
	if !m.lastFetch.IsZero() {
		elapsed := time.Since(m.lastFetch)
		if elapsed > staleThreshold {
			ind := fmt.Sprintf("⟳ %ds (stale)", int(elapsed.Seconds()))
			col4[3] = styleRed.Render(ind)
		}
	}

	// Dynamic column positions based on terminal width.
	col2Start := 40 // shortcuts column starts at char 40
	col3Start := 65 // hotkeys column starts at char 65

	// On narrow terminals, skip columns to avoid overlap.
	skipShortcuts := m.width < 100
	skipHints := m.width < 80

	lines := make([]string, 5)
	for i := range 5 {
		// Start with col1.
		line := col1[i]
		w := lipgloss.Width(line)

		// Pad to col2 position and add shortcut (skip on narrow terminals).
		if col2[i] != "" && !skipShortcuts {
			if w < col2Start {
				line += strings.Repeat(" ", col2Start-w)
			} else {
				line += "  "
			}
			line += col2[i]
		}
		w = lipgloss.Width(line)

		// Pad to col3 position and add hints (skip on narrow terminals).
		if col3[i] != "" && !skipHints {
			if w < col3Start {
				line += strings.Repeat(" ", col3Start-w)
			} else {
				line += "  "
			}
			line += col3[i]
		}
		w = lipgloss.Width(line)

		// Right-align col4 (static hints + brand).
		brandStyle := styleOrange
		if m.authExpired {
			brandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
		}
		brand := ""
		if i < len(brandLines) {
			brand = brandStyle.Render(brandLines[i])
		}
		right := ""
		if col4[i] != "" && brand != "" {
			right = col4[i] + "   " + brand
		} else if brand != "" {
			right = brand
		} else {
			right = col4[i]
		}
		rw := lipgloss.Width(right)
		gap := m.width - w - rw
		if gap < 1 {
			gap = 1
		}
		lines[i] = line + strings.Repeat(" ", gap) + right
	}

	// Context URL on its own full-width row below the grid.
	contextLine := fmt.Sprintf(" %s %s %s", styleDim.Render("Context:"), styleDim.Render(serverURL), styleDim.Render("[RW]"))
	if m.authExpired {
		badge := lipgloss.NewStyle().
			Background(lipgloss.Color("69")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1).
			Render("Session Expired")
		badgeW := lipgloss.Width(badge)
		ctxW := lipgloss.Width(contextLine)
		pad := m.width - ctxW - badgeW
		if pad < 1 {
			pad = 1
		}
		contextLine += strings.Repeat(" ", pad) + badge
	}
	return strings.Join(lines, "\n") + "\n" + contextLine
}

// renderHint renders a single hotkey hint like "<d> Describe" with dim brackets
// and white action text. keyWidth is the visual width to pad all keys to (0 = no padding).
func (m *AppModel) renderHint(hint string, keyWidth int) string {
	if strings.HasPrefix(hint, "(") {
		return styleDim.Render(hint)
	}
	idx := strings.Index(hint, ">")
	if idx < 0 {
		return styleDim.Render(hint)
	}
	key := hint[:idx+1]    // e.g. "<d>"
	action := hint[idx+2:] // e.g. "Describe" (skip the space after >)
	renderedKey := styleDim.Render(key)
	pad := keyWidth + 1 - lipgloss.Width(renderedKey)
	if pad < 1 {
		pad = 1
	}
	return renderedKey + strings.Repeat(" ", pad) + styleWhite.Render(action)
}

// viewCommandBar renders the command, filter, or prompt input bar with a border.
func (m *AppModel) viewCommandBar() string {
	var content string
	if m.promptMode {
		content = m.promptInput.View()
	} else if m.commandMode {
		content = m.commandInput.View()
	} else if m.filterMode {
		content = m.filterInput.View()
	} else {
		return ""
	}

	bs := commandBarBorderStyle
	innerW := m.width - 4
	if innerW < 10 {
		innerW = 10
	}

	top := bs.Render("┌" + strings.Repeat("─", innerW+2) + "┐")
	contentWidth := lipgloss.Width(content)
	pad := ""
	if contentWidth < innerW {
		pad = strings.Repeat(" ", innerW-contentWidth)
	}
	mid := bs.Render("│") + " " + content + pad + " " + bs.Render("│")
	bot := bs.Render("└" + strings.Repeat("─", innerW+2) + "┘")

	return top + "\n" + mid + "\n" + bot
}

// viewResourceTable renders the current resource table or view with its title bar.
func (m *AppModel) viewResourceTable() string {
	switch m.activeView {
	case "projects":
		return m.projectTable.View()
	case "agents":
		return m.agentTable.View()
	case "sessions":
		return m.sessionTable.View()
	case "inbox":
		return m.inboxTable.View()
	case "contexts":
		return m.contextTable.View()
	case "scheduledsessions":
		return m.scheduledSessionTable.View()
	case "messages":
		return m.messageStream.View()
	case "detail":
		return m.detailView.View()
	case "help":
		return m.helpView.View()
	default:
		return m.projectTable.View()
	}
}

// Hoisted breadcrumb styles to avoid allocations on every frame.
var (
	breadcrumbListStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("214")).
				Foreground(lipgloss.Color("0")).
				Bold(true).
				Padding(0, 1)
	breadcrumbLeafStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("231")).
				Bold(true).
				Padding(0, 1)
)

// viewBreadcrumb renders the navigation breadcrumb trail at the bottom.
// Each segment is an individual colored box: orange for list views, blue for leaves.
func (m *AppModel) viewBreadcrumb() string {
	listStyle := breadcrumbListStyle
	leafStyle := breadcrumbLeafStyle

	leafKinds := map[string]bool{"messages": true, "help": true, "detail": true}

	var segments []string
	for _, entry := range m.navStack {
		label := "<" + entry.Kind + ">"
		if leafKinds[entry.Kind] {
			segments = append(segments, leafStyle.Render(label))
		} else {
			segments = append(segments, listStyle.Render(label))
		}
	}
	return " " + strings.Join(segments, " ")
}

// viewInfoLine renders the ephemeral info/toast line at the very bottom.
func (m *AppModel) viewInfoLine() string {
	// Error takes priority over info.
	if m.lastError != "" {
		errText := styleRed.Render("✗ " + m.lastError)
		errWidth := lipgloss.Width(errText)
		pad := (m.width - errWidth) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + errText
	}

	if m.infoMessage != "" {
		// Center the info message.
		msgWidth := lipgloss.Width(m.infoMessage)
		pad := (m.width - msgWidth) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + styleDim.Render(m.infoMessage)
	}

	// Default: empty line.
	return ""
}
