package client

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultUserAgent is sent on every request when non-empty (e.g. lextures-cli/0.1.0).
var DefaultUserAgent string

// Client is a thin wrapper around http.Client that injects the base URL and
// API key header into every request.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New returns a Client configured with the given base URL and API key.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewRequest builds an *http.Request with the base URL prepended and the
// Authorization header set (when an API key is present).
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if DefaultUserAgent != "" {
		req.Header.Set("User-Agent", DefaultUserAgent)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// Do executes the given request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// APIKey returns the configured API key (may be empty).
func (c *Client) APIKey() string {
	return c.apiKey
}
