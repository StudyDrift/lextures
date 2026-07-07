package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const complianceExportWarning = `WARNING: Compliance exports may contain legally sensitive personal data.
Re-run with --yes to confirm you are authorized to export this data.`

type dsarRequest struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	RequestType string `json:"requestType"`
	Status      string `json:"status"`
	RequestedAt string `json:"requestedAt"`
	DueAt       string `json:"dueAt"`
}

func submitGDPRDSAR(c *client.Client, requestType string) (string, []byte, error) {
	payload, err := json.Marshal(map[string]string{"requestType": requestType})
	if err != nil {
		return "", nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/compliance/gdpr/dsar", bytes.NewReader(payload))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, err
	}
	return out.ID, body, nil
}

func listGDPRDSARs(c *client.Client, queue bool) ([]dsarRequest, []byte, error) {
	path := "/api/v1/compliance/gdpr/dsar"
	if queue {
		path += "?queue=true"
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
		Requests []dsarRequest `json:"requests"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Requests, body, nil
}

func downloadGDPRDSAR(c *client.Client, requestID string) ([]byte, error) {
	path := "/api/v1/compliance/gdpr/dsar/" + url.PathEscape(requestID) + "/download"
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

func patchGDPRDSAR(c *client.Client, requestID, status string, rejectionReason string) ([]byte, error) {
	payload := map[string]string{"status": status}
	if rejectionReason != "" {
		payload["rejectionReason"] = rejectionReason
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/compliance/gdpr/dsar/" + url.PathEscape(requestID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
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

func filterDSARsBySubject(requests []dsarRequest, subject string) []dsarRequest {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return requests
	}
	out := make([]dsarRequest, 0, len(requests))
	for _, r := range requests {
		if r.UserID == subject {
			out = append(out, r)
		}
	}
	return out
}

func waitForDSARComplete(c *client.Client, requestID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		requests, _, err := listGDPRDSARs(c, true)
		if err != nil {
			return err
		}
		for _, r := range requests {
			if r.ID == requestID {
				if r.Status == "completed" {
					return nil
				}
				if r.Status == "rejected" {
					return fmt.Errorf("DSAR %s was rejected", requestID)
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out waiting for DSAR %s to complete", requestID)
}

func fetchComplianceGET(c *client.Client, path string) ([]byte, error) {
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

func fetchComplianceExport(c *client.Client, path string) ([]byte, string, error) {
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return body, "", apiErrorBody(resp.StatusCode, body)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func postComplianceJSON(c *client.Client, path string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
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

func putComplianceJSON(c *client.Client, path string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
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

func writeComplianceExport(outDir, filename string, data []byte) (string, error) {
	path := resolveExportPath(outDir, filename)
	if err := writeExportOutput(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func readComplianceInputFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, fmt.Errorf("--file is required")
	}
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return out, nil
}

func submitCCPARequest(c *client.Client, requestType string) (string, []byte, error) {
	payload, err := json.Marshal(map[string]string{"requestType": requestType})
	if err != nil {
		return "", nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/compliance/ccpa/requests", bytes.NewReader(payload))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", body, err
	}
	return out.ID, body, nil
}

func listCCPARequests(c *client.Client, queue bool) ([]byte, error) {
	path := "/api/v1/compliance/ccpa/requests"
	if queue {
		path += "?queue=true"
	}
	return fetchComplianceGET(c, path)
}

func fetchCoppaParentDashboard(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/coppa/parent-dashboard")
}

func fetchFerpaDisclosureLog(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/ferpa/disclosure-log")
}

func fetchISOControls(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/iso/soa")
}

func fetchSOC2Evidence(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/internal/compliance/soc2/evidence-summary")
}

func fetchPIIRedactionStatus(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/internal/ops/redaction-status")
}

func fetchAIInferenceLog(c *client.Client, orgID string) ([]byte, error) {
	path := "/api/v1/compliance/ai-inference-log"
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	return fetchComplianceGET(c, path)
}

func fetchAdminAIConfig(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/admin/ai-config")
}

func fetchDPACurrent(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/dpa/current")
}

func fetchDPAAcceptances(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/dpa/acceptances")
}

func fetchSecurityReports(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/security-reports")
}

func fetchSecurityReport(c *client.Client, id string) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/compliance/security-reports/"+url.PathEscape(id))
}

func fetchResearchConsentStudies(c *client.Client) ([]byte, error) {
	return fetchComplianceGET(c, "/api/v1/admin/consent-studies")
}