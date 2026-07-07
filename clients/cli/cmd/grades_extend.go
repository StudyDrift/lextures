package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

const ferpaGradebookExportWarning = `WARNING: Gradebook export contains FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

const ferpaFinalGradeSubmitWarning = `WARNING: Final grade submission is typically irreversible.
Re-run with --yes to confirm you want to submit final grades.`

// gradebookCmd is the top-level gradebook command group.
var gradebookCmd = &cobra.Command{
	Use:   "gradebook",
	Short: "Read and bulk-import course gradebooks",
}

var gradebookGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get the gradebook matrix (students × items)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGradebookGet,
}

var gradebookImportCmd = &cobra.Command{
	Use:   "import <course>",
	Short: "Bulk-import grades from a CSV file",
	Args:  cobra.ExactArgs(1),
	RunE:  runGradebookImport,
}

var gradebookGetFlags struct {
	format string
	yes    bool
}

var gradebookImportFlags struct {
	file string
}

// finalGradesCmd is the top-level final-grades command group.
var finalGradesCmd = &cobra.Command{
	Use:   "final-grades",
	Short: "Preview, override, and submit final course grades",
}

var finalGradesListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List computed and submitted final grades",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinalGradesList,
}

var finalGradesSetCmd = &cobra.Command{
	Use:   "set <course>",
	Short: "Submit final grades with an override for one enrollment",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinalGradesSet,
}

var finalGradesSubmitCmd = &cobra.Command{
	Use:   "submit <course>",
	Short: "Submit final grades for all students",
	Args:  cobra.ExactArgs(1),
	RunE:  runFinalGradesSubmit,
}

var finalGradesSetFlags struct {
	enrollment string
	grade      string
	reason     string
	method     string
	yes        bool
}

var finalGradesSubmitFlags struct {
	file   string
	method string
	yes    bool
}

// gradingBacklogCmd lists ungraded work.
var gradingBacklogCmd = &cobra.Command{
	Use:   "grading-backlog",
	Short: "View items that still need grading",
}

var gradingBacklogListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List assignments and quizzes with ungraded submissions",
	Args:  cobra.ExactArgs(1),
	RunE:  runGradingBacklogList,
}

// gradesSchemeCmd manages the grading scheme.
var gradesSchemeCmd = &cobra.Command{
	Use:   "scheme",
	Short: "View or update the course grading scheme",
}

var gradesSchemeGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get the active grading scheme",
	Args:  cobra.ExactArgs(1),
	RunE:  runGradesSchemeGet,
}

var gradesSchemeSetCmd = &cobra.Command{
	Use:   "set <course>",
	Short: "Set the active grading scheme",
	Args:  cobra.ExactArgs(1),
	RunE:  runGradesSchemeSet,
}

var gradesSchemeSetFlags struct {
	file      string
	name      string
	schemeType string
	scaleJSON string
}

var gradesCurveFlags struct {
	course     string
	assignment string
	method     string
	bonus      float64
	targetMean float64
	targetMax  float64
	minimum    float64
	allowAbove bool
	dryRun     bool
}

var gradesCurveCmd = &cobra.Command{
	Use:   "curve",
	Short: "Apply or preview a grade curve on an assignment",
	RunE:  runGradesCurve,
}

var gradesWhatIfFlags struct {
	course    string
	user      string
	override  []string
	file      string
}

var gradesWhatIfCmd = &cobra.Command{
	Use:   "what-if",
	Short: "Project a student's final grade with hypothetical scores",
	RunE:  runGradesWhatIf,
}

var gradesHistoryFlags struct {
	course     string
	assignment string
	user       string
}

var gradesHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show grade change history for a student on an assignment",
	RunE:  runGradesHistory,
}

type gradingSchemeEnvelope struct {
	Scheme *struct {
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Type      string          `json:"type"`
		ScaleJSON json.RawMessage `json:"scaleJson"`
	} `json:"scheme"`
}

type gradingSettingsResponse struct {
	GradingScale     string                  `json:"gradingScale"`
	AssignmentGroups []assignmentGroupWeight `json:"assignmentGroups"`
}

type gradingBacklogResponse struct {
	Items []struct {
		ItemID          string `json:"itemId"`
		ItemType        string `json:"itemType"`
		AssignmentTitle string `json:"assignmentTitle"`
		UngradedCount   int64  `json:"ungradedCount"`
	} `json:"items"`
}

type finalGradesPreviewResponse struct {
	Grades []struct {
		EnrollmentID     string  `json:"enrollmentId"`
		UserID           string  `json:"userId"`
		DisplayName      string  `json:"displayName"`
		State            string  `json:"state"`
		ComputedGrade    string  `json:"computedGrade"`
		FinalGrade       string  `json:"finalGrade"`
		OverrideReason   string  `json:"overrideReason,omitempty"`
		AlreadySubmitted bool    `json:"alreadySubmitted"`
		SubmittedAt      *string `json:"submittedAt,omitempty"`
	} `json:"grades"`
}

type finalGradeOverride struct {
	EnrollmentID string `json:"enrollmentId"`
	Grade        string `json:"grade"`
	Reason       string `json:"reason,omitempty"`
}

type finalGradesSubmitResponse struct {
	Count       int    `json:"count"`
	DownloadURL string `json:"downloadUrl,omitempty"`
}

type curvePreviewResponse struct {
	Preview struct {
		EligibleCount int `json:"eligibleCount"`
		MeanBefore    *float64 `json:"meanBefore"`
		MeanAfter     *float64 `json:"meanAfter"`
		Results       []struct {
			StudentID     string  `json:"studentId"`
			RawScore      float64 `json:"rawScore"`
			AdjustedScore float64 `json:"adjustedScore"`
			Delta         float64 `json:"delta"`
			Changed       bool    `json:"changed"`
		} `json:"results"`
	} `json:"preview"`
	MaxPoints float64 `json:"maxPoints"`
}

type gradeImportSummary struct {
	Posted  int      `json:"posted"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
	Skipped int      `json:"skipped"`
}

func init() {
	gradebookGetCmd.Flags().StringVar(&gradebookGetFlags.format, "format", "csv", "output format: csv or json")
	gradebookGetCmd.Flags().BoolVar(&gradebookGetFlags.yes, "yes", false, "confirm FERPA-covered gradebook export")

	gradebookImportCmd.Flags().StringVar(&gradebookImportFlags.file, "file", "", "CSV file with grades (required)")
	_ = gradebookImportCmd.MarkFlagRequired("file")

	finalGradesSetCmd.Flags().StringVar(&finalGradesSetFlags.enrollment, "enrollment", "", "enrollment UUID to override (required)")
	_ = finalGradesSetCmd.MarkFlagRequired("enrollment")
	finalGradesSetCmd.Flags().StringVar(&finalGradesSetFlags.grade, "grade", "", "override final grade (required)")
	_ = finalGradesSetCmd.MarkFlagRequired("grade")
	finalGradesSetCmd.Flags().StringVar(&finalGradesSetFlags.reason, "reason", "", "override reason")
	finalGradesSetCmd.Flags().StringVar(&finalGradesSetFlags.method, "method", "csv", "submission method: csv or ags")
	finalGradesSetCmd.Flags().BoolVar(&finalGradesSetFlags.yes, "yes", false, "confirm final grade submission")

	finalGradesSubmitCmd.Flags().StringVar(&finalGradesSubmitFlags.file, "file", "", "JSON file with override rows")
	finalGradesSubmitCmd.Flags().StringVar(&finalGradesSubmitFlags.method, "method", "csv", "submission method: csv or ags")
	finalGradesSubmitCmd.Flags().BoolVar(&finalGradesSubmitFlags.yes, "yes", false, "confirm final grade submission")

	gradesSchemeSetCmd.Flags().StringVar(&gradesSchemeSetFlags.file, "file", "", "JSON body file (name, type, scaleJson)")
	gradesSchemeSetCmd.Flags().StringVar(&gradesSchemeSetFlags.name, "name", "", "scheme name")
	gradesSchemeSetCmd.Flags().StringVar(&gradesSchemeSetFlags.schemeType, "type", "", "scheme type (letter, percent, pass_fail, …)")
	gradesSchemeSetCmd.Flags().StringVar(&gradesSchemeSetFlags.scaleJSON, "scale-json", "", "scale JSON object")

	gradesCurveCmd.Flags().StringVar(&gradesCurveFlags.course, "course", "", "course code (required)")
	_ = gradesCurveCmd.MarkFlagRequired("course")
	gradesCurveCmd.Flags().StringVar(&gradesCurveFlags.assignment, "assignment", "", "assignment or quiz item UUID (required)")
	_ = gradesCurveCmd.MarkFlagRequired("assignment")
	gradesCurveCmd.Flags().StringVar(&gradesCurveFlags.method, "method", "", "curve method: linear, sqrt, flat, minimum, custom (required)")
	_ = gradesCurveCmd.MarkFlagRequired("method")
	gradesCurveCmd.Flags().Float64Var(&gradesCurveFlags.bonus, "bonus", 0, "flat bonus points (flat_bonus)")
	gradesCurveCmd.Flags().Float64Var(&gradesCurveFlags.targetMean, "target-mean", 0, "target mean (linear_scale)")
	gradesCurveCmd.Flags().Float64Var(&gradesCurveFlags.targetMax, "target-max", 0, "target max (linear_scale)")
	gradesCurveCmd.Flags().Float64Var(&gradesCurveFlags.minimum, "minimum", 0, "minimum score (set_minimum)")
	gradesCurveCmd.Flags().BoolVar(&gradesCurveFlags.allowAbove, "allow-above-max", false, "allow curved scores above max points")
	gradesCurveCmd.Flags().BoolVar(&gradesCurveFlags.dryRun, "dry-run", false, "preview curve without applying")

	gradesWhatIfCmd.Flags().StringVar(&gradesWhatIfFlags.course, "course", "", "course code (required)")
	_ = gradesWhatIfCmd.MarkFlagRequired("course")
	gradesWhatIfCmd.Flags().StringVar(&gradesWhatIfFlags.user, "user", "", "student user ID (required)")
	_ = gradesWhatIfCmd.MarkFlagRequired("user")
	gradesWhatIfCmd.Flags().StringArrayVar(&gradesWhatIfFlags.override, "override", nil, "hypothetical score override item-id=score")
	gradesWhatIfCmd.Flags().StringVar(&gradesWhatIfFlags.file, "file", "", "JSON file with item-id → score overrides")

	gradesHistoryCmd.Flags().StringVar(&gradesHistoryFlags.course, "course", "", "course code (required)")
	_ = gradesHistoryCmd.MarkFlagRequired("course")
	gradesHistoryCmd.Flags().StringVar(&gradesHistoryFlags.assignment, "assignment", "", "assignment item UUID (required)")
	_ = gradesHistoryCmd.MarkFlagRequired("assignment")
	gradesHistoryCmd.Flags().StringVar(&gradesHistoryFlags.user, "user", "", "student user ID (required)")
	_ = gradesHistoryCmd.MarkFlagRequired("user")

	gradebookCmd.AddCommand(gradebookGetCmd, gradebookImportCmd)
	finalGradesCmd.AddCommand(finalGradesListCmd, finalGradesSetCmd, finalGradesSubmitCmd)
	gradingBacklogCmd.AddCommand(gradingBacklogListCmd)
	gradesSchemeCmd.AddCommand(gradesSchemeGetCmd, gradesSchemeSetCmd)
}

func confirmFinalGradeSubmit(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaFinalGradeSubmitWarning)
}

func confirmGradebookExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaGradebookExportWarning)
}

func runGradebookGet(cmd *cobra.Command, args []string) error {
	if err := confirmGradebookExport(gradebookGetFlags.yes); err != nil {
		return err
	}
	format := strings.ToLower(strings.TrimSpace(gradebookGetFlags.format))
	if format != "csv" && format != "json" {
		return fmt.Errorf("unsupported format %q: use csv or json", gradebookGetFlags.format)
	}

	grid, err := fetchGradebookGrid(args[0])
	if err != nil {
		return err
	}

	if format == "json" {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(grid)
	}
	return writeGradebookMatrixCSV(cmd.OutOrStdout(), grid)
}

func writeGradebookMatrixCSV(w io.Writer, grid *gradebookGrid) error {
	cw := csv.NewWriter(w)
	header := []string{"student_id", "student_name"}
	for _, col := range grid.Columns {
		header = append(header, col.Title)
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, student := range grid.Students {
		row := []string{student.UserID, student.DisplayName}
		studentGrades := grid.Grades[student.UserID]
		for _, col := range grid.Columns {
			score := ""
			if studentGrades != nil {
				score = studentGrades[col.ID]
			}
			row = append(row, score)
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func runGradebookImport(cmd *cobra.Command, args []string) error {
	courseCode := args[0]
	f, err := os.Open(gradebookImportFlags.file)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()

	grid, err := fetchGradebookGrid(courseCode)
	if err != nil {
		return err
	}

	gradesMap, summary, err := parseGradeImportCSV(f, grid)
	if err != nil {
		return err
	}
	if len(gradesMap) == 0 {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Import complete: 0 posted, %d failed, %d skipped\n",
			summary.Failed, summary.Skipped)
		for _, e := range summary.Errors {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", e)
		}
		return nil
	}

	if err := putGradebookGrades(client.New(Cfg.Server, Cfg.APIKey), courseCode, gradesMap); err != nil {
		return err
	}
	summary.Posted = countGradeCells(gradesMap)

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Import complete: %d posted, %d failed, %d skipped\n",
		summary.Posted, summary.Failed, summary.Skipped)
	for _, e := range summary.Errors {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", e)
	}
	return nil
}

func countGradeCells(grades map[string]map[string]string) int {
	n := 0
	for _, items := range grades {
		n += len(items)
	}
	return n
}

func parseGradeImportCSV(r io.Reader, grid *gradebookGrid) (map[string]map[string]string, gradeImportSummary, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, gradeImportSummary{}, fmt.Errorf("reading CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, gradeImportSummary{}, fmt.Errorf("CSV is empty")
	}

	header := normalizeCSVHeader(records[0])
	studentIdx := headerIndex(header, "student_id")
	itemIDIdx := headerIndex(header, "item_id")
	titleIdx := firstHeaderIndex(header, "assignment_title", "item_title", "title")
	scoreIdx := headerIndex(header, "score")
	if studentIdx < 0 || scoreIdx < 0 {
		return nil, gradeImportSummary{}, fmt.Errorf("CSV must include student_id and score columns")
	}
	if itemIDIdx < 0 && titleIdx < 0 {
		return nil, gradeImportSummary{}, fmt.Errorf("CSV must include item_id or assignment_title column")
	}

	studentIDs := make(map[string]struct{}, len(grid.Students))
	for _, s := range grid.Students {
		studentIDs[s.UserID] = struct{}{}
	}
	colByID := make(map[string]gradeColumn, len(grid.Columns))
	colByTitle := make(map[string]gradeColumn, len(grid.Columns))
	for _, col := range grid.Columns {
		colByID[col.ID] = col
		colByTitle[strings.ToLower(strings.TrimSpace(col.Title))] = col
	}

	grades := make(map[string]map[string]string)
	var summary gradeImportSummary

	for rowNum, rec := range records[1:] {
		line := rowNum + 2
		if len(rec) == 0 || allEmpty(rec) {
			summary.Skipped++
			continue
		}
		studentID := strings.TrimSpace(safeField(rec, studentIdx))
		score := strings.TrimSpace(safeField(rec, scoreIdx))
		if studentID == "" {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: missing student_id", line))
			continue
		}
		if _, ok := studentIDs[studentID]; !ok {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: unknown student_id %s", line, studentID))
			continue
		}
		if score == "" {
			summary.Skipped++
			continue
		}
		if _, err := strconv.ParseFloat(score, 64); err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: invalid score %q", line, score))
			continue
		}

		itemID := ""
		if itemIDIdx >= 0 {
			itemID = strings.TrimSpace(safeField(rec, itemIDIdx))
		}
		if itemID == "" && titleIdx >= 0 {
			title := strings.ToLower(strings.TrimSpace(safeField(rec, titleIdx)))
			if col, ok := colByTitle[title]; ok {
				itemID = col.ID
			}
		}
		if itemID == "" {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: could not resolve item", line))
			continue
		}
		if _, ok := colByID[itemID]; !ok {
			summary.Failed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: unknown item_id %s", line, itemID))
			continue
		}

		if grades[studentID] == nil {
			grades[studentID] = make(map[string]string)
		}
		grades[studentID][itemID] = score
	}
	return grades, summary, nil
}

func normalizeCSVHeader(row []string) []string {
	out := make([]string, len(row))
	for i, h := range row {
		out[i] = strings.ToLower(strings.TrimSpace(h))
	}
	return out
}

func headerIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

func firstHeaderIndex(header []string, names ...string) int {
	for _, name := range names {
		if idx := headerIndex(header, name); idx >= 0 {
			return idx
		}
	}
	return -1
}

func safeField(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return rec[idx]
}

func allEmpty(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

func runFinalGradesList(cmd *cobra.Command, args []string) error {
	preview, raw, err := fetchFinalGradesPreview(args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(preview.Grades) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No final grades.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STUDENT\tSTATE\tCOMPUTED\tFINAL\tSUBMITTED")
	for _, g := range preview.Grades {
		submitted := "no"
		if g.AlreadySubmitted {
			submitted = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			g.DisplayName, g.State, g.ComputedGrade, g.FinalGrade, submitted)
	}
	return w.Flush()
}

func fetchFinalGradesPreview(courseCode string) (finalGradesPreviewResponse, []byte, error) {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/final-grades/preview", nil)
	if err != nil {
		return finalGradesPreviewResponse{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return finalGradesPreviewResponse{}, nil, fmt.Errorf("fetching final grades: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return finalGradesPreviewResponse{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return finalGradesPreviewResponse{}, nil, fmt.Errorf("permission denied (403): you do not have permission to view final grades")
	}
	if resp.StatusCode != http.StatusOK {
		return finalGradesPreviewResponse{}, nil, apiErrorBody(resp.StatusCode, body)
	}
	var preview finalGradesPreviewResponse
	if err := json.Unmarshal(body, &preview); err != nil {
		return finalGradesPreviewResponse{}, nil, fmt.Errorf("decoding response: %w", err)
	}
	return preview, body, nil
}

func runFinalGradesSet(cmd *cobra.Command, args []string) error {
	if err := confirmFinalGradeSubmit(finalGradesSetFlags.yes); err != nil {
		return err
	}
	overrides := []finalGradeOverride{{
		EnrollmentID: finalGradesSetFlags.enrollment,
		Grade:        finalGradesSetFlags.grade,
		Reason:       finalGradesSetFlags.reason,
	}}
	return submitFinalGrades(cmd, args[0], finalGradesSetFlags.method, overrides)
}

func runFinalGradesSubmit(cmd *cobra.Command, args []string) error {
	if err := confirmFinalGradeSubmit(finalGradesSubmitFlags.yes); err != nil {
		return err
	}
	var overrides []finalGradeOverride
	if finalGradesSubmitFlags.file != "" {
		var loaded []finalGradeOverride
		raw, err := os.ReadFile(finalGradesSubmitFlags.file)
		if err != nil {
			return fmt.Errorf("reading overrides file: %w", err)
		}
		if err := json.Unmarshal(raw, &loaded); err != nil {
			return fmt.Errorf("parsing overrides JSON: %w", err)
		}
		overrides = loaded
	}
	return submitFinalGrades(cmd, args[0], finalGradesSubmitFlags.method, overrides)
}

func submitFinalGrades(cmd *cobra.Command, courseCode, method string, overrides []finalGradeOverride) error {
	method = strings.TrimSpace(method)
	if method == "" {
		method = "csv"
	}
	payload := map[string]any{
		"method":    method,
		"overrides": overrides,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/final-grades/submit", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("submitting final grades: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied (403): you do not have permission to submit final grades")
	}
	if resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var result finalGradesSubmitResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submitted %d final grades\n", result.Count)
	if result.DownloadURL != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Export: %s\n", result.DownloadURL)
	}
	return nil
}

func runGradingBacklogList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+args[0]+"/grading-backlog", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("fetching grading backlog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied (403): you do not have permission to view the grading backlog")
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var backlog gradingBacklogResponse
	if err := json.Unmarshal(body, &backlog); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(backlog.Items) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Grading backlog is empty.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tTITLE\tUNGRADED")
	for _, item := range backlog.Items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\n", item.ItemType, item.AssignmentTitle, item.UngradedCount)
	}
	return w.Flush()
}

func runGradesSchemeGet(cmd *cobra.Command, args []string) error {
	scheme, raw, err := fetchGradingScheme(args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if scheme.Scheme == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grading scheme configured.")
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nName: %s\nType: %s\nScale: %s\n",
		scheme.Scheme.ID, scheme.Scheme.Name, scheme.Scheme.Type, string(scheme.Scheme.ScaleJSON))
	return nil
}

func fetchGradingScheme(courseCode string) (gradingSchemeEnvelope, []byte, error) {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/grading-scheme", nil)
	if err != nil {
		return gradingSchemeEnvelope{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return gradingSchemeEnvelope{}, nil, fmt.Errorf("fetching grading scheme: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return gradingSchemeEnvelope{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return gradingSchemeEnvelope{}, nil, fmt.Errorf("permission denied (403): you do not have permission to view grading scheme")
	}
	if resp.StatusCode != http.StatusOK {
		return gradingSchemeEnvelope{}, nil, apiErrorBody(resp.StatusCode, body)
	}
	var scheme gradingSchemeEnvelope
	if err := json.Unmarshal(body, &scheme); err != nil {
		return gradingSchemeEnvelope{}, nil, fmt.Errorf("decoding response: %w", err)
	}
	return scheme, body, nil
}

func runGradesSchemeSet(cmd *cobra.Command, args []string) error {
	var payload map[string]any
	if gradesSchemeSetFlags.file != "" {
		raw, err := os.ReadFile(gradesSchemeSetFlags.file)
		if err != nil {
			return fmt.Errorf("reading scheme file: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("parsing scheme JSON: %w", err)
		}
	} else {
		if strings.TrimSpace(gradesSchemeSetFlags.schemeType) == "" {
			return fmt.Errorf("provide --file or --type")
		}
		payload = map[string]any{"type": gradesSchemeSetFlags.schemeType}
		if gradesSchemeSetFlags.name != "" {
			payload["name"] = gradesSchemeSetFlags.name
		}
		if gradesSchemeSetFlags.scaleJSON != "" {
			var scale json.RawMessage
			if err := json.Unmarshal([]byte(gradesSchemeSetFlags.scaleJSON), &scale); err != nil {
				return fmt.Errorf("invalid --scale-json: %w", err)
			}
			payload["scaleJson"] = scale
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPut, "/api/v1/courses/"+args[0]+"/grading-scheme", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting grading scheme: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied (403): you do not have permission to edit grading scheme")
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Grading scheme saved.")
	return nil
}

func normalizeCurveMethod(method string) string {
	m := strings.ToLower(strings.TrimSpace(method))
	aliases := map[string]string{
		"linear":  "linear_scale",
		"sqrt":    "sqrt_curve",
		"flat":    "flat_bonus",
		"minimum": "set_minimum",
		"custom":  "custom_mapping",
	}
	if canon, ok := aliases[m]; ok {
		return canon
	}
	return m
}

func buildCurveParams(method string) (json.RawMessage, error) {
	switch normalizeCurveMethod(method) {
	case "flat_bonus":
		if gradesCurveFlags.bonus == 0 {
			return nil, fmt.Errorf("flat_bonus requires --bonus")
		}
		return json.Marshal(map[string]float64{"bonus": gradesCurveFlags.bonus})
	case "linear_scale":
		if gradesCurveFlags.targetMean == 0 && gradesCurveFlags.targetMax == 0 {
			return nil, fmt.Errorf("linear_scale requires --target-mean or --target-max")
		}
		params := map[string]float64{}
		if gradesCurveFlags.targetMean != 0 {
			params["targetMean"] = gradesCurveFlags.targetMean
		}
		if gradesCurveFlags.targetMax != 0 {
			params["targetMax"] = gradesCurveFlags.targetMax
		}
		return json.Marshal(params)
	case "sqrt_curve":
		return json.RawMessage(`{}`), nil
	case "set_minimum":
		if gradesCurveFlags.minimum == 0 {
			return nil, fmt.Errorf("set_minimum requires --minimum")
		}
		return json.Marshal(map[string]float64{"minimum": gradesCurveFlags.minimum})
	default:
		return json.RawMessage(`{}`), nil
	}
}

func runGradesCurve(cmd *cobra.Command, _ []string) error {
	params, err := buildCurveParams(gradesCurveFlags.method)
	if err != nil {
		return err
	}
	method := normalizeCurveMethod(gradesCurveFlags.method)
	payload, err := json.Marshal(map[string]any{
		"method":        method,
		"params":        params,
		"allowAboveMax": gradesCurveFlags.allowAbove,
	})
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	itemID := url.PathEscape(gradesCurveFlags.assignment)
	path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/curve", gradesCurveFlags.course, itemID)
	if gradesCurveFlags.dryRun {
		path += "/preview"
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("curving grades: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("permission denied (403): you do not have permission to curve grades")
	}
	if gradesCurveFlags.dryRun {
		if resp.StatusCode != http.StatusOK {
			return apiErrorBody(resp.StatusCode, body)
		}
	} else if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	if gradesCurveFlags.dryRun {
		var preview curvePreviewResponse
		if err := json.Unmarshal(body, &preview); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] eligible=%d max_points=%.1f\n",
			preview.Preview.EligibleCount, preview.MaxPoints)
		for _, r := range preview.Preview.Results {
			if !r.Changed {
				continue
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %.2f → %.2f (Δ %.2f)\n",
				r.StudentID, r.RawScore, r.AdjustedScore, r.Delta)
		}
		return nil
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Curve applied.")
	return nil
}

func runGradesWhatIf(cmd *cobra.Command, _ []string) error {
	overrides, err := parseWhatIfOverrides(gradesWhatIfFlags.override, gradesWhatIfFlags.file)
	if err != nil {
		return err
	}

	grid, err := fetchGradebookGrid(gradesWhatIfFlags.course)
	if err != nil {
		return err
	}
	settings, err := fetchGradingSettings(gradesWhatIfFlags.course)
	if err != nil {
		return err
	}

	actualGrades := grid.Grades[gradesWhatIfFlags.user]
	if actualGrades == nil {
		actualGrades = map[string]string{}
	}
	held := heldItemsForStudent(grid, gradesWhatIfFlags.user)
	excused := excusedItemsForStudent(grid, gradesWhatIfFlags.user)

	actual := computeCourseFinalPercent(grid.Columns, actualGrades, settings.AssignmentGroups, whatIfComputeOptions{
		mode:    "actual",
		excused: excused,
		now:     time.Now().UTC(),
	})
	projected := computeWhatIfFinalPercent(
		grid.Columns, actualGrades, settings.AssignmentGroups, excused, overrides, held, time.Now().UTC(),
	)

	if globalFlags.jsonOut {
		out := map[string]any{
			"userId":            gradesWhatIfFlags.user,
			"actualPercent":     actual,
			"projectedPercent":  projected,
			"overrides":         overrides,
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	actualStr := formatPercent(actual)
	projectedStr := formatPercent(projected)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Student %s\n", gradesWhatIfFlags.user)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Actual final:     %s\n", actualStr)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Projected final:  %s\n", projectedStr)
	return nil
}

func formatPercent(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f%%", *p)
}

func heldItemsForStudent(grid *gradebookGrid, userID string) map[string]bool {
	out := make(map[string]bool)
	if grid.GradeHeld == nil {
		return out
	}
	for itemID, held := range grid.GradeHeld[userID] {
		if held {
			out[itemID] = true
		}
	}
	return out
}

func excusedItemsForStudent(grid *gradebookGrid, userID string) map[string]bool {
	out := make(map[string]bool)
	if grid.ExcusedGrades == nil {
		return out
	}
	for itemID, excused := range grid.ExcusedGrades[userID] {
		if excused {
			out[itemID] = true
		}
	}
	return out
}

func parseWhatIfOverrides(flags []string, file string) (map[string]string, error) {
	overrides := make(map[string]string)
	if file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading overrides file: %w", err)
		}
		if err := json.Unmarshal(raw, &overrides); err != nil {
			return nil, fmt.Errorf("parsing overrides JSON: %w", err)
		}
	}
	for _, pair := range flags {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --override %q: use item-id=score", pair)
		}
		itemID := strings.TrimSpace(parts[0])
		score := strings.TrimSpace(parts[1])
		if itemID == "" {
			return nil, fmt.Errorf("invalid --override %q: empty item id", pair)
		}
		overrides[itemID] = score
	}
	return overrides, nil
}

func fetchGradingSettings(courseCode string) (gradingSettingsResponse, error) {
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/grading", nil)
	if err != nil {
		return gradingSettingsResponse{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return gradingSettingsResponse{}, fmt.Errorf("fetching grading settings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return gradingSettingsResponse{}, apiError(resp, 2)
	}
	var settings gradingSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return gradingSettingsResponse{}, fmt.Errorf("decoding response: %w", err)
	}
	return settings, nil
}

func runGradesHistory(cmd *cobra.Command, _ []string) error {
	itemID := url.PathEscape(gradesHistoryFlags.assignment)
	studentID := url.PathEscape(gradesHistoryFlags.user)
	path := fmt.Sprintf("/api/v1/courses/%s/assignments/%s/grades/%s/history",
		gradesHistoryFlags.course, itemID, studentID)
	req, err := client.New(Cfg.Server, Cfg.APIKey).NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(client.New(Cfg.Server, Cfg.APIKey), req)
	if err != nil {
		return fmt.Errorf("getting grade history: %w", err)
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
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var hist assignmentGradeHistoryBody
	if err := json.Unmarshal(body, &hist); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(hist.Events) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grade history.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CHANGED_AT\tACTION\tPREVIOUS\tNEW\tREASON")
	for _, e := range hist.Events {
		prev := "-"
		if e.PreviousScore != nil {
			prev = fmt.Sprintf("%.2f", *e.PreviousScore)
		}
		newScore := "-"
		if e.NewScore != nil {
			newScore = fmt.Sprintf("%.2f", *e.NewScore)
		}
		reason := "-"
		if e.Reason != nil {
			reason = *e.Reason
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ChangedAt, e.Action, prev, newScore, reason)
	}
	return w.Flush()
}