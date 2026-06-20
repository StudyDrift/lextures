package bots

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const slackAPIBase = "https://slack.com/api/"

type slackClient struct {
	http *http.Client
}

func (c *slackClient) client() *http.Client {
	if c.http != nil {
		return c.http
	}
	return http.DefaultClient
}

func (c *slackClient) postMessage(ctx context.Context, token, channel string, payload map[string]any) (int, string, time.Duration, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackAPIBase+"chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return 0, "", 0, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
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
		return resp.StatusCode, string(raw), retry, fmt.Errorf("slack rate limit")
	}
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if !out.OK {
		return resp.StatusCode, string(raw), 0, fmt.Errorf("slack: %s", out.Error)
	}
	return resp.StatusCode, string(raw), latency, nil
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Minute
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return time.Minute
}

func slackPostEphemeral(ctx context.Context, httpClient *http.Client, token, channel, userID, text string) error {
	body, _ := json.Marshal(map[string]any{
		"channel": channel,
		"user":    userID,
		"text":    text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackAPIBase+"chat.postEphemeral", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	var out struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(raw, &out)
	if !out.OK {
		return fmt.Errorf("slack ephemeral: %s", out.Error)
	}
	return nil
}

func slackOpenDM(ctx context.Context, httpClient *http.Client, token, platformUserID string) (string, error) {
	body, _ := json.Marshal(map[string]any{"users": platformUserID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackAPIBase+"conversations.open", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		OK      bool `json:"ok"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if !out.OK {
		return "", fmt.Errorf("slack conversations.open: %s", out.Error)
	}
	return out.Channel.ID, nil
}
