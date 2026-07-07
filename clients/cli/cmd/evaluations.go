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

var evaluationsCmd = &cobra.Command{
	Use:   "evaluations",
	Short: "Launch and inspect end-of-term course evaluations",
}

var evaluationsLaunchCmd = &cobra.Command{
	Use:   "launch <course>",
	Short: "Launch an evaluation window for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runEvaluationsLaunch,
}

var evaluationsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List evaluation windows for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runEvaluationsList,
}

var evaluationsGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get active evaluation status or aggregate results",
	Args:  cobra.ExactArgs(1),
	RunE:  runEvaluationsGet,
}

var evaluationsResultsCmd = &cobra.Command{
	Use:   "results <course>",
	Short: "View or export aggregate evaluation results",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runEvaluationsResults,
}

var evaluationsLaunchFlags struct {
	template string
	opens    string
	closes   string
}

var evaluationsListFlags struct {
	closedOnly bool
}

var evaluationsGetFlags struct {
	results bool
}

var evaluationsResultsFlags struct {
	format string
	out    string
}

func init() {
	evaluationsLaunchCmd.Flags().StringVar(&evaluationsLaunchFlags.template, "template", "", "evaluation template UUID (required)")
	_ = evaluationsLaunchCmd.MarkFlagRequired("template")
	evaluationsLaunchCmd.Flags().StringVar(&evaluationsLaunchFlags.opens, "opens", "", "window opens at RFC3339 timestamp (required)")
	_ = evaluationsLaunchCmd.MarkFlagRequired("opens")
	evaluationsLaunchCmd.Flags().StringVar(&evaluationsLaunchFlags.closes, "closes", "", "window closes at RFC3339 timestamp (required)")
	_ = evaluationsLaunchCmd.MarkFlagRequired("closes")

	evaluationsListCmd.Flags().BoolVar(&evaluationsListFlags.closedOnly, "closed-only", false, "include only closed windows")

	evaluationsGetCmd.Flags().BoolVar(&evaluationsGetFlags.results, "results", false, "return aggregate results for the latest closed window")

	evaluationsResultsCmd.Flags().StringVar(&evaluationsResultsFlags.format, "format", "", "export format: csv or json (use with: results export <course>)")
	evaluationsResultsCmd.Flags().StringVar(&evaluationsResultsFlags.out, "out", "", "output file path for export (- for stdout)")

	evaluationsCmd.AddCommand(
		evaluationsLaunchCmd,
		evaluationsListCmd,
		evaluationsGetCmd,
		evaluationsResultsCmd,
	)
	rootCmd.AddCommand(evaluationsCmd)
}

func runEvaluationsLaunch(cmd *cobra.Command, args []string) error {
	payload, err := json.Marshal(map[string]string{
		"templateId": strings.TrimSpace(evaluationsLaunchFlags.template),
		"opensAt":    strings.TrimSpace(evaluationsLaunchFlags.opens),
		"closesAt":   strings.TrimSpace(evaluationsLaunchFlags.closes),
	})
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := "/api/v1/admin/courses/" + url.PathEscape(args[0]) + "/evaluation-windows"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("launching evaluation: %w", err)
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
	var win map[string]any
	if err := json.Unmarshal(body, &win); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Launched evaluation window %s for %s.\n",
		stringField(win, "id"), args[0])
	return nil
}

func runEvaluationsList(cmd *cobra.Command, args []string) error {
	rows, _, err := fetchEvaluationReport(client.New(Cfg.Server, Cfg.APIKey), evaluationsListFlags.closedOnly)
	if err != nil {
		return err
	}
	course := strings.TrimSpace(args[0])
	filtered := make([]map[string]any, 0)
	for _, row := range rows {
		if strings.EqualFold(stringField(row, "courseCode"), course) {
			filtered = append(filtered, row)
		}
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"rows": filtered})
	}
	if len(filtered) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No evaluation windows found for %s.\n", course)
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "WINDOW_ID\tOPENS\tCLOSES\tRESPONSES\tENROLLED\tCOMPLETION %")
	for _, row := range filtered {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\t%v\n",
			stringField(row, "windowId"),
			stringField(row, "opensAt"),
			stringField(row, "closesAt"),
			row["responseCount"],
			row["enrolledCount"],
			row["completionPct"],
		)
	}
	return w.Flush()
}

func runEvaluationsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if evaluationsGetFlags.results {
		return printEvaluationResults(cmd, c, args[0])
	}
	path := "/api/v1/courses/" + url.PathEscape(args[0]) + "/evaluations/status"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("loading evaluation status: %w", err)
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
	var status map[string]any
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Window open: %v\n", status["windowOpen"])
	if stringField(status, "windowId") != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Window ID:   %s\n", stringField(status, "windowId"))
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Opens:       %s\n", stringField(status, "opensAt"))
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Closes:      %s\n", stringField(status, "closesAt"))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submitted:   %v\n", status["hasSubmitted"])
	return nil
}

func runEvaluationsResults(cmd *cobra.Command, args []string) error {
	if len(args) >= 2 && args[0] == "export" {
		return runEvaluationsResultsExport(cmd, args[1:])
	}
	if len(args) == 1 {
		return printEvaluationResults(cmd, client.New(Cfg.Server, Cfg.APIKey), args[0])
	}
	return fmt.Errorf("usage: evaluations results <course> | evaluations results export <course>")
}

func runEvaluationsResultsExport(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: evaluations results export <course>")
	}
	format := strings.ToLower(strings.TrimSpace(evaluationsResultsFlags.format))
	if format == "" {
		format = "json"
	}
	switch format {
	case "csv", "json":
	default:
		return fmt.Errorf("unsupported format %q: use csv or json", evaluationsResultsFlags.format)
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	results, _, err := fetchEvaluationResults(c, args[0])
	if err != nil {
		return err
	}

	var data []byte
	switch format {
	case "csv":
		data, err = evaluationResultsToCSV(results)
	default:
		data, err = json.MarshalIndent(results, "", "  ")
	}
	if err != nil {
		return err
	}
	outPath := strings.TrimSpace(evaluationsResultsFlags.out)
	if outPath == "" {
		outPath = "-"
	}
	return writeFileOrStdout(cmd, outPath, data)
}

func printEvaluationResults(cmd *cobra.Command, c *client.Client, course string) error {
	results, raw, err := fetchEvaluationResults(c, course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Window:      %s\n", stringField(results, "windowId"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Responses:   %v / %v (%.1f%%)\n",
		results["responseCount"], results["enrolledCount"], results["completionPct"])
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Threshold:   %v\n", results["meetsThreshold"])
	questions, _ := results["questions"].([]any)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Questions:   %d\n", len(questions))
	return nil
}

func fetchEvaluationReport(c *client.Client, closedOnly bool) ([]map[string]any, []byte, error) {
	path := "/api/v1/admin/evaluations/report"
	if closedOnly {
		path += "?closed_only=true"
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("loading evaluation report: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.Rows, body, nil
}

func fetchEvaluationResults(c *client.Client, course string) (map[string]any, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(course) + "/evaluations/results"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("loading evaluation results: %w", err)
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