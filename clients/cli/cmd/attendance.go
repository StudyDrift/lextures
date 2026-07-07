package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var attendanceCmd = &cobra.Command{
	Use:   "attendance",
	Short: "Record and export course attendance",
}

var attendanceListFlags struct {
	date string
}

var attendanceListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List attendance sessions for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttendanceList,
}

var attendanceRecordFlags struct {
	date    string
	user    string
	status  string
	period  string
	section string
}

var attendanceRecordCmd = &cobra.Command{
	Use:   "record <course>",
	Short: "Record attendance for one student",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttendanceRecord,
}

var attendanceImportFlags struct {
	file      string
	section   string
	chunkSize int
}

var attendanceImportCmd = &cobra.Command{
	Use:   "import <course>",
	Short: "Bulk import attendance from a CSV file",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttendanceImport,
}

var attendanceExportFlags struct {
	from   string
	to     string
	out    string
	format string
	yes    bool
}

var attendanceExportCmd = &cobra.Command{
	Use:   "export <course>",
	Short: "Export attendance records for a date range (FERPA-gated)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttendanceExport,
}

var attendanceSummaryFlags struct {
	from string
	to   string
}

var attendanceSummaryCmd = &cobra.Command{
	Use:   "summary <course>",
	Short: "Roll up attendance counts per student",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttendanceSummary,
}

func init() {
	attendanceListCmd.Flags().StringVar(&attendanceListFlags.date, "date", "", "filter by session date (YYYY-MM-DD)")

	attendanceRecordCmd.Flags().StringVar(&attendanceRecordFlags.date, "date", "", "session date (YYYY-MM-DD, default today)")
	attendanceRecordCmd.Flags().StringVar(&attendanceRecordFlags.user, "user", "", "student UUID or email (required)")
	_ = attendanceRecordCmd.MarkFlagRequired("user")
	attendanceRecordCmd.Flags().StringVar(&attendanceRecordFlags.status, "status", "present", "status: present, absent, tardy, excused")
	attendanceRecordCmd.Flags().StringVar(&attendanceRecordFlags.period, "period", "", "optional period/block label")
	attendanceRecordCmd.Flags().StringVar(&attendanceRecordFlags.section, "section", "", "optional section UUID or code")

	attendanceImportCmd.Flags().StringVar(&attendanceImportFlags.file, "file", "", "CSV file (student,date,period,status) (required)")
	_ = attendanceImportCmd.MarkFlagRequired("file")
	attendanceImportCmd.Flags().StringVar(&attendanceImportFlags.section, "section", "", "optional section UUID or code for new sessions")
	attendanceImportCmd.Flags().IntVar(&attendanceImportFlags.chunkSize, "chunk-size", defaultAttendanceImportChunk, "records per save request")

	attendanceExportCmd.Flags().StringVar(&attendanceExportFlags.from, "from", "", "start date (YYYY-MM-DD)")
	attendanceExportCmd.Flags().StringVar(&attendanceExportFlags.to, "to", "", "end date (YYYY-MM-DD)")
	attendanceExportCmd.Flags().StringVar(&attendanceExportFlags.out, "out", "", "write export to file instead of stdout")
	attendanceExportCmd.Flags().StringVar(&attendanceExportFlags.format, "format", "csv", "export format: csv or json")
	attendanceExportCmd.Flags().BoolVar(&attendanceExportFlags.yes, "yes", false, "confirm FERPA-covered attendance export")

	attendanceSummaryCmd.Flags().StringVar(&attendanceSummaryFlags.from, "from", "", "start date (YYYY-MM-DD)")
	attendanceSummaryCmd.Flags().StringVar(&attendanceSummaryFlags.to, "to", "", "end date (YYYY-MM-DD)")

	attendanceCmd.AddCommand(
		attendanceListCmd,
		attendanceRecordCmd,
		attendanceImportCmd,
		attendanceExportCmd,
		attendanceSummaryCmd,
	)
	rootCmd.AddCommand(attendanceCmd)
}

func runAttendanceList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sessions, err := fetchAttendanceSessions(c, args[0])
	if err != nil {
		return err
	}
	if strings.TrimSpace(attendanceListFlags.date) != "" {
		filtered := make([]attendanceSession, 0)
		for _, sess := range sessions {
			if sess.SessionDate == strings.TrimSpace(attendanceListFlags.date) {
				filtered = append(filtered, sess)
			}
		}
		sessions = filtered
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(sessions)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tDATE\tTITLE\tSTATUS\tMETHOD")
	for _, sess := range sessions {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			sess.ID, sess.SessionDate, sess.Title, sess.Status, sess.CollectionMethod)
	}
	return w.Flush()
}

func runAttendanceRecord(cmd *cobra.Command, args []string) error {
	date := strings.TrimSpace(attendanceRecordFlags.date)
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid --date %q (use YYYY-MM-DD)", date)
	}
	status, err := normalizeAttendanceStatus(attendanceRecordFlags.status)
	if err != nil {
		return err
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	studentID, err := resolveStudentUserID(c, attendanceRecordFlags.user)
	if err != nil {
		return err
	}

	sectionID := ""
	if strings.TrimSpace(attendanceRecordFlags.section) != "" {
		sec, err := resolveSectionForCourse(c, args[0], attendanceRecordFlags.section)
		if err != nil {
			return err
		}
		sectionID = sec.ID
	}

	sess, created, err := findOrCreateSessionForDate(c, args[0], date, attendanceRecordFlags.period, sectionID)
	if err != nil {
		return err
	}
	saved, err := putAttendanceRecords(c, args[0], sess.ID, []map[string]string{
		{"studentUserId": studentID, "status": status, "source": "instructor"},
	})
	if err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"sessionId":       sess.ID,
			"sessionCreated":  created,
			"studentUserId":   studentID,
			"status":          status,
			"saved":           saved,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Recorded %s as %s for %s (session %s)\n",
		attendanceRecordFlags.user, status, date, sess.ID)
	return nil
}

func runAttendanceImport(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(attendanceImportFlags.file)
	if err != nil {
		return fmt.Errorf("reading attendance file: %w", err)
	}
	rows, err := parseAttendanceCSV(raw)
	if err != nil {
		return err
	}

	sectionID := ""
	if strings.TrimSpace(attendanceImportFlags.section) != "" {
		c := client.New(Cfg.Server, Cfg.APIKey)
		sec, err := resolveSectionForCourse(c, args[0], attendanceImportFlags.section)
		if err != nil {
			return err
		}
		sectionID = sec.ID
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	summary, err := importAttendanceRows(c, args[0], rows, sectionID, attendanceImportFlags.chunkSize, func(done, total int) {
		if !globalFlags.jsonOut && total > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Processed %d/%d rows\r", done, total)
		}
	})
	if err != nil {
		return err
	}
	if !globalFlags.jsonOut && len(rows) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"Import complete: sessions_created=%d records_saved=%d updated=%d failed=%d\n",
		summary.SessionsCreated, summary.RecordsSaved, summary.Updated, summary.Failed)
	for _, msg := range summary.Errors {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  error: %s\n", msg)
	}
	return nil
}

func runAttendanceExport(cmd *cobra.Command, args []string) error {
	if err := confirmAttendanceExport(attendanceExportFlags.yes); err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(attendanceExportFlags.format))
	if format != "csv" && format != "json" {
		return fmt.Errorf("unsupported format %q: use csv or json", attendanceExportFlags.format)
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	sessions, err := fetchAttendanceSessions(c, args[0])
	if err != nil {
		return err
	}
	sessions = filterSessionsByDateRange(sessions, attendanceExportFlags.from, attendanceExportFlags.to)
	rows, err := collectAttendanceExportRows(c, args[0], sessions)
	if err != nil {
		return err
	}

	var w = cmd.OutOrStdout()
	if attendanceExportFlags.out != "" {
		file, err := os.Create(attendanceExportFlags.out)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		w = file
	}

	if format == "json" || globalFlags.jsonOut {
		return json.NewEncoder(w).Encode(rows)
	}
	return writeAttendanceExportCSV(w, rows)
}

func runAttendanceSummary(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	sessions, err := fetchAttendanceSessions(c, args[0])
	if err != nil {
		return err
	}
	sessions = filterSessionsByDateRange(sessions, attendanceSummaryFlags.from, attendanceSummaryFlags.to)
	exportRows, err := collectAttendanceExportRows(c, args[0], sessions)
	if err != nil {
		return err
	}
	rows := buildAttendanceSummary(exportRows)

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STUDENT\tNAME\tPRESENT\tABSENT\tTARDY\tEXCUSED\tTOTAL")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			row.StudentID, row.StudentName, row.Present, row.Absent, row.Tardy, row.Excused, row.Total)
	}
	return w.Flush()
}