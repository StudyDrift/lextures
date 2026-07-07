package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const ferpaWebhookWarning = `WARNING: Webhook event payloads may include FERPA-covered student data.
Review event filters before delivery.`

type serviceTokenItem struct {
	ID                 string   `json:"id"`
	Label              string   `json:"label"`
	TokenMask          string   `json:"tokenMask"`
	Scopes             []string `json:"scopes"`
	IsServiceToken     bool     `json:"isServiceToken"`
	ServiceAccountName *string  `json:"serviceAccountName,omitempty"`
	RevokedAt          *string  `json:"revokedAt,omitempty"`
}

type webhookSubscription struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	EndpointURL string   `json:"endpointUrl"`
	EventTypes  []string `json:"eventTypes"`
	Status      string   `json:"status"`
	Active      bool     `json:"active"`
}

type webhookDelivery struct {
	ID             int64   `json:"id"`
	EventType      string  `json:"eventType"`
	Status         string  `json:"status"`
	AttemptCount   int     `json:"attemptCount"`
	LastHTTPStatus *int    `json:"lastHttpStatus,omitempty"`
	Test           bool    `json:"test,omitempty"`
}

type cloudProviderRow struct {
	Provider  string `json:"provider"`
	Enabled   bool   `json:"enabled"`
	ClientID  string `json:"clientId"`
	APIKey    string `json:"apiKey"`
	AppKey    string `json:"appKey"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type integrationView struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

type botConnection struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	Status   string `json:"status"`
}

func redactTokenSecret(m map[string]any) {
	if _, ok := m["token"]; ok {
		m["token"] = secretPlaceholder
	}
}

func redactWebhookSigningKey(m map[string]any) {
	if _, ok := m["signingKey"]; ok {
		m["signingKey"] = secretPlaceholder
	}
}

func fetchAdminTokens(c *client.Client) ([]serviceTokenItem, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/tokens", nil)
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
		Tokens []serviceTokenItem `json:"tokens"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Tokens, body, nil
}

func createAdminToken(c *client.Client, payload map[string]any) (map[string]any, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/tokens", bytes.NewReader(raw))
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

func revokeAdminToken(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/tokens/"+url.PathEscape(id), nil)
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
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func fetchWebhooks(c *client.Client) ([]webhookSubscription, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/webhooks", nil)
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
		Subscriptions []webhookSubscription `json:"subscriptions"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Subscriptions, body, nil
}

func fetchWebhookEventTypes(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/webhooks/event-types", nil)
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

func createWebhook(c *client.Client, payload map[string]any) (map[string]any, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/webhooks", bytes.NewReader(raw))
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

func updateWebhook(c *client.Client, id string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/webhooks/" + url.PathEscape(id)
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

func deleteWebhook(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/webhooks/"+url.PathEscape(id), nil)
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

func testWebhook(c *client.Client, id, eventType string) ([]byte, error) {
	payload := map[string]any{}
	if strings.TrimSpace(eventType) != "" {
		payload["eventType"] = eventType
	}
	raw, _ := json.Marshal(payload)
	path := "/api/v1/webhooks/" + url.PathEscape(id) + "/test"
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
	if resp.StatusCode != http.StatusOK {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func fetchWebhookDeliveries(c *client.Client, id string) ([]webhookDelivery, []byte, error) {
	path := "/api/v1/webhooks/" + url.PathEscape(id) + "/deliveries"
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
		Deliveries []webhookDelivery `json:"deliveries"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Deliveries, body, nil
}

func parseWebhookEventTypes(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) == 1 && strings.Contains(raw[0], ",") {
		parts := strings.Split(raw[0], ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		if s := strings.TrimSpace(e); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func fetchAdminCloudProviders(c *client.Client) ([]cloudProviderRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/cloud-providers", nil)
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
	var rows []cloudProviderRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, body, err
	}
	for i := range rows {
		if rows[i].APIKey != "" {
			rows[i].APIKey = secretPlaceholder
		}
		if rows[i].AppKey != "" {
			rows[i].AppKey = secretPlaceholder
		}
	}
	return rows, body, nil
}

func fetchCloudProvider(c *client.Client, provider string) (cloudProviderRow, []byte, error) {
	rows, body, err := fetchAdminCloudProviders(c)
	if err != nil {
		return cloudProviderRow{}, body, err
	}
	for _, row := range rows {
		if row.Provider == provider {
			return row, body, nil
		}
	}
	return cloudProviderRow{}, body, fmt.Errorf("cloud provider %q not found", provider)
}

func setCloudProvider(c *client.Client, provider string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	path := "/api/v1/admin/cloud-providers/" + url.PathEscape(provider)
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
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

func testCloudProvider(row cloudProviderRow) error {
	if !row.Enabled {
		return fmt.Errorf("provider %q is disabled", row.Provider)
	}
	switch row.Provider {
	case "google_drive":
		if strings.TrimSpace(row.ClientID) == "" {
			return fmt.Errorf("provider %q is enabled but clientId is not configured", row.Provider)
		}
	case "onedrive":
		if strings.TrimSpace(row.ClientID) == "" {
			return fmt.Errorf("provider %q is enabled but clientId is not configured", row.Provider)
		}
	case "dropbox":
		if strings.TrimSpace(row.AppKey) == "" {
			return fmt.Errorf("provider %q is enabled but appKey is not configured", row.Provider)
		}
	default:
		return fmt.Errorf("unknown provider %q", row.Provider)
	}
	return nil
}

func fetchIntegrations(c *client.Client) ([]integrationView, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
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
		Integrations []integrationView `json:"integrations"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Integrations, body, nil
}

func fetchIntegrationSyncStatus(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/integrations/" + url.PathEscape(id) + "/sync-status"
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

func connectIntegration(c *client.Client, provider string) ([]byte, error) {
	path := "/integrations/oauth/" + url.PathEscape(provider) + "/connect"
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

func disconnectIntegration(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/integrations/"+url.PathEscape(id), nil)
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

func fetchBots(c *client.Client) ([]botConnection, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/bots", nil)
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
		Connections []botConnection `json:"connections"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Connections, body, nil
}

func registerBotSlack(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/integrations/slack/install", nil)
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

func registerBotDiscord(c *client.Client, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/bots/discord/connect", bytes.NewReader(raw))
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

func disconnectBot(c *client.Client, id string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/bots/"+url.PathEscape(id), nil)
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

func startBotLink(c *client.Client, platform string) ([]byte, error) {
	path := "/api/v1/me/bot-link/" + url.PathEscape(platform)
	req, err := c.NewRequest(http.MethodPost, path, nil)
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

func unlinkBot(c *client.Client, platform string) error {
	path := "/api/v1/me/bot-link/" + url.PathEscape(platform)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
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