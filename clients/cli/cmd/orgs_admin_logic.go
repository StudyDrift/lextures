package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type orgUnitRow struct {
	ID           string  `json:"id"`
	OrgID        string  `json:"orgId"`
	Name         string  `json:"name"`
	UnitType     string  `json:"unitType"`
	Status       string  `json:"status"`
	ParentUnitID *string `json:"parentUnitId"`
}

type orgUnitsListBody struct {
	Units []orgUnitRow `json:"units"`
}

type termPublic struct {
	ID        string `json:"id"`
	OrgID     string `json:"orgId"`
	Name      string `json:"name"`
	TermType  string `json:"termType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Status    string `json:"status"`
}

type termsListBody struct {
	Terms []termPublic `json:"terms"`
}

type orgRoleGrant struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	UserEmail string `json:"userEmail"`
	Role      string `json:"role"`
}

type orgRoleGrantsBody struct {
	Grants []orgRoleGrant `json:"grants"`
}

func adminOrgPath(id string) string {
	return "/api/v1/admin/orgs/" + url.PathEscape(id)
}

func orgPath(id string) string {
	return "/api/v1/orgs/" + url.PathEscape(id)
}

func orgUnitsPath(orgID string) string {
	return adminOrgPath(orgID) + "/units"
}

func patchAdminOrg(c *client.Client, id string, body map[string]any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPatch, adminOrgPath(id), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating org: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func deleteAdminOrg(c *client.Client, id string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodDelete, adminOrgPath(id), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("archiving org: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func fetchAdminConsoleSettings(c *client.Client, orgID string) ([]byte, error) {
	path := "/api/v1/admin-console/settings"
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting org settings: %w", err)
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

func putAdminConsoleSettings(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/admin-console/settings"
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("setting org settings: %w", err)
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

func loadJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return out, nil
}

func fetchOrgBranding(c *client.Client, orgID string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, orgPath(orgID)+"/branding", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting branding: %w", err)
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

func putOrgBranding(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, orgPath(orgID)+"/branding", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("setting branding: %w", err)
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

func fetchOrgRoleGrants(c *client.Client, orgID string) (orgRoleGrantsBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, orgPath(orgID)+"/role-grants", nil)
	if err != nil {
		return orgRoleGrantsBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return orgRoleGrantsBody{}, nil, fmt.Errorf("listing role grants: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return orgRoleGrantsBody{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return orgRoleGrantsBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out orgRoleGrantsBody
	if err := json.Unmarshal(body, &out); err != nil {
		return orgRoleGrantsBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func postOrgRoleGrant(c *client.Client, orgID, userID, role string) ([]byte, error) {
	payload := map[string]string{"userId": userID, "role": role}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, orgPath(orgID)+"/role-grants", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("adding role grant: %w", err)
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

func deleteOrgRoleGrant(c *client.Client, orgID, grantID string) error {
	req, err := c.NewRequest(http.MethodDelete, orgPath(orgID)+"/role-grants/"+url.PathEscape(grantID), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing role grant: %w", err)
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

func fetchOrgUnits(c *client.Client, orgID string) (orgUnitsListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, orgUnitsPath(orgID), nil)
	if err != nil {
		return orgUnitsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return orgUnitsListBody{}, nil, fmt.Errorf("listing org units: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return orgUnitsListBody{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return orgUnitsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out orgUnitsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return orgUnitsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func createOrgUnit(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, orgUnitsPath(orgID), bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("creating org unit: %w", err)
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

func patchOrgUnit(c *client.Client, orgID, unitID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := orgUnitsPath(orgID) + "/" + url.PathEscape(unitID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating org unit: %w", err)
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

func deleteOrgUnit(c *client.Client, orgID, unitID string) error {
	path := orgUnitsPath(orgID) + "/" + url.PathEscape(unitID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting org unit: %w", err)
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

func moveOrgUnitChild(c *client.Client, orgID, parentID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := orgUnitsPath(orgID) + "/" + url.PathEscape(parentID) + "/children"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("moving org unit: %w", err)
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

func fetchOrgTerms(c *client.Client, orgID string) (termsListBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, orgPath(orgID)+"/terms", nil)
	if err != nil {
		return termsListBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return termsListBody{}, nil, fmt.Errorf("listing terms: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return termsListBody{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return termsListBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out termsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		return termsListBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func createOrgTerm(c *client.Client, orgID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, orgPath(orgID)+"/terms", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("creating term: %w", err)
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

func patchOrgTerm(c *client.Client, orgID, termID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := orgPath(orgID) + "/terms/" + url.PathEscape(termID)
	req, err := c.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("updating term: %w", err)
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

func deleteOrgTerm(c *client.Client, orgID, termID string) error {
	path := orgPath(orgID) + "/terms/" + url.PathEscape(termID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting term: %w", err)
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

func readJSONSettingsFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, fmt.Errorf("--file is required")
	}
	return loadJSONFile(path)
}

