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

var auditLogCmd = &cobra.Command{
	Use:   "audit-log",
	Short: "List and export the compliance audit log",
}

var auditLogListFlags auditLogFilters

var auditLogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit log events",
	RunE:  runAuditLogList,
}

var auditLogExportFlags struct {
	auditLogFilters
	format string
	yes    bool
}

var auditLogExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit log events (JSON or CSV)",
	RunE:  runAuditLogExport,
}

var auditLogTailFlags struct {
	auditLogFilters
	interval time.Duration
}

var auditLogTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Poll for new audit log entries",
	RunE:  runAuditLogTail,
}

func init() {
	bindAuditLogFilters(auditLogListCmd, &auditLogListFlags)
	bindAuditLogFilters(auditLogExportCmd, &auditLogExportFlags.auditLogFilters)
	auditLogExportCmd.Flags().StringVar(&auditLogExportFlags.format, "format", "json", "output format: json or csv")
	auditLogExportCmd.Flags().BoolVar(&auditLogExportFlags.yes, "yes", false, "confirm exporting potentially sensitive audit data")

	bindAuditLogFilters(auditLogTailCmd, &auditLogTailFlags.auditLogFilters)
	auditLogTailCmd.Flags().DurationVar(&auditLogTailFlags.interval, "interval", 5*time.Second, "poll interval")

	auditLogCmd.AddCommand(auditLogListCmd, auditLogExportCmd, auditLogTailCmd)
	rootCmd.AddCommand(auditLogCmd)
}

func bindAuditLogFilters(cmd *cobra.Command, f *auditLogFilters) {
	cmd.Flags().StringVar(&f.ActorID, "actor", "", "filter by actor user UUID")
	cmd.Flags().StringVar(&f.EventType, "action", "", "filter by event type")
	cmd.Flags().StringVar(&f.From, "from", "", "start timestamp (RFC3339)")
	cmd.Flags().StringVar(&f.To, "to", "", "end timestamp (RFC3339)")
	cmd.Flags().StringVar(&f.TargetID, "target", "", "filter by target UUID")
	cmd.Flags().StringVar(&f.OrgID, "org", "", "filter by organization UUID")
}

func runAuditLogList(cmd *cobra.Command, _ []string) error {
	events, raw, err := fetchAuditLog(client.New(Cfg.Server, Cfg.APIKey), auditLogListFlags)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "EVENT_ID\tTIMESTAMP\tTYPE\tACTOR\tTARGET")
	for _, e := range events {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.EventID, e.Timestamp, e.EventType, e.ActorID, e.TargetID)
	}
	return w.Flush()
}

func runAuditLogExport(cmd *cobra.Command, _ []string) error {
	if !auditLogExportFlags.yes {
		return fmt.Errorf("audit export requires --yes (exports may contain sensitive compliance data)")
	}
	format := auditLogExportFlags.format
	if format != "json" && format != "csv" {
		return fmt.Errorf("format must be json or csv")
	}
	body, contentType, err := exportAuditLog(client.New(Cfg.Server, Cfg.APIKey), auditLogExportFlags.auditLogFilters, format)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut && format == "json" {
		var pretty any
		if json.Unmarshal(body, &pretty) == nil {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(pretty)
		}
	}
	if format == "csv" || contentType == "text/csv; charset=utf-8" {
		_, err = os.Stdout.Write(body)
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAuditLogTail(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	seen := make(map[string]struct{})
	attempt := 0
	for {
		events, _, err := fetchAuditLog(c, auditLogTailFlags.auditLogFilters)
		if err != nil {
			attempt++
			time.Sleep(auditTailBackoff(attempt))
			continue
		}
		attempt = 0
		for _, e := range events {
			if _, ok := seen[e.EventID]; ok {
				continue
			}
			seen[e.EventID] = struct{}{}
			if globalFlags.jsonOut {
				_ = json.NewEncoder(cmd.OutOrStdout()).Encode(e)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s actor=%s type=%s\n",
					e.Timestamp, e.EventID, e.ActorID, e.EventType)
			}
		}
		time.Sleep(auditLogTailFlags.interval)
	}
}