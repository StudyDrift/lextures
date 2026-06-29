package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/lextures/lextures/server/internal/repos/devicepushtokens"
)

type fcmServiceAccount struct {
	ProjectID string `json:"project_id"`
}

type fcmOAuth struct {
	credentials []byte
	tokenSource oauth2.TokenSource
	projectID   string
}

func (d *NativeDispatcher) fcmOAuth() (*fcmOAuth, error) {
	raw := strings.TrimSpace(d.Config.FCMServiceAccountJSON)
	if raw == "" {
		return nil, fmt.Errorf("fcm not configured")
	}
	o := &fcmOAuth{credentials: []byte(raw)}
	var meta fcmServiceAccount
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	o.projectID = meta.ProjectID
	cfg, err := google.JWTConfigFromJSON([]byte(raw), "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, err
	}
	o.tokenSource = cfg.TokenSource(context.Background())
	return o, nil
}

func (d *NativeDispatcher) sendFCM(ctx context.Context, row devicepushtokens.Row, title, body, actionURL string) (DeliveryResult, error) {
	oauth, err := d.fcmOAuth()
	if err != nil {
		return DeliveryResult{Retryable: true}, err
	}
	tok, err := oauth.tokenSource.Token()
	if err != nil {
		return DeliveryResult{Retryable: true}, err
	}

	msg := map[string]any{
		"message": map[string]any{
			"token": row.Token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": map[string]string{
				"title":      title,
				"body":       body,
				"action_url": actionURL,
			},
			"android": map[string]any{
				"priority": "HIGH",
			},
		},
	}
	bodyBytes, err := json.Marshal(msg)
	if err != nil {
		return DeliveryResult{}, err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", oauth.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return DeliveryResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.Client.Do(req)
	if err != nil {
		return DeliveryResult{Retryable: true}, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return DeliveryResult{}, nil
	}

	var errResp struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(respBody, &errResp)
	status := errResp.Error.Status
	msgText := errResp.Error.Message

	if strings.Contains(msgText, "registration-token-not-registered") ||
		strings.Contains(status, "NOT_FOUND") {
		return DeliveryResult{InvalidToken: true}, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return DeliveryResult{Retryable: true}, fmt.Errorf("fcm rate limited")
	}
	slog.Warn("fcm.send", "status", resp.StatusCode, "detail", msgText)
	return DeliveryResult{Retryable: resp.StatusCode >= 500}, fmt.Errorf("fcm status %d", resp.StatusCode)
}
