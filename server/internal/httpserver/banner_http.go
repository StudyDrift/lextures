package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	bannersrepo "github.com/lextures/lextures/server/internal/repos/banners"
	"github.com/lextures/lextures/server/internal/repos/organization"
	bannersservice "github.com/lextures/lextures/server/internal/service/banners"
	"github.com/lextures/lextures/server/internal/telemetry"
)

type bannerDTO struct {
	ID        string  `json:"id"`
	Scope     string  `json:"scope"`
	OrgID     *string `json:"orgId,omitempty"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	CTAText   *string `json:"ctaText,omitempty"`
	CTAURL    *string `json:"ctaUrl,omitempty"`
	StartsAt  *string `json:"startsAt,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	IsActive  bool    `json:"isActive"`
	UpdatedAt string  `json:"updatedAt"`
}

func (d Deps) maintenanceBannerEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().MaintenanceBannerEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Maintenance banners are not enabled.")
		return false
	}
	return true
}

func (d Deps) registerBannerRoutes(r chi.Router) {
	r.Get("/api/v1/status/banner", d.handleGetStatusBanner())
	r.Get("/api/v1/admin/banners", d.handleListBanners())
	r.Post("/api/v1/admin/banners", d.handleCreateBanner())
	r.Put("/api/v1/admin/banners/{id}", d.handleUpdateBanner())
	r.Delete("/api/v1/admin/banners/{id}", d.handleDeleteBanner())
	r.Post("/api/v1/admin/banners/statuspage-webhook", d.handleStatuspageBannerWebhook())
}

// GET /api/v1/status/banner — public, cacheable banner for login and app shell.
func (d Deps) handleGetStatusBanner() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().MaintenanceBannerEnabled {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=10, stale-while-revalidate=60")
			_ = json.NewEncoder(w).Encode(nil)
			return
		}
		if d.Pool == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(nil)
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()

		var orgID *uuid.UUID
		if userID, ok := d.optionalUserID(r); ok {
			if oid, err := organization.OrgIDForUser(ctx, d.Pool, userID); err == nil {
				orgID = &oid
			}
		} else if slug := orgSlugFromBrandingQuery(r); slug != "" {
			if row, err := organization.GetBySlug(ctx, d.Pool, slug); err == nil && row != nil {
				orgID = &row.ID
			}
		}

		banner, err := bannersrepo.GetActiveForOrg(ctx, d.Pool, orgID, now)
		if err != nil {
			slog.Warn("banner query failed", "err", err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(nil)
			return
		}
		d.recordBannerActiveMetric(banner)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=10, stale-while-revalidate=60")
		if banner == nil {
			_ = json.NewEncoder(w).Encode(nil)
			return
		}
		_ = json.NewEncoder(w).Encode(bannerToDTO(*banner))
	}
}

func (d Deps) recordBannerActiveMetric(b *bannersrepo.Banner) {
	m := telemetry.Default()
	if m == nil {
		return
	}
	if b == nil {
		m.SetBannerActive("", "")
		return
	}
	m.SetBannerActive(string(b.Scope), string(b.Severity))
}

func (d Deps) optionalUserID(r *http.Request) (uuid.UUID, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return uuid.UUID{}, false
	}
	if d.JWTSigner == nil {
		return uuid.UUID{}, false
	}
	user, err := d.JWTSigner.Verify(r.Context(), auth[7:])
	if err != nil || strings.TrimSpace(user.UserID) == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(user.UserID)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

// GET /api/v1/admin/banners
func (d Deps) handleListBanners() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.maintenanceBannerEnabled(w) {
			return
		}
		actor, targetOrg, globalAdmin, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		var list []bannersrepo.Banner
		var err error
		if globalAdmin && r.URL.Query().Get("scope") == "global" {
			list, err = bannersrepo.List(r.Context(), d.Pool, nil, true)
		} else {
			list, err = bannersrepo.List(r.Context(), d.Pool, &targetOrg, false)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list banners.")
			return
		}
		out := make([]bannerDTO, 0, len(list))
		for _, b := range list {
			out = append(out, bannerToDTO(b))
		}
		_ = actor
		writeJSON(w, http.StatusOK, out)
	}
}

type bannerWriteBody struct {
	Scope     string  `json:"scope"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	CTAText   *string `json:"ctaText"`
	CTAURL    *string `json:"ctaUrl"`
	StartsAt  *string `json:"startsAt"`
	ExpiresAt *string `json:"expiresAt"`
	IsActive  *bool   `json:"isActive"`
}

// POST /api/v1/admin/banners
func (d Deps) handleCreateBanner() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.maintenanceBannerEnabled(w) {
			return
		}
		actor, targetOrg, globalAdmin, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		body, err := readBannerBody(r)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		scope, err := bannersservice.ParseScope(body.Scope)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if scope == "global" {
			if !globalAdmin {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only global admins can create global banners.")
				return
			}
		}
		if err := bannersservice.ValidateMessage(body.Message); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		severity, err := bannersservice.ParseSeverity(body.Severity)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		startsAt, expiresAt, err := parseBannerTimes(body.StartsAt, body.ExpiresAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		var orgID *uuid.UUID
		if scope == "org" {
			orgID = &targetOrg
		}
		created, err := bannersrepo.Create(r.Context(), d.Pool, bannersrepo.CreateParams{
			Scope:     bannersrepo.Scope(scope),
			OrgID:     orgID,
			Message:   strings.TrimSpace(body.Message),
			Severity:  bannersrepo.Severity(severity),
			CTAText:   body.CTAText,
			CTAURL:    body.CTAURL,
			StartsAt:  startsAt,
			ExpiresAt: expiresAt,
			CreatedBy: actor,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create banner.")
			return
		}
		slog.Info("banner created", "banner_id", created.ID, "scope", created.Scope, "actor_id", actor)
		telemetry.RecordBusinessEvent("banner_created")
		writeJSON(w, http.StatusCreated, bannerToDTO(created))
	}
}

// PUT /api/v1/admin/banners/{id}
func (d Deps) handleUpdateBanner() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.maintenanceBannerEnabled(w) {
			return
		}
		_, targetOrg, globalAdmin, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid banner id.")
			return
		}
		existing, err := bannersrepo.GetByID(r.Context(), d.Pool, id)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Banner not found.")
			return
		}
		if !bannerManageAllowed(existing, targetOrg, globalAdmin) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this banner.")
			return
		}
		body, err := readBannerBody(r)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if err := bannersservice.ValidateMessage(body.Message); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		severity, err := bannersservice.ParseSeverity(body.Severity)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		startsAt, expiresAt, err := parseBannerTimes(body.StartsAt, body.ExpiresAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}
		updated, err := bannersrepo.Update(r.Context(), d.Pool, id, bannersrepo.UpdateParams{
			Message:   strings.TrimSpace(body.Message),
			Severity:  bannersrepo.Severity(severity),
			CTAText:   body.CTAText,
			CTAURL:    body.CTAURL,
			StartsAt:  startsAt,
			ExpiresAt: expiresAt,
			IsActive:  isActive,
		})
		if err != nil || updated == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update banner.")
			return
		}
		slog.Info("banner updated", "banner_id", updated.ID, "scope", updated.Scope)
		telemetry.RecordBusinessEvent("banner_updated")
		writeJSON(w, http.StatusOK, bannerToDTO(*updated))
	}
}

// DELETE /api/v1/admin/banners/{id}
func (d Deps) handleDeleteBanner() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.maintenanceBannerEnabled(w) {
			return
		}
		_, targetOrg, globalAdmin, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid banner id.")
			return
		}
		existing, err := bannersrepo.GetByID(r.Context(), d.Pool, id)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Banner not found.")
			return
		}
		if !bannerManageAllowed(existing, targetOrg, globalAdmin) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this banner.")
			return
		}
		deleted, err := bannersrepo.Delete(r.Context(), d.Pool, id)
		if err != nil || !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Banner not found.")
			return
		}
		slog.Info("banner deleted", "banner_id", id)
		telemetry.RecordBusinessEvent("banner_deleted")
		w.WriteHeader(http.StatusNoContent)
	}
}

// POST /api/v1/admin/banners/statuspage-webhook
func (d Deps) handleStatuspageBannerWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.maintenanceBannerEnabled(w) {
			return
		}
		secret := strings.TrimSpace(d.effectiveConfig().StatuspageWebhookSecret)
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read request body.")
			return
		}
		if !verifyStatuspageHMAC(r, secret, body) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid webhook signature.")
			return
		}
		payload, err := bannersservice.ParseStatuspageWebhook(body)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		inc := payload.Incident
		externalID := "statuspage:" + strings.TrimSpace(inc.ID)
		status := strings.ToLower(strings.TrimSpace(inc.Status))
		if status == "resolved" || status == "completed" {
			_ = bannersrepo.DeactivateByExternalID(r.Context(), d.Pool, externalID)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		severity := bannersservice.IncidentSeverity(inc.Impact, inc.Status)
		if severity == "" {
			severity = "warning"
		}
		message := bannersservice.IncidentMessage(inc.Name, inc.Status)
		if err := bannersservice.ValidateMessage(message); err != nil {
			message = "Service incident in progress"
		}
		systemActor, err := d.statuspageWebhookActor(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Webhook actor not configured.")
			return
		}
		ext := externalID
		_, err = bannersrepo.UpsertByExternalID(r.Context(), d.Pool, bannersrepo.CreateParams{
			Scope:      bannersrepo.ScopeGlobal,
			Message:    message,
			Severity:   bannersrepo.Severity(severity),
			ExternalID: &ext,
			CreatedBy:  systemActor,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to upsert banner.")
			return
		}
		slog.Info("banner upserted from statuspage", "external_id", externalID, "status", inc.Status)
		telemetry.RecordBusinessEvent("banner_statuspage_webhook")
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) statuspageWebhookActor(ctx context.Context) (uuid.UUID, error) {
	row := d.Pool.QueryRow(ctx, `
SELECT u.id FROM "user".users u
JOIN "user".user_app_roles uar ON uar.user_id = u.id
JOIN "user".app_roles ar ON ar.id = uar.role_id
WHERE ar.name = 'Global Admin'
ORDER BY u.created_at ASC
LIMIT 1`)
	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		return uuid.UUID{}, errNoGlobalAdmin
	}
	return id, nil
}

var errNoGlobalAdmin = errBannerWebhookActor("no global admin user found")

type errBannerWebhookActor string

func (e errBannerWebhookActor) Error() string { return string(e) }

func verifyStatuspageHMAC(r *http.Request, secret string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	sig := strings.TrimSpace(r.Header.Get("X-Statuspage-Signature"))
	if sig == "" {
		sig = strings.TrimSpace(r.Header.Get("X-Hub-Signature-256"))
		if strings.HasPrefix(strings.ToLower(sig), "sha256=") {
			sig = sig[7:]
		}
	}
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return subtleConstantTimeEqual(strings.ToLower(sig), strings.ToLower(expected))
}

func bannerManageAllowed(b *bannersrepo.Banner, targetOrg uuid.UUID, globalAdmin bool) bool {
	if b.Scope == bannersrepo.ScopeGlobal {
		return globalAdmin
	}
	if !globalAdmin && (b.OrgID == nil || *b.OrgID != targetOrg) {
		return false
	}
	return true
}

func readBannerBody(r *http.Request) (bannerWriteBody, error) {
	var body bannerWriteBody
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(&body); err != nil {
		return bannerWriteBody{}, err
	}
	if body.Scope == "" {
		body.Scope = "org"
	}
	return body, nil
}

func parseBannerTimes(startsAt, expiresAt *string) (*time.Time, *time.Time, error) {
	var startPtr, expirePtr *time.Time
	if startsAt != nil && strings.TrimSpace(*startsAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*startsAt))
		if err != nil {
			return nil, nil, err
		}
		utc := t.UTC()
		startPtr = &utc
	}
	if expiresAt != nil && strings.TrimSpace(*expiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*expiresAt))
		if err != nil {
			return nil, nil, err
		}
		utc := t.UTC()
		expirePtr = &utc
	}
	return startPtr, expirePtr, nil
}

func bannerToDTO(b bannersrepo.Banner) bannerDTO {
	dto := bannerDTO{
		ID:       b.ID.String(),
		Scope:    string(b.Scope),
		Message:  b.Message,
		Severity: string(b.Severity),
		CTAText:  b.CTAText,
		CTAURL:   b.CTAURL,
		IsActive: b.IsActive,
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if b.OrgID != nil {
		s := b.OrgID.String()
		dto.OrgID = &s
	}
	if b.StartsAt != nil {
		s := b.StartsAt.UTC().Format(time.RFC3339)
		dto.StartsAt = &s
	}
	if b.ExpiresAt != nil {
		s := b.ExpiresAt.UTC().Format(time.RFC3339)
		dto.ExpiresAt = &s
	}
	return dto
}
