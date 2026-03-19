// Package agent implements noun-style subcommands for working with a specific agent.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var agentID string

var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with agents",
	Long: `Interact with agents.

Examples:
  acpctl agent messages <id>            # list agent messages
  acpctl agent send <id> "Hello!"       # send a message to an agent`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var msgArgs struct {
	outputFormat string
}

var agentMessagesCmd = &cobra.Command{
	Use:   "messages <agent-id>",
	Short: "List messages for an agent",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentMessages,
}

var agentSendCmd = &cobra.Command{
	Use:   "send <agent-id> <message>",
	Short: "Send a message to an agent",
	Args:  cobra.ExactArgs(2),
	RunE:  runAgentSend,
}

func init() {
	agentMessagesCmd.Flags().StringVarP(&msgArgs.outputFormat, "output", "o", "", "Output format: json")
	Cmd.AddCommand(agentMessagesCmd)
	Cmd.AddCommand(agentSendCmd)
}

func runAgentMessages(cmd *cobra.Command, args []string) error {
	agentID = args[0]
	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(msgArgs.outputFormat)
	if err != nil {
		return err
	}
	printer := output.NewPrinter(format, cmd.OutOrStdout())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return listAgentMessages(ctx, client, printer)
}

func listAgentMessages(ctx context.Context, client *sdkclient.Client, printer *output.Printer) error {
	opts := sdktypes.NewListOptions().Size(100).Build()
	list, err := client.AgentMessages().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list agent messages: %w", err)
	}

	var agentMsgs []sdktypes.AgentMessage
	for _, m := range list.Items {
		if m.RecipientAgentID == agentID {
			agentMsgs = append(agentMsgs, m)
		}
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(agentMsgs)
	}

	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "FROM", Width: 20},
		{Name: "READ", Width: 5},
		{Name: "AGE", Width: 10},
		{Name: "BODY", Width: 60},
	}
	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, m := range agentMsgs {
		age := ""
		if m.CreatedAt != nil {
			age = output.FormatAge(time.Since(*m.CreatedAt))
		}
		sender := m.SenderName
		if sender == "" {
			sender = m.SenderUserID
		}
		read := "false"
		if m.Read {
			read = "true"
		}
		body := m.Body
		if len(body) > 57 {
			body = body[:57] + "..."
		}
		table.WriteRow(m.ID, sender, read, age, body)
	}
	return nil
}

func runAgentSend(cmd *cobra.Command, args []string) error {
	id := args[0]
	body := args[1]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg, err := client.AgentMessages().Create(ctx, &sdktypes.AgentMessage{
		RecipientAgentID: id,
		Body:             body,
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "sent (id=%s)\n", msg.ID)
	return nil
}
