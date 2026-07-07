package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type ltiExternalTool struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	ClientID        string  `json:"clientId"`
	ToolIssuer      string  `json:"toolIssuer"`
	ToolJWKSURL     string  `json:"toolJwksUrl"`
	ToolOidcAuthURL string  `json:"toolOidcAuthUrl"`
	ToolTokenURL    *string `json:"toolTokenUrl"`
	Active          bool    `json:"active"`
}

type ltiParentPlatform struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	ClientID         string   `json:"clientId"`
	PlatformISS      string   `json:"platformIss"`
	PlatformJWKSURL  string   `json:"platformJwksUrl"`
	PlatformAuthURL  string   `json:"platformAuthUrl"`
	PlatformTokenURL string   `json:"platformTokenUrl"`
	ToolRedirectURIs []string `json:"toolRedirectUris"`
	DeploymentIds    []string `json:"deploymentIds"`
	Active           bool     `json:"active"`
}

type developerApp struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Slug               string   `json:"slug"`
	ClientID           string   `json:"clientId"`
	ClientSecretPrefix string   `json:"clientSecretPrefix"`
	RedirectURIs       []string `json:"redirectUris"`
	RequestedScopes    []string `json:"requestedScopes"`
	Published          bool     `json:"published"`
}

type accessKeyItem struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	TokenMask string   `json:"tokenMask"`
	Scopes    []string `json:"scopes"`
	RevokedAt *string  `json:"revokedAt,omitempty"`
}

func fetchLTIRegistrations(c *client.Client) ([]ltiParentPlatform, []ltiExternalTool, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/lti/registrations", nil)
	if err != nil {
		return nil, nil, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		ParentPlatforms []ltiParentPlatform `json:"parentPlatforms"`
		ExternalTools   []ltiExternalTool   `json:"externalTools"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, nil, body, err
	}
	return out.ParentPlatforms, out.ExternalTools, body, nil
}

func registerLTIExternalTool(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/lti/external-tools", bytes.NewReader(raw))
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

func registerLTIParentPlatform(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/lti/registrations", bytes.NewReader(raw))
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

func updateLTIExternalTool(c *client.Client, id string, active bool) error {
	raw, _ := json.Marshal(map[string]bool{"active": active})
	req, err := c.NewRequest(http.MethodPut, "/api/v1/admin/lti/external-tools/"+url.PathEscape(id), bytes.NewReader(raw))
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

func deleteLTIExternalTool(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/lti/external-tools/"+url.PathEscape(id), nil)
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

func updateLTIParentPlatform(c *client.Client, id string, active bool) error {
	raw, _ := json.Marshal(map[string]bool{"active": active})
	req, err := c.NewRequest(http.MethodPut, "/api/v1/admin/lti/registrations/"+url.PathEscape(id), bytes.NewReader(raw))
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

func deleteLTIParentPlatform(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/lti/registrations/"+url.PathEscape(id), nil)
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

func fetchLTIPlatformJWKS(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/lti/provider/jwks", nil)
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

func buildLTIPlatformConfig(c *client.Client, serverURL string) (map[string]any, error) {
	jwks, err := fetchLTIPlatformJWKS(c)
	if err != nil {
		return nil, err
	}
	parents, _, _, err := fetchLTIRegistrations(c)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	deploymentIDs := []string{}
	for _, p := range parents {
		deploymentIDs = append(deploymentIDs, p.DeploymentIds...)
	}
	return map[string]any{
		"issuer":         base,
		"jwksUrl":        base + "/.well-known/jwks.json",
		"loginUrl":       base + "/api/v1/lti/provider/login",
		"tokenUrl":       base + "/api/v1/lti/provider/token",
		"deploymentIds":  deploymentIDs,
		"jwks":           json.RawMessage(jwks),
	}, nil
}

func fetchCourseLTITools(c *client.Client, courseCode string) ([]byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/lti-external-tools"
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

func fetchDeveloperApps(c *client.Client) ([]developerApp, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/developer/apps", nil)
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
		Apps []developerApp `json:"apps"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Apps, body, nil
}

func createDeveloperApp(c *client.Client, payload map[string]any) (map[string]any, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/developer/apps", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out, body, nil
}

func redactDeveloperAppSecret(m map[string]any) {
	if _, ok := m["clientSecret"]; ok {
		m["clientSecret"] = secretPlaceholder
	}
	if _, ok := m["token"]; ok {
		m["token"] = secretPlaceholder
	}
}

func fetchAccessKeys(c *client.Client) ([]accessKeyItem, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/access-keys", nil)
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
		Tokens []accessKeyItem `json:"tokens"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Tokens, body, nil
}

func createAccessKey(c *client.Client, payload map[string]any) (map[string]any, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/access-keys", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out, body, nil
}

func rotateAccessKey(c *client.Client, id string, overlapHours int) (map[string]any, []byte, error) {
	payload := map[string]any{}
	if overlapHours > 0 {
		payload["overlapHours"] = overlapHours
	}
	raw, _ := json.Marshal(payload)
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/access-keys/"+url.PathEscape(id)+"/rotate", bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out, body, nil
}

func revokeAccessKey(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/me/access-keys/"+url.PathEscape(id), nil)
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}