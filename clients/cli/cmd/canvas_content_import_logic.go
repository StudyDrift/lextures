package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/lextures/lextures/clients/cli/internal/client"
)

type canvasCourseItem struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CourseCode    string `json:"courseCode,omitempty"`
	WorkflowState string `json:"workflowState,omitempty"`
	TermName      string `json:"termName,omitempty"`
}

type canvasImportInclude struct {
	Modules       bool `json:"modules"`
	Assignments   bool `json:"assignments"`
	Quizzes       bool `json:"quizzes"`
	Enrollments   bool `json:"enrollments"`
	Grades        bool `json:"grades"`
	Settings      bool `json:"settings"`
	Files         bool `json:"files"`
	Announcements bool `json:"announcements"`
}

type canvasImportJobStatus struct {
	JobID      string `json:"jobId"`
	Status     string `json:"status"`
	Progress   string `json:"progress,omitempty"`
	CourseCode string `json:"courseCode,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

func readCanvasToken(token, tokenFile string) (string, error) {
	if tokenFile != "" {
		raw, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}
	if strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), nil
	}
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<16))
		if err != nil {
			return "", err
		}
		t := strings.TrimSpace(string(raw))
		if t != "" {
			return t, nil
		}
	}
	return "", fmt.Errorf("canvas access token required via --token-file or stdin")
}

func listCanvasCatalog(c *client.Client, canvasBase, accessToken string) ([]canvasCourseItem, []byte, error) {
	payload, _ := json.Marshal(map[string]string{
		"canvasBaseUrl": canvasBase,
		"accessToken":   accessToken,
	})
	req, err := c.NewRequest(http.MethodPost, "/api/v1/integrations/canvas/courses", bytes.NewReader(payload))
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
		Courses []canvasCourseItem `json:"courses"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Courses, body, nil
}

func filterCanvasCourses(courses []canvasCourseItem, query string) []canvasCourseItem {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return courses
	}
	filtered := make([]canvasCourseItem, 0, len(courses))
	for _, c := range courses {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.CourseCode), q) ||
			fmt.Sprintf("%d", c.ID) == q {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func includeForArtifact(artifact string) canvasImportInclude {
	switch strings.ToLower(strings.TrimSpace(artifact)) {
	case "enrollments":
		return canvasImportInclude{Enrollments: true}
	case "grades":
		return canvasImportInclude{Grades: true}
	case "submissions":
		return canvasImportInclude{Assignments: true, Grades: true}
	case "announcements":
		return canvasImportInclude{Announcements: true}
	default:
		return canvasImportInclude{
			Modules: true, Assignments: true, Quizzes: true, Enrollments: true,
			Grades: true, Settings: true, Files: true, Announcements: true,
		}
	}
}

func submitCanvasImport(
	c *client.Client,
	courseCode, mode, canvasBase, canvasCourseID, accessToken string,
	include canvasImportInclude,
	gradeSync *bool,
) (string, []byte, error) {
	payload := map[string]any{
		"mode":           mode,
		"canvasBaseUrl":  canvasBase,
		"canvasCourseId": canvasCourseID,
		"accessToken":    accessToken,
		"include":        include,
	}
	if gradeSync != nil {
		payload["canvasGradeSyncEnabled"] = *gradeSync
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/import/canvas"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return "", nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, err
	}
	return out.JobID, body, nil
}

func canvasImportTerminal(msgType string) bool {
	switch msgType {
	case "complete", "error":
		return true
	default:
		return false
	}
}

func wsURLFromHTTP(serverURL, path string) (string, error) {
	u, err := url.Parse(strings.TrimRight(serverURL, "/"))
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported server scheme %q", u.Scheme)
	}
	u.Path = path
	return u.String(), nil
}

func waitForCanvasImportJob(c *client.Client, apiToken, jobID string, timeout time.Duration, onTick func(canvasImportJobStatus)) (canvasImportJobStatus, error) {
	wsPath := "/api/v1/ws/canvas-import/" + url.PathEscape(jobID)
	wsURL, err := wsURLFromHTTP(c.BaseURL(), wsPath)
	if err != nil {
		return canvasImportJobStatus{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return canvasImportJobStatus{}, fmt.Errorf("websocket dial: %w", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	authMsg, _ := json.Marshal(map[string]string{"authToken": apiToken})
	if err := conn.Write(ctx, websocket.MessageText, authMsg); err != nil {
		return canvasImportJobStatus{}, err
	}

	last := canvasImportJobStatus{JobID: jobID}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return last, err
		}
		var msg struct {
			Type       string `json:"type"`
			Message    string `json:"message"`
			CourseCode string `json:"courseCode"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "progress":
			last.Progress = msg.Message
			last.Status = "processing"
		case "complete":
			last.Status = "completed"
			last.CourseCode = msg.CourseCode
		case "error":
			last.Status = "failed"
			last.Error = msg.Message
		}
		if onTick != nil {
			onTick(last)
		}
		if canvasImportTerminal(msg.Type) {
			if msg.Type == "error" {
				return last, fmt.Errorf("canvas import failed: %s", msg.Message)
			}
			return last, nil
		}
	}
}

func fetchCourseCanvasLink(c *client.Client, courseCode string) (map[string]any, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/canvas-link"
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
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out, body, nil
}

type contentImportJob struct {
	ID               string `json:"id"`
	CourseID         string `json:"courseId"`
	ImportType       string `json:"importType"`
	OriginalFilename string `json:"originalFilename"`
	Status           string `json:"status"`
	TotalItems       int    `json:"totalItems"`
	ProcessedItems   int    `json:"processedItems"`
	SucceededItems   int    `json:"succeededItems"`
	FailedItems      int    `json:"failedItems"`
	SkippedItems     int    `json:"skippedItems"`
}

func submitContentImport(c *client.Client, courseID, filePath string) (string, []byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = f.Close() }()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("course_id", courseID)
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", nil, err
	}
	_ = w.Close()

	req, err := c.NewRequest(http.MethodPost, "/api/v1/imports/qti", &buf)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, err
	}
	return out.JobID, body, nil
}

func fetchContentImportStatus(c *client.Client, jobID string) (contentImportJob, []byte, error) {
	path := "/api/v1/imports/" + url.PathEscape(jobID) + "/status"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return contentImportJob{}, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return contentImportJob{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return contentImportJob{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return contentImportJob{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Status         string `json:"status"`
		TotalItems     int    `json:"totalItems"`
		ProcessedItems int    `json:"processedItems"`
		SucceededItems int    `json:"succeededItems"`
		FailedItems    int    `json:"failedItems"`
		SkippedItems   int    `json:"skippedItems"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return contentImportJob{}, body, err
	}
	return contentImportJob{
		ID:             jobID,
		Status:         out.Status,
		TotalItems:     out.TotalItems,
		ProcessedItems: out.ProcessedItems,
		SucceededItems: out.SucceededItems,
		FailedItems:    out.FailedItems,
		SkippedItems:   out.SkippedItems,
	}, body, nil
}

func listContentImports(c *client.Client, courseID string) ([]contentImportJob, []byte, error) {
	path := "/api/v1/imports"
	if courseID != "" {
		path += "?course_id=" + url.QueryEscape(courseID)
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
		Imports []contentImportJob `json:"imports"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Imports, body, nil
}

func contentImportTerminal(status string) bool {
	switch strings.ToLower(status) {
	case "done", "completed", "complete", "failed", "error":
		return true
	default:
		return false
	}
}

func waitForContentImport(c *client.Client, jobID string, timeout time.Duration, onTick func(contentImportJob)) (contentImportJob, error) {
	deadline := time.Now().Add(timeout)
	var last contentImportJob
	for {
		status, _, err := fetchContentImportStatus(c, jobID)
		if err != nil {
			return last, err
		}
		last = status
		if onTick != nil {
			onTick(status)
		}
		if contentImportTerminal(status.Status) {
			if strings.EqualFold(status.Status, "failed") || strings.EqualFold(status.Status, "error") {
				return last, fmt.Errorf("import %s failed", jobID)
			}
			return last, nil
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("import %s did not complete within %s", jobID, timeout)
		}
		time.Sleep(jobPollInterval)
	}
}