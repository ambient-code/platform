// Package tui implements the terminal user interface for the ambient dashboard.
package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) rebuildMain() {
	mainW := m.width - navW - 2
	if mainW < 20 {
		mainW = 80
	}
	contentH := m.mainContentH()

	switch m.nav {
	case NavCluster:
		m.mainLines = buildClusterLines(m.data)
	case NavNamespaces:
		m.mainLines = buildNamespaceLines(m.data)
	case NavProjects:
		m.mainLines = buildProjectLines(m.data)
	case NavSessions:
		m.mainLines = m.buildSessionTiles(mainW, contentH)
	case NavAgents:
		m.mainLines = buildAgentLines(m.data)
	case NavStats:
		m.mainLines = buildStatsLines(m.data)
	}
	m.mainScroll = 0
}

func col(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w-1]) + " "
	}
	return s + strings.Repeat(" ", w-len(r))
}

func buildClusterLines(d DashData) []string {
	lines := []string{
		styleBold.Render("  Cluster Pods") + styleDim.Render("  namespace: ambient-code"),
		"",
		styleBold.Render(col("NAME", 42) + col("READY", 8) + col("STATUS", 14) + col("RESTARTS", 10) + "AGE"),
		styleDim.Render(strings.Repeat("─", 90)),
	}
	if len(d.Pods) == 0 {
		if d.Err != "" {
			lines = append(lines, styleRed.Render("  error: "+d.Err))
		} else {
			lines = append(lines, styleDim.Render("  no pods found in ambient-code namespace"))
		}
		return lines
	}
	for _, p := range d.Pods {
		statusStyle := styleGreen
		switch {
		case strings.Contains(p.Status, "Error"), strings.Contains(p.Status, "Crash"), strings.Contains(p.Status, "OOM"):
			statusStyle = styleRed
		case p.Status == "Pending", strings.Contains(p.Status, "Init"):
			statusStyle = styleYellow
		case p.Status == "Terminating":
			statusStyle = styleOrange
		}
		line := styleCyan.Render(col(p.Name, 42)) +
			col(p.Ready, 8) +
			statusStyle.Render(col(p.Status, 14)) +
			styleDim.Render(col(p.Restarts, 10)) +
			styleDim.Render(p.Age)
		lines = append(lines, "  "+line)
	}
	if d.Err != "" {
		lines = append(lines, "", styleRed.Render("  ⚠ "+d.Err))
	}
	return lines
}

func buildNamespaceLines(d DashData) []string {
	lines := []string{
		styleBold.Render("  Namespaces"),
		"",
		styleBold.Render(col("NAME", 42) + col("STATUS", 14) + "AGE"),
		styleDim.Render(strings.Repeat("─", 70)),
	}
	if len(d.Namespaces) == 0 {
		lines = append(lines, styleDim.Render("  no namespaces returned"))
		return lines
	}
	for _, ns := range d.Namespaces {
		highlight := styleWhite
		if strings.HasPrefix(ns.Name, "fleet-") || strings.HasPrefix(ns.Name, "ambient") {
			highlight = styleCyan
		}
		statusStyle := styleGreen
		if ns.Status != "Active" {
			statusStyle = styleYellow
		}
		line := highlight.Render(col(ns.Name, 42)) +
			statusStyle.Render(col(ns.Status, 14)) +
			styleDim.Render(ns.Age)
		lines = append(lines, "  "+line)
	}
	return lines
}

func buildProjectLines(d DashData) []string {
	lines := []string{
		styleBold.Render("  Projects") + styleDim.Render(fmt.Sprintf("  total: %d", len(d.Projects))),
		"",
		styleBold.Render(col("NAME", 32) + col("DISPLAY NAME", 30) + col("STATUS", 12) + "CREATED"),
		styleDim.Render(strings.Repeat("─", 90)),
	}
	if len(d.Projects) == 0 {
		lines = append(lines, styleDim.Render("  no projects"))
		return lines
	}
	for _, p := range d.Projects {
		age := ""
		if p.CreatedAt != nil {
			age = fmtAge(time.Since(*p.CreatedAt))
		}
		display := p.DisplayName
		if display == "" {
			display = p.Name
		}
		statusStyle := styleGreen
		if p.Status != "" && p.Status != "active" {
			statusStyle = styleDim
		}
		line := styleCyan.Render(col(p.Name, 32)) +
			col(display, 30) +
			statusStyle.Render(col(p.Status, 12)) +
			styleDim.Render(age)
		lines = append(lines, "  "+line)
	}
	return lines
}

func (m *Model) buildSessionTiles(w, totalH int) []string {
	sessions := m.data.Sessions
	if len(sessions) == 0 {
		return []string{
			"",
			styleBold.Render("  Sessions") + styleDim.Render("  total: 0"),
			"",
			styleDim.Render("  no sessions"),
		}
	}

	n := len(sessions)
	tileH := totalH / n
	if tileH < 6 {
		tileH = 6
	}
	msgLines := tileH - 4

	var lines []string
	for _, sess := range sessions {
		lines = append(lines, m.renderSessionTile(sess, w, msgLines)...)
		lines = append(lines, "")
	}
	return lines
}

func (m *Model) renderSessionTile(sess sdktypes.Session, w, msgLines int) []string {
	innerW := w - 2
	if innerW < 10 {
		innerW = 10
	}

	phase := sess.Phase
	if phase == "" {
		phase = "unknown"
	}
	phaseStyle := styleDim
	switch phase {
	case "Running", "running":
		phaseStyle = styleGreen
	case "Pending", "pending", "Creating":
		phaseStyle = styleYellow
	case "Failed", "failed", "Error":
		phaseStyle = styleRed
	case "Completed", "completed":
		phaseStyle = styleCyan
	}

	age := ""
	if sess.CreatedAt != nil {
		age = fmtAge(time.Since(*sess.CreatedAt))
	}

	idShort := sess.ID
	if len(idShort) > 20 {
		idShort = idShort[:20] + "…"
	}

	titleParts := styleCyan.Render(sess.ProjectID) +
		styleDim.Render("/") +
		styleWhite.Render(sess.Name) +
		"  " + phaseStyle.Render(phase) +
		styleDim.Render("  "+idShort+"  "+age)

	borderStyle := phaseStyle
	hLine := strings.Repeat("─", innerW)

	header := borderStyle.Render("┌") + borderStyle.Render(hLine) + borderStyle.Render("┐")
	titleRow := borderStyle.Render("│") + " " + padStyled(titleParts, innerW-1) + borderStyle.Render("│")
	sep := borderStyle.Render("├") + hLine + borderStyle.Render("┤")
	footer := borderStyle.Render("└") + hLine + borderStyle.Render("┘")

	msgs := m.sessionMsgs[sess.ID]
	contentLines := renderTileMessages(msgs, innerW-2, msgLines)

	out := []string{header, titleRow, sep}
	for _, l := range contentLines {
		out = append(out, borderStyle.Render("│")+" "+padStyled(l, innerW-1)+borderStyle.Render("│"))
	}
	out = append(out, footer)
	return out
}

func renderTileMessages(msgs []sdktypes.SessionMessage, w, maxLines int) []string {
	var rendered []string
	for _, msg := range msgs {
		display := tileDisplayPayload(msg)
		if display == "" {
			continue
		}
		ts := styleDim.Render(msg.CreatedAt.Format("15:04:05"))
		evStyle := eventTypeStyle(msg.EventType)
		evShort := truncate(msg.EventType, 24)
		line := ts + "  " + evStyle.Render(col(evShort, 26)) + truncate(display, w-42)
		rendered = append(rendered, line)
	}

	if len(rendered) > maxLines {
		rendered = rendered[len(rendered)-maxLines:]
	}

	padded := make([]string, maxLines)
	copy(padded[maxLines-len(rendered):], rendered)
	return padded
}

func tileDisplayPayload(msg sdktypes.SessionMessage) string {
	switch msg.EventType {
	case "user":
		return truncate(msg.Payload, 120)
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT":
		if d := extractKVField(msg.Payload, "delta"); d != "" {
			return d
		}
	case "TOOL_CALL_START":
		if name := extractKVField(msg.Payload, "tool_call_name"); name != "" {
			return "⚙ " + name
		}
		if name := extractKVField(msg.Payload, "tool_name"); name != "" {
			return "⚙ " + name
		}
	case "TOOL_CALL_RESULT":
		if c := extractKVField(msg.Payload, "content"); c != "" {
			return c
		}
	case "RUN_FINISHED":
		return "[done]"
	case "RUN_ERROR":
		if errMsg := extractKVField(msg.Payload, "message"); errMsg != "" {
			return "✗ " + errMsg
		}
	case "TEXT_MESSAGE_START":
		return "…"
	case "TEXT_MESSAGE_END", "TOOL_CALL_ARGS", "TOOL_CALL_END":
		return ""
	}
	if msg.Payload != "" && len(msg.Payload) <= 120 {
		return msg.Payload
	}
	return ""
}

func extractKVField(payload, field string) string {
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}
	needle := field + "='"
	idx := strings.Index(payload, needle)
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	var sb strings.Builder
	for i := start; i < len(payload); i++ {
		if payload[i] == '\'' && (i == start || payload[i-1] != '\\') {
			break
		}
		sb.WriteByte(payload[i])
	}
	return strings.ReplaceAll(sb.String(), `\'`, `'`)
}

func eventTypeStyle(et string) lipgloss.Style {
	switch {
	case strings.HasPrefix(et, "TEXT_MESSAGE"):
		return styleCyan
	case strings.HasPrefix(et, "TOOL_CALL"):
		return styleOrange
	case et == "RUN_FINISHED":
		return styleGreen
	case et == "RUN_ERROR":
		return styleRed
	case et == "user":
		return styleBlue
	default:
		return styleDim
	}
}

func buildAgentLines(d DashData) []string {
	lines := []string{
		styleBold.Render("  Agents") + styleDim.Render(fmt.Sprintf("  total: %d", len(d.Agents))),
		"",
		styleBold.Render(col("NAME", 24) + col("DISPLAY NAME", 24) + col("PROJECT", 22) + col("MODEL", 16) + "SESSION"),
		styleDim.Render(strings.Repeat("─", 100)),
	}
	if len(d.Agents) == 0 {
		lines = append(lines, styleDim.Render("  no agents"))
		return lines
	}
	for _, a := range d.Agents {
		display := a.DisplayName
		if display == "" {
			display = a.Name
		}
		model := a.LlmModel
		if model == "" {
			model = "—"
		}
		sess := styleDim.Render("—")
		if a.CurrentSessionID != "" {
			short := a.CurrentSessionID
			if len(short) > 16 {
				short = short[:16] + "…"
			}
			sess = styleGreen.Render(short)
		}
		line := styleCyan.Render(col(a.Name, 24)) +
			col(display, 24) +
			col(a.ProjectID, 22) +
			styleDim.Render(col(model, 16)) +
			sess
		lines = append(lines, "  "+line)
	}
	return lines
}

func buildStatsLines(d DashData) []string {
	podsByStatus := map[string]int{}
	for _, p := range d.Pods {
		podsByStatus[p.Status]++
	}

	sessionsByPhase := map[string]int{}
	for _, s := range d.Sessions {
		phase := s.Phase
		if phase == "" {
			phase = "unknown"
		}
		sessionsByPhase[phase]++
	}

	fleetNS := 0
	for _, ns := range d.Namespaces {
		if strings.HasPrefix(ns.Name, "fleet-") {
			fleetNS++
		}
	}

	age := "never"
	if !d.FetchedAt.IsZero() {
		age = fmtAge(time.Since(d.FetchedAt)) + " ago"
	}

	lines := []string{
		styleBold.Render("  Ambient Platform Stats"),
		styleDim.Render("  last refresh: " + age),
		"",
		styleBold.Render("  ── Cluster ─────────────────────────────"),
		fmt.Sprintf("  Pods (ambient-code):  %s", styleCyan.Render(fmt.Sprintf("%d", len(d.Pods)))),
		fmt.Sprintf("  Fleet namespaces:     %s", styleCyan.Render(fmt.Sprintf("%d", fleetNS))),
		fmt.Sprintf("  Total namespaces:     %s", styleDim.Render(fmt.Sprintf("%d", len(d.Namespaces)))),
	}

	if len(podsByStatus) > 0 {
		lines = append(lines, "")
		lines = append(lines, styleBold.Render("  ── Pod Status ──────────────────────────"))
		for status, count := range podsByStatus {
			style := styleGreen
			if strings.Contains(status, "Error") || strings.Contains(status, "Crash") {
				style = styleRed
			} else if status == "Pending" {
				style = styleYellow
			}
			lines = append(lines, fmt.Sprintf("  %-20s %s", status, style.Render(fmt.Sprintf("%d", count))))
		}
	}

	lines = append(lines, "",
		styleBold.Render("  ── Platform Objects ────────────────────"),
		fmt.Sprintf("  Projects:  %s", styleCyan.Render(fmt.Sprintf("%d", len(d.Projects)))),
		fmt.Sprintf("  Sessions:  %s", styleCyan.Render(fmt.Sprintf("%d", len(d.Sessions)))),
		fmt.Sprintf("  Agents:    %s", styleCyan.Render(fmt.Sprintf("%d", len(d.Agents)))),
	)

	if len(sessionsByPhase) > 0 {
		lines = append(lines, "")
		lines = append(lines, styleBold.Render("  ── Session Phases ──────────────────────"))
		for phase, count := range sessionsByPhase {
			phaseStyle := styleDim
			switch phase {
			case "Running", "running":
				phaseStyle = styleGreen
			case "Pending", "pending", "Creating":
				phaseStyle = styleYellow
			case "Failed", "failed":
				phaseStyle = styleRed
			case "Completed", "completed":
				phaseStyle = styleCyan
			}
			lines = append(lines, fmt.Sprintf("  %-20s %s", phase, phaseStyle.Render(fmt.Sprintf("%d", count))))
		}
	}

	if d.Err != "" {
		lines = append(lines, "", styleRed.Render("  ⚠ fetch errors: "+d.Err))
	}

	return lines
}

func (m *Model) execCommand(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			m.msgCh <- cmdDoneMsg{}
			return nil
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			m.msgCh <- cmdOutputMsg{text: "pipe error: " + err.Error(), kind: lkRed}
			m.msgCh <- cmdDoneMsg{}
			return nil
		}
		errPipe, err := cmd.StderrPipe()
		if err != nil {
			m.msgCh <- cmdOutputMsg{text: "pipe error: " + err.Error(), kind: lkRed}
			m.msgCh <- cmdDoneMsg{}
			return nil
		}

		if err := cmd.Start(); err != nil {
			m.msgCh <- cmdOutputMsg{text: "exec error: " + err.Error(), kind: lkRed}
			m.msgCh <- cmdDoneMsg{}
			return nil
		}

		var wg sync.WaitGroup
		scanPipe := func(pipe interface{ Read([]byte) (int, error) }, kind lineKind) {
			defer wg.Done()
			scanner := bufio.NewScanner(pipe)
			for scanner.Scan() {
				m.msgCh <- cmdOutputMsg{text: scanner.Text(), kind: kind}
			}
		}

		wg.Add(2)
		go scanPipe(outPipe, lkNormal)
		go scanPipe(errPipe, lkRed)
		wg.Wait()

		if err := cmd.Wait(); err != nil {
			m.msgCh <- cmdOutputMsg{text: "exit: " + err.Error(), kind: lkRed}
		}
		m.msgCh <- cmdDoneMsg{}
		return nil
	}
}
