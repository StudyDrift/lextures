package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

const secretPlaceholder = "••••••••••••"

var validSettingsScopes = map[string]string{
	"platform":       "/api/v1/settings/platform",
	"locale":         "/api/v1/settings/locale",
	"timezone":       "/api/v1/settings/timezone",
	"system-prompts": "/api/v1/settings/system-prompts",
}

type settingsExportFile struct {
	Version        int                    `json:"version"`
	Platform       map[string]any         `json:"platform,omitempty"`
	Locale         map[string]any         `json:"locale,omitempty"`
	Timezone       map[string]any         `json:"timezone,omitempty"`
	SystemPrompts  map[string]any         `json:"systemPrompts,omitempty"`
	PasswordPolicy map[string]any         `json:"passwordPolicy,omitempty"`
	AIProvider     map[string]any         `json:"aiProvider,omitempty"`
	DataResidency  map[string]any         `json:"dataResidency,omitempty"`
}

type settingsApplyDiff struct {
	Platform       bool     `json:"platform,omitempty"`
	Locale         bool     `json:"locale,omitempty"`
	Timezone       bool     `json:"timezone,omitempty"`
	SystemPrompts  []string `json:"systemPrompts,omitempty"`
	PasswordPolicy bool     `json:"passwordPolicy,omitempty"`
	AIProvider     bool     `json:"aiProvider,omitempty"`
}

func settingsScopePath(scope string) (string, error) {
	path, ok := validSettingsScopes[strings.ToLower(strings.TrimSpace(scope))]
	if !ok {
		return "", fmt.Errorf("unknown settings scope %q (use platform, locale, timezone, system-prompts)", scope)
	}
	return path, nil
}

func getSettingsScope(c *client.Client, scope string) ([]byte, error) {
	path, err := settingsScopePath(scope)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("getting settings: %w", err)
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

func putSettingsScope(c *client.Client, scope string, payload map[string]any) ([]byte, error) {
	path, err := settingsScopePath(scope)
	if err != nil {
		return nil, err
	}
	if scope == "system-prompts" {
		return nil, fmt.Errorf("use settings set system-prompts <key> --file for individual prompts")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("setting settings: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return body, apiErrorBody(resp.StatusCode, body)
	}
	return body, nil
}

func putSystemPrompt(c *client.Client, key string, content string) ([]byte, error) {
	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	path := "/api/v1/settings/system-prompts/" + url.PathEscape(key)
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("setting system prompt: %w", err)
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

func getPasswordPolicy(c *client.Client, institutionID string) ([]byte, error) {
	path := "/api/v1/admin/password-policy"
	if institutionID != "" {
		path += "?institutionId=" + url.QueryEscape(institutionID)
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

func putPasswordPolicy(c *client.Client, institutionID string, payload map[string]any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/admin/password-policy"
	if institutionID != "" {
		path += "?institutionId=" + url.QueryEscape(institutionID)
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

func getAIProviderSettings(c *client.Client) (map[string]any, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/ai-settings", nil)
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
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, body, err
	}
	redactAIProviderMap(out)
	return out, body, nil
}

func putAIProviderSettings(c *client.Client, payload map[string]any) ([]byte, error) {
	if key, ok := payload["byokApiKey"].(string); ok && key == secretPlaceholder {
		delete(payload, "byokApiKey")
	}
	// Deprecated legacy field — strip empty; warn callers via docs/CLI Long text.
	if _, ok := payload["openRouterApiKey"]; ok {
		delete(payload, "openRouterApiKey")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPut, "/api/v1/admin/ai-settings", bytes.NewReader(raw))
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

func testAIProviderSettings(c *client.Client) ([]byte, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/v1/admin/ai-settings/test", nil)
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

func getDataResidency(c *client.Client, orgID string) ([]byte, error) {
	path := "/api/v1/internal/compliance/data-residency/org/" + url.PathEscape(orgID)
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

func redactAIProviderMap(m map[string]any) {
	if m == nil {
		return
	}
	delete(m, "byokApiKey")
	hadDeprecatedOpenRouterKey := false
	if _, ok := m["openRouterApiKey"]; ok {
		hadDeprecatedOpenRouterKey = true
		delete(m, "openRouterApiKey")
	}
	if byok, ok := m["byokConfigured"].(bool); ok && byok {
		m["byokConfigured"] = true
	}
	redactSettingsSecrets(m)
	if hadDeprecatedOpenRouterKey {
		m["openRouterApiKey"] = "[deprecated — use provider credentials / byokApiKey]"
	}
}

func redactSettingsSecrets(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			kl := strings.ToLower(k)
			if strings.Contains(kl, "password") || strings.Contains(kl, "secret") ||
				strings.Contains(kl, "apikey") || strings.Contains(kl, "privatekey") ||
				(strings.Contains(kl, "token") && kl != "prompttokens" && kl != "completiontokens") {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					t[k] = secretPlaceholder
					continue
				}
			}
			redactSettingsSecrets(val)
		}
	case []any:
		for _, item := range t {
			redactSettingsSecrets(item)
		}
	}
}

func loadSettingsPayload(scope, filePath, key string) (map[string]any, string, error) {
	if scope == "system-prompts" {
		if key == "" {
			return nil, "", fmt.Errorf("system-prompts set requires <key> argument")
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("reading file: %w", err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil, "", fmt.Errorf("prompt file is empty")
		}
		return map[string]any{"content": content}, key, nil
	}
	payload, err := loadJSONFile(filePath)
	if err != nil {
		return nil, "", err
	}
	return payload, "", nil
}

func exportTenantSettings(c *client.Client, orgID string) (settingsExportFile, error) {
	out := settingsExportFile{Version: 1}
	if body, err := getSettingsScope(c, "platform"); err == nil {
		_ = json.Unmarshal(body, &out.Platform)
		redactSettingsSecrets(out.Platform)
	}
	if body, err := getSettingsScope(c, "locale"); err == nil {
		_ = json.Unmarshal(body, &out.Locale)
	}
	if body, err := getSettingsScope(c, "timezone"); err == nil {
		_ = json.Unmarshal(body, &out.Timezone)
	}
	if body, err := getSettingsScope(c, "system-prompts"); err == nil {
		_ = json.Unmarshal(body, &out.SystemPrompts)
	}
	if body, err := getPasswordPolicy(c, ""); err == nil {
		_ = json.Unmarshal(body, &out.PasswordPolicy)
	}
	if m, _, err := getAIProviderSettings(c); err == nil {
		out.AIProvider = m
	}
	if orgID != "" {
		if body, err := getDataResidency(c, orgID); err == nil {
			_ = json.Unmarshal(body, &out.DataResidency)
		}
	}
	return out, nil
}

func computeSettingsApplyDiff(current, desired settingsExportFile) settingsApplyDiff {
	var diff settingsApplyDiff
	if desired.Platform != nil && !jsonEqual(current.Platform, desired.Platform) {
		diff.Platform = true
	}
	if desired.Locale != nil && !jsonEqual(current.Locale, desired.Locale) {
		diff.Locale = true
	}
	if desired.Timezone != nil && !jsonEqual(current.Timezone, desired.Timezone) {
		diff.Timezone = true
	}
	if desired.PasswordPolicy != nil && !jsonEqual(current.PasswordPolicy, desired.PasswordPolicy) {
		diff.PasswordPolicy = true
	}
	if desired.AIProvider != nil && !jsonEqual(redactCopy(current.AIProvider), redactCopy(desired.AIProvider)) {
		diff.AIProvider = true
	}
	if desired.SystemPrompts != nil {
		desiredPrompts := extractPromptsByKey(desired.SystemPrompts)
		currentPrompts := extractPromptsByKey(current.SystemPrompts)
		for key, val := range desiredPrompts {
			if !jsonEqual(currentPrompts[key], val) {
				diff.SystemPrompts = append(diff.SystemPrompts, key)
			}
		}
	}
	return diff
}

func redactCopy(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	raw, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	redactAIProviderMap(out)
	return out
}

func extractPromptsByKey(root map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	if root == nil {
		return out
	}
	if prompts, ok := root["prompts"].([]any); ok {
		for _, item := range prompts {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key, _ := m["key"].(string)
			if key != "" {
				out[key] = m
			}
		}
		return out
	}
	for key, val := range root {
		if m, ok := val.(map[string]any); ok {
			out[key] = m
		}
	}
	return out
}

func jsonEqual(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func applyTenantSettings(c *client.Client, desired settingsExportFile, yes bool) error {
	if desired.DataResidency != nil {
		return fmt.Errorf("data residency is read-only after org provisioning (region is immutable)")
	}
	if desired.Platform != nil {
		if _, err := putSettingsScope(c, "platform", desired.Platform); err != nil {
			return err
		}
	}
	if desired.Locale != nil {
		if _, err := putSettingsScope(c, "locale", desired.Locale); err != nil {
			return err
		}
	}
	if desired.Timezone != nil {
		if _, err := putSettingsScope(c, "timezone", desired.Timezone); err != nil {
			return err
		}
	}
	if desired.PasswordPolicy != nil {
		if _, err := putPasswordPolicy(c, "", desired.PasswordPolicy); err != nil {
			return err
		}
	}
	if desired.AIProvider != nil {
		if _, err := putAIProviderSettings(c, desired.AIProvider); err != nil {
			return err
		}
	}
	if desired.SystemPrompts != nil {
		promptsRaw, ok := desired.SystemPrompts["prompts"].([]any)
		if !ok {
			for key, val := range desired.SystemPrompts {
				if m, ok := val.(map[string]any); ok {
					if content, ok := m["content"].(string); ok {
						if _, err := putSystemPrompt(c, key, content); err != nil {
							return err
						}
					}
				}
			}
		} else {
			for _, item := range promptsRaw {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key, _ := m["key"].(string)
				content, _ := m["content"].(string)
				if key != "" && content != "" {
					if _, err := putSystemPrompt(c, key, content); err != nil {
						return err
					}
				}
			}
		}
	}
	_ = yes
	return nil
}

type storageQuotaRow struct {
	Scope       string `json:"scope"`
	ScopeID     string `json:"scope_id"`
	LimitBytes  *int64 `json:"limit_bytes"`
	UsedBytes   int64  `json:"used_bytes"`
	PercentUsed float64 `json:"percent_used"`
}

type courseStorageUsage struct {
	UsedBytes   int64   `json:"used_bytes"`
	LimitBytes  *int64  `json:"limit_bytes"`
	PercentUsed float64 `json:"percent_used"`
}

func listStorageQuotas(c *client.Client) ([]storageQuotaRow, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/admin/storage-quotas", nil)
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
	var rows []storageQuotaRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, body, err
	}
	return rows, body, nil
}

func setStorageQuota(c *client.Client, scope, scopeID string, limitBytes *int64) error {
	payload, err := json.Marshal(map[string]any{"limit_bytes": limitBytes})
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/admin/storage-quotas/%s/%s", url.PathEscape(scope), url.PathEscape(scopeID))
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
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

func fetchCourseStorageUsage(c *client.Client, courseCode string) (courseStorageUsage, []byte, error) {
	path := "/api/v1/courses/" + url.PathEscape(courseCode) + "/storage-usage"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return courseStorageUsage{}, nil, err
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return courseStorageUsage{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readResponseBody(resp)
	if err != nil {
		return courseStorageUsage{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return courseStorageUsage{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out courseStorageUsage
	if err := json.Unmarshal(body, &out); err != nil {
		return courseStorageUsage{}, body, err
	}
	return out, body, nil
}