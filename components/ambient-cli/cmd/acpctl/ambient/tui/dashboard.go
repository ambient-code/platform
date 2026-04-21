// Package tui implements the terminal user interface for the ambient dashboard.
package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sdkClientIface interface {
	Sessions() *sdkclient.SessionAPI
}

var (
	styleSelected       = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("255"))
	styleSelectedGutter = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("▌")
)

func (m *Model) rebuildMain() {
	mainW := m.width - navW - 2
	if mainW < 20 {
		mainW = 80
	}
	contentH := m.mainContentH()

	if m.agentEditMode {
		m.mainLines = m.renderAgentEditLines(mainW)
		m.mainScroll = 0
		return
	}

	switch m.nav {
	case NavDashboard:
		m.mainLines = m.buildDashboardLines()
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
	}

	if m.panelFocus && !m.detailMode {
		m.applyPanelCursor()
	}

	m.mainScroll = 0
}

func navHeaderRows(nav NavSection) int {
	switch nav {
	case NavDashboard:
		return 4
	case NavCluster, NavNamespaces, NavProjects, NavAgents, NavSessions:
		return 4
	default:
		return 2
	}
}

func applyRowCursor(lines []string, row, mainW int) {
	if row < 0 || row >= len(lines) {
		return
	}
	line := lines[row]
	if strings.HasPrefix(line, "  ") {
		line = styleSelectedGutter + line[2:]
	} else {
		line = styleSelectedGutter + line
	}
	vis := lipgloss.Width(line)
	if vis < mainW {
		line = line + strings.Repeat(" ", mainW-vis)
	}
	lines[row] = styleSelected.Render(line)
}

func (m *Model) applyPanelCursor() {
	row := m.panelRow + navHeaderRows(m.nav)
	applyRowCursor(m.mainLines, row, m.width-navW-2)
}

func (m *Model) applyDetailCursor(headerLines int) {
	row := m.detailRow + headerLines
	if row < 0 || row >= len(m.detailLines) {
		return
	}
	rebuilt := make([]string, len(m.detailLines))
	copy(rebuilt, m.detailLines)
	applyRowCursor(rebuilt, row, m.width-navW-2)
	m.detailLines = rebuilt
}

func col(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w-1]) + " "
	}
	return s + strings.Repeat(" ", w-len(r))
}

func (m *Model) buildDashboardLines() []string {
	lines := []string{
		styleBold.Render("  System Controls"),
		"",
		styleBold.Render("  ── Port Forwards ───────────────────────"),
		"",
	}

	for i, pf := range m.portForwards {
		var statusIcon, statusLabel string
		if pf.Running {
			statusIcon = styleGreen.Render("●")
			statusLabel = styleGreen.Render("running")
		} else {
			statusIcon = styleRed.Render("○")
			statusLabel = styleDim.Render("stopped")
		}
		pidStr := ""
		if pf.Running && pf.PID > 0 {
			pidStr = styleDim.Render(fmt.Sprintf("  pid %d", pf.PID))
		}
		toggle := styleDim.Render("[Enter/Space to toggle]")
		if m.panelFocus && m.panelRow == i {
			if pf.Running {
				toggle = styleRed.Render("[Enter/Space: stop]")
			} else {
				toggle = styleGreen.Render("[Enter/Space: start]")
			}
		}
		line := "  " + statusIcon + "  " +
			styleBlue.Render(col(pf.Label, 12)) +
			styleDim.Render(fmt.Sprintf("localhost:%-6d → %s:%d", pf.LocalPort, pf.SvcName, pf.SvcPort)) +
			"  " + statusLabel + pidStr + "  " + toggle
		lines = append(lines, line)
	}

	lines = append(lines, "",
		styleBold.Render("  ── Login ───────────────────────────────"),
		"",
	)

	loginRow := len(m.portForwards)
	if m.loginStatus.LoggedIn {
		loginLine := "  " + styleGreen.Render("●") + "  " +
			styleGreen.Render(col("Logged in", 12)) +
			styleDim.Render("user: "+m.loginStatus.User+"  ctx: "+m.loginStatus.Server+"  ns: "+m.loginStatus.Namespace)
		lines = append(lines, loginLine)
	} else {
		toggle := styleDim.Render("[Enter/Space: refresh login status]")
		if m.panelFocus && m.panelRow == loginRow {
			toggle = styleBlue.Render("[Enter/Space: check login]")
		}
		loginLine := "  " + styleRed.Render("○") + "  " +
			styleDim.Render(col("Not logged in", 14)) + "  " + toggle
		lines = append(lines, loginLine)
	}

	lines = append(lines, "",
		styleBold.Render("  ── Platform Stats ──────────────────────"),
		"",
	)
	lines = append(lines, buildStatsLines(m.data)[1:]...)

	return lines
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
		line := styleBlue.Render(col(p.Name, 42)) +
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
			highlight = styleBlue
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
		display := p.Name
		statusStyle := styleGreen
		if p.Status != "" && p.Status != "active" {
			statusStyle = styleDim
		}
		line := styleBlue.Render(col(p.Name, 32)) +
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

	nameW := 38
	header := []string{
		"",
		styleBold.Render(col("  PROJECT/SESSION", nameW)) + styleBold.Render("MESSAGE"),
		styleDim.Render(strings.Repeat("─", w-2)),
	}

	msgColW := w - nameW - 4
	if msgColW < 10 {
		msgColW = 10
	}

	for _, sess := range sessions {
		name := styleBlue.Render(sess.ProjectID) + styleDim.Render("/") + styleWhite.Render(sess.Name)
		var msgCol string
		if m.composeMode && m.composeSessionID == sess.ID {
			msgCol = styleOrange.Render("▶ ") + m.composeInput.render()
			if m.composeStatus != "" {
				msgCol += "  " + m.composeStatus
			}
		} else {
			msgs := m.sessionMsgs[sess.ID]
			last := lastMessageSnippet(msgs, msgColW)
			msgCol = styleDim.Render(last)
		}
		row := "  " + padStyled(name, nameW-2) + msgCol
		header = append(header, row)
	}
	header = append(header, "", styleDim.Render("  ▼ live messages"), "")

	n := len(sessions)
	tableRows := len(header)
	remaining := totalH - tableRows
	if remaining < 0 {
		remaining = 0
	}
	tileH := remaining / n
	const minTileH = 8
	const maxTileH = 16
	if tileH < minTileH {
		tileH = minTileH
	}
	if tileH > maxTileH {
		tileH = maxTileH
	}
	msgLines := tileH - 4

	if m.sessionTileContent == nil {
		m.sessionTileContent = make(map[string][2]int)
	}
	for _, sess := range sessions {
		tile := m.renderSessionTile(sess, w, msgLines)
		tileStart := len(header)
		header = append(header, tile...)
		header = append(header, "")
		contentStart := tileStart + 3
		contentEnd := tileStart + len(tile) - 1
		m.sessionTileContent[sess.ID] = [2]int{contentStart, contentEnd}
	}
	return header
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
		phaseStyle = styleGreen
	}

	age := ""
	if sess.CreatedAt != nil {
		age = fmtAge(time.Since(*sess.CreatedAt))
	}

	idShort := sess.ID
	if len(idShort) > 20 {
		idShort = idShort[:20] + "…"
	}

	titleParts := styleBlue.Render(sess.ProjectID) +
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
		ts := styleDim.Render("--:--:--")
		if msg.CreatedAt != nil {
			ts = styleDim.Render(msg.CreatedAt.Format("15:04:05"))
		}
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

func lastMessageSnippet(msgs []sdktypes.SessionMessage, maxW int) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		d := tileDisplayPayload(msgs[i])
		if d != "" {
			return truncate(d, maxW)
		}
	}
	return ""
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
		return styleBlue
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
		styleBold.Render(col("NAME", 24) + col("OWNER", 27) + col("VERSION", 10) + "PROMPT"),
		styleDim.Render(strings.Repeat("─", 100)),
	}
	if len(d.Agents) == 0 {
		lines = append(lines, styleDim.Render("  no agents"))
		return lines
	}
	for _, a := range d.Agents {
		prompt := a.Prompt
		if len(prompt) > 30 {
			prompt = prompt[:27] + "…"
		}
		line := styleBlue.Render(col(a.Name, 24)) +
			col(a.OwnerUserID, 27) +
			styleDim.Render(prompt)
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
		fmt.Sprintf("  Pods (ambient-code):  %s", styleBlue.Render(fmt.Sprintf("%d", len(d.Pods)))),
		fmt.Sprintf("  Fleet namespaces:     %s", styleBlue.Render(fmt.Sprintf("%d", fleetNS))),
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
		fmt.Sprintf("  Projects:  %s", styleBlue.Render(fmt.Sprintf("%d", len(d.Projects)))),
		fmt.Sprintf("  Sessions:  %s", styleBlue.Render(fmt.Sprintf("%d", len(d.Sessions)))),
		fmt.Sprintf("  Agents:    %s", styleBlue.Render(fmt.Sprintf("%d", len(d.Agents)))),
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
				phaseStyle = styleGreen
			}
			lines = append(lines, fmt.Sprintf("  %-20s %s", phase, phaseStyle.Render(fmt.Sprintf("%d", count))))
		}
	}

	if d.Err != "" {
		lines = append(lines, "", styleRed.Render("  ⚠ fetch errors: "+d.Err))
	}

	return lines
}

func fetchPodLogs(namespace, podName string) []string {
	out, err := exec.Command("kubectl", "logs", "--namespace", namespace, podName, "--tail=200", "--all-containers=true").Output()
	if err != nil {
		out2, _ := exec.Command("kubectl", "logs", "--namespace", namespace, podName, "--tail=200").Output()
		if len(out2) == 0 {
			return []string{styleRed.Render("error fetching logs: " + err.Error())}
		}
		out = out2
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	return append([]string(nil), lines...)
}

func fetchNamespacePodsDetail(namespace string) detailReadyMsg {
	out, err := exec.Command("kubectl", "get", "pods", "--namespace", namespace,
		"--no-headers", "-o", "custom-columns=NAME:.metadata.name,READY:.status.containerStatuses[*].ready,STATUS:.status.phase,RESTARTS:.status.containerStatuses[*].restartCount,AGE:.metadata.creationTimestamp").Output()
	title := "Pods in namespace: " + namespace
	header := styleBold.Render(col("NAME", 52) + col("READY", 8) + col("STATUS", 14) + col("RESTARTS", 10) + "AGE")
	const headerLines = 2
	lines := []string{header, styleDim.Render(strings.Repeat("─", 110))}
	var items []detailItem
	if err != nil {
		lines = append(lines, styleRed.Render("  error: "+err.Error()))
		return detailReadyMsg{title: title, lines: lines}
	}
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if l == "" {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) > 0 {
			items = append(items, detailItem{namespace: namespace, name: fields[0]})
		}
		lines = append(lines, "  "+l)
	}
	if len(items) == 0 {
		lines = append(lines, styleDim.Render("  no pods"))
		return detailReadyMsg{title: title, lines: lines}
	}
	return detailReadyMsg{
		title:       title,
		lines:       lines,
		selectable:  true,
		items:       items,
		headerLines: headerLines,
	}
}

func resolveSessionPod(sess sdktypes.Session) (namespace, podName string) {
	ns := sess.KubeNamespace
	if ns == "" {
		ns = sess.ProjectID
	}
	if ns == "" {
		return "", ""
	}

	if sess.KubeCrName != "" {
		candidate := sess.KubeCrName + "-runner"
		out, err := exec.Command("kubectl", "get", "pod", candidate, "--namespace", ns, "--no-headers", "-o", "name").Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return ns, candidate
		}
		out, err = exec.Command("kubectl", "get", "pod", sess.KubeCrName, "--namespace", ns, "--no-headers", "-o", "name").Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return ns, sess.KubeCrName
		}
	}

	if sess.ID != "" {
		labelSel := "ambient-code.io/session-id=" + sess.ID
		out, err := exec.Command("kubectl", "get", "pods", "--namespace", ns, "-l", labelSel, "--no-headers", "-o", "custom-columns=NAME:.metadata.name").Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					return ns, line
				}
			}
		}
	}

	return "", ""
}

func fetchSessionSplitDetail(sess sdktypes.Session, msgs []sdktypes.SessionMessage) splitDetailReadyMsg {
	title := "Session: " + sess.ProjectID + "/" + sess.Name

	phase := sess.Phase
	if phase == "" {
		phase = "unknown"
	}
	age := "—"
	if sess.CreatedAt != nil {
		age = fmtAge(time.Since(*sess.CreatedAt))
	}

	topLines := []string{
		styleBold.Render("  Session Detail"),
		"",
		"  " + col("ID:", 16) + styleBlue.Render(sess.ID),
		"  " + col("Name:", 16) + styleWhite.Render(sess.Name),
		"  " + col("Project:", 16) + sess.ProjectID,
		"  " + col("Phase:", 16) + phase,
		"  " + col("Age:", 16) + age,
		"  " + col("Model:", 16) + sess.LlmModel,
		"",
		styleBold.Render("  ── Messages ────────────────────────────────"),
		"",
	}
	if len(msgs) == 0 {
		topLines = append(topLines, styleDim.Render("  no messages"))
	}
	for _, msg := range msgs {
		display := tileDisplayPayload(msg)
		if display == "" {
			continue
		}
		ts := styleDim.Render("--:--:--")
		if msg.CreatedAt != nil {
			ts = styleDim.Render(msg.CreatedAt.Format("15:04:05"))
		}
		evStyle := eventTypeStyle(msg.EventType)
		seqStr := styleDim.Render(fmt.Sprintf("#%-4d", msg.Seq))
		topLines = append(topLines, "  "+seqStr+"  "+ts+"  "+evStyle.Render(col(msg.EventType, 22))+"  "+display)
	}

	var bottomLines []string
	podNS, podName := resolveSessionPod(sess)
	if podNS != "" && podName != "" {
		bottomLines = append(bottomLines,
			styleBold.Render("  ── Pod Logs: "+podNS+"/"+podName+" ────────────────"),
			"",
		)
		bottomLines = append(bottomLines, fetchPodLogs(podNS, podName)...)
	} else {
		bottomLines = []string{
			styleBold.Render("  ── Pod Logs ────────────────────────────────"),
			"",
			styleDim.Render("  no pod info available"),
		}
	}

	return splitDetailReadyMsg{
		title:       title,
		topLines:    topLines,
		bottomLines: bottomLines,
	}
}

func fetchProjectSessionsDetail(ctx context.Context, client sdkClientIface, proj sdktypes.Project) detailReadyMsg {
	title := "Project: " + proj.Name
	const headerLines = 2
	lines := []string{
		styleBold.Render(col("NAME", 30) + col("PHASE", 14) + col("MODEL", 22) + "AGE"),
		styleDim.Render(strings.Repeat("─", 100)),
	}
	sessionList, err := client.Sessions().List(ctx, nil)
	if err != nil {
		return detailReadyMsg{title: title, lines: append(lines, styleRed.Render("  error: "+err.Error()))}
	}
	var items []detailItem
	for _, sess := range sessionList.Items {
		if sess.ProjectID != proj.ID {
			continue
		}
		phase := sess.Phase
		if phase == "" {
			phase = "unknown"
		}
		phaseStyle := styleDim
		switch phase {
		case "Running", "running":
			phaseStyle = styleGreen
		case "Pending", "Creating":
			phaseStyle = styleYellow
		case "Failed", "Error":
			phaseStyle = styleRed
		case "Completed":
			phaseStyle = styleGreen
		}
		age := "—"
		if sess.CreatedAt != nil {
			age = fmtAge(time.Since(*sess.CreatedAt))
		}
		line := styleBlue.Render(col(sess.Name, 30)) +
			phaseStyle.Render(col(phase, 14)) +
			styleDim.Render(col(sess.LlmModel, 22)) +
			styleDim.Render(age)
		lines = append(lines, "  "+line)
		items = append(items, detailItem{kind: "session", id: sess.ID, name: sess.Name, namespace: proj.ID})
	}
	if len(items) == 0 {
		lines = append(lines, styleDim.Render("  no sessions"))
		return detailReadyMsg{title: title, lines: lines}
	}
	return detailReadyMsg{
		title:       title,
		lines:       lines,
		selectable:  true,
		items:       items,
		headerLines: headerLines,
	}
}

func renderAgentDetail(agent sdktypes.Agent) []string {
	return []string{
		styleBold.Render("  Agent Detail"),
		"",
		"  " + col("ID:", 20) + styleBlue.Render(agent.ID),
		"  " + col("Name:", 20) + styleWhite.Render(agent.Name),
		"  " + col("Owner:", 20) + agent.OwnerUserID,
		"",
		styleBold.Render("  ── Prompt ──────────────────────────────────"),
		"",
		"  " + styleDim.Render(agent.Prompt),
	}
}

func (m *Model) renderAgentEditLines(mainW int) []string {
	agent := m.agentEditAgent
	runes := []rune(m.agentEditPrompt)
	cur := m.agentEditCursor
	if cur > len(runes) {
		cur = len(runes)
	}

	before := string(runes[:cur])
	cursorCh := "█"
	after := ""
	if cur < len(runes) {
		cursorCh = string(runes[cur : cur+1])
		after = string(runes[cur+1:])
	}

	dirtyMark := ""
	if m.agentEditDirty {
		dirtyMark = " " + styleOrange.Render("●")
	}

	lines := []string{
		styleBold.Render("  ✎ Edit Agent") + dirtyMark,
		"",
		"  " + col("ID:", 20) + styleBlue.Render(agent.ID),
		"  " + col("Name:", 20) + styleWhite.Render(agent.Name),
		"  " + col("Owner:", 20) + agent.OwnerUserID,
		"",
		styleOrange.Render("  ── Prompt (editing) ────────────────────────"),
		"",
	}

	promptW := mainW - 4
	if promptW < 20 {
		promptW = 20
	}
	fullPrompt := before + styleBold.Render(cursorCh) + after
	var promptLines []string
	remaining := fullPrompt
	for len([]rune(remaining)) > 0 {
		if lipgloss.Width(remaining) <= promptW {
			promptLines = append(promptLines, "  "+remaining)
			break
		}
		promptLines = append(promptLines, "  "+string([]rune(remaining)[:promptW]))
		remaining = string([]rune(remaining)[promptW:])
	}
	if len(promptLines) == 0 {
		promptLines = []string{"  " + styleBold.Render("█")}
	}
	lines = append(lines, promptLines...)
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
