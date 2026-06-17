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

// ChatCompletion sends a non-streaming chat request and returns the assistant text, if any.
func (c *Client) ChatCompletion(model string, messages []Message) (ChatResult, error) {
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
	messages := []VisionMessage{
		{Role: "system", Content: []ContentPart{{Type: "text", Text: systemPrompt}}},
		{
			Role: "user",
			Content: []ContentPart{
				{Type: "text", Text: userText},
				{Type: "image_url", ImageURL: &ContentPartImage{URL: imageURL}},
			},
		},
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
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
