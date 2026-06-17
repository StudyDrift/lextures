package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/api"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/jackc/pgx/v5/pgxpool"
)

type accessKeyCourse struct {
	ID         string `json:"id"`
	CourseCode string `json:"courseCode"`
	Title      string `json:"title"`
}

type accessKeyItem struct {
	ID         string             `json:"id"`
	Label      string             `json:"label"`
	TokenMask  string             `json:"tokenMask"`
	Scopes     []string           `json:"scopes"`
	CourseIds  []string           `json:"courseIds"`
	Courses    []accessKeyCourse  `json:"courses,omitempty"`
	AllCourses bool               `json:"allCourses"`
	ExpiresAt  *time.Time         `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time         `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time         `json:"revokedAt,omitempty"`
	CreatedAt  time.Time          `json:"createdAt"`
	UnusedDays *int               `json:"unusedDays,omitempty"`
}

func validateAccessKeyCourseIDsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) error {
	if len(courseIDs) == 0 {
		return nil
	}
	for _, cid := range courseIDs {
		ok, err := enrollment.UserHasAccessByCourseID(ctx, pool, cid, userID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("course not accessible")
		}
	}
	return nil
}

func accessKeyItemFromRow(r apitokens.Row, courses []accessKeyCourse, now time.Time) accessKeyItem {
	courseIds := make([]string, 0, len(r.CourseIDs))
	for _, id := range r.CourseIDs {
		courseIds = append(courseIds, id.String())
	}
	item := accessKeyItem{
		ID:         r.ID.String(),
		Label:      r.Label,
		TokenMask:  apitokens.MaskedDisplay(r.TokenPrefix),
		Scopes:     r.Scopes,
		CourseIds:  courseIds,
		Courses:    courses,
		AllCourses: len(r.CourseIDs) == 0,
		ExpiresAt:  r.ExpiresAt,
		LastUsedAt: r.LastUsedAt,
		RevokedAt:  r.RevokedAt,
		CreatedAt:  r.CreatedAt,
	}
	if r.RevokedAt == nil {
		ref := r.CreatedAt
		if r.LastUsedAt != nil {
			ref = *r.LastUsedAt
		}
		days := int(now.Sub(ref.UTC()).Hours() / 24)
		if days >= 90 {
			item.UnusedDays = &days
		}
	}
	return item
}

func (d Deps) apiBaseURL(r *http.Request) string {
	base := strings.TrimRight(strings.TrimSpace(d.effectiveConfig().LTIAPIBaseURL), "/")
	if base != "" {
		return base
	}
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	host := strings.TrimSpace(r.Host)
	if host != "" {
		return proto + "://" + host
	}
	return "http://localhost:8080"
}

func (d Deps) handleListAccessKeyScopes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meSessionUserID(w, r); !ok {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"scopes": api.AllScopes()})
	}
}

func (d Deps) handleListMyAccessKeys() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		rows, err := apitokens.ListByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load access keys.")
			return
		}
		now := time.Now().UTC()
		items := make([]accessKeyItem, 0, len(rows))
		for _, row := range rows {
			var summaries []course.Summary
			if len(row.CourseIDs) > 0 {
				var err error
				summaries, err = course.SummariesByIDs(r.Context(), d.Pool, row.CourseIDs)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load access keys.")
					return
				}
			}
			courses := make([]accessKeyCourse, 0, len(summaries))
			for _, s := range summaries {
				courses = append(courses, accessKeyCourse{
					ID:         s.ID.String(),
					CourseCode: s.CourseCode,
					Title:      s.Title,
				})
			}
			items = append(items, accessKeyItemFromRow(row, courses, now))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"tokens": items})
	}
}

type postAccessKeyBody struct {
	Label      string   `json:"label"`
	Scopes     []string `json:"scopes"`
	CourseIds  []string `json:"courseIds"`
	ExpiresAt  *string  `json:"expiresAt"`
}

func (d Deps) handlePostMyAccessKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		var body postAccessKeyBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		scopes, valid := api.NormalizeScopes(body.Scopes)
		if !valid || len(scopes) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Select at least one valid scope.")
			return
		}
		label := strings.TrimSpace(body.Label)
		if label == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Label is required.")
			return
		}
		var expiresAt *time.Time
		if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "expiresAt must be RFC3339.")
				return
			}
			utc := t.UTC()
			if !utc.After(time.Now().UTC()) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "expiresAt must be in the future.")
				return
			}
			expiresAt = &utc
		}
		courseIDs, err := apitokens.NormalizeCourseIDs(body.CourseIds)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Each courseId must be a UUID.")
			return
		}
		if err := validateAccessKeyCourseIDsForUser(r.Context(), d.Pool, userID, courseIDs); err != nil {
			if err.Error() == "course not accessible" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "One or more courses are not accessible to your account.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not validate courses.")
			return
		}
		row, secret, err := apitokens.Insert(r.Context(), d.Pool, userID, label, scopes, courseIDs, expiresAt)
		if err != nil {
			if err.Error() == "maximum number of access keys reached" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create access key.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         row.ID.String(),
			"token":      secret,
			"label":      row.Label,
			"scopes":     row.Scopes,
			"courseIds":  courseIDsToStrings(row.CourseIDs),
			"allCourses": len(row.CourseIDs) == 0,
			"expiresAt":  row.ExpiresAt,
			"createdAt":  row.CreatedAt,
		})
	}
}

func (d Deps) handleDeleteMyAccessKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		raw := chi.URLParam(r, "id")
		tokenID, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid access key id.")
			return
		}
		okRevoke, err := apitokens.RevokeForUser(r.Context(), d.Pool, userID, tokenID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not revoke access key.")
			return
		}
		if !okRevoke {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Access key not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func courseIDsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func (d Deps) handleMyMCPConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meSessionUserID(w, r); !ok {
			return
		}
		base := d.apiBaseURL(r)
		mcpServer := map[string]any{
			"command": "node",
			"args":    []string{"clients/mcp/dist/index.js"},
			"env": map[string]string{
				"LEXTURES_API_URL":   base,
				"LEXTURES_API_TOKEN": "<paste-your-access-key>",
			},
		}
		cursorConfig := map[string]any{
			"mcpServers": map[string]any{
				"lextures": mcpServer,
			},
		}
		claudeConfig := map[string]any{
			"mcpServers": map[string]any{
				"lextures": mcpServer,
			},
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"apiBaseUrl":          base,
			"cursorConfig":        cursorConfig,
			"claudeDesktopConfig": claudeConfig,
			"instructions": []string{
				"From your Lextures repo clone, build the MCP server: cd clients/mcp && npm install && npm run build.",
				"Open Cursor (or Claude Desktop) with the repo as the project root so clients/mcp/dist/index.js resolves.",
				"Create an access key above with the MCP: Connect scope (and any data scopes your agent needs).",
				"Copy the key when shown — it is only displayed once.",
				"Paste the key into LEXTURES_API_TOKEN in the MCP config below.",
				"Add the config to Cursor (Settings → MCP) or Claude Desktop (~/.claude/claude_desktop_config.json).",
			},
		})
	}
}

func (d Deps) registerIntegrationsRoutes(r chi.Router) {
	r.Get("/api/v1/me/access-keys/scopes", d.handleListAccessKeyScopes())
	r.Get("/api/v1/me/access-keys", d.handleListMyAccessKeys())
	r.Post("/api/v1/me/access-keys", d.handlePostMyAccessKey())
	r.Delete("/api/v1/me/access-keys/{id}", d.handleDeleteMyAccessKey())
	r.Get("/api/v1/me/integrations/mcp", d.handleMyMCPConfig())
}
