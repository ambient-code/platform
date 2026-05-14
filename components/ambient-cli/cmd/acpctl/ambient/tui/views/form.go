package views

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ACPTheme returns a huh theme matching the TUI's orange/blue palette.
func ACPTheme() *huh.Theme {
	t := huh.ThemeBase()

	orange := lipgloss.Color("214")
	blue := lipgloss.Color("69")
	white := lipgloss.Color("255")
	dim := lipgloss.Color("240")
	black := lipgloss.Color("0")
	red := lipgloss.Color("196")

	t.Focused.Base = t.Focused.Base.BorderForeground(dim)
	t.Focused.Title = t.Focused.Title.Foreground(orange).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(dim)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(orange)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(orange)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(orange)
	t.Focused.Option = t.Focused.Option.Foreground(white)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(orange)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(orange).SetString("✓ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(dim).SetString("• ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(white)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(black).Background(orange)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(white).Background(lipgloss.Color("237"))
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(orange)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(dim)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(blue)
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(white)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(orange).Bold(true)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(dim)

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}

// NewProjectForm creates a huh form for creating a new project.
func NewProjectForm(name, description *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Name").
				Placeholder("my-project").
				Validate(huh.ValidateNotEmpty()).
				Value(name),
			huh.NewInput().
				Key("description").
				Title("Description").
				Placeholder("(optional)").
				Value(description),
		),
	).WithTheme(ACPTheme()).WithShowHelp(false)
}

// NewAgentForm creates a huh form for creating a new agent.
func NewAgentForm(name, prompt *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Name").
				Placeholder("my-agent").
				Validate(huh.ValidateNotEmpty()).
				Value(name),
			huh.NewInput().
				Key("prompt").
				Title("Prompt").
				Placeholder("(optional)").
				Value(prompt),
		),
	).WithTheme(ACPTheme()).WithShowHelp(false)
}

// NewSessionForm creates a huh form for creating a new session.
// projectOptions must have at least one entry. agentOptions should include a
// "(none)" entry for standalone sessions; the agent Select is only shown when
// there are 2+ options.
func NewSessionForm(name, prompt, repoURL, projectID *string, projectOptions []huh.Option[string], agentID *string, agentOptions []huh.Option[string]) *huh.Form {
	fields := []huh.Field{
		huh.NewSelect[string]().
			Key("project").
			Title("Project").
			Options(projectOptions...).
			Value(projectID),
		huh.NewInput().
			Key("name").
			Title("Name").
			Placeholder("my-session").
			Validate(huh.ValidateNotEmpty()).
			Value(name),
		huh.NewInput().
			Key("prompt").
			Title("Prompt").
			Placeholder("(optional)").
			Value(prompt),
		huh.NewInput().
			Key("repo_url").
			Title("Repo URL").
			Placeholder("https://github.com/org/repo (optional)").
			Value(repoURL),
	}
	if len(agentOptions) > 1 {
		fields = append(fields,
			huh.NewSelect[string]().
				Key("agent").
				Title("Agent").
				Options(agentOptions...).
				Value(agentID),
		)
	}
	return huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(ACPTheme()).WithShowHelp(false)
}
