package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Dialog message types
// ---------------------------------------------------------------------------

// DialogConfirmMsg is emitted when the user presses Enter on a dialog button.
type DialogConfirmMsg struct {
	Confirmed bool   // true if OK/Delete was selected
	Value     string // text input value (for input dialogs)
}

// DialogCancelMsg is emitted when the user presses Esc to dismiss a dialog.
type DialogCancelMsg struct{}

// ---------------------------------------------------------------------------
// Dialog colors (local to views package to avoid circular import)
// ---------------------------------------------------------------------------

var (
	dlgColorDim    = lipgloss.Color("240")
	dlgColorOrange = lipgloss.Color("214")
	dlgColorWhite  = lipgloss.Color("255")
	dlgColorBlack  = lipgloss.Color("0")
)

// ---------------------------------------------------------------------------
// Dialog
// ---------------------------------------------------------------------------

// Dialog represents a centered overlay dialog box that can display a
// confirmation prompt or collect text input, styled after the k9s delete
// confirmation pattern.
type Dialog struct {
	Title    string   // e.g. "Delete", "Confirm", "New Agent"
	Message  string   // e.g. "Delete agent test-agent?"
	Buttons  []string // e.g. ["Cancel", "OK"] or ["Cancel", "Delete"]
	Selected int      // which button is highlighted (0=Cancel, 1=OK)
	Input    *textinput.Model
	Width    int // dialog width (auto-calculated from content if 0)
}

// NewConfirmDialog creates a two-button dialog with Cancel and OK.
func NewConfirmDialog(title, message string) Dialog {
	return Dialog{
		Title:    title,
		Message:  message,
		Buttons:  []string{"Cancel", "OK"},
		Selected: 1, // default to OK
	}
}

// NewDeleteDialog creates a delete confirmation dialog with Cancel and Delete
// buttons. The message is formatted as "Delete <kind> <name>?".
func NewDeleteDialog(kind, name string) Dialog {
	return Dialog{
		Title:    "Delete",
		Message:  fmt.Sprintf("Delete %s %s?", kind, name),
		Buttons:  []string{"Cancel", "Delete"},
		Selected: 0, // default to Cancel for safety
	}
}

// NewErrorDialog creates a single-button dialog with ASCII art and an error message.
func NewErrorDialog(title, message, ascii string) Dialog {
	return Dialog{
		Title:    title,
		Message:  ascii + "\n" + message,
		Buttons:  []string{"Dismiss"},
		Selected: 0,
		Width:    50,
	}
}

// NewInputDialog creates a dialog with a text input field and Cancel/OK buttons.
func NewInputDialog(title, prompt string) Dialog {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.CharLimit = 1024
	ti.Focus()
	return Dialog{
		Title:    title,
		Message:  "",
		Buttons:  []string{"Cancel", "OK"},
		Selected: 1,
		Input:    &ti,
	}
}

// Confirmed returns true if the currently selected button is not the first
// (Cancel) button — i.e. OK or Delete is selected.
func (d Dialog) Confirmed() bool {
	return d.Selected > 0
}

// InputValue returns the text input value, or empty string if there is no input.
func (d Dialog) InputValue() string {
	if d.Input != nil {
		return d.Input.Value()
	}
	return ""
}

// Update handles key events for the dialog. Left/Right/Tab switch the selected
// button, Enter confirms, Esc cancels. For input dialogs, typing is delegated
// to the embedded textinput.
func (d *Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return *d, func() tea.Msg { return DialogCancelMsg{} }

		case tea.KeyEnter:
			return *d, func() tea.Msg {
				return DialogConfirmMsg{
					Confirmed: d.Confirmed(),
					Value:     d.InputValue(),
				}
			}

		case tea.KeyLeft, tea.KeyShiftTab:
			if d.Selected > 0 {
				d.Selected--
			}
			return *d, nil

		case tea.KeyRight, tea.KeyTab:
			if d.Selected < len(d.Buttons)-1 {
				d.Selected++
			}
			return *d, nil

		default:
			// Delegate typing to the text input if present.
			if d.Input != nil {
				var cmd tea.Cmd
				*d.Input, cmd = d.Input.Update(msg)
				return *d, cmd
			}
		}
	}

	return *d, nil
}

// View renders the dialog as a bordered box that can be overlaid on top of
// other content. The dialog is centered within the given container dimensions.
// The returned string contains the full output with centering padding so the
// caller can replace lines in the underlying content.
func (d Dialog) View(containerWidth, containerHeight int) string {
	borderStyle := lipgloss.NewStyle().Foreground(dlgColorDim)
	titleStyle := lipgloss.NewStyle().Foreground(dlgColorOrange).Bold(true)
	messageStyle := lipgloss.NewStyle().Foreground(dlgColorWhite)
	btnActiveStyle := lipgloss.NewStyle().
		Background(dlgColorOrange).
		Foreground(dlgColorBlack).
		Bold(true).
		Padding(0, 1)
	btnInactiveStyle := lipgloss.NewStyle().
		Foreground(dlgColorDim).
		Padding(0, 1)

	// Calculate dialog width: max(40, widest message line + 8, input prompt + 16),
	// capped at containerWidth - 10.
	dlgWidth := 40
	if d.Message != "" {
		for _, line := range strings.Split(d.Message, "\n") {
			if msgW := lipgloss.Width(line) + 8; msgW > dlgWidth {
				dlgWidth = msgW
			}
		}
	}
	if d.Input != nil {
		if promptW := lipgloss.Width(d.Input.Prompt) + 24; promptW > dlgWidth {
			dlgWidth = promptW
		}
	}
	if d.Width > 0 && d.Width > dlgWidth {
		dlgWidth = d.Width
	}
	maxWidth := containerWidth - 10
	if maxWidth < 30 {
		maxWidth = 30
	}
	if dlgWidth > maxWidth {
		dlgWidth = maxWidth
	}

	// Inner width is the space between the left and right border characters.
	innerWidth := dlgWidth - 2

	// Build the title bar: ┌────<Delete>────┐
	titleText := titleStyle.Render(d.Title)
	titleVisualWidth := lipgloss.Width(titleText)
	titleDecorated := borderStyle.Render("<") + titleText + borderStyle.Render(">")
	titleDecoratedWidth := titleVisualWidth + 2 // < and >

	remaining := innerWidth - titleDecoratedWidth
	if remaining < 2 {
		remaining = 2
	}
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes

	topLine := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleDecorated +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	// Empty line.
	emptyLine := borderStyle.Render("│") +
		strings.Repeat(" ", innerWidth) +
		borderStyle.Render("│")

	// Message lines (centered within inner width, supports multiline).
	var msgLines []string
	if d.Message != "" {
		for _, line := range strings.Split(d.Message, "\n") {
			lineRendered := messageStyle.Render(line)
			lineVisualWidth := lipgloss.Width(lineRendered)
			linePadLeft := (innerWidth - lineVisualWidth) / 2
			if linePadLeft < 1 {
				linePadLeft = 1
			}
			linePadRight := innerWidth - lineVisualWidth - linePadLeft
			if linePadRight < 0 {
				linePadRight = 0
			}
			msgLines = append(msgLines,
				borderStyle.Render("│")+
					strings.Repeat(" ", linePadLeft)+
					lineRendered+
					strings.Repeat(" ", linePadRight)+
					borderStyle.Render("│"))
		}
	}

	// Input line (if present).
	var inputLine string
	if d.Input != nil {
		inputRendered := d.Input.View()
		inputVisualWidth := lipgloss.Width(inputRendered)
		inputPadLeft := 4
		inputPadRight := innerWidth - inputVisualWidth - inputPadLeft
		if inputPadRight < 0 {
			inputPadRight = 0
		}
		inputLine = borderStyle.Render("│") +
			strings.Repeat(" ", inputPadLeft) +
			inputRendered +
			strings.Repeat(" ", inputPadRight) +
			borderStyle.Render("│")
	}

	// Button line (centered within inner width).
	var btnParts []string
	for i, label := range d.Buttons {
		if i == d.Selected {
			btnParts = append(btnParts, btnActiveStyle.Render(label))
		} else {
			btnParts = append(btnParts, btnInactiveStyle.Render(label))
		}
	}
	btnRow := strings.Join(btnParts, "     ")
	btnVisualWidth := lipgloss.Width(btnRow)
	btnPadLeft := (innerWidth - btnVisualWidth) / 2
	if btnPadLeft < 1 {
		btnPadLeft = 1
	}
	btnPadRight := innerWidth - btnVisualWidth - btnPadLeft
	if btnPadRight < 0 {
		btnPadRight = 0
	}
	btnLine := borderStyle.Render("│") +
		strings.Repeat(" ", btnPadLeft) +
		btnRow +
		strings.Repeat(" ", btnPadRight) +
		borderStyle.Render("│")

	// Bottom border.
	bottomLine := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	// Assemble dialog lines.
	var dialogLines []string
	dialogLines = append(dialogLines, topLine)
	dialogLines = append(dialogLines, emptyLine)
	if len(msgLines) > 0 {
		dialogLines = append(dialogLines, msgLines...)
		dialogLines = append(dialogLines, emptyLine)
	}
	if d.Input != nil {
		dialogLines = append(dialogLines, inputLine)
		dialogLines = append(dialogLines, emptyLine)
	}
	dialogLines = append(dialogLines, btnLine)
	dialogLines = append(dialogLines, emptyLine)
	dialogLines = append(dialogLines, bottomLine)

	// Center the dialog horizontally within the container width.
	dlgVisualWidth := lipgloss.Width(dialogLines[0])
	hPad := (containerWidth - dlgVisualWidth) / 2
	if hPad < 0 {
		hPad = 0
	}

	for i, line := range dialogLines {
		dialogLines[i] = strings.Repeat(" ", hPad) + line
	}

	// Center the dialog vertically within the container height.
	dlgHeight := len(dialogLines)
	vPad := (containerHeight - dlgHeight) / 2
	if vPad < 0 {
		vPad = 0
	}

	// Build full output: vPad empty lines, dialog lines, remaining empty lines.
	var result []string
	for range vPad {
		result = append(result, "")
	}
	result = append(result, dialogLines...)
	remaining = containerHeight - vPad - dlgHeight
	for range remaining {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// OverlayDialog renders the dialog on top of background content. It splits
// both the background and the dialog output into lines, and replaces the
// background lines where the dialog appears. Lines in the dialog output that
// are empty are treated as transparent (the background shows through).
func OverlayDialog(background string, dialog Dialog, containerWidth, containerHeight int) string {
	bgLines := strings.Split(background, "\n")
	dlgOutput := dialog.View(containerWidth, containerHeight)
	dlgLines := strings.Split(dlgOutput, "\n")

	// Ensure bgLines has enough lines.
	for len(bgLines) < containerHeight {
		bgLines = append(bgLines, "")
	}

	// Overlay: replace background lines with dialog lines where non-empty.
	for i, dlgLine := range dlgLines {
		if i >= len(bgLines) {
			break
		}
		if strings.TrimSpace(dlgLine) != "" {
			bgLines[i] = dlgLine
		}
	}

	return strings.Join(bgLines, "\n")
}

// OverlayForm renders a huh form inside a bordered box matching the confirm
// dialog aesthetic (dim single-line border, orange title), centered on top of
// background content. The title is displayed as ┌───<Title>───┐.
func OverlayForm(background, formView, title string, containerWidth, containerHeight int) string {
	bgLines := strings.Split(background, "\n")
	for len(bgLines) < containerHeight {
		bgLines = append(bgLines, "")
	}

	borderStyle := lipgloss.NewStyle().Foreground(dlgColorDim)
	titleStyle := lipgloss.NewStyle().Foreground(dlgColorOrange).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(dlgColorDim)

	// Strip trailing blank lines from the form view so the box is tight.
	formLines := strings.Split(formView, "\n")
	for len(formLines) > 0 && strings.TrimSpace(formLines[len(formLines)-1]) == "" {
		formLines = formLines[:len(formLines)-1]
	}

	// Determine inner width: max(form content, 56) to ensure comfortable padding.
	innerWidth := 56
	for _, fl := range formLines {
		if w := lipgloss.Width(fl) + 4; w > innerWidth {
			innerWidth = w
		}
	}
	maxInner := containerWidth - 12
	if maxInner < 30 {
		maxInner = 30
	}
	if innerWidth > maxInner {
		innerWidth = maxInner
	}

	// Top border with title: ┌────<New Session>────┐
	titleText := titleStyle.Render(title)
	titleVisualWidth := lipgloss.Width(titleText)
	titleDecorated := borderStyle.Render("<") + titleText + borderStyle.Render(">")
	titleDecoratedWidth := titleVisualWidth + 2
	remaining := innerWidth - titleDecoratedWidth
	if remaining < 2 {
		remaining = 2
	}
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes
	topLine := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		titleDecorated +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	emptyLine := borderStyle.Render("│") +
		strings.Repeat(" ", innerWidth) +
		borderStyle.Render("│")

	bottomLine := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	// Hint line: "Tab: next  Enter: submit  Esc: cancel"
	hint := hintStyle.Render("Tab: next  Enter: submit  Esc: cancel")
	hintW := lipgloss.Width(hint)
	hintPadL := (innerWidth - hintW) / 2
	if hintPadL < 1 {
		hintPadL = 1
	}
	hintPadR := innerWidth - hintW - hintPadL
	if hintPadR < 0 {
		hintPadR = 0
	}
	hintLine := borderStyle.Render("│") +
		strings.Repeat(" ", hintPadL) + hint + strings.Repeat(" ", hintPadR) +
		borderStyle.Render("│")

	// Assemble the dialog lines.
	var dialogLines []string
	dialogLines = append(dialogLines, topLine, emptyLine)
	for _, fl := range formLines {
		lineW := lipgloss.Width(fl)
		padL := 2
		padR := innerWidth - lineW - padL
		if padR < 0 {
			padR = 0
		}
		dialogLines = append(dialogLines,
			borderStyle.Render("│")+
				strings.Repeat(" ", padL)+fl+strings.Repeat(" ", padR)+
				borderStyle.Render("│"))
	}
	dialogLines = append(dialogLines, emptyLine, hintLine, emptyLine, bottomLine)

	// Center the dialog in the container.
	dlgHeight := len(dialogLines)
	vOffset := (containerHeight - dlgHeight) / 2
	if vOffset < 0 {
		vOffset = 0
	}

	dlgVisualWidth := lipgloss.Width(dialogLines[0])
	hPad := (containerWidth - dlgVisualWidth) / 2
	if hPad < 0 {
		hPad = 0
	}

	for i, dLine := range dialogLines {
		target := vOffset + i
		if target >= len(bgLines) {
			break
		}
		bgLines[target] = strings.Repeat(" ", hPad) + dLine
	}

	return strings.Join(bgLines, "\n")
}
