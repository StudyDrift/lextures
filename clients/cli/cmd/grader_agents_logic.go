package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/lextures/lextures/clients/cli/internal/wsclient"
)

const graderAgentAIOptOutMessage = `AI processing is disabled for this account.
Visit Settings → AI processing to re-enable AI features before using grader agents.
See /ai-disclosure for how student data is used with AI providers.`

const graderAgentFullRunWarning = `WARNING: Running the grader agent on all submissions may overwrite existing grades.
Re-run with --yes to confirm.`

// dryRunExecutionEvent mirrors gradingagentsvc.ExecutionEvent for CLI parsing.
type dryRunExecutionEvent struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Level   string `json:"level,omitempty"`
	NodeID  string `json:"nodeId,omitempty"`
	Status  string `json:"status,omitempty"`
	Result  *struct {
		SuggestedPoints  float64 `json:"suggestedPoints"`
		Comment          string  `json:"comment"`
		Confidence       float64 `json:"confidence"`
		PromptTokens     int     `json:"promptTokens,omitempty"`
		CompletionTokens int     `json:"completionTokens,omitempty"`
		CostUSD          float64 `json:"costUsd,omitempty"`
	} `json:"result,omitempty"`
}

type dryRunSampleResult struct {
	SubmissionID     string  `json:"submissionId"`
	SuggestedPoints  float64 `json:"suggestedPoints,omitempty"`
	Comment          string  `json:"comment,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
	PromptTokens     int     `json:"promptTokens,omitempty"`
	CompletionTokens int     `json:"completionTokens,omitempty"`
	CostUSD          float64 `json:"costUsd,omitempty"`
	Error            string  `json:"error,omitempty"`
	Events           []dryRunExecutionEvent `json:"events,omitempty"`
}

func graderAgentCoursePath(course, suffix string) string {
	return "/api/v1/courses/" + url.PathEscape(course) + suffix
}

func graderAgentAssignmentPath(course, assignment, suffix string) string {
	return graderAgentCoursePath(course, "/assignments/"+url.PathEscape(assignment)+suffix)
}

func httpToWSURL(baseURL, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("server URL is required")
	}
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	case strings.HasPrefix(base, "wss://"), strings.HasPrefix(base, "ws://"):
	default:
		return "", fmt.Errorf("unsupported server URL scheme: %s", baseURL)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path, nil
}

func normalizeGraderAgentSetPayload(raw []byte) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if cfg, ok := doc["config"]; ok && len(cfg) > 0 && string(cfg) != "null" {
		return cfg, nil
	}
	return raw, nil
}

func validateGraderAgentConfigPayload(raw []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	_, hasPrompt := doc["prompt"]
	_, hasGraph := doc["workflowGraph"]
	if !hasPrompt && !hasGraph {
		return fmt.Errorf("agent config must include prompt or workflowGraph")
	}
	if prompt, ok := doc["prompt"].(string); ok && strings.TrimSpace(prompt) == "" && !hasGraph {
		return fmt.Errorf("agent config prompt cannot be empty without workflowGraph")
	}
	return nil
}

func loadGraderAgentSetPayload(path string) ([]byte, error) {
	raw, err := readInputFile(path)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeGraderAgentSetPayload(raw)
	if err != nil {
		return nil, err
	}
	if err := validateGraderAgentConfigPayload(normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func parseAIOptOutResponse(body []byte) (bool, error) {
	var out struct {
		OptOut bool `json:"aiProcessingOptOut"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return false, fmt.Errorf("decoding AI opt-out response: %w", err)
	}
	return out.OptOut, nil
}

func fetchAIOptOut(c *client.Client) (bool, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/settings/ai-opt-out", nil)
	if err != nil {
		return false, nil, fmt.Errorf("building AI opt-out request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return false, nil, fmt.Errorf("fetching AI opt-out: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, nil, fmt.Errorf("reading AI opt-out response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, body, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, body, apiErrorBody(resp.StatusCode, body)
	}
	optedOut, err := parseAIOptOutResponse(body)
	return optedOut, body, err
}

func ensureAIGradingAllowed(c *client.Client) error {
	optedOut, _, err := fetchAIOptOut(c)
	if err != nil {
		return err
	}
	if optedOut {
		return fmt.Errorf("%s", graderAgentAIOptOutMessage)
	}
	return nil
}

func printGraderAgentDisclosure(c *client.Client, w io.Writer) error {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/public/ai-disclosure", nil)
	if err != nil {
		return nil
	}
	resp, err := c.Do(req)
	if err != nil || resp == nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var doc struct {
		Provider string `json:"provider"`
		Models   []struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Provider string   `json:"provider"`
			Purposes []string `json:"purposes"`
		} `json:"models"`
	}
	body, _ := io.ReadAll(resp.Body)
	if json.Unmarshal(body, &doc) != nil {
		return nil
	}
	if len(doc.Models) == 0 {
		return nil
	}
	model := doc.Models[0]
	_, _ = fmt.Fprintf(w, "AI disclosure: provider=%s model=%s (%s)\n",
		doc.Provider, model.ID, model.Name)
	return nil
}

func limitSubmissionIDs(ids []string, sample int) []string {
	if sample <= 0 || sample >= len(ids) {
		return ids
	}
	return ids[:sample]
}

func gradableSubmissionIDs(entries []assignmentSubmissionEntry) []string {
	var out []string
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) != "" {
			out = append(out, entry.ID)
		}
	}
	return out
}

func parseDryRunExecutionEvent(raw []byte) (dryRunExecutionEvent, error) {
	var ev dryRunExecutionEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return dryRunExecutionEvent{}, err
	}
	return ev, nil
}

func collectDryRunSampleResult(submissionID string, events []dryRunExecutionEvent) dryRunSampleResult {
	out := dryRunSampleResult{
		SubmissionID: submissionID,
		Events:       events,
	}
	for _, ev := range events {
		switch ev.Type {
		case "error":
			out.Error = ev.Message
		case "complete":
			if ev.Result != nil {
				out.SuggestedPoints = ev.Result.SuggestedPoints
				out.Comment = ev.Result.Comment
				out.Confidence = ev.Result.Confidence
				out.PromptTokens = ev.Result.PromptTokens
				out.CompletionTokens = ev.Result.CompletionTokens
				out.CostUSD = ev.Result.CostUSD
			}
		}
	}
	return out
}

func formatDryRunProgressLine(ev dryRunExecutionEvent) string {
	switch ev.Type {
	case "log":
		if ev.Message != "" {
			return ev.Message
		}
	case "node_start":
		if ev.NodeID != "" {
			return fmt.Sprintf("node %s started", ev.NodeID)
		}
	case "node_complete":
		if ev.Status != "" {
			return fmt.Sprintf("node %s %s", ev.NodeID, ev.Status)
		}
	case "complete":
		if ev.Result != nil {
			return fmt.Sprintf("suggested %.2f pts (confidence %.0f%%)",
				ev.Result.SuggestedPoints, ev.Result.Confidence*100)
		}
		return "dry run complete"
	case "error":
		return "error: " + ev.Message
	}
	return ""
}

func graderAgentRunIsTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func fetchGraderAgentConfig(c *client.Client, course, assignment string) (map[string]any, []byte, error) {
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("getting grader agent: %w", err)
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
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Config, body, nil
}

func workflowGraphFromConfig(cfg map[string]any) (map[string]any, error) {
	if cfg == nil {
		return nil, fmt.Errorf("no grader agent config found; run grader-agents set first")
	}
	graph, ok := cfg["workflowGraph"].(map[string]any)
	if !ok || len(graph) == 0 {
		return nil, fmt.Errorf("grader agent config has no workflowGraph; dry-run requires a workflow canvas config")
	}
	return graph, nil
}

func runGraderAgentDryRunWS(
	ctx context.Context,
	c *client.Client,
	course, assignment, submissionID string,
	workflowGraph map[string]any,
	onEvent func(dryRunExecutionEvent) error,
) ([]dryRunExecutionEvent, error) {
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/dry-run/ws")
	wsURL, err := httpToWSURL(c.BaseURL(), path)
	if err != nil {
		return nil, err
	}
	first, err := json.Marshal(map[string]any{
		"authToken":     c.APIKey(),
		"submissionId":  submissionID,
		"workflowGraph": workflowGraph,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding dry-run payload: %w", err)
	}
	var events []dryRunExecutionEvent
	err = wsclient.Stream(ctx, wsclient.StreamOptions{
		URL:        wsURL,
		FirstFrame: first,
		OnMessage: func(payload []byte) error {
			ev, parseErr := parseDryRunExecutionEvent(payload)
			if parseErr != nil {
				return parseErr
			}
			events = append(events, ev)
			if onEvent != nil {
				return onEvent(ev)
			}
			return nil
		},
	})
	if err != nil {
		return events, err
	}
	return events, nil
}

func fetchGraderAgentRunEstimate(c *client.Client, course, assignment, scope string, overwrite bool) (map[string]any, []byte, error) {
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/run-estimate")
	q := url.Values{}
	if scope != "" {
		q.Set("scope", scope)
	}
	if overwrite {
		q.Set("overwrite", "true")
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching run estimate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, body, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func formatRunEstimateLine(estimate map[string]any) string {
	if estimate == nil {
		return ""
	}
	var parts []string
	if v, ok := estimate["submissionCount"].(float64); ok {
		parts = append(parts, fmt.Sprintf("%d submissions", int(v)))
	}
	if v, ok := estimate["estimatedCostMinUsd"].(float64); ok {
		if max, ok2 := estimate["estimatedCostMaxUsd"].(float64); ok2 {
			parts = append(parts, fmt.Sprintf("est. cost $%.4f–$%.4f", v, max))
		} else {
			parts = append(parts, fmt.Sprintf("est. cost $%.4f", v))
		}
	}
	if v, ok := estimate["estimatedPromptTokens"].(float64); ok {
		parts = append(parts, fmt.Sprintf("est. prompt tokens %.0f", v))
	}
	return strings.Join(parts, "; ")
}

func postGraderAgentRun(c *client.Client, course, assignment string, payload map[string]any) (map[string]any, []byte, error) {
	raw, _ := json.Marshal(payload)
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/runs")
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("starting grader agent run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchGraderAgentRun(c *client.Client, course, assignment, runID string) (map[string]any, []byte, error) {
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/runs/"+url.PathEscape(runID))
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("getting grader agent run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func waitForGraderAgentRun(c *client.Client, course, assignment, runID string, timeout time.Duration, onTick func(map[string]any)) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for {
		run, _, err := fetchGraderAgentRun(c, course, assignment, runID)
		if err != nil {
			return nil, err
		}
		if onTick != nil {
			onTick(run)
		}
		status, _ := run["status"].(string)
		if graderAgentRunIsTerminal(status) {
			return run, nil
		}
		if time.Now().After(deadline) {
			return run, fmt.Errorf("timed out waiting for run %s (status=%s)", runID, status)
		}
		time.Sleep(2 * time.Second)
	}
}

func fetchLatestGraderAgentRunID(c *client.Client, course, assignment string) (string, error) {
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/runs")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", fmt.Errorf("listing grader agent runs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("run history is not enabled; pass --run <run-id> from grader-agents run output")
	}
	if resp.StatusCode != http.StatusOK {
		return "", apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Runs []map[string]any `json:"runs"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if len(out.Runs) == 0 {
		return "", fmt.Errorf("no grader agent runs found; pass --run <run-id>")
	}
	return stringField(out.Runs[0], "id"), nil
}

func postGraderAgentReviewBulk(c *client.Client, course, assignment string, payload map[string]any) (map[string]any, []byte, error) {
	raw, _ := json.Marshal(payload)
	path := graderAgentAssignmentPath(course, assignment, "/grader-agent/review/bulk")
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("bulk review: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func formatRunUsageLines(run map[string]any) []string {
	var lines []string
	if v, ok := run["model"].(string); ok && v != "" {
		lines = append(lines, "Model: "+v)
	}
	if v, ok := run["promptTokens"].(float64); ok && v > 0 {
		lines = append(lines, fmt.Sprintf("Prompt tokens: %.0f", v))
	}
	if v, ok := run["completionTokens"].(float64); ok && v > 0 {
		lines = append(lines, fmt.Sprintf("Completion tokens: %.0f", v))
	}
	if v, ok := run["costUsd"].(float64); ok && v > 0 {
		lines = append(lines, fmt.Sprintf("Cost USD: %.4f", v))
	}
	return lines
}