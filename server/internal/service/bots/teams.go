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

type teamsClient struct {
	http *http.Client
}

func (c *teamsClient) client() *http.Client {
	if c.http != nil {
		return c.http
	}
	return http.DefaultClient
}

func (c *teamsClient) postActivity(ctx context.Context, serviceURL, conversationID, token string, card map[string]any) (int, string, time.Duration, error) {
	payload := map[string]any{
		"type": "message",
		"attachments": []map[string]any{{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"content":     card,
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", 0, err
	}
	url := serviceURL + "/v3/conversations/" + conversationID + "/activities"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
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
		return resp.StatusCode, string(raw), retry, fmt.Errorf("teams rate limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(raw), latency, fmt.Errorf("teams: status %d", resp.StatusCode)
	}
	return resp.StatusCode, string(raw), latency, nil
}
