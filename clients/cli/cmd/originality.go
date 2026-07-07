package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var originalityCmd = &cobra.Command{
	Use:   "originality",
	Short: "View and manage assignment originality reports",
}

var originalityStatusCmd = &cobra.Command{
	Use:   "status <assignment>",
	Short: "Show originality scan status for a student's submission",
	Args:  cobra.ExactArgs(1),
	RunE:  runOriginalityStatus,
}

var originalityGetCmd = &cobra.Command{
	Use:   "get <assignment>",
	Short: "Get originality score and report reference for a student's submission",
	Args:  cobra.ExactArgs(1),
	RunE:  runOriginalityGet,
}

var originalityListCmd = &cobra.Command{
	Use:   "list <assignment>",
	Short: "List originality scores for all submissions on an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runOriginalityList,
}

var originalitySubmitCmd = &cobra.Command{
	Use:   "submit <assignment>",
	Short: "Trigger or retry an originality check for a student's submission",
	Args:  cobra.ExactArgs(1),
	RunE:  runOriginalitySubmit,
}

var originalityExportCmd = &cobra.Command{
	Use:   "export <assignment>",
	Short: "Export originality scores for an assignment to CSV",
	Args:  cobra.ExactArgs(1),
	RunE:  runOriginalityExport,
}

var originalityStatusFlags struct {
	course string
	user   string
}

var originalityGetFlags struct {
	course string
	user   string
}

var originalityListFlags struct {
	course string
	page   int
	limit  int
}

var originalitySubmitFlags struct {
	course string
	user   string
}

var originalityExportFlags struct {
	course string
	out    string
	yes    bool
}

func init() {
	for _, cmd := range []*cobra.Command{originalityStatusCmd, originalityGetCmd, originalitySubmitCmd} {
		flags := originalitySharedFlags(cmd)
		cmd.Flags().StringVar(flags.course, "course", "", "course code (required)")
		_ = cmd.MarkFlagRequired("course")
		cmd.Flags().StringVar(flags.user, "user", "", "student user UUID (required)")
		_ = cmd.MarkFlagRequired("user")
	}

	originalityListCmd.Flags().StringVar(&originalityListFlags.course, "course", "", "course code (required)")
	_ = originalityListCmd.MarkFlagRequired("course")
	originalityListCmd.Flags().IntVar(&originalityListFlags.page, "page", 1, "page number (1-based)")
	originalityListCmd.Flags().IntVar(&originalityListFlags.limit, "limit", 50, "maximum results per page")

	originalityExportCmd.Flags().StringVar(&originalityExportFlags.course, "course", "", "course code (required)")
	_ = originalityExportCmd.MarkFlagRequired("course")
	originalityExportCmd.Flags().StringVar(&originalityExportFlags.out, "out", "", "output CSV file path (required)")
	_ = originalityExportCmd.MarkFlagRequired("out")
	originalityExportCmd.Flags().BoolVar(&originalityExportFlags.yes, "yes", false, "confirm FERPA-covered originality export")

	originalityCmd.AddCommand(
		originalityStatusCmd,
		originalityGetCmd,
		originalityListCmd,
		originalitySubmitCmd,
		originalityExportCmd,
	)
	rootCmd.AddCommand(originalityCmd)
}

type originalitySharedFlagRefs struct {
	course *string
	user   *string
}

func originalitySharedFlags(cmd *cobra.Command) originalitySharedFlagRefs {
	switch cmd {
	case originalityStatusCmd:
		return originalitySharedFlagRefs{course: &originalityStatusFlags.course, user: &originalityStatusFlags.user}
	case originalityGetCmd:
		return originalitySharedFlagRefs{course: &originalityGetFlags.course, user: &originalityGetFlags.user}
	default:
		return originalitySharedFlagRefs{course: &originalitySubmitFlags.course, user: &originalitySubmitFlags.user}
	}
}

func runOriginalityStatus(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := originalityStatusFlags.course
	itemID := args[0]

	entry, err := resolveAssignmentSubmission(c, courseCode, itemID, originalityStatusFlags.user)
	if err != nil {
		return err
	}
	summaryBody, raw, err := fetchOriginalitySummary(c, courseCode, itemID, entry.ID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	summary := summaryBody.Summary
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "User:       %s\n", entry.SubmittedBy)
	_, _ = fmt.Fprintf(out, "Submission: %s\n", entry.ID)
	if summary.Provider != "" {
		_, _ = fmt.Fprintf(out, "Provider:   %s\n", summary.Provider)
	}
	_, _ = fmt.Fprintf(out, "Status:     %s\n", originalityStatusFromSummary(summary))
	_, _ = fmt.Fprintf(out, "Similarity: %s\n", formatOptionalPercent(summary.SimilarityPct))
	_, _ = fmt.Fprintf(out, "AI score:   %s\n", formatOptionalPercent(summary.AIProbability))
	return nil
}

func runOriginalityGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := originalityGetFlags.course
	itemID := args[0]

	entry, err := resolveAssignmentSubmission(c, courseCode, itemID, originalityGetFlags.user)
	if err != nil {
		return err
	}
	reportsBody, reportsRaw, err := fetchOriginalityReports(c, courseCode, itemID, entry.ID)
	if err != nil {
		return err
	}
	embedBody, _, err := fetchOriginalityEmbed(c, courseCode, itemID, entry.ID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		payload := map[string]any{
			"userId":       entry.SubmittedBy,
			"submissionId": entry.ID,
			"summary":      embedBody.Summary,
			"reports":      reportsBody.Reports,
		}
		if embedBody.EmbedURL != nil {
			payload["reportUrl"] = *embedBody.EmbedURL
		} else {
			payload["reportUrl"] = reportURLFromReports(reportsBody.Reports)
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
	}
	_ = reportsRaw
	best := bestOriginalityReport(reportsBody.Reports)
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "User:       %s\n", entry.SubmittedBy)
	_, _ = fmt.Fprintf(out, "Submission: %s\n", entry.ID)
	if best != nil {
		if best.Provider != "" {
			_, _ = fmt.Fprintf(out, "Provider:   %s\n", best.Provider)
		}
		_, _ = fmt.Fprintf(out, "Status:     %s\n", best.Status)
		_, _ = fmt.Fprintf(out, "Similarity: %s\n", formatOptionalPercent(best.SimilarityPct))
		_, _ = fmt.Fprintf(out, "AI score:   %s\n", formatOptionalPercent(best.AIProbability))
	}
	reportURL := reportURLFromReports(reportsBody.Reports)
	if embedBody.EmbedURL != nil && strings.TrimSpace(*embedBody.EmbedURL) != "" {
		reportURL = strings.TrimSpace(*embedBody.EmbedURL)
	}
	if reportURL != "" {
		_, _ = fmt.Fprintf(out, "Report URL: %s\n", reportURL)
	} else if embedBody.Summary.FullReportUnavailable {
		msg := strings.TrimSpace(embedBody.Summary.FullReportUnavailableMessage)
		if msg == "" {
			msg = "Full provider report link is unavailable."
		}
		_, _ = fmt.Fprintf(out, "Report URL: %s\n", msg)
	}
	return nil
}

func runOriginalityList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := originalityListFlags.course
	itemID := args[0]

	submissionsBody, _, err := fetchAssignmentSubmissions(c, courseCode, itemID, "")
	if err != nil {
		return err
	}
	rows, err := buildOriginalityListRows(c, courseCode, itemID, submissionsBody.Submissions)
	if err != nil {
		return err
	}
	pageRows := paginateSlice(rows, originalityListFlags.page, originalityListFlags.limit)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"items": pageRows,
			"page":  originalityListFlags.page,
			"limit": originalityListFlags.limit,
			"total": len(rows),
		})
	}
	if len(pageRows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No submissions.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USER\tNAME\tSUBMISSION\tPROVIDER\tSTATUS\tSIMILARITY\tAI")
	for _, row := range pageRows {
		name := row.DisplayName
		if name == "" {
			name = row.UserID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.UserID,
			name,
			row.SubmissionID,
			row.Provider,
			row.Status,
			formatOptionalPercent(row.SimilarityPct),
			formatOptionalPercent(row.AIProbability),
		)
	}
	return w.Flush()
}

func runOriginalitySubmit(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := originalitySubmitFlags.course
	itemID := args[0]

	entry, err := resolveAssignmentSubmission(c, courseCode, itemID, originalitySubmitFlags.user)
	if err != nil {
		return err
	}
	body, result, err := postOriginalityRetry(c, courseCode, itemID, entry.ID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		payload := map[string]any{
			"userId":       entry.SubmittedBy,
			"submissionId": entry.ID,
		}
		for key, value := range result {
			payload[key] = value
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
	}
	retried, _ := result["retried"].(float64)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Triggered originality check for submission %s (retried %d provider job(s))\n", entry.ID, int(retried))
	_ = body
	return nil
}

func runOriginalityExport(cmd *cobra.Command, args []string) error {
	if err := confirmOriginalityExport(originalityExportFlags.yes); err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	courseCode := originalityExportFlags.course
	itemID := args[0]

	submissionsBody, _, err := fetchAssignmentSubmissions(c, courseCode, itemID, "")
	if err != nil {
		return err
	}
	rows, err := buildOriginalityListRows(c, courseCode, itemID, submissionsBody.Submissions)
	if err != nil {
		return err
	}
	for i, entry := range submissionsBody.Submissions {
		if strings.TrimSpace(entry.ID) == "" {
			continue
		}
		reportsBody, _, err := fetchOriginalityReports(c, courseCode, itemID, entry.ID)
		if err != nil {
			return err
		}
		rows[i].ReportURL = reportURLFromReports(reportsBody.Reports)
		if rows[i].ReportURL == "" {
			embedBody, _, err := fetchOriginalityEmbed(c, courseCode, itemID, entry.ID)
			if err != nil {
				return err
			}
			if embedBody.EmbedURL != nil {
				rows[i].ReportURL = strings.TrimSpace(*embedBody.EmbedURL)
			}
		}
	}
	csvBytes, err := writeOriginalityCSV(rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile(originalityExportFlags.out, csvBytes, 0o600); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"file":  originalityExportFlags.out,
			"rows":  len(rows),
			"bytes": len(csvBytes),
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported %d row(s) to %s\n", len(rows), originalityExportFlags.out)
	return nil
}