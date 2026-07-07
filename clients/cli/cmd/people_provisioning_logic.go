package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const (
	userImportAsyncThreshold = 100
	importJobPollInterval    = 2 * time.Second
)

type adminConsoleUser struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type userImportRow struct {
	Email  string
	Name   string
	Role   string
	SISID  string
	Line   int
}

type userImportSummary struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

type importJobStatus struct {
	JobID         string `json:"jobId"`
	Status        string `json:"status"`
	ProcessedRows int    `json:"processedRows"`
	ErrorRows     int    `json:"errorRows"`
	CreatedCount  int    `json:"createdCount"`
	UpdatedCount  int    `json:"updatedCount"`
	SkippedCount  int    `json:"skippedCount"`
	HasResult     bool   `json:"hasResult"`
}

func adminConsoleUserPath(userID string) string {
	return "/api/v1/admin-console/users/" + url.PathEscape(userID)
}

func fetchAdminConsoleUser(c *client.Client, userID, orgID string) (adminConsoleUser, []byte, error) {
	path := adminConsoleUserPath(userID)
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return adminConsoleUser{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return adminConsoleUser{}, nil, fmt.Errorf("getting user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return adminConsoleUser{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return adminConsoleUser{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out adminConsoleUser
	if err := json.Unmarshal(body, &out); err != nil {
		return adminConsoleUser{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func patchAdminConsoleUser(c *client.Client, userID, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := adminConsoleUserPath(userID)
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
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

func deleteAdminConsoleUser(c *client.Client, userID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/people/"+url.PathEscape(userID), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func parseUserImportCSV(path string) ([]userImportRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening CSV: %w", err)
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must include a header row and at least one data row")
	}
	header := make(map[string]int)
	for i, col := range records[0] {
		header[strings.ToLower(strings.TrimSpace(col))] = i
	}
	emailIdx, ok := header["email"]
	if !ok {
		return nil, fmt.Errorf("CSV must include an email column")
	}
	nameIdx := header["name"]
	roleIdx := header["role"]
	sisIdx, ok := header["sis_id"]
	if !ok {
		sisIdx = header["sisid"]
	}

	rows := make([]userImportRow, 0, len(records)-1)
	for i, rec := range records[1:] {
		if len(rec) == 0 {
			continue
		}
		email := ""
		if emailIdx < len(rec) {
			email = strings.TrimSpace(rec[emailIdx])
		}
		if email == "" {
			continue
		}
		row := userImportRow{Email: email, Line: i + 2}
		if nameIdx >= 0 && nameIdx < len(rec) {
			row.Name = strings.TrimSpace(rec[nameIdx])
		}
		if roleIdx >= 0 && roleIdx < len(rec) {
			row.Role = strings.TrimSpace(rec[roleIdx])
		}
		if sisIdx >= 0 && sisIdx < len(rec) {
			row.SISID = strings.TrimSpace(rec[sisIdx])
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func redactSecretsFromJSON(raw []byte) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	redactSecretFields(v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

func redactSecretFields(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			kl := strings.ToLower(k)
			if strings.Contains(kl, "password") || strings.Contains(kl, "secret") || strings.Contains(kl, "token") {
				t[k] = "[REDACTED]"
				continue
			}
			redactSecretFields(val)
		}
	case []any:
		for _, item := range t {
			redactSecretFields(item)
		}
	}
}

func submitImportJob(c *client.Client, orgID, filePath string, dryRun bool, mergeStrategy, profile string) (map[string]any, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filePath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	_ = w.WriteField("dry_run", fmt.Sprintf("%t", dryRun))
	if mergeStrategy != "" {
		_ = w.WriteField("merge_strategy", mergeStrategy)
	}
	if profile != "" {
		_ = w.WriteField("profile", profile)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	path := "/api/v1/admin-console/imports"
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodPost, path, &buf)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("submitting import: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func fetchImportJobStatus(c *client.Client, orgID, jobID string) (importJobStatus, []byte, error) {
	path := "/api/v1/admin-console/imports/" + url.PathEscape(jobID)
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return importJobStatus{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return importJobStatus{}, nil, fmt.Errorf("getting import status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return importJobStatus{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return importJobStatus{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out importJobStatus
	if err := json.Unmarshal(body, &out); err != nil {
		return importJobStatus{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func waitForImportJob(c *client.Client, orgID, jobID string, timeout time.Duration, onTick func(importJobStatus)) (importJobStatus, error) {
	deadline := time.Now().Add(timeout)
	for {
		status, _, err := fetchImportJobStatus(c, orgID, jobID)
		if err != nil {
			return importJobStatus{}, err
		}
		if onTick != nil {
			onTick(status)
		}
		switch strings.ToLower(status.Status) {
		case "complete", "completed", "failed", "error":
			return status, nil
		}
		if time.Now().After(deadline) {
			return status, fmt.Errorf("import job %s did not complete within %s", jobID, timeout)
		}
		time.Sleep(importJobPollInterval)
	}
}

func writeSecretsFile(path string, rows []map[string]string) error {
	if path == "" || len(rows) == 0 {
		return nil
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"email", "temporaryPassword"})
	for _, row := range rows {
		_ = w.Write([]string{row["email"], row["password"]})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}