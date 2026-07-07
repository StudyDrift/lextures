package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var onerosterCmd = &cobra.Command{
	Use:   "oneroster",
	Short: "Operate OneRoster CSV and REST feeds",
}

var onerosterCommonFlags struct {
	institution string
}

var onerosterPullFlags struct {
	files []string
}

var onerosterPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Upload OneRoster CSV files and start a sync",
	RunE:  runOneRosterPull,
}

var onerosterValidateFlags struct {
	url   string
	token string
	files []string
}

var onerosterValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a OneRoster CSV bundle or REST endpoint",
	RunE:  runOneRosterValidate,
}

var onerosterStatusFlags struct {
	run string
}

var onerosterStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show OneRoster sync run status",
	RunE:  runOneRosterStatus,
}

func init() {
	onerosterPullCmd.Flags().StringVar(&onerosterCommonFlags.institution, "institution", "", "institution UUID (required)")
	_ = onerosterPullCmd.MarkFlagRequired("institution")
	onerosterPullCmd.Flags().StringSliceVar(&onerosterPullFlags.files, "file", nil, "OneRoster CSV files (repeatable)")
	onerosterPullCmd.Flags().StringSliceVar(&onerosterPullFlags.files, "files", nil, "alias for --file")

	onerosterValidateCmd.Flags().StringVar(&onerosterValidateFlags.url, "url", "", "OneRoster REST endpoint to probe")
	onerosterValidateCmd.Flags().StringVar(&onerosterValidateFlags.token, "token", "", "bearer token for REST probe")
	onerosterValidateCmd.Flags().StringSliceVar(&onerosterValidateFlags.files, "file", nil, "CSV files to validate")

	onerosterStatusCmd.Flags().StringVar(&onerosterCommonFlags.institution, "institution", "", "institution UUID (required)")
	_ = onerosterStatusCmd.MarkFlagRequired("institution")
	onerosterStatusCmd.Flags().StringVar(&onerosterStatusFlags.run, "run", "", "sync run id (default: latest)")

	onerosterCmd.AddCommand(onerosterPullCmd, onerosterValidateCmd, onerosterStatusCmd)
	rootCmd.AddCommand(onerosterCmd)
}

func runOneRosterPull(cmd *cobra.Command, args []string) error {
	if len(onerosterPullFlags.files) == 0 {
		return fmt.Errorf("at least one --file is required")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := uploadOneRosterCSV(c, onerosterCommonFlags.institution, onerosterPullFlags.files)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		SyncRunID string `json:"syncRunId"`
	}
	_ = json.Unmarshal(body, &out)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OneRoster sync started: %s\n", out.SyncRunID)
	return nil
}

func runOneRosterValidate(cmd *cobra.Command, args []string) error {
	if onerosterValidateFlags.url != "" {
		if strings.TrimSpace(onerosterValidateFlags.token) == "" {
			return fmt.Errorf("--token is required with --url")
		}
		c := client.New(Cfg.Server, Cfg.APIKey)
		body, err := probeOneRosterURL(c, onerosterValidateFlags.url, onerosterValidateFlags.token)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"ok": true, "body": json.RawMessage(body)})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "OneRoster endpoint reachable.")
		return nil
	}
	if len(onerosterValidateFlags.files) == 0 {
		return fmt.Errorf("provide --file for CSV validation or --url with --token for REST validation")
	}
	if err := validateOneRosterCSVFiles(onerosterValidateFlags.files); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"ok": true, "files": onerosterValidateFlags.files})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "OneRoster CSV bundle looks valid.")
	return nil
}

func runOneRosterStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if onerosterStatusFlags.run != "" {
		body, err := fetchOneRosterSyncRunDetail(c, onerosterStatusFlags.run)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(body)
			return err
		}
		var out struct {
			Events []map[string]any `json:"events"`
		}
		_ = json.Unmarshal(body, &out)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Run %s: %d events\n", onerosterStatusFlags.run, len(out.Events))
		return nil
	}
	runs, raw, err := fetchOneRosterSyncRuns(c, onerosterCommonFlags.institution)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "RUN\tSTATUS\tCREATED\tUPDATED\tERRORS\tSTARTED")
	for _, run := range runs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\n",
			run.ID, run.Status, run.CreatedCount, run.UpdatedCount, run.ErrorCount, run.StartedAt)
	}
	return w.Flush()
}