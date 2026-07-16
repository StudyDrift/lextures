// Package openrouter implements the OpenAI-compatible OpenRouter chat API used by the Rust server.
package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the public OpenRouter API base (chat, models list).
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Client calls OpenRouter's /chat/completions endpoint.
type Client struct {
	HTTP    *http.Client
	apiKey  string
	baseURL string
}

// NewClient returns a client with the public OpenRouter base URL.
func NewClient(apiKey string) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 120 * time.Second},
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: DefaultBaseURL,
	}
}

// NewClientWithBaseURL is for tests (httptest server).
func NewClientWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// HTTPConfig exposes transport settings for adapters (e.g. images API).
func (c *Client) HTTPConfig() (apiKey, baseURL string, httpClient *http.Client) {
	if c == nil {
		return "", "", nil
	}
	base := c.baseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return c.apiKey, base, c.HTTP
}

// Message is one chat message (OpenAI format).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ContentPart is a multimodal message segment (text or image URL).
type ContentPart struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *ContentPartImage `json:"image_url,omitempty"`
}

// ContentPartImage references an image for vision models.
type ContentPartImage struct {
	URL string `json:"url"`
}

// VisionMessage is a chat message with multimodal content.
type VisionMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage usagePayload `json:"usage"`
}

type usagePayload struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

func usageFromPayload(u usagePayload) UsageInfo {
	total := u.TotalTokens
	if total == 0 {
		total = u.PromptTokens + u.CompletionTokens
	}
	return UsageInfo{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      total,
		CostUSD:          u.Cost,
	}
}

// ChatOptions configures optional chat completion behavior.
type ChatOptions struct {
	// JSONMode requests structured JSON output when the model supports it.
	JSONMode bool
	// MaxTokens caps the completion length when > 0, bounding generation time.
	MaxTokens int
}

// WithTimeout returns a shallow copy of the client using a different HTTP timeout.
// Used for longer single-shot generations (e.g. workflow building) without
// changing the shared client used elsewhere.
func (c *Client) WithTimeout(d time.Duration) *Client {
	if c == nil {
		return nil
	}
	return &Client{
		HTTP:    &http.Client{Timeout: d},
		apiKey:  c.apiKey,
		baseURL: c.baseURL,
	}
}

// ChatCompletion sends a non-streaming chat request and returns the assistant text, if any.
func (c *Client) ChatCompletion(model string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	var opt ChatOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if c == nil {
		return ChatResult{}, fmt.Errorf("openrouter: nil client")
	}
	if c.apiKey == "" {
		return ChatResult{}, fmt.Errorf("openrouter: missing API key")
	}
	base := c.baseURL
	if base == "" {
		base = DefaultBaseURL
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if opt.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if opt.MaxTokens > 0 {
		body["max_tokens"] = opt.MaxTokens
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return ChatResult{}, err
	}
	u := base + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return ChatResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return ChatResult{}, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return ChatResult{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := string(b)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return ChatResult{}, fmt.Errorf("openrouter: status %d: %s", res.StatusCode, msg)
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("openrouter: parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("openrouter: no choices in response")
	}
	out := ChatResult{Usage: usageFromPayload(parsed.Usage)}
	if parsed.Choices[0].Message.Content != nil {
		out.Text = *parsed.Choices[0].Message.Content
	}
	return out, nil
}

// VisionCompletion sends a vision-capable chat request with one image URL.
func (c *Client) VisionCompletion(model, systemPrompt, userText, imageURL string) (ChatResult, error) {
	urls := []string{imageURL}
	if strings.TrimSpace(imageURL) == "" {
		urls = nil
	}
	return c.VisionCompletionMulti(model, systemPrompt, userText, urls, false)
}

// VisionCompletionMulti sends a vision request with zero or more image/data URLs.
func (c *Client) VisionCompletionMulti(model, systemPrompt, userText string, imageURLs []string, jsonMode bool) (ChatResult, error) {
	if c == nil {
		return ChatResult{}, fmt.Errorf("openrouter: nil client")
	}
	if c.apiKey == "" {
		return ChatResult{}, fmt.Errorf("openrouter: missing API key")
	}
	base := c.baseURL
	if base == "" {
		base = DefaultBaseURL
	}
	userParts := []ContentPart{{Type: "text", Text: userText}}
	for _, imageURL := range imageURLs {
		if u := strings.TrimSpace(imageURL); u != "" {
			userParts = append(userParts, ContentPart{Type: "image_url", ImageURL: &ContentPartImage{URL: u}})
		}
	}
	messages := []VisionMessage{
		{Role: "system", Content: []ContentPart{{Type: "text", Text: systemPrompt}}},
		{Role: "user", Content: userParts},
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if jsonMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return ChatResult{}, err
	}
	u := base + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return ChatResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return ChatResult{}, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return ChatResult{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := string(b)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return ChatResult{}, fmt.Errorf("openrouter: status %d: %s", res.StatusCode, msg)
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("openrouter: parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("openrouter: no choices in response")
	}
	out := ChatResult{Usage: usageFromPayload(parsed.Usage)}
	if parsed.Choices[0].Message.Content != nil {
		out.Text = *parsed.Choices[0].Message.Content
	}
	return out, nil
}
