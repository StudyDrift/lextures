package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var enrollmentsCmd = &cobra.Command{
	Use:   "enrollments",
	Short: "Manage course enrollments and rosters",
}

var enrollmentsListFlags struct {
	role    string
	section string
	state   string
}

var enrollmentsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List enrollments for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsList,
}

var enrollmentsExportFlags struct {
	out    string
	format string
	yes    bool
}

var enrollmentsExportCmd = &cobra.Command{
	Use:   "export <course>",
	Short: "Export roster for reconciliation (FERPA-gated)",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsExport,
}

var enrollmentsImportFlags struct {
	file           string
	role           string
	chunkSize      int
	createMissing  bool
}

var enrollmentsImportCmd = &cobra.Command{
	Use:   "import <course>",
	Short: "Bulk enroll users from a CSV roster",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsImport,
}

var enrollmentsAddFlags struct {
	user    string
	role    string
	section string
}

var enrollmentsAddCmd = &cobra.Command{
	Use:   "add <course>",
	Short: "Enroll a single user",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsAdd,
}

var enrollmentsRemoveFlags struct {
	user string
	role string
}

var enrollmentsRemoveCmd = &cobra.Command{
	Use:   "remove <course>",
	Short: "Remove a user's enrollment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsRemove,
}

var enrollmentsSetStateFlags struct {
	user   string
	state  string
	reason string
	role   string
}

var enrollmentsSetStateCmd = &cobra.Command{
	Use:   "set-state <course>",
	Short: "Change enrollment state (conclude, deactivate, reactivate)",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsSetState,
}

var enrollmentsSelfEnrollCmd = &cobra.Command{
	Use:   "self-enroll <course>",
	Short: "Self-enroll the authenticated user in an open course",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnrollmentsSelfEnroll,
}

func init() {
	enrollmentsListCmd.Flags().StringVar(&enrollmentsListFlags.role, "role", "", "filter by role")
	enrollmentsListCmd.Flags().StringVar(&enrollmentsListFlags.section, "section", "", "filter by section UUID or code")
	enrollmentsListCmd.Flags().StringVar(&enrollmentsListFlags.state, "state", "", "filter by state (active, invited, withdrawn, dropped, ...)")

	enrollmentsExportCmd.Flags().StringVar(&enrollmentsExportFlags.out, "out", "", "write export to file instead of stdout")
	enrollmentsExportCmd.Flags().StringVar(&enrollmentsExportFlags.format, "format", "csv", "export format: csv or json")
	enrollmentsExportCmd.Flags().BoolVar(&enrollmentsExportFlags.yes, "yes", false, "confirm FERPA-covered roster export")

	enrollmentsImportCmd.Flags().StringVar(&enrollmentsImportFlags.file, "file", "", "CSV roster file (required)")
	_ = enrollmentsImportCmd.MarkFlagRequired("file")
	enrollmentsImportCmd.Flags().StringVar(&enrollmentsImportFlags.role, "role", "student", "default role when CSV has no role column")
	enrollmentsImportCmd.Flags().IntVar(&enrollmentsImportFlags.chunkSize, "chunk-size", defaultEnrollmentImportChunk, "emails per bulk enroll request")
	enrollmentsImportCmd.Flags().BoolVar(&enrollmentsImportFlags.createMissing, "create-missing", false, "provision missing users before enrolling (delegates to users create)")

	enrollmentsAddCmd.Flags().StringVar(&enrollmentsAddFlags.user, "user", "", "user UUID or email (required)")
	_ = enrollmentsAddCmd.MarkFlagRequired("user")
	enrollmentsAddCmd.Flags().StringVar(&enrollmentsAddFlags.role, "role", "student", "enrollment role")
	enrollmentsAddCmd.Flags().StringVar(&enrollmentsAddFlags.section, "section", "", "section UUID or code")

	enrollmentsRemoveCmd.Flags().StringVar(&enrollmentsRemoveFlags.user, "user", "", "user UUID or email (required)")
	_ = enrollmentsRemoveCmd.MarkFlagRequired("user")
	enrollmentsRemoveCmd.Flags().StringVar(&enrollmentsRemoveFlags.role, "role", "", "disambiguate when user has multiple roles")

	enrollmentsSetStateCmd.Flags().StringVar(&enrollmentsSetStateFlags.user, "user", "", "user UUID or email (required)")
	_ = enrollmentsSetStateCmd.MarkFlagRequired("user")
	enrollmentsSetStateCmd.Flags().StringVar(&enrollmentsSetStateFlags.state, "state", "", "target state: active, withdrawn/concluded, dropped/deactivated (required)")
	_ = enrollmentsSetStateCmd.MarkFlagRequired("state")
	enrollmentsSetStateCmd.Flags().StringVar(&enrollmentsSetStateFlags.reason, "reason", "", "optional reason recorded in state history")
	enrollmentsSetStateCmd.Flags().StringVar(&enrollmentsSetStateFlags.role, "role", "student", "enrollment role to update")

	enrollmentsCmd.AddCommand(
		enrollmentsListCmd,
		enrollmentsExportCmd,
		enrollmentsImportCmd,
		enrollmentsAddCmd,
		enrollmentsRemoveCmd,
		enrollmentsSetStateCmd,
		enrollmentsSelfEnrollCmd,
	)
	rootCmd.AddCommand(enrollmentsCmd)
}

func runEnrollmentsList(cmd *cobra.Command, args []string) error {
	rows, err := fetchEnrollments(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	rows = filterEnrollments(rows, enrollmentsListFlags.role, enrollmentsListFlags.section, enrollmentsListFlags.state)

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tUSER\tNAME\tROLE\tSECTION\tSTATE\tINVITED")
	for _, row := range rows {
		name := ""
		if row.DisplayName != nil {
			name = *row.DisplayName
		}
		section := ""
		if row.SectionCode != nil {
			section = *row.SectionCode
		}
		state := "active"
		if row.State != nil && strings.TrimSpace(*row.State) != "" {
			state = *row.State
		}
		if row.InvitationPending {
			state = "invited"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t\n",
			row.ID, row.UserID, name, row.Role, section, state, row.InvitationPending)
	}
	return w.Flush()
}

func runEnrollmentsExport(cmd *cobra.Command, args []string) error {
	if err := confirmRosterExport(enrollmentsExportFlags.yes); err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(enrollmentsExportFlags.format))
	if format != "csv" && format != "json" {
		return fmt.Errorf("unsupported format %q: use csv or json", enrollmentsExportFlags.format)
	}

	rows, err := fetchEnrollments(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}

	var w io.Writer = cmd.OutOrStdout()
	var file *os.File
	if enrollmentsExportFlags.out != "" {
		file, err = os.Create(enrollmentsExportFlags.out)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		w = file
	}

	if format == "json" || globalFlags.jsonOut {
		return json.NewEncoder(w).Encode(rows)
	}
	return writeEnrollmentsExportCSV(w, rows)
}

func runEnrollmentsImport(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(enrollmentsImportFlags.file)
	if err != nil {
		return fmt.Errorf("reading roster file: %w", err)
	}
	rows, err := parseRosterCSV(raw, enrollmentsImportFlags.role)
	if err != nil {
		return err
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	summary, err := importRosterRows(c, args[0], rows, enrollmentsImportFlags.chunkSize, enrollmentsImportFlags.createMissing, func(done, total int) {
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
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Import complete: added=%d updated=%d skipped=%d not_found=%d failed=%d\n",
		summary.Added, summary.Updated, summary.Skipped, summary.NotFound, summary.Failed)
	for _, msg := range summary.Errors {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  error: %s\n", msg)
	}
	return nil
}

func runEnrollmentsAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	email, err := resolveUserEmail(c, enrollmentsAddFlags.user)
	if err != nil {
		return fmt.Errorf("resolving user: %w", err)
	}

	resp, err := postEnrollments(c, args[0], []string{email}, enrollmentsAddFlags.role)
	if err != nil {
		return err
	}
	if len(resp.AlreadyEnrolled) > 0 {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"status":  "already_enrolled",
				"course":  args[0],
				"user":    enrollmentsAddFlags.user,
				"role":    enrollmentsAddFlags.role,
			})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "warning: already enrolled")
	} else if len(resp.NotFound) > 0 {
		return fmt.Errorf("user %q not found in course org", email)
	} else if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"status": "enrolled",
			"course": args[0],
			"user":   enrollmentsAddFlags.user,
			"role":   enrollmentsAddFlags.role,
			"added":  resp.Added,
		})
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enrolled %s as %s in %s\n", email, enrollmentsAddFlags.role, args[0])
	}

	if strings.TrimSpace(enrollmentsAddFlags.section) == "" {
		return nil
	}
	en, err := resolveEnrollmentForUser(c, args[0], enrollmentsAddFlags.user, enrollmentsAddFlags.role)
	if err != nil {
		return err
	}
	sections, err := fetchSections(c, args[0])
	if err != nil {
		return err
	}
	sec, err := resolveSectionRef(sections, enrollmentsAddFlags.section)
	if err != nil {
		return err
	}
	if err := transferEnrollmentSection(c, en.ID, sec.ID); err != nil {
		return err
	}
	if !globalFlags.jsonOut {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Moved enrollment to section %s\n", sec.SectionCode)
	}
	return nil
}

func runEnrollmentsRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	en, err := resolveEnrollmentForUser(c, args[0], enrollmentsRemoveFlags.user, enrollmentsRemoveFlags.role)
	if err != nil {
		return err
	}
	if err := deleteEnrollment(c, args[0], en.ID); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"removed": en.ID,
			"course":  args[0],
			"user":    enrollmentsRemoveFlags.user,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed enrollment %s from %s\n", en.ID, args[0])
	return nil
}

func runEnrollmentsSetState(cmd *cobra.Command, args []string) error {
	state, err := normalizeEnrollmentStateAlias(enrollmentsSetStateFlags.state)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	en, err := resolveEnrollmentForUser(c, args[0], enrollmentsSetStateFlags.user, enrollmentsSetStateFlags.role)
	if err != nil {
		return err
	}
	out, err := patchEnrollmentState(c, args[0], en.ID, state, enrollmentsSetStateFlags.reason)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enrollment %s state set to %s\n", out.ID, out.State)
	return nil
}

func runEnrollmentsSelfEnroll(cmd *cobra.Command, args []string) error {
	body, err := postSelfEnroll(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, fmt.Sprintf("Self-enrolled in %s", args[0]))
}