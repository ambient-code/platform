package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorOrange = lipgloss.Color("214")
	colorCyan   = lipgloss.Color("36")
	colorGreen  = lipgloss.Color("32")
	colorRed    = lipgloss.Color("31")
	colorYellow = lipgloss.Color("33")
	colorDim    = lipgloss.Color("240")
	colorWhite  = lipgloss.Color("255")
	colorBlue   = lipgloss.Color("69")

	styleOrange = lipgloss.NewStyle().Foreground(colorOrange)
	styleCyan   = lipgloss.NewStyle().Foreground(colorCyan)
	styleGreen  = lipgloss.NewStyle().Foreground(colorGreen)
	styleRed    = lipgloss.NewStyle().Foreground(colorRed)
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleYellow = lipgloss.NewStyle().Foreground(colorYellow)
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleWhite  = lipgloss.NewStyle().Foreground(colorWhite)
	styleBlue   = lipgloss.NewStyle().Foreground(colorBlue)
)

const navW = 22

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	header := m.renderHeader()
	bar := styleDim.Render(strings.Repeat("─", m.width))

	sidebar := m.renderSidebar()
	main := m.renderMain()

	body := joinSidebarMain(sidebar, main, m.width)

	footer := m.renderFooter()

	return header + "\n" + bar + "\n" + body + "\n" + bar + "\n" + footer
}

func (m *Model) renderHeader() string {
	age := ""
	if !m.lastFetch.IsZero() {
		age = styleDim.Render("  refreshed " + fmtAge(time.Since(m.lastFetch)) + " ago")
	}
	spin := ""
	if m.refreshing {
		spin = styleYellow.Render("  ⟳")
	}
	title := styleBold.Render("Ambient") +
		styleCyan.Render(" Dashboard") +
		age + spin +
		styleDim.Render("  ↑↓/jk nav  ·  Tab cmd  ·  r refresh  ·  q quit")
	return " " + title
}

func (m *Model) renderFooter() string {
	if m.cmdFocus {
		return styleBlue.Render("▶") + " " + m.input.render()
	}
	if m.cmdRunning {
		return styleYellow.Render("⏳") + " " + styleDim.Render("running…  (Tab to focus cmd bar)")
	}
	hint := styleDim.Render("Tab to focus command bar  ·  Esc to unfocus")
	return "  " + hint
}

func (m *Model) renderSidebar() []string {
	lines := make([]string, 0, len(navLabels)+4)
	lines = append(lines, styleBlue.Render("┌"+strings.Repeat("─", navW-2)+"┐"))
	lines = append(lines, styleBlue.Render("│")+styleBold.Render(" Navigation")+styleBlue.Render(padTo(" ", navW-13))+"│")
	lines = append(lines, styleBlue.Render("├"+strings.Repeat("─", navW-2)+"┤"))
	for i, label := range navLabels {
		nav := NavSection(i)
		count := m.navCount(nav)
		countStr := ""
		if count >= 0 {
			countStr = styleDim.Render(fmt.Sprintf(" (%d)", count))
		}
		if m.nav == nav {
			text := styleGreen.Render("▶ "+label) + countStr
			lines = append(lines, styleBlue.Render("│")+" "+padStyled(text, navW-3)+styleBlue.Render("│"))
		} else {
			text := styleDim.Render("  "+label) + countStr
			lines = append(lines, styleBlue.Render("│")+" "+padStyled(text, navW-3)+styleBlue.Render("│"))
		}
	}
	lines = append(lines, styleBlue.Render("└"+strings.Repeat("─", navW-2)+"┘"))
	return lines
}

func (m *Model) navCount(nav NavSection) int {
	switch nav {
	case NavCluster:
		return len(m.data.Pods)
	case NavNamespaces:
		return len(m.data.Namespaces)
	case NavProjects:
		return len(m.data.Projects)
	case NavSessions:
		return len(m.data.Sessions)
	case NavAgents:
		return len(m.data.Agents)
	}
	return -1
}

func (m *Model) renderMain() []string {
	mainW := m.width - navW - 1
	contentH := m.mainContentH()

	visible := m.mainLines
	if m.mainScroll < len(visible) {
		visible = visible[m.mainScroll:]
	}
	if len(visible) > contentH {
		visible = visible[:contentH]
	}

	lines := make([]string, contentH)
	for i, l := range visible {
		lines[i] = truncateLine(l, mainW)
	}

	scrollInfo := ""
	if len(m.mainLines) > contentH {
		pct := 0
		if len(m.mainLines) > 0 {
			pct = (m.mainScroll + contentH) * 100 / len(m.mainLines)
			if pct > 100 {
				pct = 100
			}
		}
		scrollInfo = styleDim.Render(fmt.Sprintf("  %d%%  PgUp/PgDn to scroll", pct))
	}
	_ = scrollInfo

	return lines
}

func joinSidebarMain(sidebar, main []string, totalW int) string {
	mainW := totalW - navW - 1
	h := len(main)
	if len(sidebar) > h {
		h = len(sidebar)
	}

	var sb strings.Builder
	for i := 0; i < h; i++ {
		sLine := ""
		if i < len(sidebar) {
			sLine = sidebar[i]
		}
		mLine := ""
		if i < len(main) {
			mLine = main[i]
		}
		sLine = padTo(sLine, navW)
		_ = mainW
		sb.WriteString(sLine)
		sb.WriteString(styleDim.Render("│"))
		sb.WriteString(" ")
		sb.WriteString(mLine)
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func padTo(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

func padStyled(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

func truncateLine(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	if len(runes) > w-1 {
		return string(runes[:w-1]) + "…"
	}
	return s
}

func fmtAge(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
