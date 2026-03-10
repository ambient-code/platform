// Package get implements the get subcommand for listing and retrieving resources.
package get

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var args struct {
	outputFormat string
	limit        int
	watch        bool
	watchTimeout time.Duration
}

var Cmd = &cobra.Command{
	Use:   "get <resource> [name]",
	Short: "Display one or many resources",
	Long: `Display one or many resources.

Valid resource types:
  sessions         (aliases: session, sess)
  projects         (aliases: project, proj)
  project-settings (aliases: projectsettings, ps)
  users            (aliases: user, usr)`,
	Args:    cobra.RangeArgs(1, 2),
	RunE:    run,
	Example: "  acpctl get sessions\n  acpctl get session my-session-id\n  acpctl get projects -o json\n  acpctl get sessions -w  # Watch for real-time session changes",
}

func init() {
	Cmd.Flags().StringVarP(&args.outputFormat, "output", "o", "", "Output format: json|wide")
	Cmd.Flags().IntVar(&args.limit, "limit", 100, "Maximum number of items to return")
	Cmd.Flags().BoolVarP(&args.watch, "watch", "w", false, "Watch for real-time changes (sessions only)")
	Cmd.Flags().DurationVar(&args.watchTimeout, "watch-timeout", 30*time.Minute, "Timeout for watch mode (e.g. 1h, 10m)")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := normalizeResource(cmdArgs[0])

	var name string
	if len(cmdArgs) > 1 {
		name = cmdArgs[1]
	}

	if args.watch {
		if resource != "sessions" {
			return fmt.Errorf("--watch is only supported for sessions, not %s", resource)
		}
		if name != "" {
			return fmt.Errorf("watch cannot be used with a specific resource name")
		}
		if args.outputFormat == "json" {
			return fmt.Errorf("watch is not supported with JSON output format")
		}
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(args.outputFormat)
	if err != nil {
		return err
	}
	printer := output.NewPrinter(format)

	if args.watch {
		return watchSessions(cmd, client, printer)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
	defer cancel()

	switch resource {
	case "sessions":
		return getSessions(ctx, client, printer, name)
	case "projects":
		return getProjects(ctx, client, printer, name)
	case "project-settings":
		return getProjectSettings(ctx, client, printer, name)
	case "users":
		return getUsers(ctx, client, printer, name)
	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: sessions, projects, project-settings, users", cmdArgs[0])
	}
}

func normalizeResource(r string) string {
	switch strings.ToLower(r) {
	case "session", "sessions", "sess":
		return "sessions"
	case "project", "projects", "proj":
		return "projects"
	case "project-settings", "projectsettings", "project-setting", "ps":
		return "project-settings"
	case "user", "users", "usr":
		return "users"
	default:
		return r
	}
}

func getSessions(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		session, err := client.Sessions().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get session %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(session)
		}
		return printSessionTable(printer, []sdktypes.Session{*session})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.Sessions().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printSessionTable(printer, list.Items)
}

func printSessionTable(printer *output.Printer, sessions []sdktypes.Session) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 30},
		{Name: "PROJECT", Width: 20},
		{Name: "PHASE", Width: 12},
		{Name: "MODEL", Width: 16},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, s := range sessions {
		age := ""
		if s.CreatedAt != nil {
			age = output.FormatAge(time.Since(*s.CreatedAt))
		}
		table.WriteRow(s.ID, s.Name, s.ProjectID, s.Phase, s.LlmModel, age)
	}
	return nil
}

func getProjects(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		project, err := client.Projects().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get project %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(project)
		}
		return printProjectTable(printer, []sdktypes.Project{*project})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.Projects().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printProjectTable(printer, list.Items)
}

func printProjectTable(printer *output.Printer, projects []sdktypes.Project) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 30},
		{Name: "DISPLAY NAME", Width: 30},
		{Name: "STATUS", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, p := range projects {
		table.WriteRow(p.ID, p.Name, p.DisplayName, p.Status)
	}
	return nil
}

func getProjectSettings(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		settings, err := client.ProjectSettings().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get project-settings %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(settings)
		}
		return printProjectSettingsTable(printer, []sdktypes.ProjectSettings{*settings})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.ProjectSettings().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list project-settings: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printProjectSettingsTable(printer, list.Items)
}

func printProjectSettingsTable(printer *output.Printer, settings []sdktypes.ProjectSettings) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "PROJECT ID", Width: 27},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, s := range settings {
		age := ""
		if s.CreatedAt != nil {
			age = output.FormatAge(time.Since(*s.CreatedAt))
		}
		table.WriteRow(s.ID, s.ProjectID, age)
	}
	return nil
}

func getUsers(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		user, err := client.Users().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get user %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(user)
		}
		return printUserTable(printer, []sdktypes.User{*user})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.Users().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printUserTable(printer, list.Items)
}

func printUserTable(printer *output.Printer, users []sdktypes.User) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "USERNAME", Width: 30},
		{Name: "NAME", Width: 30},
		{Name: "EMAIL", Width: 40},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, u := range users {
		table.WriteRow(u.ID, u.Username, u.Name, u.Email)
	}
	return nil
}

func watchSessions(cmd *cobra.Command, client *sdkclient.Client, printer *output.Printer) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), args.watchTimeout)
	defer cancel()
	ctx, sigCancel := signal.NotifyContext(ctx, os.Interrupt)
	defer sigCancel()

	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 30},
		{Name: "PROJECT", Width: 20},
		{Name: "PHASE", Width: 12},
		{Name: "MODEL", Width: 16},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	// Try gRPC streaming watch first, fall back to polling if unavailable
	watcher, err := client.Sessions().Watch(ctx, &sdkclient.WatchOptions{
		Timeout: args.watchTimeout,
	})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "gRPC watch unavailable (%v), falling back to polling...\n", err)
		return watchSessionsPolling(cmd.ErrOrStderr(), ctx, client, table)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcher.Done():
			return nil
		case err := <-watcher.Errors():
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Watch error: %v\n", err)
				continue
			}
		case event := <-watcher.Events():
			if event == nil {
				continue
			}

			// Display the session
			if event.Session != nil {
				age := ""
				if event.Session.CreatedAt != nil {
					age = output.FormatAge(time.Since(*event.Session.CreatedAt))
				}
				table.WriteRow(event.Session.ID, event.Session.Name, event.Session.ProjectID, event.Session.Phase, event.Session.LlmModel, age)
			}
		}
	}
}

const maxConsecutiveErrors = 5

// watchSessionsPolling implements the fallback polling-based watch
func watchSessionsPolling(stderr io.Writer, ctx context.Context, client *sdkclient.Client, table *output.Table) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ticker := time.NewTicker(cfg.GetPollingInterval())
	defer ticker.Stop()

	sessionTracker := newSessionTracker()
	firstPoll := true
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
			err := processAllSessionPages(listCtx, client, args.limit, sessionTracker, table, firstPoll)
			listCancel()

			if err != nil {
				consecutiveErrors++
				fmt.Fprintf(stderr, "Error processing sessions (%d/%d): %v\n", consecutiveErrors, maxConsecutiveErrors, err)
				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("too many consecutive errors, stopping watch: %w", err)
				}
				continue
			}

			consecutiveErrors = 0
			firstPoll = false
		}
	}
}

// sessionTracker tracks session state across polling cycles for change detection
type sessionTracker struct {
	sessions map[string]sdktypes.Session
}

func newSessionTracker() *sessionTracker {
	return &sessionTracker{
		sessions: make(map[string]sdktypes.Session),
	}
}

// processAllSessionPages processes all pages of sessions, handling each page individually
func processAllSessionPages(ctx context.Context, client *sdkclient.Client, pageSize int, tracker *sessionTracker, table *output.Table, firstPoll bool) error {
	page := 1
	seenSessions := make(map[string]bool) // Track which sessions we've seen this cycle

	for {
		listOpts := sdktypes.NewListOptions().
			Page(page).
			Size(pageSize).
			Build()

		list, err := client.Sessions().List(ctx, listOpts)
		if err != nil {
			return err
		}

		// Process this page of sessions immediately
		for _, session := range list.Items {
			seenSessions[session.ID] = true

			// Calculate age
			age := ""
			if session.CreatedAt != nil {
				age = output.FormatAge(time.Since(*session.CreatedAt))
			}

			if firstPoll {
				// On first poll, show all sessions
				table.WriteRow(session.ID, session.Name, session.ProjectID, session.Phase, session.LlmModel, age)
			} else {
				// Check for changes against previous state
				if oldSession, exists := tracker.sessions[session.ID]; exists {
					if sessionChanged(oldSession, session) {
						// Session changed - show current state
						table.WriteRow(session.ID, session.Name, session.ProjectID, session.Phase, session.LlmModel, age)
					}
				} else {
					// New session
					table.WriteRow(session.ID, session.Name, session.ProjectID, session.Phase, session.LlmModel, age)
				}
			}

			// Update tracker with this session immediately
			tracker.sessions[session.ID] = session
		}

		// If we got fewer items than the page size, we've reached the end
		if len(list.Items) < pageSize {
			break
		}

		page++
	}

	// Check for deleted sessions (only if not first poll)
	if !firstPoll {
		for id, oldSession := range tracker.sessions {
			if !seenSessions[id] {
				table.WriteRow(oldSession.ID, oldSession.Name, oldSession.ProjectID, "DELETED", oldSession.LlmModel, "")
				delete(tracker.sessions, id) // Remove from tracker
			}
		}
	}

	return nil
}

func sessionChanged(old, current sdktypes.Session) bool {
	return old.Phase != current.Phase ||
		old.Name != current.Name ||
		old.LlmModel != current.LlmModel ||
		(old.UpdatedAt != nil && current.UpdatedAt != nil && !old.UpdatedAt.Equal(*current.UpdatedAt))
}
