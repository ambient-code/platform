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
	colorGreen  = lipgloss.Color("28")
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
	title := styleOrange.Render(styleBold.Render("Ambient")) +
		styleBlue.Render(" Dashboard") +
		age + spin
	return " " + title
}

func (m *Model) renderBreadcrumb() string {
	sep := styleDim.Render(" › ")
	crumb := styleOrange.Render(navLabels[m.nav])
	if m.panelFocus && !m.detailMode {
		enterHint := "Enter/→ open"
		extra := ""
		if m.nav == NavSessions {
			enterHint = "Enter/→ send message"
		}
		if m.nav == NavAgents {
			enterHint = "Enter/→ edit"
			extra = "  ·  D delete"
		}
		return crumb + sep + styleOrange.Render("panel") + styleDim.Render("  ↑↓/jk navigate  ·  "+enterHint+extra+"  ·  Esc/← back")
	}
	if m.detailMode && m.detailSelectable {
		return crumb + sep + styleDim.Render("panel") + sep + styleOrange.Render(m.detailTitle) + styleDim.Render("  ↑↓/jk navigate  ·  Enter/→ open  ·  Esc/← back")
	}
	if m.detailMode && m.detailSplit {
		panel := "messages"
		if m.detailSplitFocus == 1 {
			panel = "pod logs"
		}
		return crumb + sep + styleDim.Render("panel") + sep + styleOrange.Render(m.detailTitle) + styleDim.Render("  ["+panel+"]  ↑↓/jk scroll  ·  ←/→ switch  ·  Esc back")
	}
	if m.detailMode {
		return crumb + sep + styleDim.Render("panel") + sep + styleOrange.Render(m.detailTitle) + styleDim.Render("  ↑↓/jk scroll  ·  Esc/← back")
	}
	return styleOrange.Render(navLabels[m.nav]) + styleDim.Render("  ↑↓/jk nav  ·  Enter/→ panel  ·  Tab cmd  ·  r refresh  ·  q quit")
}

func (m *Model) renderFooter() string {
	if m.agentConfirmDelete {
		return styleRed.Render("⚠") + " " + styleBold.Render("Delete agent "+m.agentDeleteName+"?") + styleDim.Render("  y yes  ·  n/Esc cancel")
	}
	if m.agentEditMode {
		dirty := ""
		if m.agentEditDirty {
			dirty = styleOrange.Render("  ●")
		}
		status := ""
		if m.agentEditStatus != "" {
			status = "  " + m.agentEditStatus
		}
		return styleOrange.Render("✎") + " " + styleBold.Render("Editing: "+m.agentEditAgent.Name) + dirty + styleDim.Render("  Enter save  ·  Esc cancel") + status
	}
	if m.composeMode {
		return styleOrange.Render("▶") + " " + styleDim.Render("Enter send  ·  Esc cancel")
	}
	if m.cmdFocus {
		return styleBlue.Render("▶") + " " + m.input.render()
	}
	if m.cmdRunning {
		return styleYellow.Render("⏳") + " " + styleDim.Render("running…  (Tab to focus cmd bar)")
	}
	return " " + m.renderBreadcrumb()
}

func (m *Model) renderSidebar() []string {
	lines := make([]string, 0, len(navLabels)+4)
	lines = append(lines, styleOrange.Render("┌"+strings.Repeat("─", navW-2)+"┐"))
	lines = append(lines, styleOrange.Render("│")+styleBold.Render(" Navigation")+styleOrange.Render(padTo(" ", navW-13))+"│")
	lines = append(lines, styleOrange.Render("├"+strings.Repeat("─", navW-2)+"┤"))
	for i, label := range navLabels {
		nav := NavSection(i)
		count := m.navCount(nav)
		countStr := ""
		if count >= 0 {
			countStr = styleDim.Render(fmt.Sprintf(" (%d)", count))
		}
		if m.nav == nav {
			text := styleOrange.Render("▶ "+label) + countStr
			lines = append(lines, styleOrange.Render("│")+" "+padStyled(text, navW-3)+styleOrange.Render("│"))
		} else {
			text := styleDim.Render("  "+label) + countStr
			lines = append(lines, styleOrange.Render("│")+" "+padStyled(text, navW-3)+styleOrange.Render("│"))
		}
	}
	lines = append(lines, styleOrange.Render("└"+strings.Repeat("─", navW-2)+"┘"))
	return lines
}

func (m *Model) navCount(nav NavSection) int {
	switch nav {
	case NavDashboard:
		return -1
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

	if m.detailMode && m.detailSplit {
		halfH := contentH / 2
		if halfH < 2 {
			halfH = 2
		}
		bottomH := contentH - halfH - 1

		renderPanel := func(src []string, scroll, h int, focused bool) []string {
			visible := src
			if scroll < len(visible) {
				visible = visible[scroll:]
			}
			if len(visible) > h {
				visible = visible[:h]
			}
			out := make([]string, h)
			for i, l := range visible {
				out[i] = truncateLine(l, mainW)
			}
			return out
		}

		topFocused := m.detailSplitFocus == 0
		botFocused := m.detailSplitFocus == 1

		sepChar := "─"
		sepStyle := styleDim
		if topFocused {
			sepStyle = styleOrange
		}
		topIndicator := styleDim.Render(" ↑↓/jk  Tab switch")
		if topFocused {
			topIndicator = styleOrange.Render(" ↑↓/jk") + styleDim.Render("  Tab switch")
		}

		botSepStyle := styleDim
		if botFocused {
			botSepStyle = styleOrange
		}
		botIndicator := styleDim.Render(" ↑↓/jk")
		if botFocused {
			botIndicator = styleOrange.Render(" ↑↓/jk") + styleDim.Render("  Tab switch")
		}
		_ = botIndicator

		sep := sepStyle.Render(strings.Repeat(sepChar, mainW/2)) + topIndicator

		var lines []string
		lines = append(lines, renderPanel(m.detailTopLines, m.detailTopScroll, halfH, topFocused)...)
		lines = append(lines, sep)
		botLines := renderPanel(m.detailBottomLines, m.detailBottomScroll, bottomH, botFocused)
		if botFocused && len(botLines) > 0 {
			botLines[0] = botSepStyle.Render(botLines[0])
		}
		lines = append(lines, botLines...)
		return lines
	}

	var source []string
	var scroll int
	if m.detailMode {
		source = m.detailLines
		scroll = m.detailScroll
	} else {
		source = m.mainLines
		scroll = m.mainScroll
	}

	visible := source
	if scroll < len(visible) {
		visible = visible[scroll:]
	}
	if len(visible) > contentH {
		visible = visible[:contentH]
	}

	lines := make([]string, contentH)
	for i, l := range visible {
		lines[i] = truncateLine(l, mainW)
	}
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
