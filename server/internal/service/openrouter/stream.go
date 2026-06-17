package openrouter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChunkHandler receives each streamed content token from the LLM.
// Returning a non-nil error stops streaming.
type ChunkHandler func(text string) error

// ChatCompletionStream sends a streaming chat request to OpenRouter and calls onChunk for each
// content delta. It returns the concatenated full response text and any usage from the final chunk.
func (c *Client) ChatCompletionStream(model string, messages []Message, onChunk ChunkHandler) (ChatResult, error) {
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
		"stream":   true,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return ChatResult{}, err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/chat/completions", bytes.NewReader(buf))
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
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		msg := string(b)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return ChatResult{}, fmt.Errorf("openrouter: status %d: %s", res.StatusCode, msg)
	}

	var sb strings.Builder
	var usage UsageInfo
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(payload) == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content *string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage usagePayload `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if u := usageFromPayload(chunk.Usage); u.HasData() {
			usage = u
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if chunk.Choices[0].Delta.Content == nil {
			continue
		}
		text := *chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		sb.WriteString(text)
		if onChunk != nil {
			if err := onChunk(text); err != nil {
				return ChatResult{Text: sb.String(), Usage: usage}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResult{Text: sb.String(), Usage: usage}, fmt.Errorf("openrouter: scan stream: %w", err)
	}
	return ChatResult{Text: sb.String(), Usage: usage}, nil
}
