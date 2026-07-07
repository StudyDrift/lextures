package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type emailTemplateSlot struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	HasCustom   bool              `json:"hasCustom"`
	UpdatedAt   *string           `json:"updatedAt,omitempty"`
	MergeFields map[string]string `json:"mergeFields,omitempty"`
}

type emailTemplateDetail struct {
	emailTemplateSlot
	Active *struct {
		HTMLBody string  `json:"htmlBody"`
		TextBody *string `json:"textBody"`
	} `json:"active,omitempty"`
	DefaultHTML string `json:"defaultHtml"`
	DefaultText string `json:"defaultText"`
}

type bannerRow struct {
	ID        string  `json:"id"`
	Scope     string  `json:"scope"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	StartsAt  *string `json:"startsAt,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	IsActive  bool    `json:"isActive"`
	UpdatedAt string  `json:"updatedAt"`
}

func resolveEmailTemplateSlot(key, locale string) string {
	key = strings.TrimSpace(key)
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return key
	}
	return key + "." + locale
}

func fetchEmailTemplates(c *client.Client) ([]emailTemplateSlot, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin-console/email-templates", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing templates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var slots []emailTemplateSlot
	if err := json.Unmarshal(body, &slots); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return slots, body, nil
}

func fetchEmailTemplate(c *client.Client, slotID string) (emailTemplateDetail, []byte, error) {
	path := "/api/v1/admin-console/email-templates/" + url.PathEscape(slotID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return emailTemplateDetail{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return emailTemplateDetail{}, nil, fmt.Errorf("getting template: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return emailTemplateDetail{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return emailTemplateDetail{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out emailTemplateDetail
	if err := json.Unmarshal(body, &out); err != nil {
		return emailTemplateDetail{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func putEmailTemplate(c *client.Client, slotID string, htmlBody string, textBody *string) ([]byte, error) {
	payload := map[string]any{"htmlBody": htmlBody}
	if textBody != nil {
		payload["textBody"] = *textBody
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/admin-console/email-templates/" + url.PathEscape(slotID)
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("setting template: %w", err)
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

func previewEmailTemplate(c *client.Client, slotID string, htmlBody string, textBody *string) ([]byte, error) {
	payload := map[string]any{}
	if strings.TrimSpace(htmlBody) != "" {
		payload["htmlBody"] = htmlBody
		if textBody != nil {
			payload["textBody"] = *textBody
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/admin-console/email-templates/" + url.PathEscape(slotID) + "/preview"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("previewing template: %w", err)
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

func testSendEmailTemplate(c *client.Client, slotID string) error {
	path := "/api/v1/admin-console/email-templates/" + url.PathEscape(slotID) + "/test"
	req, err := c.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("sending test email: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return nil
}

func readTemplateFile(path string) (html string, text *string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("reading file: %w", err)
	}
	content := string(data)
	if strings.HasSuffix(strings.ToLower(path), ".txt") {
		t := content
		return "", &t, nil
	}
	return content, nil, nil
}

func fetchBanners(c *client.Client, scope string) ([]bannerRow, []byte, error) {
	q := ""
	if scope != "" {
		q = "?scope=" + url.QueryEscape(scope)
	}
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/banners"+q, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing banners: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, body, apiErrorBody(resp.StatusCode, body)
	}
	var rows []bannerRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return rows, body, nil
}

type bannerWriteInput struct {
	Scope     string
	Message   string
	Severity  string
	From      string
	Until     string
	Audience  string
	IsActive  *bool
	CTAText   *string
	CTAURL    *string
}

func createBanner(c *client.Client, in bannerWriteInput) (bannerRow, []byte, error) {
	payload, err := bannerPayload(in)
	if err != nil {
		return bannerRow{}, nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return bannerRow{}, nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/banners", bytes.NewReader(raw))
	if err != nil {
		return bannerRow{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return bannerRow{}, nil, fmt.Errorf("creating banner: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return bannerRow{}, nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return bannerRow{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var row bannerRow
	if err := json.Unmarshal(body, &row); err != nil {
		return bannerRow{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return row, body, nil
}

func updateBanner(c *client.Client, id string, in bannerWriteInput) (bannerRow, []byte, error) {
	payload, err := bannerPayload(in)
	if err != nil {
		return bannerRow{}, nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return bannerRow{}, nil, err
	}
	path := "/api/v1/admin/banners/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return bannerRow{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return bannerRow{}, nil, fmt.Errorf("updating banner: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return bannerRow{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return bannerRow{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var row bannerRow
	if err := json.Unmarshal(body, &row); err != nil {
		return bannerRow{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return row, body, nil
}

func findBanner(c *client.Client, id string) (bannerRow, error) {
	rows, _, err := fetchBanners(c, "")
	if err != nil {
		return bannerRow{}, err
	}
	for _, row := range rows {
		if row.ID == id {
			return row, nil
		}
	}
	return bannerRow{}, fmt.Errorf("banner %q not found", id)
}

func deleteBanner(c *client.Client, id string) error {
	path := "/api/v1/admin/banners/" + url.PathEscape(id)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting banner: %w", err)
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

func bannerPayload(in bannerWriteInput) (map[string]any, error) {
	scope := strings.TrimSpace(in.Scope)
	if scope == "" {
		scope = "org"
	}
	if strings.TrimSpace(in.Audience) == "global" {
		scope = "global"
	}
	severity := strings.TrimSpace(in.Severity)
	if severity == "" {
		severity = "info"
	}
	if strings.TrimSpace(in.Message) == "" {
		return nil, fmt.Errorf("message is required")
	}
	payload := map[string]any{
		"scope":    scope,
		"message":  in.Message,
		"severity": severity,
	}
	if in.From != "" {
		payload["startsAt"] = in.From
	}
	if in.Until != "" {
		payload["expiresAt"] = in.Until
	}
	if in.IsActive != nil {
		payload["isActive"] = *in.IsActive
	}
	if in.CTAText != nil {
		payload["ctaText"] = *in.CTAText
	}
	if in.CTAURL != nil {
		payload["ctaUrl"] = *in.CTAURL
	}
	return payload, nil
}

func validateBannerWindow(from, until string) error {
	if from == "" && until == "" {
		return nil
	}
	var start, end time.Time
	var err error
	if from != "" {
		start, err = time.Parse(time.RFC3339, from)
		if err != nil {
			return fmt.Errorf("invalid --from: use RFC3339")
		}
	}
	if until != "" {
		end, err = time.Parse(time.RFC3339, until)
		if err != nil {
			return fmt.Errorf("invalid --until: use RFC3339")
		}
	}
	if from != "" && until != "" && end.Before(start) {
		return fmt.Errorf("--until must be after --from")
	}
	return nil
}