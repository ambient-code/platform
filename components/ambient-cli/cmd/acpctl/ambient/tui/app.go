package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ASCII art branding rendered in the header.
var brandLines = []string{
	`   _    ___ ___  `,
	`  /_\  / __| _ \ `,
	` / _ \| (__|  _/ `,
	`/_/ \_\\___|_|   `,
	`                 `,
}

// View implements tea.Model. It renders the k9s-style full-screen layout.
func (m *AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// 1. Header block.
	sections = append(sections, m.viewHeader())

	// 2. Separator.
	sections = append(sections, styleDim.Render(strings.Repeat("─", m.width)))

	// 3. Command/filter/prompt bar (only when active).
	if m.commandMode || m.filterMode || m.promptMode {
		sections = append(sections, m.viewCommandBar())
	}

	// 4. Resource table with title bar.
	sections = append(sections, m.viewResourceTable())

	// 5. Separator.
	sections = append(sections, styleDim.Render(strings.Repeat("─", m.width)))

	// 6. Breadcrumb trail.
	sections = append(sections, m.viewBreadcrumb())

	// 7. Info line.
	sections = append(sections, m.viewInfoLine())

	return strings.Join(sections, "\n")
}

// viewHeader renders the multi-line header block with context info on the left,
// project shortcuts in the center, and contextual hotkeys + static hints + branding
// on the right.
func (m *AppModel) viewHeader() string {
	// Left side: context metadata lines.
	contextName := "none"
	serverURL := "unknown"
	project := "none"
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

	// Refresh indicator.
	refreshIndicator := ""
	if !m.lastFetch.IsZero() {
		elapsed := time.Since(m.lastFetch)
		indicator := fmt.Sprintf("%ds", int(elapsed.Seconds()))
		if elapsed > staleThreshold {
			indicator += " (stale)"
			refreshIndicator = styleRed.Render("  ⟳ " + indicator)
		} else {
			refreshIndicator = styleDim.Render("  ⟳ " + indicator)
		}
	}

	leftLines := []string{
		fmt.Sprintf("  %s %s %s",
			styleDim.Render("Context:"),
			styleOrange.Render(contextName),
			styleDim.Render("[RW]"),
		),
		fmt.Sprintf("  %s %s",
			styleDim.Render("Server: "),
			styleWhite.Render(serverURL),
		),
		fmt.Sprintf("  %s %s",
			styleDim.Render("User:   "),
			styleWhite.Render("user"),
		),
		fmt.Sprintf("  %s %s",
			styleDim.Render("Project:"),
			styleOrange.Render(project),
		),
		refreshIndicator,
	}

	// Build stacked project shortcuts (only below project/context level).
	// Rendered vertically like k9s namespace shortcuts:
	//   <0> all
	//   <1> test
	//   <2> test-jsell
	showShortcuts := m.activeView != "projects" && m.activeView != "contexts" && len(m.projectShortcuts) > 0
	var shortcutLines []string
	if showShortcuts {
		shortcutLines = append(shortcutLines, styleCyan.Render("<0>")+" "+styleCyan.Render("all"))
		maxShortcuts := min(len(m.projectShortcuts), 4)
		for i := range maxShortcuts {
			shortcutLines = append(shortcutLines,
				styleCyan.Render(fmt.Sprintf("<%d>", i+1))+" "+styleCyan.Render(m.projectShortcuts[i]))
		}
	}

	// Build contextual hints (two rows, ~4 per row).
	ctxHints := m.contextualHints()
	var ctxRow1, ctxRow2 []string
	splitAt := (len(ctxHints) + 1) / 2
	for i, h := range ctxHints {
		rendered := m.renderHint(h)
		if i < splitAt {
			ctxRow1 = append(ctxRow1, rendered)
		} else {
			ctxRow2 = append(ctxRow2, rendered)
		}
	}
	ctxLine1 := strings.Join(ctxRow1, "  ")
	ctxLine2 := strings.Join(ctxRow2, "  ")

	// Static hints (always shown).
	staticHints := []string{
		styleDim.Render("<?>") + " " + styleWhite.Render("Help"),
		styleDim.Render("<:>") + " " + styleWhite.Render("Command"),
		styleDim.Render("</>") + " " + styleWhite.Render("Filter"),
	}
	staticLine := strings.Join(staticHints, "  ")

	// Right side layout:
	//   Line 0: ctx hints row1 + static hints
	//   Line 1: ctx hints row2
	//   Lines 2+: (empty, branding fills in)
	rightHintLines := make([]string, 5)
	if len(ctxRow1) > 0 {
		rightHintLines[0] = ctxLine1 + "   " + staticLine
	} else {
		rightHintLines[0] = staticLine
	}
	if len(ctxRow2) > 0 {
		rightHintLines[1] = ctxLine2
	}

	// Combine left metadata, shortcuts (middle), right hints + branding.
	headerLines := make([]string, 5)
	for i := range 5 {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}

		// Stacked project shortcuts (middle column).
		shortcut := ""
		if i < len(shortcutLines) {
			shortcut = "  " + shortcutLines[i]
		}

		hint := rightHintLines[i]

		brand := ""
		if i < len(brandLines) {
			brand = styleOrange.Render(brandLines[i])
		}

		leftContent := left + shortcut
		leftWidth := lipgloss.Width(leftContent)

		var rightContent string
		var rightWidth int
		if hint != "" {
			rightContent = hint + "  " + brand
			rightWidth = lipgloss.Width(hint) + 2 + lipgloss.Width(brand)
		} else {
			rightContent = brand
			rightWidth = lipgloss.Width(brand)
		}

		gap := m.width - leftWidth - rightWidth
		if gap < 1 {
			gap = 1
		}

		headerLines[i] = leftContent + strings.Repeat(" ", gap) + rightContent
	}

	return strings.Join(headerLines, "\n")
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
	default:
		return m.projectTable.View()
	}
}

// viewBreadcrumb renders the navigation breadcrumb trail at the bottom.
func (m *AppModel) viewBreadcrumb() string {
	var segments []string
	for _, entry := range m.navStack {
		segments = append(segments, styleOrange.Render("<"+entry.Kind+">"))
	}
	return "  " + strings.Join(segments, styleDim.Render("  "))
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
