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
	"path/filepath"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const ferpaReportsExportWarning = `WARNING: Report exports may include FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

type reportTypeInfo struct {
	ID          string
	Description string
	Scope       string
	Formats     []string
	Async       bool
}

var reportCatalog = []reportTypeInfo{
	{ID: "learning-activity", Description: "Platform learning activity summary", Scope: "platform", Formats: []string{"json", "csv", "pdf"}, Async: false},
	{ID: "gradebook", Description: "Course gradebook PDF", Scope: "course", Formats: []string{"pdf"}, Async: false},
	{ID: "progress", Description: "Student progress PDF", Scope: "course", Formats: []string{"pdf"}, Async: false},
}

func lookupReportType(id string) (reportTypeInfo, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, r := range reportCatalog {
		if r.ID == id {
			return r, true
		}
	}
	return reportTypeInfo{}, false
}

type learningActivityReport struct {
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`
	Summary struct {
		TotalEvents   int64 `json:"totalEvents"`
		UniqueUsers   int64 `json:"uniqueUsers"`
		UniqueCourses int64 `json:"uniqueCourses"`
	} `json:"summary"`
	ByDay []struct {
		Day          string `json:"day"`
		CourseVisit  int64  `json:"courseVisit"`
		ContentOpen  int64  `json:"contentOpen"`
		ContentLeave int64  `json:"contentLeave"`
	} `json:"byDay"`
	ByEventKind []struct {
		EventKind string `json:"eventKind"`
		Count     int64  `json:"count"`
	} `json:"byEventKind"`
	TopCourses []struct {
		CourseID   string `json:"courseId"`
		CourseCode string `json:"courseCode"`
		Title      string `json:"title"`
		EventCount int64  `json:"eventCount"`
	} `json:"topCourses"`
}

func fetchLearningActivityReport(c *client.Client, from, to string) (learningActivityReport, []byte, error) {
	q := url.Values{}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	path := "/api/v1/reports/learning-activity"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return learningActivityReport{}, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return learningActivityReport{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return learningActivityReport{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return learningActivityReport{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out learningActivityReport
	if err := json.Unmarshal(body, &out); err != nil {
		return learningActivityReport{}, body, err
	}
	return out, body, nil
}

func fetchCourseEngagementOverview(c *client.Client, courseCode string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/analytics/engagement-overview"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func downloadBinaryReport(c *client.Client, path string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func learningActivityToCSV(report learningActivityReport) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := 0
	_ = w.Write([]string{"section", "key", "value"})
	rows++
	_ = w.Write([]string{"summary", "totalEvents", fmt.Sprintf("%d", report.Summary.TotalEvents)})
	rows++
	_ = w.Write([]string{"summary", "uniqueUsers", fmt.Sprintf("%d", report.Summary.UniqueUsers)})
	rows++
	_ = w.Write([]string{"summary", "uniqueCourses", fmt.Sprintf("%d", report.Summary.UniqueCourses)})
	rows++
	for _, d := range report.ByDay {
		_ = w.Write([]string{"byDay", d.Day, fmt.Sprintf("visits=%d open=%d leave=%d", d.CourseVisit, d.ContentOpen, d.ContentLeave)})
		rows++
	}
	for _, k := range report.ByEventKind {
		_ = w.Write([]string{"byEventKind", k.EventKind, fmt.Sprintf("%d", k.Count)})
		rows++
	}
	for _, c := range report.TopCourses {
		_ = w.Write([]string{"topCourses", c.CourseCode, fmt.Sprintf("%s (%d)", c.Title, c.EventCount)})
		rows++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), rows, nil
}

func writeExportOutput(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func resolveExportPath(outDir, filename string) string {
	if outDir == "" {
		return filename
	}
	return filepath.Join(outDir, filename)
}

type reportScheduleRow struct {
	ID         string `json:"id"`
	ReportType string `json:"reportType"`
	Cadence    string `json:"cadence"`
	Enabled    bool   `json:"enabled"`
	NextRunAt  string `json:"nextRunAt"`
}

func fetchReportSchedules(c *client.Client) ([]reportScheduleRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/reports/schedules", nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var rows []reportScheduleRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, body, err
	}
	return rows, body, nil
}

func createReportSchedule(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/reports/schedules", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func deleteReportSchedule(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/reports/schedules/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func normalizeExportFormat(format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "csv", "json", "ndjson", "pdf":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported format %q: use csv, json, ndjson, or pdf", format)
	}
}