package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func (d Deps) transcriptsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFTranscripts {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Transcripts is not enabled.")
		return true
	}
	return false
}

type transcriptStudentJSON struct {
	UserID      string  `json:"userId"`
	Email       string  `json:"email"`
	FirstName   *string `json:"firstName,omitempty"`
	LastName    *string `json:"lastName,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	StudentID   *string `json:"studentId,omitempty"`
}

type transcriptWebhookPayload struct {
	RequestID   string                `json:"requestId"`
	RequestedAt string                `json:"requestedAt"`
	Student     transcriptStudentJSON `json:"student"`
}

type transcriptRequestJSON struct {
	ID                  string  `json:"id"`
	Status              string  `json:"status"`
	RequestedAt         string  `json:"requestedAt"`
	SubmittedAt         *string `json:"submittedAt,omitempty"`
	ErrorMessage        *string `json:"errorMessage,omitempty"`
	WebhookResponseCode *int    `json:"webhookResponseCode,omitempty"`
}

func requestToJSON(r transcriptsrepo.Request) transcriptRequestJSON {
	out := transcriptRequestJSON{
		ID:                  r.ID.String(),
		Status:              string(r.Status),
		RequestedAt:         r.RequestedAt.UTC().Format(time.RFC3339),
		ErrorMessage:        r.ErrorMessage,
		WebhookResponseCode: r.WebhookResponseCode,
	}
	if r.SubmittedAt != nil {
		s := r.SubmittedAt.UTC().Format(time.RFC3339)
		out.SubmittedAt = &s
	}
	return out
}

type transcriptsConfigJSON struct {
	WebhookURL        string `json:"webhookUrl"`
	WebhookSecret     string `json:"webhookSecret"`
	HasWebhookSecret  bool   `json:"hasWebhookSecret"`
}

func configToJSON(c *transcriptsrepo.Config) transcriptsConfigJSON {
	out := transcriptsConfigJSON{}
	if c.WebhookURL != nil {
		out.WebhookURL = *c.WebhookURL
	}
	if c.WebhookSecret != nil && strings.TrimSpace(*c.WebhookSecret) != "" {
		out.HasWebhookSecret = true
		out.WebhookSecret = placeholderSecretResponse
	}
	return out
}

func (d Deps) registerTranscriptsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/transcripts/config", d.handleGetAdminTranscriptsConfig())
	r.Put("/api/v1/admin/transcripts/config", d.handlePutAdminTranscriptsConfig())
	r.Post("/api/v1/transcripts/requests", d.handlePostTranscriptRequest())
	r.Get("/api/v1/transcripts/requests", d.handleGetTranscriptRequests())
}

// GET /api/v1/admin/transcripts/config
func (d Deps) handleGetAdminTranscriptsConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToJSON(cfg))
	}
}

type putTranscriptsConfigBody struct {
	WebhookURL    string  `json:"webhookUrl"`
	WebhookSecret *string `json:"webhookSecret"`
}

// PUT /api/v1/admin/transcripts/config
func (d Deps) handlePutAdminTranscriptsConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putTranscriptsConfigBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		url := strings.TrimSpace(body.WebhookURL)
		if url == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "webhookUrl is required.")
			return
		}
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "webhookUrl must be an http or https URL.")
			return
		}
		var secret *string
		if body.WebhookSecret != nil {
			s := strings.TrimSpace(*body.WebhookSecret)
			if s != "" && s != placeholderSecretResponse {
				secret = &s
			}
		}
		cfg, err := transcriptsrepo.UpsertConfig(r.Context(), d.Pool, url, secret)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save transcripts config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(configToJSON(cfg))
	}
}

// POST /api/v1/transcripts/requests
func (d Deps) handlePostTranscriptRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		if cfg.WebhookURL == nil || strings.TrimSpace(*cfg.WebhookURL) == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Transcript requests are not configured yet. Contact your institution.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		req, err := transcriptsrepo.InsertRequest(r.Context(), d.Pool, userID, &orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create transcript request.")
			return
		}
		go d.deliverTranscriptWebhook(context.Background(), req.ID, userID, cfg)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"request": requestToJSON(*req)})
	}
}

// GET /api/v1/transcripts/requests
func (d Deps) handleGetTranscriptRequests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		list, err := transcriptsrepo.ListByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load transcript requests.")
			return
		}
		out := make([]transcriptRequestJSON, 0, len(list))
		for _, item := range list {
			out = append(out, requestToJSON(item))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": out})
	}
}

func (d Deps) deliverTranscriptWebhook(ctx context.Context, requestID uuid.UUID, userID uuid.UUID, cfg *transcriptsrepo.Config) {
	if d.Pool == nil || cfg.WebhookURL == nil {
		return
	}
	u, err := user.FindByID(ctx, d.Pool, userID)
	if err != nil || u == nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, requestID, "Could not load student profile.", nil)
		return
	}
	payload := transcriptWebhookPayload{
		RequestID:   requestID.String(),
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
		Student: transcriptStudentJSON{
			UserID:      u.ID,
			Email:       u.Email,
			FirstName:   u.FirstName,
			LastName:    u.LastName,
			DisplayName: u.DisplayName,
			StudentID:   u.Sid,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, requestID, "Failed to encode webhook payload.", nil)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(*cfg.WebhookURL), bytes.NewReader(body))
	if err != nil {
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, requestID, "Invalid webhook URL.", nil)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Lextures-Transcripts/1.0")
	if cfg.WebhookSecret != nil && strings.TrimSpace(*cfg.WebhookSecret) != "" {
		mac := hmac.New(sha256.New, []byte(strings.TrimSpace(*cfg.WebhookSecret)))
		_, _ = mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Lextures-Signature", "sha256="+sig)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		msg := "Webhook delivery failed: " + err.Error()
		_ = transcriptsrepo.MarkFailed(ctx, d.Pool, requestID, msg, nil)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	code := resp.StatusCode
	if code >= 200 && code < 300 {
		_ = transcriptsrepo.MarkSubmitted(ctx, d.Pool, requestID, code)
		return
	}
	msg := "Institution webhook returned an error."
	_ = transcriptsrepo.MarkFailed(ctx, d.Pool, requestID, msg, &code)
}
