package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var reportsCmd = &cobra.Command{
	Use:   "reports",
	Short: "Run and export platform reports",
}

var reportsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available report types",
	RunE:  runReportsList,
}

var reportsCommonFlags struct {
	from   string
	to     string
	course string
	org    string
	format string
	out    string
	yes    bool
	wait   bool
}

var reportsLearningActivityCmd = &cobra.Command{
	Use:   "learning-activity",
	Short: "Fetch the platform learning-activity report",
	RunE:  runReportsLearningActivity,
}

var reportsRunCmd = &cobra.Command{
	Use:   "run <type>",
	Short: "Generate a report (sync; --wait is a no-op for sync reports)",
	Args:  cobra.ExactArgs(1),
	RunE:  runReportsRun,
}

var reportsExportCmd = &cobra.Command{
	Use:   "export <type>",
	Short: "Export a report to a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runReportsExport,
}

var reportsScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage scheduled report exports",
}

var reportsScheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List report schedules",
	RunE:  runReportsScheduleList,
}

var reportsScheduleCreateFlags struct {
	file string
}

var reportsScheduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a report schedule from JSON",
	RunE:  runReportsScheduleCreate,
}

var reportsScheduleDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a report schedule",
	Args:  cobra.ExactArgs(1),
	RunE:  runReportsScheduleDelete,
}

func init() {
	reportsLearningActivityCmd.Flags().StringVar(&reportsCommonFlags.from, "from", "", "range start (RFC3339)")
	reportsLearningActivityCmd.Flags().StringVar(&reportsCommonFlags.to, "to", "", "range end (RFC3339)")
	reportsLearningActivityCmd.Flags().StringVar(&reportsCommonFlags.org, "org", "", "reserved; platform report is org-global via permissions")
	reportsLearningActivityCmd.Flags().StringVar(&reportsCommonFlags.course, "course", "", "reserved for future course-scoped activity")

	for _, cmd := range []*cobra.Command{reportsRunCmd, reportsExportCmd} {
		cmd.Flags().StringVar(&reportsCommonFlags.from, "from", "", "range start (RFC3339)")
		cmd.Flags().StringVar(&reportsCommonFlags.to, "to", "", "range end (RFC3339)")
		cmd.Flags().StringVar(&reportsCommonFlags.course, "course", "", "course code (required for course-scoped reports)")
		cmd.Flags().StringVar(&reportsCommonFlags.format, "format", "csv", "output format: csv, json, ndjson, pdf")
		cmd.Flags().StringVar(&reportsCommonFlags.out, "out", "", "output directory or file path")
		cmd.Flags().BoolVar(&reportsCommonFlags.yes, "yes", false, "confirm FERPA-covered export")
		cmd.Flags().BoolVar(&reportsCommonFlags.wait, "wait", false, "wait for async generation (no-op for sync reports)")
	}

	reportsScheduleCreateCmd.Flags().StringVar(&reportsScheduleCreateFlags.file, "file", "", "schedule JSON (required)")
	_ = reportsScheduleCreateCmd.MarkFlagRequired("file")

	reportsScheduleCmd.AddCommand(reportsScheduleListCmd, reportsScheduleCreateCmd, reportsScheduleDeleteCmd)
	reportsCmd.AddCommand(reportsListCmd, reportsLearningActivityCmd, reportsRunCmd, reportsExportCmd, reportsScheduleCmd)
	rootCmd.AddCommand(reportsCmd)
}

func runReportsList(cmd *cobra.Command, args []string) error {
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"reports": reportCatalog})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tSCOPE\tFORMATS\tASYNC\tDESCRIPTION")
	for _, r := range reportCatalog {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", r.ID, r.Scope, strings.Join(r.Formats, ","), r.Async, r.Description)
	}
	return w.Flush()
}

func runReportsLearningActivity(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	report, raw, err := fetchLearningActivityReport(c, reportsCommonFlags.from, reportsCommonFlags.to)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "range %s → %s\n", report.Range.From.Format(time.RFC3339), report.Range.To.Format(time.RFC3339))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "events=%d users=%d courses=%d\n",
		report.Summary.TotalEvents, report.Summary.UniqueUsers, report.Summary.UniqueCourses)
	return nil
}

func runReportsRun(cmd *cobra.Command, args []string) error {
	if reportsCommonFlags.wait {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "report is synchronous; --wait completed immediately")
	}
	return runReportsExport(cmd, args)
}

func runReportsExport(cmd *cobra.Command, args []string) error {
	reportType, ok := lookupReportType(args[0])
	if !ok {
		return fmt.Errorf("unknown report type %q; run `lextures reports list`", args[0])
	}
	if !reportsCommonFlags.yes {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), ferpaReportsExportWarning)
		return fmt.Errorf("re-run with --yes to export")
	}
	format, err := normalizeExportFormat(reportsCommonFlags.format)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)

	switch reportType.ID {
	case "learning-activity":
		return exportLearningActivity(c, cmd, format)
	case "gradebook", "progress":
		if strings.TrimSpace(reportsCommonFlags.course) == "" {
			return fmt.Errorf("course-scoped report %q requires --course", reportType.ID)
		}
		if format != "pdf" {
			return fmt.Errorf("report %q only supports pdf export", reportType.ID)
		}
		return exportCoursePDF(c, cmd, reportType.ID)
	default:
		return fmt.Errorf("unsupported report type %q", reportType.ID)
	}
}

func exportLearningActivity(c *client.Client, cmd *cobra.Command, format string) error {
	report, raw, err := fetchLearningActivityReport(c, reportsCommonFlags.from, reportsCommonFlags.to)
	if err != nil {
		return err
	}
	var data []byte
	var rowCount int
	filename := "learning-activity-" + time.Now().UTC().Format("20060102")
	switch format {
	case "json", "ndjson":
		data = raw
		rowCount = len(report.ByDay) + len(report.ByEventKind) + len(report.TopCourses) + 3
		if format == "ndjson" {
			filename += ".ndjson"
		} else {
			filename += ".json"
		}
	case "csv":
		data, rowCount, err = learningActivityToCSV(report)
		if err != nil {
			return err
		}
		filename += ".csv"
	case "pdf":
		data, err = downloadBinaryReport(c, "/api/v1/reports/learning-activity/export.pdf")
		if err != nil {
			return err
		}
		filename += ".pdf"
		rowCount = 1
	}
	outPath := resolveExportPath(reportsCommonFlags.out, filename)
	if reportsCommonFlags.out != "" {
		if err := writeExportOutput(outPath, data); err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"path": outPath, "rows": rowCount, "format": format,
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s (%d records)\n", outPath, rowCount)
		return nil
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func exportCoursePDF(c *client.Client, cmd *cobra.Command, reportType string) error {
	path := fmt.Sprintf("/api/v1/courses/%s/reports/%s/export.pdf",
		strings.TrimSpace(reportsCommonFlags.course), reportType)
	data, err := downloadBinaryReport(c, path)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s-%s-%s.pdf", reportType, reportsCommonFlags.course, time.Now().UTC().Format("20060102"))
	outPath := resolveExportPath(reportsCommonFlags.out, filename)
	if reportsCommonFlags.out != "" {
		if err := writeExportOutput(outPath, data); err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"path": outPath, "bytes": len(data)})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s (%d bytes)\n", outPath, len(data))
		return nil
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func runReportsScheduleList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := fetchReportSchedules(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tCADENCE\tENABLED\tNEXT_RUN")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", row.ID, row.ReportType, row.Cadence, row.Enabled, row.NextRunAt)
	}
	return w.Flush()
}

func runReportsScheduleCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := loadJSONFile(reportsScheduleCreateFlags.file)
	if err != nil {
		return err
	}
	body, err := createReportSchedule(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runReportsScheduleDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := deleteReportSchedule(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Schedule deleted.")
	return nil
}