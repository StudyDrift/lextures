package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var reportCardsCmd = &cobra.Command{
	Use:   "report-cards",
	Short: "List, inspect, and export term report cards",
}

var reportCardsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List report cards for a grading period",
	Args:  cobra.ExactArgs(1),
	RunE:  runReportCardsList,
}

var reportCardsGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get one report card by card ID or student",
	RunE:  runReportCardsGet,
}

var reportCardsExportCmd = &cobra.Command{
	Use:   "export <course>",
	Short: "Export report cards for a section or student",
	Args:  cobra.ExactArgs(1),
	RunE:  runReportCardsExport,
}

var reportCardsListFlags struct {
	period string
}

var reportCardsGetFlags struct {
	period string
	card   string
	user   string
}

var reportCardsExportFlags struct {
	period  string
	section string
	user    string
	format  string
	out     string
	yes     bool
}

func init() {
	reportCardsListCmd.Flags().StringVar(&reportCardsListFlags.period, "period", "", "grading period code (required)")
	_ = reportCardsListCmd.MarkFlagRequired("period")

	reportCardsGetCmd.Flags().StringVar(&reportCardsGetFlags.period, "period", "", "grading period code (required)")
	_ = reportCardsGetCmd.MarkFlagRequired("period")
	reportCardsGetCmd.Flags().StringVar(&reportCardsGetFlags.card, "card", "", "report card UUID")
	reportCardsGetCmd.Flags().StringVar(&reportCardsGetFlags.user, "user", "", "student user UUID or email")

	reportCardsExportCmd.Flags().StringVar(&reportCardsExportFlags.period, "period", "", "grading period code (required)")
	_ = reportCardsExportCmd.MarkFlagRequired("period")
	reportCardsExportCmd.Flags().StringVar(&reportCardsExportFlags.section, "section", "", "section UUID to export")
	reportCardsExportCmd.Flags().StringVar(&reportCardsExportFlags.user, "user", "", "student user UUID or email")
	reportCardsExportCmd.Flags().StringVar(&reportCardsExportFlags.format, "format", "pdf", "export format: pdf, csv, or json")
	reportCardsExportCmd.Flags().StringVar(&reportCardsExportFlags.out, "out", "", "output directory (required for pdf)")
	reportCardsExportCmd.Flags().BoolVar(&reportCardsExportFlags.yes, "yes", false, "confirm FERPA-covered bulk export")

	reportCardsCmd.AddCommand(reportCardsListCmd, reportCardsGetCmd, reportCardsExportCmd)
	rootCmd.AddCommand(reportCardsCmd)
}

func runReportCardsList(cmd *cobra.Command, args []string) error {
	cards, raw, err := fetchCourseReportCards(client.New(Cfg.Server, Cfg.APIKey), args[0], reportCardsListFlags.period)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(cards) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No report cards.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTUDENT\tSTATUS\tGRADE\tPERIOD")
	for _, card := range cards {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			stringField(card, "id"),
			stringField(card, "studentId"),
			stringField(card, "status"),
			reportCardGradeLabel(card),
			stringField(card, "gradingPeriod"),
		)
	}
	return w.Flush()
}

func runReportCardsGet(cmd *cobra.Command, args []string) error {
	cardID := strings.TrimSpace(reportCardsGetFlags.card)
	userRef := strings.TrimSpace(reportCardsGetFlags.user)
	if cardID == "" && userRef == "" {
		return fmt.Errorf("specify --card or --user")
	}
	if cardID != "" && userRef != "" {
		return fmt.Errorf("specify only one of --card or --user")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	if cardID != "" {
		return printReportCard(cmd, c, cardID)
	}
	userID, _, err := resolveUserID(c, userRef)
	if err != nil {
		return err
	}
	cards, _, err := fetchCourseReportCards(c, args[0], reportCardsGetFlags.period)
	if err != nil {
		return err
	}
	for _, card := range cards {
		if stringField(card, "studentId") == userID {
			if globalFlags.jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(card)
			}
			printReportCardSummary(cmd, card)
			return nil
		}
	}
	return fmt.Errorf("no report card found for student %s in period %s", userID, reportCardsGetFlags.period)
}

func runReportCardsExport(cmd *cobra.Command, args []string) error {
	if err := confirmFerpaExport(reportCardsExportFlags.yes); err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(reportCardsExportFlags.format))
	switch format {
	case "pdf", "csv", "json":
	default:
		return fmt.Errorf("unsupported format %q: use pdf, csv, or json", reportCardsExportFlags.format)
	}
	if format == "pdf" && strings.TrimSpace(reportCardsExportFlags.out) == "" {
		return fmt.Errorf("--out directory is required for pdf export")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	cards, _, err := fetchCourseReportCards(c, course, reportCardsExportFlags.period)
	if err != nil {
		return err
	}
	cards, err = filterReportCards(c, course, cards)
	if err != nil {
		return err
	}
	if len(cards) == 0 {
		return fmt.Errorf("no report cards matched the export filters")
	}

	switch format {
	case "json":
		payload := map[string]any{"reportCards": cards, "period": reportCardsExportFlags.period}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		outPath := reportCardsExportFlags.out
		if outPath == "" {
			outPath = "-"
		}
		return writeFileOrStdout(cmd, outPath, data)
	case "csv":
		data, err := reportCardsToCSV(cards)
		if err != nil {
			return err
		}
		outPath := reportCardsExportFlags.out
		if outPath == "" {
			outPath = "-"
		}
		return writeFileOrStdout(cmd, outPath, data)
	default:
		return exportReportCardPDFs(cmd, c, cards)
	}
}

func fetchCourseReportCards(c *client.Client, course, period string) ([]reportCardRecord, []byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/report-cards/%s", url.PathEscape(course), url.PathEscape(period))
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing report cards: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ReportCards []reportCardRecord `json:"reportCards"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.ReportCards, body, nil
}

func filterReportCards(c *client.Client, course string, cards []reportCardRecord) ([]reportCardRecord, error) {
	section := strings.TrimSpace(reportCardsExportFlags.section)
	userRef := strings.TrimSpace(reportCardsExportFlags.user)
	if section == "" && userRef == "" {
		return cards, nil
	}
	if section != "" && userRef != "" {
		return nil, fmt.Errorf("specify only one of --section or --user")
	}

	if userRef != "" {
		userID, _, err := resolveUserID(c, userRef)
		if err != nil {
			return nil, err
		}
		var filtered []reportCardRecord
		for _, card := range cards {
			if stringField(card, "studentId") == userID {
				filtered = append(filtered, card)
			}
		}
		return filtered, nil
	}

	allowed, err := studentIDsForSection(c, course, section)
	if err != nil {
		return nil, err
	}
	var filtered []reportCardRecord
	for _, card := range cards {
		if allowed[stringField(card, "studentId")] {
			filtered = append(filtered, card)
		}
	}
	return filtered, nil
}

func studentIDsForSection(c *client.Client, course, sectionID string) (map[string]bool, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/enrollments", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("loading enrollments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Enrollments []struct {
			UserID    string  `json:"userId"`
			Role      string  `json:"role"`
			SectionID *string `json:"sectionId"`
		} `json:"enrollments"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	allowed := map[string]bool{}
	for _, e := range out.Enrollments {
		if e.Role != "student" {
			continue
		}
		if e.SectionID != nil && *e.SectionID == sectionID {
			allowed[e.UserID] = true
		}
	}
	return allowed, nil
}

func exportReportCardPDFs(cmd *cobra.Command, c *client.Client, cards []reportCardRecord) error {
	outDir := strings.TrimSpace(reportCardsExportFlags.out)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	manifest := make([]reportCardManifestEntry, 0, len(cards))
	var written, failed int

	for _, card := range cards {
		cardID := stringField(card, "id")
		studentID := stringField(card, "studentId")
		entry := reportCardManifestEntry{CardID: cardID, StudentID: studentID}

		pdfBytes, filename, err := downloadReportCardPDF(c, cardID, studentID)
		if err != nil {
			entry.Error = err.Error()
			failed++
			manifest = append(manifest, entry)
			continue
		}
		target := filepath.Join(outDir, filename)
		if err := os.WriteFile(target, pdfBytes, 0o644); err != nil {
			entry.Error = err.Error()
			failed++
			manifest = append(manifest, entry)
			continue
		}
		entry.File = target
		entry.Bytes = len(pdfBytes)
		written++
		manifest = append(manifest, entry)
	}

	summary := map[string]any{
		"written":  written,
		"failed":   failed,
		"outDir":   outDir,
		"manifest": manifest,
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %d PDF(s) to %s (%d failed).\n", written, outDir, failed)
	manifestPath := filepath.Join(outDir, "manifest.csv")
	if err := writeReportCardManifest(manifestPath, manifest); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Manifest: %s\n", manifestPath)
	if failed > 0 {
		return fmt.Errorf("%d PDF(s) failed", failed)
	}
	return nil
}

func downloadReportCardPDF(c *client.Client, cardID, studentID string) ([]byte, string, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/report-cards/"+url.PathEscape(cardID)+"/generate-pdf", nil)
	if err != nil {
		return nil, "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", apiErrorBody(resp.StatusCode, body)
	}
	filename := fmt.Sprintf("%s.pdf", cardID)
	if studentID != "" {
		filename = fmt.Sprintf("%s_%s.pdf", studentID, reportCardsExportFlags.period)
	}
	return body, filename, nil
}

type reportCardManifestEntry struct {
	CardID    string `json:"cardId"`
	StudentID string `json:"studentId"`
	File      string `json:"file"`
	Bytes     int    `json:"bytes"`
	Error     string `json:"error,omitempty"`
}

func writeReportCardManifest(path string, entries []reportCardManifestEntry) error {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"card_id", "student_id", "file", "bytes", "error"})
	for _, e := range entries {
		_ = w.Write([]string{e.CardID, e.StudentID, e.File, fmt.Sprintf("%d", e.Bytes), e.Error})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

func printReportCard(cmd *cobra.Command, c *client.Client, cardID string) error {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/report-cards/"+url.PathEscape(cardID)+"/pdf", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("downloading report card: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"cardId": cardID,
			"bytes":  len(body),
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Downloaded PDF for card %s (%d bytes).\n", cardID, len(body))
	return nil
}

func printReportCardSummary(cmd *cobra.Command, card reportCardRecord) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Card:    %s\n", stringField(card, "id"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Student: %s\n", stringField(card, "studentId"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Period:  %s\n", stringField(card, "gradingPeriod"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status:  %s\n", stringField(card, "status"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Grade:   %s\n", reportCardGradeLabel(card))
	if c := stringField(card, "comment"); c != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Comment: %s\n", c)
	}
}

func reportCardGradeLabel(card reportCardRecord) string {
	if g := stringField(card, "letterGrade"); g != "" {
		return g
	}
	if p := numberField(card, "finalGradePct"); p != "" {
		return p + "%"
	}
	return ""
}