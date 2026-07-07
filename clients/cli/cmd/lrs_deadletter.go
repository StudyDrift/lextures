package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var lrsDeadLetterCmd = &cobra.Command{
	Use:   "lrs-dead-letter",
	Short: "Inspect and retry LRS forwarding dead-letter queue",
}

var lrsDeadLetterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List LRS dead-letter statements",
	RunE:  runLRSDeadLetterList,
}

var lrsDeadLetterRetryCmd = &cobra.Command{
	Use:   "retry <id>",
	Short: "Retry forwarding a dead-letter statement",
	Args:  cobra.ExactArgs(1),
	RunE:  runLRSDeadLetterRetry,
}

func init() {
	lrsDeadLetterCmd.AddCommand(lrsDeadLetterListCmd, lrsDeadLetterRetryCmd)
	rootCmd.AddCommand(lrsDeadLetterCmd)
}

func runLRSDeadLetterList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := listLRSDeadLetter(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	return printLRSDeadLetterTable(cmd, body)
}

func runLRSDeadLetterRetry(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := retryLRSDeadLetter(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dead-letter statement queued for retry.")
	return nil
}