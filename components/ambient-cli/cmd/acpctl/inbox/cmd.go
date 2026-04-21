// Package inbox implements the inbox subcommand for managing inbox messages.
package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "inbox",
	Short: "Interact with agent inbox messages",
	Long: `Interact with agent inbox messages.

Subcommands:
  list       List inbox messages for a project-agent
  send       Send a message to a project-agent's inbox
  mark-read  Mark an inbox message as read
  delete     Delete an inbox message`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(sendCmd)
	Cmd.AddCommand(markReadCmd)
	Cmd.AddCommand(deleteCmd)

	listCmd.Flags().StringVar(&listArgs.projectID, "project-id", "", "Project ID (required)")
	listCmd.Flags().StringVar(&listArgs.paID, "pa-id", "", "Project-agent ID (required)")
	listCmd.Flags().StringVarP(&listArgs.outputFormat, "output", "o", "", "Output format: json|wide")
	listCmd.Flags().IntVar(&listArgs.limit, "limit", 100, "Maximum number of items to return")

	sendCmd.Flags().StringVar(&sendArgs.projectID, "project-id", "", "Project ID (required)")
	sendCmd.Flags().StringVar(&sendArgs.paID, "pa-id", "", "Project-agent ID (required)")
	sendCmd.Flags().StringVar(&sendArgs.body, "body", "", "Message body (required)")
	sendCmd.Flags().StringVar(&sendArgs.fromName, "from-name", "", "Sender display name")
	sendCmd.Flags().StringVar(&sendArgs.fromPAID, "from-pa-id", "", "Sender project-agent ID")
	sendCmd.Flags().StringVarP(&sendArgs.outputFormat, "output", "o", "", "Output format: json")

	markReadCmd.Flags().StringVar(&markReadArgs.projectID, "project-id", "", "Project ID (required)")
	markReadCmd.Flags().StringVar(&markReadArgs.paID, "pa-id", "", "Project-agent ID (required)")
	markReadCmd.Flags().StringVar(&markReadArgs.msgID, "msg-id", "", "Message ID (required)")

	deleteCmd.Flags().StringVar(&deleteArgs.projectID, "project-id", "", "Project ID (required)")
	deleteCmd.Flags().StringVar(&deleteArgs.paID, "pa-id", "", "Project-agent ID (required)")
	deleteCmd.Flags().StringVar(&deleteArgs.msgID, "msg-id", "", "Message ID (required)")
}

var listArgs struct {
	projectID    string
	paID         string
	outputFormat string
	limit        int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List inbox messages for a project-agent",
	Example: `  acpctl inbox list --project-id <id> --pa-id <id>
  acpctl inbox list --project-id <id> --pa-id <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listArgs.projectID == "" {
			return fmt.Errorf("--project-id is required")
		}
		if listArgs.paID == "" {
			return fmt.Errorf("--pa-id is required")
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		format, err := output.ParseFormat(listArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		opts := sdktypes.NewListOptions().Size(listArgs.limit).Build()
		list, err := client.InboxMessages().ListByAgent(ctx, listArgs.projectID, listArgs.paID, opts)
		if err != nil {
			return fmt.Errorf("list inbox messages: %w", err)
		}

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printInboxTable(printer, list.Items)
	},
}

var sendArgs struct {
	projectID    string
	paID         string
	body         string
	fromName     string
	fromPAID     string
	outputFormat string
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message to a project-agent's inbox",
	Example: `  acpctl inbox send --project-id <id> --pa-id <id> --body "please review PR #42"
  acpctl inbox send --project-id <id> --pa-id <id> --body "task complete" --from-name "agent-alpha"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if sendArgs.projectID == "" {
			return fmt.Errorf("--project-id is required")
		}
		if sendArgs.paID == "" {
			return fmt.Errorf("--pa-id is required")
		}
		if sendArgs.body == "" {
			return fmt.Errorf("--body is required")
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		builder := sdktypes.NewInboxMessageBuilder().
			AgentID(sendArgs.paID).
			Body(sendArgs.body)

		if sendArgs.fromName != "" {
			builder = builder.FromName(sendArgs.fromName)
		}
		if sendArgs.fromPAID != "" {
			builder = builder.FromAgentID(sendArgs.fromPAID)
		}

		msg, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build inbox message: %w", err)
		}

		created, err := client.InboxMessages().Send(ctx, sendArgs.projectID, sendArgs.paID, msg)
		if err != nil {
			return fmt.Errorf("send inbox message: %w", err)
		}

		format, err := output.ParseFormat(sendArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(created)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "inbox-message/%s sent\n", created.ID)
		return nil
	},
}

var markReadArgs struct {
	projectID string
	paID      string
	msgID     string
}

var markReadCmd = &cobra.Command{
	Use:     "mark-read",
	Short:   "Mark an inbox message as read",
	Example: `  acpctl inbox mark-read --project-id <id> --pa-id <id> --msg-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if markReadArgs.projectID == "" {
			return fmt.Errorf("--project-id is required")
		}
		if markReadArgs.paID == "" {
			return fmt.Errorf("--pa-id is required")
		}
		if markReadArgs.msgID == "" {
			return fmt.Errorf("--msg-id is required")
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		if err := client.InboxMessages().MarkRead(ctx, markReadArgs.projectID, markReadArgs.paID, markReadArgs.msgID); err != nil {
			return fmt.Errorf("mark-read inbox message: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "inbox-message/%s marked as read\n", markReadArgs.msgID)
		return nil
	},
}

var deleteArgs struct {
	projectID string
	paID      string
	msgID     string
}

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an inbox message",
	Example: `  acpctl inbox delete --project-id <id> --pa-id <id> --msg-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteArgs.projectID == "" {
			return fmt.Errorf("--project-id is required")
		}
		if deleteArgs.paID == "" {
			return fmt.Errorf("--pa-id is required")
		}
		if deleteArgs.msgID == "" {
			return fmt.Errorf("--msg-id is required")
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		if err := client.InboxMessages().DeleteMessage(ctx, deleteArgs.projectID, deleteArgs.paID, deleteArgs.msgID); err != nil {
			return fmt.Errorf("delete inbox message: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "inbox-message/%s deleted\n", deleteArgs.msgID)
		return nil
	},
}

func printInboxTable(printer *output.Printer, msgs []sdktypes.InboxMessage) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "FROM", Width: 20},
		{Name: "BODY", Width: 50},
		{Name: "READ", Width: 6},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, m := range msgs {
		age := ""
		if m.CreatedAt != nil {
			age = output.FormatAge(time.Since(*m.CreatedAt))
		}
		from := m.FromName
		if from == "" {
			from = m.FromAgentID
		}
		read := "false"
		if m.Read {
			read = "true"
		}
		body := m.Body
		if len(body) > 48 {
			body = body[:45] + "..."
		}
		table.WriteRow(m.ID, from, body, read, age)
	}
	return nil
}
