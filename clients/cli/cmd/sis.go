package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var sisCmd = &cobra.Command{
	Use:   "sis",
	Short: "Configure and operate SIS roster sync",
}

var sisCommonFlags struct {
	org string
}

var sisConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage SIS connection configuration",
}

var sisConfigGetFlags struct {
	connection string
}

var sisConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get SIS connection configuration",
	RunE:  runSISConfigGet,
}

var sisConfigSetFlags struct {
	file       string
	connection string
}

var sisConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Create or update a SIS connection from JSON",
	RunE:  runSISConfigSet,
}

var sisConfigTestFlags struct {
	connection string
}

var sisConfigTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test SIS connectivity and credentials",
	RunE:  runSISConfigTest,
}

var sisSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trigger and inspect SIS sync jobs",
}

var sisSyncRunFlags struct {
	connection string
	wait       bool
	timeout    time.Duration
	yes        bool
}

var sisSyncRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a SIS sync for a connection",
	RunE:  runSISSyncRun,
}

var sisSyncStatusFlags struct {
	log string
}

var sisSyncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get the latest sync log for a connection",
	RunE:  runSISSyncStatus,
}

var sisSyncHistoryFlags struct {
	limit int
}

var sisSyncHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List SIS sync history",
	RunE:  runSISSyncHistory,
}

var sisReconcileFlags struct {
	connection string
	dryRun     bool
}

var sisReconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Show roster drift from the latest sync summary (dry-run only)",
	RunE:  runSISReconcile,
}

func init() {
	sisConfigGetCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisConfigGetCmd.MarkFlagRequired("org")
	sisConfigGetCmd.Flags().StringVar(&sisConfigGetFlags.connection, "connection", "", "connection id or vendor")

	sisConfigSetCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisConfigSetCmd.MarkFlagRequired("org")
	sisConfigSetCmd.Flags().StringVar(&sisConfigSetFlags.file, "file", "", "connection JSON file (required)")
	_ = sisConfigSetCmd.MarkFlagRequired("file")
	sisConfigSetCmd.Flags().StringVar(&sisConfigSetFlags.connection, "connection", "", "existing connection id to patch")

	sisConfigTestCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisConfigTestCmd.MarkFlagRequired("org")
	sisConfigTestCmd.Flags().StringVar(&sisConfigTestFlags.connection, "connection", "", "connection id or vendor (required)")
	_ = sisConfigTestCmd.MarkFlagRequired("connection")

	sisSyncRunCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisSyncRunCmd.MarkFlagRequired("org")
	sisSyncRunCmd.Flags().StringVar(&sisSyncRunFlags.connection, "connection", "", "connection id or vendor (required)")
	_ = sisSyncRunCmd.MarkFlagRequired("connection")
	sisSyncRunCmd.Flags().BoolVar(&sisSyncRunFlags.wait, "wait", false, "poll until the sync completes")
	sisSyncRunCmd.Flags().DurationVar(&sisSyncRunFlags.timeout, "timeout", 30*time.Minute, "max wait time")
	sisSyncRunCmd.Flags().BoolVar(&sisSyncRunFlags.yes, "yes", false, "confirm sync when deletions may occur")

	sisSyncStatusCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisSyncStatusCmd.MarkFlagRequired("org")
	sisSyncStatusCmd.Flags().StringVar(&sisSyncStatusFlags.log, "log", "", "sync log id (default: latest for connection)")
	sisSyncStatusCmd.Flags().StringVar(&sisSyncRunFlags.connection, "connection", "", "connection id or vendor")

	sisSyncHistoryCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisSyncHistoryCmd.MarkFlagRequired("org")
	sisSyncHistoryCmd.Flags().IntVar(&sisSyncHistoryFlags.limit, "limit", 20, "max rows")

	sisReconcileCmd.Flags().StringVar(&sisCommonFlags.org, "org", "", "target org UUID (required)")
	_ = sisReconcileCmd.MarkFlagRequired("org")
	sisReconcileCmd.Flags().StringVar(&sisReconcileFlags.connection, "connection", "", "connection id or vendor (required)")
	_ = sisReconcileCmd.MarkFlagRequired("connection")
	sisReconcileCmd.Flags().BoolVar(&sisReconcileFlags.dryRun, "dry-run", true, "show drift without applying changes")

	sisConfigCmd.AddCommand(sisConfigGetCmd, sisConfigSetCmd, sisConfigTestCmd)
	sisSyncCmd.AddCommand(sisSyncRunCmd, sisSyncStatusCmd, sisSyncHistoryCmd)
	sisCmd.AddCommand(sisConfigCmd, sisSyncCmd, sisReconcileCmd)
	rootCmd.AddCommand(sisCmd)
}

func runSISConfigGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	conns, raw, err := fetchSISConnections(c, sisCommonFlags.org)
	if err != nil {
		return err
	}
	if sisConfigGetFlags.connection != "" {
		connID, err := resolveSISConnectionID(c, sisCommonFlags.org, sisConfigGetFlags.connection)
		if err != nil {
			return err
		}
		for _, conn := range conns {
			if conn.ID == connID {
				if globalFlags.jsonOut {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"connection": conn})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\tactive=%v\n", conn.ID, conn.Vendor, conn.BaseURL, conn.Active)
				return nil
			}
		}
		return fmt.Errorf("connection %q not found", sisConfigGetFlags.connection)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tVENDOR\tBASE_URL\tACTIVE")
	for _, conn := range conns {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", conn.ID, conn.Vendor, conn.BaseURL, conn.Active)
	}
	return w.Flush()
}

func runSISConfigSet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, created, err := applySISConfigFile(c, sisCommonFlags.org, sisConfigSetFlags.file, sisConfigSetFlags.connection)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	if created {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "SIS connection created.")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "SIS connection updated.")
	}
	return nil
}

func runSISConfigTest(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	connID, err := resolveSISConnectionID(c, sisCommonFlags.org, sisConfigTestFlags.connection)
	if err != nil {
		return err
	}
	body, err := testSISConnection(c, sisCommonFlags.org, connID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
		Vendor  string `json:"vendor"`
	}
	_ = json.Unmarshal(body, &out)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (vendor=%s)\n", out.Message, out.Vendor)
	return nil
}

func runSISSyncRun(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	connID, err := resolveSISConnectionID(c, sisCommonFlags.org, sisSyncRunFlags.connection)
	if err != nil {
		return err
	}
	if !sisSyncRunFlags.yes {
		logs, _, err := fetchSISSyncLogs(c, sisCommonFlags.org, 1)
		if err == nil && len(logs) > 0 && logs[0].Summary != nil {
			if deleted, ok := logs[0].Summary["deleted"]; ok {
				switch v := deleted.(type) {
				case float64:
					if v > 0 {
						return fmt.Errorf("previous sync deleted %.0f records; re-run with --yes to confirm", v)
					}
				case int:
					if v > 0 {
						return fmt.Errorf("previous sync deleted %d records; re-run with --yes to confirm", v)
					}
				}
			}
		}
	}
	log, raw, err := runSISSync(c, sisCommonFlags.org, connID)
	if err != nil {
		return err
	}
	if sisSyncRunFlags.wait && log.ID != "" {
		log, err = waitForSISSyncLog(c, sisCommonFlags.org, log.ID, sisSyncRunFlags.timeout, func(s sisSyncLog) {
			if !globalFlags.jsonOut {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status=%s\n", s.Status)
			}
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(log)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sync %s finished: status=%s summary=%v\n", log.ID, log.Status, log.Summary)
		return nil
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started sync %s (status=%s)\n", log.ID, log.Status)
	return nil
}

func runSISSyncStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	logs, raw, err := fetchSISSyncLogs(c, sisCommonFlags.org, 50)
	if err != nil {
		return err
	}
	var selected *sisSyncLog
	if sisSyncStatusFlags.log != "" {
		for i := range logs {
			if logs[i].ID == sisSyncStatusFlags.log {
				selected = &logs[i]
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("sync log %q not found", sisSyncStatusFlags.log)
		}
	} else if sisSyncRunFlags.connection != "" {
		connID, err := resolveSISConnectionID(c, sisCommonFlags.org, sisSyncRunFlags.connection)
		if err != nil {
			return err
		}
		for i := range logs {
			if logs[i].ConnectionID == connID {
				selected = &logs[i]
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("no sync logs for connection %q", sisSyncRunFlags.connection)
		}
	} else if len(logs) > 0 {
		selected = &logs[0]
	}
	if globalFlags.jsonOut {
		if selected != nil {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(selected)
		}
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if selected == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No sync logs found.")
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\tsummary=%v\n", selected.ID, selected.Status, selected.StartedAt, selected.Summary)
	return nil
}

func runSISSyncHistory(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	logs, raw, err := fetchSISSyncLogs(c, sisCommonFlags.org, sisSyncHistoryFlags.limit)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "LOG\tCONNECTION\tSTATUS\tSTARTED")
	for _, log := range logs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", log.ID, log.ConnectionID, log.Status, log.StartedAt)
	}
	return w.Flush()
}

func runSISReconcile(cmd *cobra.Command, args []string) error {
	if !sisReconcileFlags.dryRun {
		return fmt.Errorf("only --dry-run is supported; use sis sync run to apply changes")
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	connID, err := resolveSISConnectionID(c, sisCommonFlags.org, sisReconcileFlags.connection)
	if err != nil {
		return err
	}
	logs, _, err := fetchSISSyncLogs(c, sisCommonFlags.org, 20)
	if err != nil {
		return err
	}
	for _, log := range logs {
		if log.ConnectionID != connID {
			continue
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"dryRun":  true,
				"logId":   log.ID,
				"status":  log.Status,
				"summary": log.Summary,
				"errors":  log.Errors,
			})
		}
		_, _ = fmt.Fprintf(os.Stdout, "Latest sync %s (%s): summary=%v errors=%d\n",
			log.ID, log.Status, log.Summary, len(log.Errors))
		return nil
	}
	return fmt.Errorf("no sync history for connection %q", sisReconcileFlags.connection)
}