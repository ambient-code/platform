package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ASCII art branding rendered in the header.
var brandLines = []string{
	` _    __  __  `,
	`/_\  |  \/  | `,
	`/ _ \ | |\/| | `,
	`/_/ \_\|_|  |_| `,
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

	// 3. Command/filter bar (only when active).
	if m.commandMode || m.filterMode {
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

// viewHeader renders the multi-line header block with context info on the left
// and branding + key hints on the right.
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

	// Right side: key hints + branding (column-aligned).
	hintLines := []string{
		styleDim.Render("<?>") + "  " + styleWhite.Render("Help   "),
		styleDim.Render("<:>") + "  " + styleWhite.Render("Command"),
		styleDim.Render("</>") + "  " + styleWhite.Render("Filter "),
		"",
		"",
	}

	// Combine left, hints, and branding into header lines.
	headerLines := make([]string, 5)
	for i := 0; i < 5; i++ {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}

		hint := ""
		if i < len(hintLines) {
			hint = hintLines[i]
		}

		brand := ""
		if i < len(brandLines) {
			brand = styleOrange.Render(brandLines[i])
		}

		// Calculate padding to right-align hints and branding.
		leftWidth := lipgloss.Width(left)
		hintWidth := lipgloss.Width(hint)
		brandWidth := lipgloss.Width(brand)
		rightContent := hint + "  " + brand
		rightWidth := hintWidth + 2 + brandWidth

		gap := m.width - leftWidth - rightWidth
		if gap < 1 {
			gap = 1
		}

		headerLines[i] = left + strings.Repeat(" ", gap) + rightContent
	}

	return strings.Join(headerLines, "\n")
}

// viewCommandBar renders the command or filter input bar with completion hints.
func (m *AppModel) viewCommandBar() string {
	if m.commandMode {
		bar := "  " + m.commandInput.View()
		if m.commandHint != "" {
			bar += "\n  " + styleDim.Render(m.commandHint)
		}
		return bar
	}
	if m.filterMode {
		return "  " + m.filterInput.View()
	}
	return ""
}

// viewResourceTable renders the current resource table with its title bar.
func (m *AppModel) viewResourceTable() string {
	return m.projectTable.View()
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
