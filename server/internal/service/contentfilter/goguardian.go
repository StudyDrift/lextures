package contentfilter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const goGuardianActivityURL = "https://api.goguardian.com/v1/activity"

// ActivityEvent is the payload sent to GoGuardian's activity classification API.
type ActivityEvent struct {
	URL           string `json:"url"`
	Category      string `json:"category"`
	Title         string `json:"title"`
	StudentIDHash string `json:"student_id_hash"`
}

// StudentIDHash returns SHA-256 hex of student_id + salt (plan 13.14).
func StudentIDHash(studentID uuid.UUID, salt string) string {
	key := strings.TrimSpace(salt)
	if key == "" {
		key = "lextures-content-filter-default"
	}
	sum := sha256.Sum256([]byte(studentID.String() + key))
	return hex.EncodeToString(sum[:])
}

// EmitActivity sends a fire-and-forget activity event to GoGuardian.
func EmitActivity(ctx context.Context, apiKey string, ev ActivityEvent) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return
	}
	body, err := json.Marshal(ev)
	if err != nil {
		slog.Warn("contentfilter: marshal activity event", "err", err)
		return
	}
	go func() {
		reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, goGuardianActivityURL, bytes.NewReader(body))
		if err != nil {
			slog.Warn("contentfilter: build goguardian request", "err", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Warn("contentfilter: goguardian api unreachable", "err", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			slog.Warn("contentfilter: goguardian api error", "status", resp.StatusCode, "body", string(b))
		}
	}()
}
