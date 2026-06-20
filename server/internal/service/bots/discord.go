package bots

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const discordAPIBase = "https://discord.com/api/v10/"

type discordClient struct {
	http *http.Client
}

func (c *discordClient) client() *http.Client {
	if c.http != nil {
		return c.http
	}
	return http.DefaultClient
}

func (c *discordClient) postMessage(ctx context.Context, token, channelID string, payload map[string]any) (int, string, time.Duration, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", 0, err
	}
	url := discordAPIBase + "channels/" + channelID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+token)
	start := time.Now()
	resp, err := c.client().Do(req)
	if err != nil {
		return 0, "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	latency := time.Since(start)
	if resp.StatusCode == http.StatusTooManyRequests {
		retry := parseRetryAfter(resp.Header.Get("Retry-After"))
		return resp.StatusCode, string(raw), retry, fmt.Errorf("discord rate limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(raw), latency, fmt.Errorf("discord: status %d", resp.StatusCode)
	}
	return resp.StatusCode, string(raw), latency, nil
}
