package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

func printLRSDeadLetterTable(cmd *cobra.Command, body []byte) error {
	var rows []struct {
		ID          string  `json:"id"`
		StatementID string  `json:"statementId"`
		LastError   *string `json:"lastError,omitempty"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTATEMENT\tERROR")
	for _, row := range rows {
		errMsg := ""
		if row.LastError != nil {
			errMsg = *row.LastError
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", row.ID, row.StatementID, errMsg)
	}
	return w.Flush()
}

var lrsCmd = &cobra.Command{
	Use:   "lrs",
	Short: "Manage xAPI LRS emission configuration",
}

var lrsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Read and write LRS endpoints",
}

var lrsConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get LRS configuration",
	RunE:  runLRSConfigGet,
}

var lrsConfigSetFlags struct {
	file string
	id   string
}

var lrsConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Create or update an LRS endpoint from JSON",
	RunE:  runLRSConfigSet,
}

var lrsConfigTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Send a test statement to an LRS endpoint",
	Args:  cobra.ExactArgs(1),
	RunE:  runLRSConfigTest,
}

var lrsDeadLetterNestedCmd = &cobra.Command{
	Use:   "dead-letter",
	Short: "Inspect and retry LRS forwarding dead-letter queue",
}

var lrsDeadLetterListNestedCmd = &cobra.Command{
	Use:   "list",
	Short: "List LRS dead-letter statements",
	RunE:  runLRSDeadLetterListNested,
}

var lrsDeadLetterRetryNestedCmd = &cobra.Command{
	Use:   "retry <id>",
	Short: "Retry forwarding a dead-letter statement",
	Args:  cobra.ExactArgs(1),
	RunE:  runLRSDeadLetterRetryNested,
}

var lrsDeadLetterPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge dead-letter queue (not supported by server)",
	RunE:  runLRSDeadLetterPurge,
}

func init() {
	lrsConfigSetCmd.Flags().StringVar(&lrsConfigSetFlags.file, "file", "", "LRS config JSON (required)")
	_ = lrsConfigSetCmd.MarkFlagRequired("file")
	lrsConfigSetCmd.Flags().StringVar(&lrsConfigSetFlags.id, "id", "", "existing config id to update")

	lrsDeadLetterNestedCmd.AddCommand(lrsDeadLetterListNestedCmd, lrsDeadLetterRetryNestedCmd, lrsDeadLetterPurgeCmd)
	lrsConfigCmd.AddCommand(lrsConfigGetCmd, lrsConfigSetCmd, lrsConfigTestCmd)
	lrsCmd.AddCommand(lrsConfigCmd, lrsDeadLetterNestedCmd)
	rootCmd.AddCommand(lrsCmd)
}

func runLRSConfigGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchLRSConfig(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return nil
}

func runLRSConfigSet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := setLRSConfig(c, lrsConfigSetFlags.file, lrsConfigSetFlags.id)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	if lrsConfigSetFlags.id != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "LRS configuration updated.")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "LRS configuration created.")
	}
	return nil
}

func runLRSConfigTest(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := testLRSEndpoint(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runLRSDeadLetterListNested(cmd *cobra.Command, args []string) error {
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

func runLRSDeadLetterRetryNested(cmd *cobra.Command, args []string) error {
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

func runLRSDeadLetterPurge(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("server does not expose a dead-letter purge endpoint; retry entries individually")
}