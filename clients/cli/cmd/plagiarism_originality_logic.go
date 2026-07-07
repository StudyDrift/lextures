package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const ferpaOriginalityExportWarning = `WARNING: Originality export contains FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

type plagiarismSettingsJSON struct {
	PlagiarismChecksEnabled     bool     `json:"plagiarismChecksEnabled"`
	PlagiarismProvider          *string  `json:"plagiarismProvider"`
	PlagiarismAlertThresholdPct float64  `json:"plagiarismAlertThresholdPct"`
}

type originalityReportJSON struct {
	Provider      string   `json:"provider"`
	Status        string   `json:"status"`
	SimilarityPct *float64 `json:"similarityPct"`
	AIProbability *float64 `json:"aiProbability"`
	ReportURL     *string  `json:"reportUrl"`
	ReportToken   *string  `json:"reportToken"`
	ErrorMessage  *string  `json:"errorMessage"`
}

type originalityReportsBody struct {
	Reports []originalityReportJSON `json:"reports"`
}

type originalitySummaryJSON struct {
	Provider                     string   `json:"provider"`
	SimilarityPct                *float64 `json:"similarityPct"`
	AIProbability                *float64 `json:"aiProbability"`
	DetectedAt                   string   `json:"detectedAt"`
	FullReportUnavailable        bool     `json:"fullReportUnavailable"`
	FullReportUnavailableMessage string   `json:"fullReportUnavailableMessage"`
}

type originalitySummaryBody struct {
	Summary originalitySummaryJSON `json:"summary"`
}

type originalityEmbedBody struct {
	Summary  originalitySummaryJSON `json:"summary"`
	EmbedURL *string                `json:"embedUrl"`
}

type originalityListRow struct {
	UserID          string   `json:"userId"`
	DisplayName     string   `json:"displayName"`
	SubmissionID    string   `json:"submissionId"`
	Provider        string   `json:"provider"`
	Status          string   `json:"status"`
	SimilarityPct   *float64 `json:"similarityPct"`
	AIProbability   *float64 `json:"aiProbability"`
	ReportURL       string   `json:"reportUrl,omitempty"`
	StatusMessage   string   `json:"statusMessage,omitempty"`
}

func confirmOriginalityExport(confirmed bool) error {
	if confirmed {
		return nil
	}
	return fmt.Errorf("%s", ferpaOriginalityExportWarning)
}

func parsePlagiarismPolicyJSON(raw []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	allowed := map[string]bool{
		"plagiarismChecksEnabled":     true,
		"plagiarismProvider":          true,
		"plagiarismAlertThresholdPct": true,
	}
	for key := range payload {
		if !allowed[key] {
			return nil, fmt.Errorf("unknown field %q in policy file", key)
		}
	}
	if provider, ok := payload["plagiarismProvider"]; ok && provider != nil {
		providerStr, ok := provider.(string)
		if !ok {
			return nil, fmt.Errorf("plagiarismProvider must be a string")
		}
		normalized := strings.ToLower(strings.TrimSpace(providerStr))
		switch normalized {
		case "":
			delete(payload, "plagiarismProvider")
		case "none", "turnitin", "copyleaks", "gptzero":
			payload["plagiarismProvider"] = normalized
		default:
			return nil, fmt.Errorf("invalid plagiarismProvider %q", providerStr)
		}
	}
	if threshold, ok := payload["plagiarismAlertThresholdPct"]; ok && threshold != nil {
		value, err := coerceFloat64(threshold)
		if err != nil {
			return nil, fmt.Errorf("plagiarismAlertThresholdPct must be a number")
		}
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("plagiarismAlertThresholdPct must be between 0 and 100")
		}
		payload["plagiarismAlertThresholdPct"] = value
	}
	if enabled, ok := payload["plagiarismChecksEnabled"]; ok && enabled != nil {
		if _, ok := enabled.(bool); !ok {
			return nil, fmt.Errorf("plagiarismChecksEnabled must be a boolean")
		}
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("policy file is empty")
	}
	return payload, nil
}

func coerceFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case json.Number:
		return n.Float64()
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("not a number")
	}
}

func paginateSlice[T any](items []T, page, limit int) []T {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	start := (page - 1) * limit
	if start >= len(items) {
		return nil
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func originalitySubmissionPath(courseCode, itemID, submissionID, suffix string) string {
	base := assignmentAPIPath(
		courseCode,
		"/assignments/"+url.PathEscape(itemID)+"/submissions/"+url.PathEscape(submissionID)+"/originality",
	)
	if suffix == "" {
		return base
	}
	return base + suffix
}

func resolveAssignmentSubmission(c *client.Client, courseCode, itemID, userID string) (assignmentSubmissionEntry, error) {
	body, _, err := fetchAssignmentSubmissions(c, courseCode, itemID, "")
	if err != nil {
		return assignmentSubmissionEntry{}, err
	}
	entries := filterSubmissions(body.Submissions, "submitted", userID, false, nil)
	if len(entries) == 0 {
		return assignmentSubmissionEntry{}, fmt.Errorf("no submission found for user %s", userID)
	}
	return entries[0], nil
}

func fetchPlagiarismSettings(c *client.Client, courseCode string) ([]byte, plagiarismSettingsJSON, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/plagiarism-settings"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, plagiarismSettingsJSON{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, plagiarismSettingsJSON{}, fmt.Errorf("getting plagiarism settings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, plagiarismSettingsJSON{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return body, plagiarismSettingsJSON{}, apiErrorBody(resp.StatusCode, body)
	}
	var settings plagiarismSettingsJSON
	if err := json.Unmarshal(body, &settings); err != nil {
		return body, plagiarismSettingsJSON{}, fmt.Errorf("decoding response: %w", err)
	}
	return body, settings, nil
}

func patchPlagiarismSettings(c *client.Client, courseCode string, patch map[string]any) ([]byte, error) {
	raw, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("encoding patch: %w", err)
	}
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/plagiarism-settings"
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating plagiarism settings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchOriginalitySummary(c *client.Client, courseCode, itemID, submissionID string) (originalitySummaryBody, []byte, error) {
	path := originalitySubmissionPath(courseCode, itemID, submissionID, "/summary")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return originalitySummaryBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return originalitySummaryBody{}, nil, fmt.Errorf("getting originality status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return originalitySummaryBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return originalitySummaryBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out originalitySummaryBody
	if err := json.Unmarshal(body, &out); err != nil {
		return originalitySummaryBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchOriginalityReports(c *client.Client, courseCode, itemID, submissionID string) (originalityReportsBody, []byte, error) {
	path := originalitySubmissionPath(courseCode, itemID, submissionID, "")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return originalityReportsBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return originalityReportsBody{}, nil, fmt.Errorf("getting originality report: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return originalityReportsBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return originalityReportsBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out originalityReportsBody
	if err := json.Unmarshal(body, &out); err != nil {
		return originalityReportsBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func fetchOriginalityEmbed(c *client.Client, courseCode, itemID, submissionID string) (originalityEmbedBody, []byte, error) {
	path := originalitySubmissionPath(courseCode, itemID, submissionID, "/embed-url")
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return originalityEmbedBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return originalityEmbedBody{}, nil, fmt.Errorf("getting originality embed URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return originalityEmbedBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return originalityEmbedBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out originalityEmbedBody
	if err := json.Unmarshal(body, &out); err != nil {
		return originalityEmbedBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func postOriginalityRetry(c *client.Client, courseCode, itemID, submissionID string) ([]byte, map[string]any, error) {
	path := originalitySubmissionPath(courseCode, itemID, submissionID, "/retry")
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("submitting originality check: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return body, nil, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return body, nil, fmt.Errorf("decoding response: %w", err)
	}
	return body, out, nil
}

func originalityStatusFromSummary(summary originalitySummaryJSON) string {
	if summary.FullReportUnavailable {
		if msg := strings.TrimSpace(summary.FullReportUnavailableMessage); msg != "" {
			return msg
		}
		return "pending"
	}
	if summary.DetectedAt != "" {
		return "complete"
	}
	return "unknown"
}

func bestOriginalityReport(reports []originalityReportJSON) *originalityReportJSON {
	var best *originalityReportJSON
	for i := range reports {
		report := &reports[i]
		if report.Status != "done" {
			continue
		}
		if best == nil {
			best = report
			continue
		}
		if report.SimilarityPct != nil && (best.SimilarityPct == nil || *report.SimilarityPct > *best.SimilarityPct) {
			best = report
		}
	}
	if best != nil {
		return best
	}
	if len(reports) > 0 {
		return &reports[len(reports)-1]
	}
	return nil
}

func reportURLFromReports(reports []originalityReportJSON) string {
	for _, report := range reports {
		if report.ReportURL != nil && strings.TrimSpace(*report.ReportURL) != "" {
			return strings.TrimSpace(*report.ReportURL)
		}
	}
	return ""
}

func formatOptionalPercent(value *float64) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64) + "%"
}

func buildOriginalityListRows(c *client.Client, courseCode, itemID string, entries []assignmentSubmissionEntry) ([]originalityListRow, error) {
	rows := make([]originalityListRow, 0, len(entries))
	for _, entry := range entries {
		row := originalityListRow{
			UserID:       entry.SubmittedBy,
			DisplayName:  entry.SubmittedByDisplayName,
			SubmissionID: entry.ID,
		}
		if strings.TrimSpace(entry.ID) == "" {
			row.Status = "missing"
			rows = append(rows, row)
			continue
		}
		summaryBody, _, err := fetchOriginalitySummary(c, courseCode, itemID, entry.ID)
		if err != nil {
			return nil, err
		}
		summary := summaryBody.Summary
		row.Provider = summary.Provider
		row.SimilarityPct = summary.SimilarityPct
		row.AIProbability = summary.AIProbability
		row.Status = originalityStatusFromSummary(summary)
		row.StatusMessage = strings.TrimSpace(summary.FullReportUnavailableMessage)
		rows = append(rows, row)
	}
	return rows, nil
}

func writeOriginalityCSV(rows []originalityListRow) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{
		"user_id", "display_name", "submission_id", "provider", "status",
		"similarity_pct", "ai_probability", "report_url",
	}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.UserID,
			row.DisplayName,
			row.SubmissionID,
			row.Provider,
			row.Status,
			formatOptionalPercent(row.SimilarityPct),
			formatOptionalPercent(row.AIProbability),
			row.ReportURL,
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}