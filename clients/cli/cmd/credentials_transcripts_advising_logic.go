package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const transcriptExportWarning = `WARNING: Transcripts and CCR exports are FERPA official records.
Re-run with --yes to confirm you are authorized to export this data.`

type credentialSummary struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	SourceType      string `json:"sourceType"`
	SourceID        string `json:"sourceId"`
	IssuedAt        string `json:"issuedAt"`
	VerificationURL string `json:"verificationUrl"`
	Revoked         bool   `json:"revoked"`
}

type transcriptRequest struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	DeliveryType string  `json:"deliveryType"`
	RequestedAt  string  `json:"requestedAt"`
	SubmittedAt  *string `json:"submittedAt,omitempty"`
}

type advisingNote struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"studentId"`
	AdvisorID        string  `json:"advisorId"`
	Content          string  `json:"content"`
	VisibleToStudent bool    `json:"visibleToStudent"`
	CreatedAt        string  `json:"createdAt"`
	AdvisorEmail     string  `json:"advisorEmail,omitempty"`
	AdvisorDisplay   *string `json:"advisorDisplayName,omitempty"`
}

type credentialRecipient struct {
	Email      string
	CourseCode string
	UserID     string
}

func parseCredentialRecipientsCSV(path string) ([]credentialRecipient, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV must include a header row and at least one recipient")
	}
	header := make(map[string]int)
	for i, col := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(col))] = i
	}
	emailIdx, okEmail := header["email"]
	userIdx, okUser := header["userid"]
	if !okEmail && !okUser {
		if e, ok := header["user_id"]; ok {
			userIdx = e
			okUser = true
		}
	}
	if !okEmail && !okUser {
		return nil, fmt.Errorf("CSV header must include email or userId")
	}
	courseIdx, okCourse := header["course"]
	if !okCourse {
		if c, ok := header["course_code"]; ok {
			courseIdx = c
			okCourse = true
		}
	}
	if !okCourse {
		if c, ok := header["coursecode"]; ok {
			courseIdx = c
			okCourse = true
		}
	}
	out := make([]credentialRecipient, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		rec := credentialRecipient{}
		if okEmail && emailIdx < len(row) {
			rec.Email = strings.TrimSpace(row[emailIdx])
		}
		if okUser && userIdx < len(row) {
			rec.UserID = strings.TrimSpace(row[userIdx])
		}
		if okCourse && courseIdx < len(row) {
			rec.CourseCode = strings.TrimSpace(row[courseIdx])
		}
		if rec.Email == "" && rec.UserID == "" {
			continue
		}
		out = append(out, rec)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid recipients found in CSV")
	}
	return out, nil
}

func dedupeCredentialRecipients(items []credentialRecipient) []credentialRecipient {
	seen := make(map[string]struct{}, len(items))
	out := make([]credentialRecipient, 0, len(items))
	for _, item := range items {
		key := item.UserID + "|" + strings.ToLower(item.Email) + "|" + strings.ToLower(item.CourseCode)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func fetchMyCredentials(c *client.Client) ([]credentialSummary, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/credentials", nil)
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
		Credentials []credentialSummary `json:"credentials"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Credentials, body, nil
}

func verifyCredential(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/credentials/" + url.PathEscape(id) + "/verify"
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

func downloadCredentialPDF(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/credentials/" + url.PathEscape(id) + "/download"
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

func fetchTranscriptRequests(c *client.Client, admin bool) ([]transcriptRequest, []byte, error) {
	path := "/api/v1/transcripts/requests"
	if admin {
		path = "/api/v1/admin/transcripts/requests"
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
		Requests []transcriptRequest `json:"requests"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Requests, body, nil
}

func submitTranscriptRequest(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/transcripts/requests", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func generateCCR(c *client.Client, sharePublicly bool) ([]byte, error) {
	raw, err := json.Marshal(map[string]bool{"sharePublicly": sharePublicly})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/ccr/generate", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func downloadCCR(c *client.Client, docID, format string) ([]byte, error) {
	q := url.Values{}
	if format != "" {
		q.Set("format", format)
	}
	path := "/api/v1/me/ccr/" + url.PathEscape(docID) + "/download"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
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

func fetchDegreeProgress(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/degree-progress", nil)
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

func fetchAdvisorNotes(c *client.Client, studentID string) ([]advisingNote, []byte, error) {
	path := "/api/v1/advisor/students/" + url.PathEscape(studentID) + "/notes"
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
		Notes []advisingNote `json:"notes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Notes, body, nil
}

func addAdvisorNote(c *client.Client, studentID, content string) ([]byte, error) {
	raw, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	path := "/api/v1/advisor/students/" + url.PathEscape(studentID) + "/notes"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchMyGoals(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/goals", nil)
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

func patchMyGoals(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/me/goals", bytes.NewReader(raw))
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

func readTextFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
