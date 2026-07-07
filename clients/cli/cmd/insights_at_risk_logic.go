package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const ferpaAtRiskExportWarning = `WARNING: At-risk exports include FERPA-covered student records.
Re-run with --yes to confirm you are authorized to export this data.`

type atRiskAlert struct {
	ID             string   `json:"id"`
	EnrollmentID   string   `json:"enrollmentId"`
	UserID         string   `json:"userId"`
	DisplayName    string   `json:"displayName"`
	Score          float32  `json:"score"`
	Status         string   `json:"status"`
	TopFactor      string   `json:"topFactor"`
	TopFactorLabel string   `json:"topFactorLabel"`
	CourseCode     string   `json:"courseCode,omitempty"`
	TriggeredDate  string   `json:"triggeredDate"`
	MissingPct     *float32 `json:"missingPct,omitempty"`
	QuizAvg        *float32 `json:"quizAvg,omitempty"`
	DaysInactive   *int     `json:"daysInactive,omitempty"`
}

type atRiskScorePoint struct {
	Date         string   `json:"date"`
	Score        float32  `json:"score"`
	MissingPct   *float32 `json:"missingPct,omitempty"`
	QuizAvg      *float32 `json:"quizAvg,omitempty"`
	DaysInactive int      `json:"daysInactive"`
	TopFactor    string   `json:"topFactor"`
}

func parseAtRiskThreshold(raw string) (float32, bool, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return 0, false, nil
	}
	switch raw {
	case "high":
		return 75, true, nil
	case "medium", "med":
		return 50, true, nil
	case "low":
		return 25, true, nil
	default:
		v, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return 0, false, fmt.Errorf("threshold must be high, medium, low, or a numeric score")
		}
		return float32(v), true, nil
	}
}

func filterAtRiskAlerts(alerts []atRiskAlert, threshold float32, hasThreshold bool, factor string) []atRiskAlert {
	factor = strings.ToLower(strings.TrimSpace(factor))
	out := make([]atRiskAlert, 0, len(alerts))
	for _, a := range alerts {
		if hasThreshold && a.Score < threshold {
			continue
		}
		if factor != "" && strings.ToLower(a.TopFactor) != factor {
			continue
		}
		out = append(out, a)
	}
	return out
}

func fetchCourseAtRiskAlerts(c *client.Client, courseCode string, includeResolved bool) ([]atRiskAlert, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/at-risk"
	if includeResolved {
		path += "?includeResolved=true"
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	var out struct {
		Alerts   []atRiskAlert `json:"alerts"`
		Resolved []atRiskAlert `json:"resolved"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	alerts := append(out.Alerts, out.Resolved...)
	for i := range alerts {
		alerts[i].CourseCode = courseCode
	}
	return alerts, body, nil
}

func fetchOrgAtRiskAlerts(c *client.Client, orgID string, includeResolved bool) ([]atRiskAlert, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses", nil)
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
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var listed coursesListBody
	if err := json.Unmarshal(body, &listed); err != nil {
		return nil, err
	}
	orgID = strings.TrimSpace(orgID)
	var alerts []atRiskAlert
	for _, course := range listed.Courses {
		if course.Archived {
			continue
		}
		if orgID != "" {
			if course.OrgID == nil || strings.TrimSpace(*course.OrgID) != orgID {
				continue
			}
		}
		rows, _, err := fetchCourseAtRiskAlerts(c, course.CourseCode, includeResolved)
		if err != nil {
			continue
		}
		alerts = append(alerts, rows...)
	}
	return alerts, nil
}

func postAtRiskRecompute(c *client.Client, courseCode string) ([]byte, error) {
	var path string
	var payload []byte
	if strings.TrimSpace(courseCode) != "" {
		path = "/api/v1/courses/" + url.PathEscape(courseCode) + "/at-risk/run"
	} else {
		path = "/api/v1/admin/at-risk/run"
		if courseCode != "" {
			raw, err := json.Marshal(map[string]string{"courseCode": courseCode})
			if err != nil {
				return nil, err
			}
			payload = raw
		}
	}
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
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

func fetchEnrollmentAtRiskHistory(c *client.Client, courseCode, enrollmentID string) ([]atRiskScorePoint, []byte, error) {
	path := fmt.Sprintf("/api/v1/courses/%s/enrollments/%s/at-risk-history",
		url.PathEscape(courseCode), url.PathEscape(enrollmentID))
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	var out struct {
		Scores []atRiskScorePoint `json:"scores"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Scores, body, nil
}

func atRiskAlertsToCSV(alerts []atRiskAlert) ([]byte, int, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"course", "userId", "enrollmentId", "displayName", "score", "topFactor", "topFactorLabel", "status", "triggeredDate"})
	rows := 1
	for _, a := range alerts {
		_ = w.Write([]string{
			a.CourseCode, a.UserID, a.EnrollmentID, a.DisplayName,
			fmt.Sprintf("%.1f", a.Score), a.TopFactor, a.TopFactorLabel, a.Status, a.TriggeredDate,
		})
		rows++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), rows, nil
}

func fetchCourseInsights(c *client.Client, courseCode string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/analytics/insights"
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

func fetchCourseCrossSection(c *client.Client, courseCode string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/analytics/cross-section"
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

type classroomQuestion struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Addressed bool   `json:"addressed"`
	CreatedAt string `json:"createdAt"`
}

func fetchCourseSignals(c *client.Client, courseCode string, includeAddressed bool) ([]classroomQuestion, []byte, error) {
	co, _, err := fetchCourseDetail(c, courseCode)
	if err != nil {
		return nil, nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(co.ID) + "/questions"
	if includeAddressed {
		path += "?includeAddressed=true"
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	var out struct {
		Questions []classroomQuestion `json:"questions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Questions, body, nil
}