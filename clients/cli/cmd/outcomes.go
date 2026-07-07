package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var outcomesCmd = &cobra.Command{
	Use:   "outcomes",
	Short: "Manage course learning outcomes and mastery reports",
}

var outcomesListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List outcomes with alignment progress",
	Args:  cobra.ExactArgs(1),
	RunE:  runOutcomesList,
}

var outcomesCreateCmd = &cobra.Command{
	Use:   "create <course>",
	Short: "Create a learning outcome",
	Args:  cobra.ExactArgs(1),
	RunE:  runOutcomesCreate,
}

var outcomesAlignCmd = &cobra.Command{
	Use:   "align <course>",
	Short: "Bulk-align outcomes to course items from a JSON or CSV file",
	Args:  cobra.ExactArgs(1),
	RunE:  runOutcomesAlign,
}

var outcomesReportCmd = &cobra.Command{
	Use:   "report <course>",
	Short: "Pull outcome mastery rollups for analytics",
	Args:  cobra.ExactArgs(1),
	RunE:  runOutcomesReport,
}

var outcomesMasteryCmd = &cobra.Command{
	Use:   "mastery <course>",
	Short: "Get a student's SBG mastery rollup for a grading period",
	RunE:  runOutcomesMastery,
}

var outcomesCreateFlags struct {
	title       string
	description string
	file        string
}

var outcomesAlignFlags struct {
	file string
}

var outcomesReportFlags struct {
	section string
	group   string
}

var outcomesMasteryFlags struct {
	user   string
	period string
	method string
}

func init() {
	outcomesCreateCmd.Flags().StringVar(&outcomesCreateFlags.title, "title", "", "outcome title")
	outcomesCreateCmd.Flags().StringVar(&outcomesCreateFlags.description, "description", "", "outcome description")
	outcomesCreateCmd.Flags().StringVar(&outcomesCreateFlags.file, "file", "", "JSON body with title and description")

	outcomesAlignCmd.Flags().StringVar(&outcomesAlignFlags.file, "file", "", "alignment JSON or CSV file (required)")
	_ = outcomesAlignCmd.MarkFlagRequired("file")

	outcomesReportCmd.Flags().StringVar(&outcomesReportFlags.section, "section", "", "section UUID filter")
	outcomesReportCmd.Flags().StringVar(&outcomesReportFlags.group, "group", "", "group UUID filter")

	outcomesMasteryCmd.Flags().StringVar(&outcomesMasteryFlags.user, "user", "", "student user UUID or email (required)")
	_ = outcomesMasteryCmd.MarkFlagRequired("user")
	outcomesMasteryCmd.Flags().StringVar(&outcomesMasteryFlags.period, "period", "", "grading period code (required)")
	_ = outcomesMasteryCmd.MarkFlagRequired("period")
	outcomesMasteryCmd.Flags().StringVar(&outcomesMasteryFlags.method, "method", "", "aggregation method (most_recent, mode, highest)")

	outcomesCmd.AddCommand(
		outcomesListCmd,
		outcomesCreateCmd,
		outcomesAlignCmd,
		outcomesReportCmd,
		outcomesMasteryCmd,
	)
	rootCmd.AddCommand(outcomesCmd)
}

func runOutcomesList(cmd *cobra.Command, args []string) error {
	body, err := fetchCourseOutcomes(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(body)
	}
	if len(body.Outcomes) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No outcomes.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Enrolled learners: %d\n", body.EnrolledLearners)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tLINKS\tROLLUP %")
	for _, o := range body.Outcomes {
		rollup := ""
		if o.RollupAvgScorePercent != nil {
			rollup = fmt.Sprintf("%.1f", *o.RollupAvgScorePercent)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", o.ID, o.Title, len(o.Links), rollup)
	}
	return w.Flush()
}

type courseOutcomesListBody struct {
	EnrolledLearners int `json:"enrolledLearners"`
	Outcomes         []struct {
		ID                    string  `json:"id"`
		Title                 string  `json:"title"`
		Description           string  `json:"description"`
		RollupAvgScorePercent *float64 `json:"rollupAvgScorePercent"`
		Links                 []any   `json:"links"`
	} `json:"outcomes"`
}

func fetchCourseOutcomes(c *client.Client, course string) (courseOutcomesListBody, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+url.PathEscape(course)+"/outcomes", nil)
	if err != nil {
		return courseOutcomesListBody{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return courseOutcomesListBody{}, fmt.Errorf("listing outcomes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return courseOutcomesListBody{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return courseOutcomesListBody{}, apiErrorBody(resp.StatusCode, body)
	}
	var out courseOutcomesListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return courseOutcomesListBody{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func runOutcomesCreate(cmd *cobra.Command, args []string) error {
	var payload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if outcomesCreateFlags.file != "" {
		raw, err := readInputFile(outcomesCreateFlags.file)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
	} else {
		payload.Title = strings.TrimSpace(outcomesCreateFlags.title)
		payload.Description = strings.TrimSpace(outcomesCreateFlags.description)
	}
	if payload.Title == "" {
		return fmt.Errorf("title is required (--title or --file)")
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+url.PathEscape(args[0])+"/outcomes", bytes.NewReader(mustJSON(payload)))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating outcome: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var created struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	_ = json.Unmarshal(body, &created)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created outcome %s (%s).\n", created.ID, created.Title)
	return nil
}

func runOutcomesAlign(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(outcomesAlignFlags.file)
	if err != nil {
		return fmt.Errorf("reading alignment file: %w", err)
	}
	rows, err := parseOutcomeAlignFile(raw)
	if err != nil {
		return err
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	var posted, failed int
	var errs []string
	for i, row := range rows {
		payload := map[string]any{
			"structureItemId": row.StructureItemID,
			"targetKind":      row.TargetKind,
		}
		if row.QuizQuestionID != nil {
			payload["quizQuestionId"] = *row.QuizQuestionID
		}
		if row.MeasurementLevel != nil {
			payload["measurementLevel"] = *row.MeasurementLevel
		}
		if row.IntensityLevel != nil {
			payload["intensityLevel"] = *row.IntensityLevel
		}
		if row.SubOutcomeID != nil {
			payload["subOutcomeId"] = *row.SubOutcomeID
		}

		path := fmt.Sprintf("/api/v1/courses/%s/outcomes/%s/links",
			url.PathEscape(course), url.PathEscape(row.OutcomeID))
		req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(mustJSON(payload)))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := doWithRetry(c, req)
		if err != nil {
			failed++
			errs = append(errs, fmt.Sprintf("row %d: %v", i+1, err))
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			failed++
			errs = append(errs, fmt.Sprintf("row %d: %s", i+1, apiErrorBody(resp.StatusCode, body)))
			continue
		}
		posted++
	}

	summary := map[string]any{
		"posted": posted,
		"failed": failed,
		"errors": errs,
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(summary)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Aligned %d link(s); %d failed.\n", posted, failed)
	for _, e := range errs {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", e)
	}
	if failed > 0 {
		return fmt.Errorf("%d alignment(s) failed", failed)
	}
	return nil
}

func runOutcomesReport(cmd *cobra.Command, args []string) error {
	params := url.Values{}
	if s := strings.TrimSpace(outcomesReportFlags.section); s != "" {
		params.Set("sectionId", s)
	}
	if g := strings.TrimSpace(outcomesReportFlags.group); g != "" {
		params.Set("groupId", g)
	}
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/analytics/outcomes"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("loading outcomes report: %w", err)
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

	var report struct {
		MasteryThreshold float64 `json:"masteryThreshold"`
		DataAsOf         string  `json:"dataAsOf"`
		Outcomes         []struct {
			OutcomeID      string   `json:"outcomeId"`
			Title          string   `json:"title"`
			NStudents      int      `json:"nStudents"`
			NAssessed      int      `json:"nAssessed"`
			MeanScore      *float64 `json:"meanScore"`
			PctMet         float64  `json:"pctMet"`
			PctNotMet      float64  `json:"pctNotMet"`
			AlignmentCount int      `json:"alignmentCount"`
		} `json:"outcomes"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Threshold: %.1f%%  Data as of: %s\n", report.MasteryThreshold, report.DataAsOf)
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "OUTCOME\tASSESSED\tMEAN\t% MET\t% NOT MET\tALIGNMENTS")
	for _, o := range report.Outcomes {
		mean := ""
		if o.MeanScore != nil {
			mean = fmt.Sprintf("%.1f", *o.MeanScore)
		}
		_, _ = fmt.Fprintf(w, "%s\t%d/%d\t%s\t%.1f\t%.1f\t%d\n",
			o.Title, o.NAssessed, o.NStudents, mean, o.PctMet, o.PctNotMet, o.AlignmentCount)
	}
	return w.Flush()
}

func runOutcomesMastery(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	userID, _, err := resolveUserID(c, strings.TrimSpace(outcomesMasteryFlags.user))
	if err != nil {
		return err
	}
	params := url.Values{}
	if m := strings.TrimSpace(outcomesMasteryFlags.method); m != "" {
		params.Set("method", m)
	}
	path := fmt.Sprintf("/api/v1/students/%s/sbg/%s",
		url.PathEscape(userID), url.PathEscape(strings.TrimSpace(outcomesMasteryFlags.period)))
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("loading student mastery: %w", err)
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

	var report struct {
		StudentID string `json:"studentId"`
		Period    string `json:"period"`
		Method    string `json:"method"`
		Scores    []struct {
			StandardID string `json:"standardId"`
			ScoreValue int    `json:"scoreValue"`
		} `json:"scores"`
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Student %s — period %s (%s)\n", report.StudentID, report.Period, report.Method)
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STANDARD\tSCORE")
	for _, s := range report.Scores {
		_, _ = fmt.Fprintf(w, "%s\t%d\n", s.StandardID, s.ScoreValue)
	}
	return w.Flush()
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}