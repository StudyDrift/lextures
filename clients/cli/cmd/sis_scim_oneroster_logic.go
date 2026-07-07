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
	"path/filepath"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type sisConnection struct {
	ID              string  `json:"id"`
	OrgID           string  `json:"orgId"`
	Vendor          string  `json:"vendor"`
	Market          string  `json:"market"`
	BaseURL         string  `json:"baseUrl"`
	ClientIDRef     string  `json:"clientIdRef"`
	ClientSecretRef string  `json:"clientSecretRef"`
	SyncSchedule    string  `json:"syncSchedule"`
	SyncMode        string  `json:"syncMode"`
	Active          bool    `json:"active"`
	LastSyncAt      *string `json:"lastSyncAt"`
}

type sisSyncLog struct {
	ID           string         `json:"id"`
	ConnectionID string         `json:"connectionId"`
	StartedAt    string         `json:"startedAt"`
	FinishedAt   *string        `json:"finishedAt"`
	Status       string         `json:"status"`
	Summary      map[string]any `json:"summary"`
	Errors       []any          `json:"errors"`
}

func redactSISConnection(m map[string]any) {
	for _, key := range []string{"clientIdRef", "clientSecretRef", "clientId", "clientSecret", "token", "password", "secret"} {
		if _, ok := m[key]; ok {
			m[key] = secretPlaceholder
		}
	}
}

func redactSISConnectionsList(body []byte) []byte {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	if conns, ok := root["connections"].([]any); ok {
		for _, item := range conns {
			if m, ok := item.(map[string]any); ok {
				redactSISConnection(m)
			}
		}
	}
	if conn, ok := root["connection"].(map[string]any); ok {
		redactSISConnection(conn)
	}
	out, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return out
}

func sisOrgPath(orgID, subpath string) string {
	return "/api/v1/admin/orgs/" + url.PathEscape(strings.TrimSpace(orgID)) + subpath
}

func fetchSISConnections(c *client.Client, orgID string) ([]sisConnection, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, sisOrgPath(orgID, "/sis/connections"), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing SIS connections: %w", err)
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
		Connections []sisConnection `json:"connections"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Connections, redactSISConnectionsList(body), nil
}

func createSISConnection(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, sisOrgPath(orgID, "/sis/connections"), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("creating SIS connection: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return redactSISConnectionsList(body), nil
}

func patchSISConnection(c *client.Client, orgID, connectionID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := sisOrgPath(orgID, "/sis/connections/"+url.PathEscape(connectionID))
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating SIS connection: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return redactSISConnectionsList(body), nil
}

func testSISConnection(c *client.Client, orgID, connectionID string) ([]byte, error) {
	path := sisOrgPath(orgID, "/sis/connections/"+url.PathEscape(connectionID)+"/test")
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("testing SIS connection: %w", err)
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

func runSISSync(c *client.Client, orgID, connectionID string) (sisSyncLog, []byte, error) {
	path := sisOrgPath(orgID, "/sis/connections/"+url.PathEscape(connectionID)+"/sync")
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return sisSyncLog{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return sisSyncLog{}, nil, fmt.Errorf("starting SIS sync: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return sisSyncLog{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return sisSyncLog{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		LogID   string         `json:"logId"`
		Status  string         `json:"status"`
		Summary map[string]any `json:"summary"`
		Errors  []any          `json:"errors"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return sisSyncLog{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return sisSyncLog{
		ID:           out.LogID,
		ConnectionID: connectionID,
		Status:       out.Status,
		Summary:      out.Summary,
		Errors:       out.Errors,
	}, body, nil
}

func fetchSISSyncLogs(c *client.Client, orgID string, limit int) ([]sisSyncLog, []byte, error) {
	path := sisOrgPath(orgID, "/sis/sync-logs")
	if limit > 0 {
		path += "?limit=" + url.QueryEscape(fmt.Sprintf("%d", limit))
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing sync logs: %w", err)
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
		Logs []sisSyncLog `json:"logs"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Logs, body, nil
}

func sisSyncTerminal(status string) bool {
	switch strings.ToLower(status) {
	case "completed", "complete", "failed", "error":
		return true
	default:
		return false
	}
}

func waitForSISSyncLog(c *client.Client, orgID, logID string, timeout time.Duration, onTick func(sisSyncLog)) (sisSyncLog, error) {
	deadline := time.Now().Add(timeout)
	var last sisSyncLog
	for {
		logs, _, err := fetchSISSyncLogs(c, orgID, 50)
		if err != nil {
			return last, err
		}
		found := false
		for _, l := range logs {
			if l.ID == logID {
				last = l
				found = true
				break
			}
		}
		if !found {
			return last, fmt.Errorf("sync log %q not found", logID)
		}
		if onTick != nil {
			onTick(last)
		}
		if sisSyncTerminal(last.Status) {
			if strings.EqualFold(last.Status, "failed") || strings.EqualFold(last.Status, "error") {
				return last, fmt.Errorf("sync log %s finished with status %s", logID, last.Status)
			}
			return last, nil
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("sync log %s did not complete within %s", logID, timeout)
		}
		time.Sleep(jobPollInterval)
	}
}

func resolveSISConnectionID(c *client.Client, orgID, idOrVendor string) (string, error) {
	idOrVendor = strings.TrimSpace(idOrVendor)
	if idOrVendor == "" {
		return "", fmt.Errorf("connection id or vendor is required")
	}
	conns, _, err := fetchSISConnections(c, orgID)
	if err != nil {
		return "", err
	}
	for _, conn := range conns {
		if conn.ID == idOrVendor || strings.EqualFold(conn.Vendor, idOrVendor) {
			return conn.ID, nil
		}
	}
	return idOrVendor, nil
}

func applySISConfigFile(c *client.Client, orgID, filePath, connectionID string) ([]byte, bool, error) {
	payload, err := loadJSONFile(filePath)
	if err != nil {
		return nil, false, err
	}
	if connectionID != "" {
		body, err := patchSISConnection(c, orgID, connectionID, payload)
		return body, false, err
	}
	if id, ok := payload["id"].(string); ok && strings.TrimSpace(id) != "" {
		delete(payload, "id")
		body, err := patchSISConnection(c, orgID, strings.TrimSpace(id), payload)
		return body, false, err
	}
	body, err := createSISConnection(c, orgID, payload)
	return body, true, err
}

type scimTokenRow struct {
	ID            string  `json:"id"`
	InstitutionID string  `json:"institutionId"`
	Label         string  `json:"label"`
	CreatedAt     string  `json:"createdAt"`
	RevokedAt     *string `json:"revokedAt,omitempty"`
}

type scimEventRow struct {
	ID           string  `json:"id"`
	Operation    string  `json:"operation"`
	ScimResource string  `json:"scimResource"`
	UserEmail    *string `json:"userEmail"`
	CreatedAt    string  `json:"createdAt"`
}

func fetchScimTokens(c *client.Client, institutionID string) ([]scimTokenRow, []byte, error) {
	path := "/api/v1/admin/provisioning/scim/tokens?institutionId=" + url.QueryEscape(institutionID)
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
		Tokens []scimTokenRow `json:"tokens"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Tokens, body, nil
}

func fetchScimEvents(c *client.Client, institutionID string) ([]scimEventRow, []byte, error) {
	path := "/api/v1/admin/provisioning/scim/events?institutionId=" + url.QueryEscape(institutionID)
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
		Events []scimEventRow `json:"events"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Events, body, nil
}

func fetchSCIMCollection(c *client.Client, resource, bearerToken string) ([]byte, error) {
	path := "/scim/v2/" + resource
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	req.Header.Set("Accept", "application/scim+json")
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

type onerosterSyncRun struct {
	ID               string  `json:"id"`
	InstitutionID    string  `json:"institutionId"`
	Trigger          string  `json:"trigger"`
	Status           string  `json:"status"`
	CreatedCount     int     `json:"createdCount"`
	UpdatedCount     int     `json:"updatedCount"`
	DeactivatedCount int     `json:"deactivatedCount"`
	ErrorCount       int     `json:"errorCount"`
	StartedAt        string  `json:"startedAt"`
	CompletedAt      *string `json:"completedAt,omitempty"`
	ErrorMessage     *string `json:"errorMessage,omitempty"`
}

func fetchOneRosterSyncRuns(c *client.Client, institutionID string) ([]onerosterSyncRun, []byte, error) {
	path := "/api/v1/admin/provisioning/oneroster/sync-runs?institutionId=" + url.QueryEscape(institutionID)
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
		SyncRuns []onerosterSyncRun `json:"syncRuns"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.SyncRuns, body, nil
}

func fetchOneRosterSyncRunDetail(c *client.Client, runID string) ([]byte, error) {
	path := "/api/v1/admin/provisioning/oneroster/sync-runs/" + url.PathEscape(runID)
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

func uploadOneRosterCSV(c *client.Client, institutionID string, files []string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("institutionId", institutionID)
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
		part, err := w.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if _, err := io.Copy(part, f); err != nil {
			_ = f.Close()
			return nil, err
		}
		_ = f.Close()
	}
	_ = w.Close()

	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/provisioning/oneroster/upload", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
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

func validateOneRosterCSVFiles(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("at least one .csv file is required")
	}
	requiredByFile := map[string][]string{
		"users.csv":        {"sourcedId"},
		"classes.csv":      {"sourcedId"},
		"enrollments.csv":  {"sourcedId", "classSourcedId", "userSourcedId", "role"},
	}
	foundUsers := false
	for _, path := range paths {
		base := strings.ToLower(filepath.Base(path))
		if !strings.HasSuffix(base, ".csv") {
			return fmt.Errorf("%s: expected a .csv file", path)
		}
		if base == "users.csv" {
			foundUsers = true
		}
		reqCols, ok := requiredByFile[base]
		if !ok {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		cr := csv.NewReader(f)
		header, err := cr.Read()
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", base, err)
		}
		idx := map[string]bool{}
		for _, h := range header {
			idx[strings.ToLower(strings.TrimSpace(h))] = true
		}
		for _, col := range reqCols {
			if !idx[strings.ToLower(col)] {
				return fmt.Errorf("%s: missing required column %q", base, col)
			}
		}
	}
	if !foundUsers {
		return fmt.Errorf("users.csv is required")
	}
	return nil
}

func probeOneRosterURL(c *client.Client, endpointURL, bearerToken string) ([]byte, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(endpointURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/oneroster/v1p2/users"
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
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

func fetchLRSConfig(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/lrs-config", nil)
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
	return redactLRSSecrets(body), nil
}

func redactLRSSecrets(body []byte) []byte {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return body
		}
		redactLRSRow(single)
		out, err := json.Marshal(single)
		if err != nil {
			return body
		}
		return out
	}
	for _, row := range rows {
		redactLRSRow(row)
	}
	out, err := json.Marshal(rows)
	if err != nil {
		return body
	}
	return out
}

func redactLRSRow(m map[string]any) {
	for _, key := range []string{"password", "oauthClientSecret", "secret"} {
		if _, ok := m[key]; ok {
			m[key] = secretPlaceholder
		}
	}
}

func setLRSConfig(c *client.Client, filePath, configID string) ([]byte, error) {
	payload, err := loadJSONFile(filePath)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	method := http.MethodPost
	path := "/api/v1/admin/lrs-config"
	if configID != "" {
		method = http.MethodPut
		path += "/" + url.PathEscape(configID)
	}
	req, err := c.NewRequest(method, path, bytes.NewReader(raw))
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