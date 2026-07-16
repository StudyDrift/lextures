package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = defaultHardTimeout

type httpClient struct {
	http    *http.Client
	apiKey  string
	baseURL string
	headers map[string]string
}

func newHTTPClient(apiKey, baseURL string, headers map[string]string) *httpClient {
	return &httpClient{
		http:    &http.Client{Timeout: defaultHTTPTimeout},
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		headers: headers,
	}
}

func (c *httpClient) withTimeout(d time.Duration) *httpClient {
	if c == nil {
		return nil
	}
	if d <= 0 {
		return c
	}
	return &httpClient{
		http:    &http.Client{Timeout: d},
		apiKey:  c.apiKey,
		baseURL: c.baseURL,
		headers: c.headers,
	}
}

func (c *httpClient) postJSON(ctx context.Context, provider ProviderName, path string, body any) ([]byte, int, error) {
	res, err := c.postJSONRaw(ctx, provider, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := string(b)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return nil, res.StatusCode, newProviderError(provider, res.StatusCode, msg)
	}
	return b, res.StatusCode, nil
}

// postJSONRaw performs the POST and returns the raw response (caller must Close Body).
func (c *httpClient) postJSONRaw(ctx context.Context, provider ProviderName, path string, body any) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("aiprovider: nil http client")
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range c.headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	client := c.http
	if client == nil {
		client = http.DefaultClient
	}
	_ = provider
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
