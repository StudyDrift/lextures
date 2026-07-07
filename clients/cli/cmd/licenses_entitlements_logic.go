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

	"github.com/lextures/lextures/clients/cli/internal/client"
)

type licenseRow struct {
	OrgID      string  `json:"orgId"`
	OrgName    string  `json:"orgName,omitempty"`
	OrgSlug    string  `json:"orgSlug,omitempty"`
	Tier       string  `json:"tier"`
	MaxSeats   int     `json:"maxSeats"`
	UsedSeats  int     `json:"usedSeats"`
	Unlimited  bool    `json:"unlimited"`
	PercentUsed float64 `json:"percentUsed,omitempty"`
	ContractStart string `json:"contractStart,omitempty"`
	ContractEnd   string `json:"contractEnd,omitempty"`
}

type entitlementRow struct {
	ID              string  `json:"id"`
	EntitlementType string  `json:"entitlementType"`
	CourseID        *string `json:"courseId,omitempty"`
	AmountPaidCents int     `json:"amountPaidCents"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	ValidFrom       string  `json:"validFrom"`
	ValidUntil      *string `json:"validUntil,omitempty"`
}

type marketplaceApp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	LogoURL     *string `json:"logoUrl,omitempty"`
}

func readLicenseKeyMaterial(key, keyFile string) (map[string]any, error) {
	if keyFile != "" {
		return loadJSONFile(keyFile)
	}
	if key != "" {
		trim := strings.TrimSpace(key)
		if strings.HasPrefix(trim, "{") {
			var out map[string]any
			if err := json.Unmarshal([]byte(trim), &out); err != nil {
				return nil, fmt.Errorf("parsing --key JSON: %w", err)
			}
			return out, nil
		}
		return map[string]any{"tier": trim}, nil
	}
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<16))
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		trim := strings.TrimSpace(string(raw))
		if trim == "" {
			return nil, fmt.Errorf("stdin is empty")
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(trim), &out); err != nil {
			return map[string]any{"tier": trim}, nil
		}
		return out, nil
	}
	return nil, fmt.Errorf("one of --key, --key-file, or --file is required")
}

func redactLicensePayload(payload map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "key") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") {
			out[k] = "[REDACTED]"
			continue
		}
		out[k] = v
	}
	return out
}

func fetchLicensesList(c *client.Client, limit, offset int) ([]licenseRow, []byte, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	path := "/api/v1/admin/licenses"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing licenses: %w", err)
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
		Items []licenseRow `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Items, body, nil
}

func fetchLicenseStatus(c *client.Client, orgID string) (licenseRow, []byte, error) {
	if orgID != "" {
		items, raw, err := fetchLicensesList(c, 500, 0)
		if err != nil {
			return licenseRow{}, raw, err
		}
		for _, item := range items {
			if item.OrgID == orgID || item.OrgSlug == orgID {
				return item, nil, nil
			}
		}
		return licenseRow{}, nil, fmt.Errorf("no license found for org %q", orgID)
	}
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin-console/license", nil)
	if err != nil {
		return licenseRow{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return licenseRow{}, nil, fmt.Errorf("getting license status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return licenseRow{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return licenseRow{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out licenseRow
	if err := json.Unmarshal(body, &out); err != nil {
		return licenseRow{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func patchLicense(c *client.Client, orgID string, payload map[string]any) (licenseRow, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return licenseRow{}, err
	}
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/admin/licenses/"+url.PathEscape(orgID), bytes.NewReader(raw))
	if err != nil {
		return licenseRow{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return licenseRow{}, fmt.Errorf("applying license: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return licenseRow{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return licenseRow{}, apiErrorBody(resp.StatusCode, body)
	}
	var out licenseRow
	if err := json.Unmarshal(body, &out); err != nil {
		return licenseRow{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func resyncLicenseSeats(c *client.Client, orgID string) (licenseRow, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/licenses/"+url.PathEscape(orgID)+"/resync", nil)
	if err != nil {
		return licenseRow{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return licenseRow{}, fmt.Errorf("resyncing seats: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return licenseRow{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return licenseRow{}, apiErrorBody(resp.StatusCode, body)
	}
	var out licenseRow
	if err := json.Unmarshal(body, &out); err != nil {
		return licenseRow{}, fmt.Errorf("decoding response: %w", err)
	}
	return out, nil
}

func fetchEntitlements(c *client.Client) ([]entitlementRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/me/entitlements", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing entitlements: %w", err)
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
		Entitlements []entitlementRow `json:"entitlements"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Entitlements, body, nil
}

func fetchMarketplaceApps(c *client.Client) ([]marketplaceApp, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/marketplace/apps", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, nil, fmt.Errorf("listing marketplace apps: %w", err)
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
		Apps []marketplaceApp `json:"apps"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, fmt.Errorf("decoding response: %w", err)
	}
	return out.Apps, body, nil
}

func fetchMarketplaceApp(c *client.Client, slug string) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/marketplace/apps/"+url.PathEscape(slug), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting marketplace app: %w", err)
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

func fetchAdminInstalledApps(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/marketplace/installed", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("listing installed apps: %w", err)
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

func revokeInstalledApp(c *client.Client, installID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/admin/marketplace/installed/"+url.PathEscape(installID), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("revoking app: %w", err)
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

func fetchRevenueSummary(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/revenue/summary", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting revenue summary: %w", err)
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

func fetchCreatorEarnings(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/creator/earnings", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting creator earnings: %w", err)
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

func formatSeatUsage(lic licenseRow) string {
	if lic.Unlimited || lic.MaxSeats < 0 {
		return fmt.Sprintf("%d used (unlimited)", lic.UsedSeats)
	}
	return fmt.Sprintf("%d/%d seats (%.0f%% used)", lic.UsedSeats, lic.MaxSeats, lic.PercentUsed)
}