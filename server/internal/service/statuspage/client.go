package statuspage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultAPIBase = "https://api.statuspage.io/v1"

// Config drives the Statuspage integration.
type Config struct {
	Enabled         bool
	PageURL         string
	APIKey          string
	PageID          string
	ComponentMap    ComponentMap
	CacheTTL        time.Duration
	HTTPClient      *http.Client
	APIBaseURL      string
}

// Client proxies Statuspage summary data and updates component status.
type Client struct {
	cfg Config

	mu          sync.RWMutex
	cached      Summary
	cachedAt    time.Time
	httpClient  *http.Client
	apiBaseURL  string
}

func NewClient(cfg Config) *Client {
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = time.Minute
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if base == "" {
		base = defaultAPIBase
	}
	pageURL := strings.TrimSpace(cfg.PageURL)
	if pageURL == "" {
		pageURL = "https://status.lextures.io"
	}
	return &Client{
		cfg: Config{
			Enabled:      cfg.Enabled,
			PageURL:      pageURL,
			APIKey:       strings.TrimSpace(cfg.APIKey),
			PageID:       strings.TrimSpace(cfg.PageID),
			ComponentMap: cfg.ComponentMap,
			CacheTTL:     ttl,
		},
		httpClient: hc,
		apiBaseURL: base,
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.cfg.Enabled && c.cfg.APIKey != "" && c.cfg.PageID != ""
}

func (c *Client) Summary(ctx context.Context) (Summary, error) {
	if c == nil || !c.Configured() {
		pageURL := "https://status.lextures.io"
		if c != nil && strings.TrimSpace(c.cfg.PageURL) != "" {
			pageURL = c.cfg.PageURL
		}
		return emptySummary(pageURL, false), nil
	}

	c.mu.RLock()
	if !c.cachedAt.IsZero() && time.Since(c.cachedAt) < c.cfg.CacheTTL {
		out := c.cached
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	raw, err := c.fetchUpstreamSummary(ctx)
	if err != nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		if !c.cachedAt.IsZero() {
			return c.cached, nil
		}
		return emptySummary(c.cfg.PageURL, true), err
	}
	out := normalizeSummary(c.cfg.PageURL, raw)

	c.mu.Lock()
	c.cached = out
	c.cachedAt = time.Now()
	c.mu.Unlock()
	return out, nil
}

func (c *Client) fetchUpstreamSummary(ctx context.Context) (upstreamSummary, error) {
	url := fmt.Sprintf("%s/pages/%s/summary.json", c.apiBaseURL, c.cfg.PageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return upstreamSummary{}, err
	}
	req.Header.Set("Authorization", "OAuth "+c.cfg.APIKey)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return upstreamSummary{}, err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return upstreamSummary{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return upstreamSummary{}, fmt.Errorf("statuspage summary: status %d", res.StatusCode)
	}
	var raw upstreamSummary
	if err := json.Unmarshal(body, &raw); err != nil {
		return upstreamSummary{}, err
	}
	return raw, nil
}

func (c *Client) UpdateComponentStatus(ctx context.Context, componentID, status string) error {
	if c == nil || !c.Configured() {
		return fmt.Errorf("statuspage is not configured")
	}
	componentID = strings.TrimSpace(componentID)
	status = strings.TrimSpace(status)
	if componentID == "" || status == "" {
		return fmt.Errorf("component id and status are required")
	}
	url := fmt.Sprintf("%s/pages/%s/components/%s", c.apiBaseURL, c.cfg.PageID, componentID)
	payload := map[string]any{
		"component": map[string]string{"status": status},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "OAuth "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("statuspage component update: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}