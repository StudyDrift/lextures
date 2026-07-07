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

var surveysCmd = &cobra.Command{
	Use:   "surveys",
	Short: "Create course surveys and export aggregated responses",
}

var surveysListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List surveys in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runSurveysList,
}

var surveysGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get one survey by item UUID",
	Args:  cobra.ExactArgs(1),
	RunE:  runSurveysGet,
}

var surveysCreateCmd = &cobra.Command{
	Use:   "create <course>",
	Short: "Create a survey in a course module",
	Args:  cobra.ExactArgs(1),
	RunE:  runSurveysCreate,
}

var surveysResultsCmd = &cobra.Command{
	Use:   "results <id>",
	Short: "View or export aggregated survey results",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSurveysResults,
}

var surveysCreateFlags struct {
	title  string
	module string
	file   string
}

var surveysResultsFlags struct {
	format string
	out    string
}

func init() {
	surveysCreateCmd.Flags().StringVar(&surveysCreateFlags.title, "title", "", "survey title")
	surveysCreateCmd.Flags().StringVar(&surveysCreateFlags.module, "module", "", "module UUID (required unless --file includes moduleId)")
	surveysCreateCmd.Flags().StringVar(&surveysCreateFlags.file, "file", "", "JSON body matching CreateCourseSurveyRequest")

	surveysResultsCmd.Flags().StringVar(&surveysResultsFlags.format, "format", "", "export format: csv or json (use with: results export <id>)")
	surveysResultsCmd.Flags().StringVar(&surveysResultsFlags.out, "out", "", "output file path for export (- for stdout)")

	surveysCmd.AddCommand(surveysListCmd, surveysGetCmd, surveysCreateCmd, surveysResultsCmd)
	rootCmd.AddCommand(surveysCmd)
}

func runSurveysList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/surveys"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing surveys: %w", err)
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
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No surveys found.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tANONYMITY\tOPENS\tCLOSES")
	for _, row := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			stringField(row, "id"),
			stringField(row, "title"),
			stringField(row, "anonymityMode"),
			stringField(row, "opensAt"),
			stringField(row, "closesAt"),
		)
	}
	return w.Flush()
}

func runSurveysGet(cmd *cobra.Command, args []string) error {
	survey, raw, err := fetchSurvey(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:          %s\n", stringField(survey, "id"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title:       %s\n", stringField(survey, "title"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Anonymity:   %s\n", stringField(survey, "anonymityMode"))
	questions, _ := survey["questions"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Questions:   %d\n", len(questions))
	return nil
}

func runSurveysCreate(cmd *cobra.Command, args []string) error {
	payload, err := surveyCreatePayload()
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/surveys"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("creating survey: %w", err)
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
	var survey map[string]any
	if err := json.Unmarshal(body, &survey); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created survey %s (%s).\n",
		stringField(survey, "id"), stringField(survey, "title"))
	return nil
}

func runSurveysResults(cmd *cobra.Command, args []string) error {
	if len(args) >= 2 && args[0] == "export" {
		return runSurveysResultsExport(cmd, args[1:])
	}
	if len(args) == 1 {
		return runSurveysResultsShow(cmd, args[0])
	}
	return fmt.Errorf("usage: surveys results <id> | surveys results export <id>")
}

func runSurveysResultsShow(cmd *cobra.Command, surveyID string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	results, raw, err := fetchSurveyResults(c, surveyID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Responses: %v\n", results["responseCount"])
	questions, _ := results["questions"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Questions: %d\n", len(questions))
	return nil
}

func runSurveysResultsExport(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: surveys results export <id>")
	}
	format := strings.ToLower(strings.TrimSpace(surveysResultsFlags.format))
	if format == "" {
		format = "json"
	}
	switch format {
	case "csv", "json":
	default:
		return fmt.Errorf("unsupported format %q: use csv or json", surveysResultsFlags.format)
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	surveyID := args[0]
	survey, _, err := fetchSurvey(c, surveyID)
	if err != nil {
		return err
	}
	results, _, err := fetchSurveyResults(c, surveyID)
	if err != nil {
		return err
	}
	exportDoc, err := prepareSurveyResultsExport(survey, results)
	if err != nil {
		return err
	}

	var data []byte
	switch format {
	case "csv":
		data, err = surveyResultsToCSV(exportDoc)
	default:
		data, err = json.MarshalIndent(exportDoc, "", "  ")
	}
	if err != nil {
		return err
	}
	outPath := strings.TrimSpace(surveysResultsFlags.out)
	if outPath == "" {
		outPath = "-"
	}
	return writeFileOrStdout(cmd, outPath, data)
}

func surveyCreatePayload() ([]byte, error) {
	if path := strings.TrimSpace(surveysCreateFlags.file); path != "" {
		raw, err := readInputFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading survey file: %w", err)
		}
		return raw, nil
	}
	title := strings.TrimSpace(surveysCreateFlags.title)
	moduleID := strings.TrimSpace(surveysCreateFlags.module)
	if title == "" {
		return nil, fmt.Errorf("specify --title or --file")
	}
	if moduleID == "" {
		return nil, fmt.Errorf("specify --module or --file")
	}
	return json.Marshal(map[string]any{
		"moduleId":      moduleID,
		"title":         title,
		"anonymityMode": "identified",
		"questions":     []any{},
	})
}

func fetchSurvey(c *client.Client, surveyID string) (map[string]any, []byte, error) {
	path := "/api/v1/surveys/" + url.PathEscape(surveyID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("loading survey: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, apiErrorBody(resp.StatusCode, body)
	}
	var survey map[string]any
	if err := json.Unmarshal(body, &survey); err != nil {
		return nil, nil, fmt.Errorf("decoding response: %w", err)
	}
	return survey, body, nil
}

func fetchSurveyResults(c *client.Client, surveyID string) (map[string]any, []byte, error) {
	path := "/api/v1/surveys/" + url.PathEscape(surveyID) + "/results"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("loading survey results: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, apiErrorBody(resp.StatusCode, body)
	}
	var results map[string]any
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, nil, fmt.Errorf("decoding response: %w", err)
	}
	return results, body, nil
}