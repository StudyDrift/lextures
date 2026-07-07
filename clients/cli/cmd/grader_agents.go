package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var graderAgentsCmd = &cobra.Command{
	Use:   "grader-agents",
	Short: "Manage per-assignment AI grading agents",
}

var graderAgentsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List grading agents configured in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsList,
}

var graderAgentsGetCmd = &cobra.Command{
	Use:   "get <course>",
	Short: "Get the grader-agent config for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsGet,
}

var graderAgentsSetCmd = &cobra.Command{
	Use:   "set <course>",
	Short: "Create or update a grader-agent config from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsSet,
}

var graderAgentsDeleteCmd = &cobra.Command{
	Use:   "delete <course>",
	Short: "Delete a grader-agent config for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsDelete,
}

var graderAgentsDryRunCmd = &cobra.Command{
	Use:   "dry-run <course>",
	Short: "Preview AI grading on sample submissions (no gradebook changes)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsDryRun,
}

var graderAgentsRunCmd = &cobra.Command{
	Use:   "run <assignment>",
	Short: "Trigger AI grading for an assignment",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsRun,
}

var graderAgentsResultsCmd = &cobra.Command{
	Use:   "results <assignment>",
	Short: "Show suggested scores from a grader-agent run",
	Args:  cobra.ExactArgs(1),
	RunE:  runGraderAgentsResults,
}

var graderAgentsGetFlags struct {
	assignment string
}

var graderAgentsSetFlags struct {
	assignment string
	file       string
}

var graderAgentsDeleteFlags struct {
	assignment string
}

var graderAgentsDryRunFlags struct {
	assignment string
	sample     int
	submission string
}

var graderAgentsRunFlags struct {
	course   string
	scope    string
	mode     string
	yes      bool
	wait     bool
	timeout  int
	overwrite bool
}

var graderAgentsResultsFlags struct {
	course string
	run    string
	accept bool
	yes    bool
}

func init() {
	graderAgentsGetCmd.Flags().StringVar(&graderAgentsGetFlags.assignment, "assignment", "", "assignment item id (required)")
	_ = graderAgentsGetCmd.MarkFlagRequired("assignment")

	graderAgentsSetCmd.Flags().StringVar(&graderAgentsSetFlags.assignment, "assignment", "", "assignment item id (required)")
	_ = graderAgentsSetCmd.MarkFlagRequired("assignment")
	graderAgentsSetCmd.Flags().StringVar(&graderAgentsSetFlags.file, "file", "", "JSON agent config (path or -)")

	graderAgentsDeleteCmd.Flags().StringVar(&graderAgentsDeleteFlags.assignment, "assignment", "", "assignment item id (required)")
	_ = graderAgentsDeleteCmd.MarkFlagRequired("assignment")

	graderAgentsDryRunCmd.Flags().StringVar(&graderAgentsDryRunFlags.assignment, "assignment", "", "assignment item id (required)")
	_ = graderAgentsDryRunCmd.MarkFlagRequired("assignment")
	graderAgentsDryRunCmd.Flags().IntVar(&graderAgentsDryRunFlags.sample, "sample", 1, "number of submissions to dry-run (limits token spend)")
	graderAgentsDryRunCmd.Flags().StringVar(&graderAgentsDryRunFlags.submission, "submission", "", "specific submission id (overrides --sample selection)")

	graderAgentsRunCmd.Flags().StringVar(&graderAgentsRunFlags.course, "course", "", "course code (required)")
	_ = graderAgentsRunCmd.MarkFlagRequired("course")
	graderAgentsRunCmd.Flags().StringVar(&graderAgentsRunFlags.scope, "scope", "ungraded", "run scope: ungraded, all, current")
	graderAgentsRunCmd.Flags().StringVar(&graderAgentsRunFlags.mode, "mode", "suggest", "run mode: suggest or apply")
	graderAgentsRunCmd.Flags().BoolVar(&graderAgentsRunFlags.yes, "yes", false, "confirm destructive/full-class runs")
	graderAgentsRunCmd.Flags().BoolVar(&graderAgentsRunFlags.overwrite, "overwrite", false, "allow overwriting existing grades (required for scope=all)")
	graderAgentsRunCmd.Flags().BoolVar(&graderAgentsRunFlags.wait, "wait", false, "poll until the run finishes")
	graderAgentsRunCmd.Flags().IntVar(&graderAgentsRunFlags.timeout, "timeout", 600, "seconds to wait when --wait is set")

	graderAgentsResultsCmd.Flags().StringVar(&graderAgentsResultsFlags.course, "course", "", "course code (required)")
	_ = graderAgentsResultsCmd.MarkFlagRequired("course")
	graderAgentsResultsCmd.Flags().StringVar(&graderAgentsResultsFlags.run, "run", "", "run id (defaults to latest when run history is enabled)")
	graderAgentsResultsCmd.Flags().BoolVar(&graderAgentsResultsFlags.accept, "accept", false, "apply all held suggestions to the gradebook")
	graderAgentsResultsCmd.Flags().BoolVar(&graderAgentsResultsFlags.yes, "yes", false, "confirm --accept")

	graderAgentsCmd.AddCommand(
		graderAgentsListCmd,
		graderAgentsGetCmd,
		graderAgentsSetCmd,
		graderAgentsDeleteCmd,
		graderAgentsDryRunCmd,
		graderAgentsRunCmd,
		graderAgentsResultsCmd,
	)
	rootCmd.AddCommand(graderAgentsCmd)
}

func runGraderAgentsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentCoursePath(args[0], "/grader-agents")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("listing grader agents: %w", err)
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
	var out struct {
		Agents []map[string]any `json:"agents"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(out.Agents) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grader agents configured.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ITEM_ID\tTITLE\tSTATUS\tAUTO_GRADE\tREVIEW_COUNT\tUPDATED")
	for _, agent := range out.Agents {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\t%s\n",
			stringField(agent, "itemId"),
			stringField(agent, "assignmentTitle"),
			stringField(agent, "status"),
			agent["autoGradeNew"],
			agent["reviewCount"],
			stringField(agent, "updatedAt"),
		)
	}
	return w.Flush()
}

func runGraderAgentsGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	cfg, raw, err := fetchGraderAgentConfig(c, args[0], graderAgentsGetFlags.assignment)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if cfg == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No grader agent configured for this assignment.")
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:         %s\n", stringField(cfg, "id"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status:     %s\n", stringField(cfg, "status"))
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "PostPolicy: %s\n", stringField(cfg, "postPolicy"))
	if model := stringField(cfg, "modelId"); model != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Model:      %s\n", model)
	}
	if _, ok := cfg["workflowGraph"]; ok {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Workflow:   yes")
	}
	return nil
}

func runGraderAgentsSet(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(graderAgentsSetFlags.file) == "" {
		return fmt.Errorf("--file is required")
	}
	payload, err := loadGraderAgentSetPayload(graderAgentsSetFlags.file)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentAssignmentPath(args[0], graderAgentsSetFlags.assignment, "/grader-agent")
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("saving grader agent: %w", err)
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
	var out struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Saved grader agent %s (status=%s).\n",
		stringField(out.Config, "id"), stringField(out.Config, "status"))
	return nil
}

func runGraderAgentsDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := graderAgentAssignmentPath(args[0], graderAgentsDeleteFlags.assignment, "/grader-agent")
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting grader agent: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"deleted": graderAgentsDeleteFlags.assignment,
		})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Deleted grader agent config.")
	return nil
}

func runGraderAgentsDryRun(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := ensureAIGradingAllowed(c); err != nil {
		return err
	}
	_ = printGraderAgentDisclosure(c, cmd.OutOrStdout())

	course := args[0]
	assignment := graderAgentsDryRunFlags.assignment
	cfg, _, err := fetchGraderAgentConfig(c, course, assignment)
	if err != nil {
		return err
	}
	graph, err := workflowGraphFromConfig(cfg)
	if err != nil {
		return err
	}

	var submissionIDs []string
	if sub := strings.TrimSpace(graderAgentsDryRunFlags.submission); sub != "" {
		submissionIDs = []string{sub}
	} else {
		subs, _, err := fetchAssignmentSubmissions(c, course, assignment, "ungraded")
		if err != nil {
			return err
		}
		submissionIDs = gradableSubmissionIDs(subs.Submissions)
		if len(submissionIDs) == 0 {
			subs, _, err = fetchAssignmentSubmissions(c, course, assignment, "")
			if err != nil {
				return err
			}
			submissionIDs = gradableSubmissionIDs(subs.Submissions)
		}
		if len(submissionIDs) == 0 {
			return fmt.Errorf("no gradable submissions found for dry-run")
		}
		submissionIDs = limitSubmissionIDs(submissionIDs, graderAgentsDryRunFlags.sample)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var samples []dryRunSampleResult
	for _, submissionID := range submissionIDs {
		if !globalFlags.jsonOut {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nDry-running submission %s…\n", submissionID)
		}
		events, wsErr := runGraderAgentDryRunWS(ctx, c, course, assignment, submissionID, graph, func(ev dryRunExecutionEvent) error {
			if globalFlags.jsonOut {
				return nil
			}
			if line := formatDryRunProgressLine(ev); line != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", line)
			}
			return nil
		})
		sample := collectDryRunSampleResult(submissionID, events)
		if wsErr != nil && sample.Error == "" {
			sample.Error = wsErr.Error()
		}
		samples = append(samples, sample)
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"course":     course,
			"assignment": assignment,
			"samples":    samples,
		})
	}

	var totalCost float64
	var totalPrompt, totalCompletion int
	for _, sample := range samples {
		if sample.Error != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submission %s: error: %s\n", sample.SubmissionID, sample.Error)
			continue
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Submission %s: %.2f pts (confidence %.0f%%)\n",
			sample.SubmissionID, sample.SuggestedPoints, sample.Confidence*100)
		totalCost += sample.CostUSD
		totalPrompt += sample.PromptTokens
		totalCompletion += sample.CompletionTokens
	}
	if totalPrompt > 0 || totalCompletion > 0 || totalCost > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nToken usage: prompt=%d completion=%d cost=$%.4f\n",
			totalPrompt, totalCompletion, totalCost)
	}
	return nil
}

func runGraderAgentsRun(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := ensureAIGradingAllowed(c); err != nil {
		return err
	}
	_ = printGraderAgentDisclosure(c, cmd.OutOrStdout())

	course := graderAgentsRunFlags.course
	assignment := args[0]
	scope := strings.ToLower(strings.TrimSpace(graderAgentsRunFlags.scope))
	if scope == "all" && !graderAgentsRunFlags.yes {
		return fmt.Errorf("%s", graderAgentFullRunWarning)
	}
	overwrite := graderAgentsRunFlags.overwrite
	if scope == "all" && !overwrite {
		overwrite = true
	}

	estimate, _, _ := fetchGraderAgentRunEstimate(c, course, assignment, scope, overwrite)
	if line := formatRunEstimateLine(estimate); line != "" && !globalFlags.jsonOut {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Run estimate: %s\n", line)
	}

	payload := map[string]any{
		"scope": scope,
		"mode":  strings.ToLower(strings.TrimSpace(graderAgentsRunFlags.mode)),
	}
	if overwrite {
		payload["overwrite"] = true
	}
	started, raw, err := postGraderAgentRun(c, course, assignment, payload)
	if err != nil {
		return err
	}
	runID := stringField(started, "runId")
	if runID == "" {
		return fmt.Errorf("server did not return runId")
	}

	if !graderAgentsRunFlags.wait {
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started grader agent run %s (%v queued).\n",
			runID, started["queuedCount"])
		if summary, ok := started["targetSummary"].(string); ok && summary != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", summary)
		}
		return nil
	}

	timeout := time.Duration(graderAgentsRunFlags.timeout) * time.Second
	var lastStatus string
	run, err := waitForGraderAgentRun(c, course, assignment, runID, timeout, func(run map[string]any) {
		status, _ := run["status"].(string)
		if status != lastStatus && !globalFlags.jsonOut {
			completed, _ := run["completedCount"].(float64)
			total, _ := run["totalCount"].(float64)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Run %s: %s (%.0f/%.0f)\n", runID, status, completed, total)
		}
		lastStatus = status
	})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(run)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Run %s finished with status %s.\n", runID, stringField(run, "status"))
	for _, line := range formatRunUsageLines(run) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}

func runGraderAgentsResults(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := graderAgentsResultsFlags.course
	assignment := args[0]

	if graderAgentsResultsFlags.accept {
		if !graderAgentsResultsFlags.yes {
			return fmt.Errorf("bulk accept writes grades to the gradebook; re-run with --yes to confirm")
		}
		if err := ensureAIGradingAllowed(c); err != nil {
			return err
		}
		out, raw, err := postGraderAgentReviewBulk(c, course, assignment, map[string]any{
			"action": "approve_all",
		})
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		applied := 0
		if outcomes, ok := out["outcomes"].([]any); ok {
			for _, row := range outcomes {
				if m, ok := row.(map[string]any); ok {
					switch stringField(m, "status") {
					case "applied", "overridden":
						applied++
					}
				}
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Accepted %d suggestion(s).\n", applied)
		return nil
	}

	runID := strings.TrimSpace(graderAgentsResultsFlags.run)
	if runID == "" {
		var err error
		runID, err = fetchLatestGraderAgentRunID(c, course, assignment)
		if err != nil {
			return err
		}
	}
	run, raw, err := fetchGraderAgentRun(c, course, assignment, runID)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Run %s status=%s (%.0f/%.0f complete, %.0f failed)\n",
		runID,
		stringField(run, "status"),
		run["completedCount"],
		run["totalCount"],
		run["failedCount"],
	)
	for _, line := range formatRunUsageLines(run) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	results, _ := run["results"].([]any)
	if len(results) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No results.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "RESULT_ID\tSUBMISSION_ID\tSTATUS\tSUGGESTED\tCONFIDENCE")
	for _, row := range results {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		conf := ""
		if v, ok := m["confidence"].(float64); ok {
			conf = fmt.Sprintf("%.0f%%", v*100)
		}
		points := ""
		if v, ok := m["suggestedPoints"].(float64); ok {
			points = fmt.Sprintf("%.2f", v)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			stringField(m, "id"),
			stringField(m, "submissionId"),
			stringField(m, "status"),
			points,
			conf,
		)
	}
	return w.Flush()
}