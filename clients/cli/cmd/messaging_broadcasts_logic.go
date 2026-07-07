package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const broadcastWarning = `WARNING: Org-wide broadcasts reach a large audience.
Re-run with --yes to confirm you intend to send this broadcast.`

type mailboxMessage struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	FromEmail string `json:"fromEmail"`
	Folder    string `json:"folder"`
	CreatedAt string `json:"createdAt"`
}

type broadcastItem struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
}

type notificationRow struct {
	ID        string `json:"id"`
	EventType string `json:"eventType"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"createdAt"`
}

func buildAudienceJSON(audience string) json.RawMessage {
	audience = strings.TrimSpace(strings.ToLower(audience))
	switch audience {
	case "students", "student":
		return json.RawMessage(`{"roles":["student"]}`)
	case "staff", "faculty":
		return json.RawMessage(`{"roles":["instructor","admin"]}`)
	case "all", "":
		return json.RawMessage(`{"all":true}`)
	default:
		return json.RawMessage(fmt.Sprintf(`{"segment":%q}`, audience))
	}
}

func withIdempotencyKey(req *http.Request, key string) {
	key = strings.TrimSpace(key)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
}

func listMailboxMessages(c *client.Client, folder, query string) ([]mailboxMessage, []byte, error) {
	q := url.Values{}
	q.Set("folder", folder)
	if query != "" {
		q.Set("q", query)
	}
	path := "/api/v1/communication/messages?" + q.Encode()
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
		Messages []mailboxMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Messages, body, nil
}

func getMailboxMessage(c *client.Client, id string) ([]byte, error) {
	path := "/api/v1/communication/messages/" + url.PathEscape(id)
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

func sendMailboxMessage(c *client.Client, toEmail, subject, body string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{
		"toEmail": toEmail,
		"subject": subject,
		"body":    body,
	})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/communication/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func listOrgBroadcasts(c *client.Client, orgID string, limit int) ([]broadcastItem, []byte, error) {
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/broadcasts"
	if limit > 0 {
		path += "?limit=" + url.QueryEscape(fmt.Sprintf("%d", limit))
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
		Broadcasts []broadcastItem `json:"broadcasts"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	return out.Broadcasts, body, nil
}

func sendOrgBroadcast(c *client.Client, orgID string, payload map[string]any, idempotencyKey string) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/orgs/" + url.PathEscape(orgID) + "/broadcasts"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	withIdempotencyKey(req, idempotencyKey)
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

func fetchBroadcastDeliveryReport(c *client.Client, orgID, broadcastID string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/broadcasts/%s/delivery-report",
		url.PathEscape(orgID), url.PathEscape(broadcastID))
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

func listNotifications(c *client.Client) ([]notificationRow, int, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/notifications", nil)
	if err != nil {
		return nil, 0, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, 0, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, body, apiErrorBody(resp.StatusCode, body)
	}
	var out struct {
		Notifications []notificationRow `json:"notifications"`
		UnreadCount   int               `json:"unreadCount"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, 0, body, err
	}
	return out.Notifications, out.UnreadCount, body, nil
}

func markNotificationRead(c *client.Client, id string) error {
	path := "/api/v1/me/notifications/" + url.PathEscape(id) + "/read"
	req, err := c.NewRequest(http.MethodPost, path, nil)
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

func markAllNotificationsRead(c *client.Client) error {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/me/notifications/read-all", nil)
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

func getNotificationPreferences(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/notification-preferences", nil)
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

func setNotificationPreferences(c *client.Client, prefs []map[string]any) ([]byte, error) {
	raw, err := json.Marshal(map[string]any{"preferences": prefs})
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, "/api/v1/me/notification-preferences", bytes.NewReader(raw))
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

func parseRFC3339Schedule(s string) (*string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("schedule must be RFC3339: %w", err)
	}
	out := t.UTC().Format(time.RFC3339)
	return &out, nil
}
