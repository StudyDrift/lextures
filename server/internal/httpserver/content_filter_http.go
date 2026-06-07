package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/contentfilter"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/repos/organization"
	cfsvc "github.com/lextures/lextures/server/internal/service/contentfilter"
)

const contentFilterSecretPlaceholder = "••••••••••••"

func (d Deps) contentFilterEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFContentFilterIntegration {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
			"Content filter integration is not enabled.")
		return false
	}
	return true
}

func (d Deps) handleContentFilterAllowlist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(serverdata.ContentFilterAllowlistJSON)
	}
}

type contentFilterSettingsJSON struct {
	OrgID               string `json:"orgId"`
	GoGuardianEnabled   bool   `json:"goGuardianEnabled"`
	GoGuardianAPIKey    string `json:"goGuardianApiKey"`
	HasGoGuardianAPIKey bool   `json:"hasGoGuardianApiKey"`
	SecurlyEnabled      bool   `json:"securlyEnabled"`
	UpdatedAt           string `json:"updatedAt"`
	AllowlistURL        string `json:"allowlistUrl"`
}

func toContentFilterJSON(orgID uuid.UUID, row *repo.Row) contentFilterSettingsJSON {
	out := contentFilterSettingsJSON{
		OrgID:        orgID.String(),
		AllowlistURL: "/.well-known/content-filter-allowlist.json",
	}
	if row == nil {
		return out
	}
	out.GoGuardianEnabled = row.GoGuardianEnabled
	out.SecurlyEnabled = row.SecurlyEnabled
	out.HasGoGuardianAPIKey = row.HasGoGuardianAPIKey
	if row.HasGoGuardianAPIKey {
		out.GoGuardianAPIKey = contentFilterSecretPlaceholder
	}
	out.UpdatedAt = row.UpdatedAt.UTC().Format(time.RFC3339)
	return out
}

// GET/PATCH /api/v1/orgs/{orgId}/settings/content-filter
func (d Deps) handleOrgContentFilterSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentFilterEnabled(w) {
			return
		}
		orgStr := strings.TrimSpace(chi.URLParam(r, "orgId"))
		orgID, err := uuid.Parse(orgStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
			return
		}
		ctx := r.Context()
		cfg := d.effectiveConfig()

		switch r.Method {
		case http.MethodGet:
			row, err := repo.Get(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content filter settings.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(toContentFilterJSON(orgID, row))

		case http.MethodPatch:
			var body struct {
				GoGuardianEnabled   *bool   `json:"goGuardianEnabled"`
				GoGuardianAPIKey    *string `json:"goGuardianApiKey"`
				ClearGoGuardianAPIKey *bool `json:"clearGoGuardianApiKey"`
				SecurlyEnabled      *bool   `json:"securlyEnabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			cur, _, err := repo.GetWithSecrets(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content filter settings.")
				return
			}
			ggEnabled := false
			secEnabled := false
			if cur != nil {
				ggEnabled = cur.GoGuardianEnabled
				secEnabled = cur.SecurlyEnabled
			}
			if body.GoGuardianEnabled != nil {
				ggEnabled = *body.GoGuardianEnabled
			}
			if body.SecurlyEnabled != nil {
				secEnabled = *body.SecurlyEnabled
			}
			var apiKeyPlain *string
			clearKey := body.ClearGoGuardianAPIKey != nil && *body.ClearGoGuardianAPIKey
			if body.GoGuardianAPIKey != nil {
				s := strings.TrimSpace(*body.GoGuardianAPIKey)
				if s != "" && s != contentFilterSecretPlaceholder {
					apiKeyPlain = &s
				}
			}
			if err := repo.Upsert(ctx, d.Pool, repo.UpsertInput{
				OrgID:             orgID,
				GoGuardianEnabled: ggEnabled,
				SecurlyEnabled:    secEnabled,
				APIKeyPlaintext:   apiKeyPlain,
				ClearAPIKey:       clearKey,
				SecretsKey:        cfg.PlatformSecretsKey,
			}); err != nil {
				if strings.Contains(err.Error(), "secrets key") {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
						"Platform secrets key is not configured; cannot store API key.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save content filter settings.")
				return
			}
			updated, err := repo.Get(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload content filter settings.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(toContentFilterJSON(orgID, updated))

		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPatch}, ", "))
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

type contentFilterActivityBody struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// POST /api/v1/content-filter/activity — authenticated student activity for GoGuardian.
func (d Deps) handleContentFilterActivity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentFilterEnabled(w) {
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body contentFilterActivityBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		url := strings.TrimSpace(body.URL)
		if url == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "url is required.")
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			title = "Lextures"
		}

		ctx := r.Context()
		orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		row, sec, err := repo.GetWithSecrets(ctx, d.Pool, orgID)
		if err != nil || row == nil || !row.GoGuardianEnabled {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		cfg := d.effectiveConfig()
		apiKey, err := repo.DecryptAPIKey(sec, cfg.PlatformSecretsKey)
		if err != nil || apiKey == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		salt := cfg.JWTSecret
		if len(cfg.PlatformSecretsKey) == 32 {
			salt = string(cfg.PlatformSecretsKey)
		}
		cfsvc.EmitActivity(ctx, apiKey, cfsvc.ActivityEvent{
			URL:           url,
			Category:      "educational",
			Title:         title,
			StudentIDHash: cfsvc.StudentIDHash(userID, salt),
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) registerContentFilterRoutes(r chi.Router) {
	r.Get("/.well-known/content-filter-allowlist.json", d.handleContentFilterAllowlist())
	r.Get("/api/v1/orgs/{orgId}/settings/content-filter", d.handleOrgContentFilterSettings())
	r.Patch("/api/v1/orgs/{orgId}/settings/content-filter", d.handleOrgContentFilterSettings())
	r.Post("/api/v1/content-filter/activity", d.handleContentFilterActivity())
}
