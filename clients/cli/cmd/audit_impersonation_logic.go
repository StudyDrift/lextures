package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type auditEventRow struct {
	EventID   string          `json:"eventId"`
	EventType string          `json:"eventType"`
	ActorID   string          `json:"actorId"`
	Timestamp string          `json:"timestamp"`
	OrgID     string          `json:"orgId,omitempty"`
	TargetID  string          `json:"targetId,omitempty"`
	TargetType string         `json:"targetType,omitempty"`
	ActorIP   string          `json:"actorIp,omitempty"`
	BeforeValue json.RawMessage `json:"beforeValue,omitempty"`
	AfterValue  json.RawMessage `json:"afterValue,omitempty"`
}

type auditLogFilters struct {
	ActorID   string
	EventType string
	From      string
	To        string
	TargetID  string
	OrgID     string
}

type adminSearchResult struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle"`
	Snippet  string  `json:"snippet,omitempty"`
	Path     string  `json:"path"`
	Score    float64 `json:"score,omitempty"`
}

func buildAuditLogQuery(filters auditLogFilters) string {
	q := url.Values{}
	if filters.ActorID != "" {
		q.Set("actorId", filters.ActorID)
	}
	if filters.EventType != "" {
		q.Set("eventType", filters.EventType)
	}
	if filters.From != "" {
		q.Set("from", filters.From)
	}
	if filters.To != "" {
		q.Set("to", filters.To)
	}
	if filters.TargetID != "" {
		q.Set("targetId", filters.TargetID)
	}
	if filters.OrgID != "" {
		q.Set("orgId", filters.OrgID)
	}
	if s := q.Encode(); s != "" {
		return "?" + s
	}
	return ""
}

func fetchAuditLog(c *client.Client, filters auditLogFilters) ([]auditEventRow, []byte, error) {
	path := "/api/v1/compliance/audit-log" + buildAuditLogQuery(filters)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing audit log: %w", err)
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
		Events []auditEventRow `json:"events"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Events, body, nil
}

func exportAuditLog(c *client.Client, filters auditLogFilters, format string) ([]byte, string, error) {
	q := buildAuditLogQuery(filters)
	sep := "?"
	if strings.Contains(q, "?") {
		sep = "&"
	} else if q == "" {
		sep = "?"
		q = ""
	}
	if format == "csv" {
		q += sep + "format=csv"
		sep = "&"
	}
	path := "/api/v1/compliance/audit-log/export" + q
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, "", fmt.Errorf("exporting audit log: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return body, "", apiErrorBody(resp.StatusCode, body)
	}
	contentType := resp.Header.Get("Content-Type")
	return body, contentType, nil
}

func mapAdminSearchType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "user", "users":
		return "users"
	case "course", "courses":
		return "courses"
	case "content":
		return "content"
	case "org", "orgs":
		return ""
	default:
		return ""
	}
}

func fetchAdminSearch(c *client.Client, query, typeFilter string) ([]adminSearchResult, []byte, error) {
	if len(strings.TrimSpace(query)) < 2 {
		return nil, nil, fmt.Errorf("query must be at least 2 characters")
	}
	mapped := mapAdminSearchType(typeFilter)
	if typeFilter != "" && mapped == "" {
		return nil, nil, fmt.Errorf("unsupported search type %q (use user, course, or content)", typeFilter)
	}
	q := url.Values{"q": {query}}
	if mapped != "" {
		q.Set("types", mapped)
	}
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/search?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("admin search: %w", err)
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
		Users   []adminSearchResult `json:"users"`
		Courses []adminSearchResult `json:"courses"`
		Content []adminSearchResult `json:"content"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	results := make([]adminSearchResult, 0, len(out.Users)+len(out.Courses)+len(out.Content))
	results = append(results, out.Users...)
	results = append(results, out.Courses...)
	results = append(results, out.Content...)
	return results, body, nil
}

func flattenAdminSearchResults(body []byte) ([]adminSearchResult, error) {
	var out struct {
		Users   []adminSearchResult `json:"users"`
		Courses []adminSearchResult `json:"courses"`
		Content []adminSearchResult `json:"content"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	results := make([]adminSearchResult, 0, len(out.Users)+len(out.Courses)+len(out.Content))
	results = append(results, out.Users...)
	results = append(results, out.Courses...)
	results = append(results, out.Content...)
	return results, nil
}

type identityMeResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   *string `json:"displayName"`
	Impersonating *struct {
		AdminID string `json:"adminId"`
	} `json:"impersonating,omitempty"`
}

func fetchIdentityMe(c *client.Client) (identityMeResponse, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return identityMeResponse{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return identityMeResponse{}, nil, fmt.Errorf("fetching identity: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return identityMeResponse{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return identityMeResponse{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out identityMeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return identityMeResponse{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func startImpersonation(c *client.Client, targetUserID string) (string, string, map[string]any, error) {
	payload, err := json.Marshal(map[string]string{"target_user_id": targetUserID})
	if err != nil {
		return "", "", nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin-console/impersonate", strings.NewReader(string(payload)))
	if err != nil {
		return "", "", nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return "", "", nil, fmt.Errorf("starting impersonation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return "", "", nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", nil, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ImpersonationToken string         `json:"impersonation_token"`
		ExpiresAt          string         `json:"expires_at"`
		Target             map[string]any `json:"target"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", nil, fmt.Errorf("decoding response: %w", err)
	}
	return out.ImpersonationToken, out.ExpiresAt, out.Target, nil
}

func stopImpersonation(c *client.Client) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin-console/impersonate/session", nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("stopping impersonation: %w", err)
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

func isImpersonationWriteBlock(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "writes_blocked_during_impersonation")
}

func auditTailBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 2 * time.Second
	}
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 30*time.Second {
		return 30 * time.Second
	}
	return d
}