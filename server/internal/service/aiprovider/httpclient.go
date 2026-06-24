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

const defaultHTTPTimeout = 60 * time.Second

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

func (c *httpClient) postJSON(ctx context.Context, provider ProviderName, path string, body any) ([]byte, int, error) {
	if c == nil {
		return nil, 0, fmt.Errorf("aiprovider: nil http client")
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return nil, 0, err
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
	res, err := client.Do(req)
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