package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
)

// ASCII art branding rendered in the header (Fix 9: extra left padding).
var brandLines = []string{
	`    _    ___ ___  `,
	`   /_\  / __| _ \ `,
	`  / _ \| (__|  _/ `,
	` /_/ \_\\___|_|   `,
	`                  `,
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

	// 3. Resource table with title bar (+ dialog overlay if active).
	tableOutput := m.viewResourceTable()
	if m.dialog != nil {
		tableH := m.height - 10
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
	contextName, serverURL, project := "none", "unknown", "none"
	if m.config != nil {
		if m.config.CurrentContext != "" {
			contextName = m.config.CurrentContext
		}
		if ctx := m.config.Current(); ctx != nil {
			if ctx.Server != "" {
				serverURL = ctx.Server
			}
			if ctx.Project != "" {
				project = ctx.Project
			}
		}
	}
	// Col 1: metadata.
	col1 := [5]string{
		fmt.Sprintf(" %s %s %s", styleDim.Render("Context:"), styleOrange.Render(contextName), styleDim.Render("[RW]")),
		fmt.Sprintf(" %s %s", styleDim.Render("User:   "), styleWhite.Render("user")),
		fmt.Sprintf(" %s %s", styleDim.Render("Project:"), styleOrange.Render(project)),
		fmt.Sprintf(" %s %s", styleDim.Render("Server: "), styleDim.Render(serverURL)),
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

	// Col 3: contextual hotkey hints (two rows).
	var col3 [5]string
	hints := m.contextualHints()
	split := (len(hints) + 1) / 2
	var row1, row2 []string
	for i, h := range hints {
		if i < split {
			row1 = append(row1, m.renderHint(h))
		} else {
			row2 = append(row2, m.renderHint(h))
		}
	}
	col3[0] = strings.Join(row1, "  ")
	col3[1] = strings.Join(row2, "  ")

	// Col 4: static hints + logo + refresh.
	var col4 [5]string
	col4[0] = styleDim.Render("<?>") + " " + styleWhite.Render("Help   ")
	col4[1] = styleDim.Render("<:>") + " " + styleWhite.Render("Command")
	col4[2] = styleDim.Render("</>") + " " + styleWhite.Render("Filter ")
	if !m.lastFetch.IsZero() {
		elapsed := time.Since(m.lastFetch)
		ind := fmt.Sprintf("⟳ %ds", int(elapsed.Seconds()))
		if elapsed > staleThreshold {
			col4[3] = styleRed.Render(ind + " (stale)")
		} else {
			col4[3] = styleDim.Render(ind)
		}
	}

	// Fixed column positions (visual widths).
	const col2Start = 40 // shortcuts column starts at char 40
	const col3Start = 65 // hotkeys column starts at char 65

	lines := make([]string, 5)
	for i := range 5 {
		// Start with col1.
		line := col1[i]
		w := lipgloss.Width(line)

		// Pad to col2 position and add shortcut.
		if col2[i] != "" {
			if w < col2Start {
				line += strings.Repeat(" ", col2Start-w)
			} else {
				line += "  "
			}
			line += col2[i]
		}
		w = lipgloss.Width(line)

		// Pad to col3 position and add hints.
		if col3[i] != "" {
			if w < col3Start {
				line += strings.Repeat(" ", col3Start-w)
			} else {
				line += "  "
			}
			line += col3[i]
		}
		w = lipgloss.Width(line)

		// Right-align col4 (static hints + brand).
		brand := ""
		if i < len(brandLines) {
			brand = styleOrange.Render(brandLines[i])
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

	return strings.Join(lines, "\n")
}

// renderHint renders a single hotkey hint like "<d> Describe" with dim brackets
// and white action text.
func (m *AppModel) renderHint(hint string) string {
	// Parse hints of the form "<key> Action" or "(text)".
	if strings.HasPrefix(hint, "(") {
		return styleDim.Render(hint)
	}
	// Find the closing bracket.
	idx := strings.Index(hint, ">")
	if idx < 0 {
		return styleDim.Render(hint)
	}
	key := hint[:idx+1]   // e.g. "<d>"
	action := hint[idx+1:] // e.g. " Describe"
	return styleDim.Render(key) + styleWhite.Render(action)
}

// viewCommandBar renders the command, filter, or prompt input bar.
func (m *AppModel) viewCommandBar() string {
	if m.promptMode {
		return "  " + m.promptInput.View()
	}
	if m.commandMode {
		return "  " + m.commandInput.View()
	}
	if m.filterMode {
		return "  " + m.filterInput.View()
	}
	return ""
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

// viewBreadcrumb renders the navigation breadcrumb trail at the bottom
// with a full-width background color: orange for list views, blue for leaf views.
func (m *AppModel) viewBreadcrumb() string {
	var segments []string
	for _, entry := range m.navStack {
		segments = append(segments, "<"+entry.Kind+">")
	}
	text := "  " + strings.Join(segments, "  ")

	// Determine if current view is a "leaf" view (messages, help, detail)
	// or a "list" view (projects, agents, sessions, inbox, contexts).
	isLeaf := m.activeView == "messages" || m.activeView == "help" || m.activeView == "detail"

	var bgStyle lipgloss.Style
	if isLeaf {
		bgStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("69")).
			Foreground(lipgloss.Color("255")).
			Width(m.width)
	} else {
		bgStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Width(m.width)
	}
	return bgStyle.Render(text)
}

// viewInfoLine renders the ephemeral info/toast line at the very bottom.
func (m *AppModel) viewInfoLine() string {
	// Error takes priority over info.
	if m.lastError != "" {
		return "  " + styleRed.Render("✗ "+m.lastError)
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
