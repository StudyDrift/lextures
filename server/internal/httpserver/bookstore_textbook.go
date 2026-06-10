package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// requireBookstoreEnabled returns false and writes a 501 when the bookstore feature flag is off.
func (d Deps) requireBookstoreEnabled(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFBookstoreIntegration && !d.Config.FFBookstoreIntegration {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Bookstore integration is not enabled.")
		return false
	}
	return true
}

// validBookstoreProvider reports whether p is a supported bookstore provider.
func validBookstoreProvider(p string) bool {
	switch p {
	case "vitalsource", "redshelf":
		return true
	default:
		return false
	}
}

// ─── DB types ──────────────────────────────────────────────────────────────────

type textbookResourceRow struct {
	StructureItemID uuid.UUID
	Provider        string
	ExternalToolID  *uuid.UUID
	Metadata        json.RawMessage
	UpdatedAt       time.Time
}

type inclusiveAccessRow struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	ISBN      string
	Title     string
	OptOutURL string
	Provider  string
	Enabled   bool
	UpdatedAt time.Time
}

type bookstoreConfigRow struct {
	ID                int
	DefaultProvider   string
	VitalSourceToolID *uuid.UUID
	RedShelfToolID    *uuid.UUID
	UpdatedAt         time.Time
}

// ─── DB helpers ─────────────────────────────────────────────────────────────────

func getTextbookResource(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*textbookResourceRow, error) {
	var r textbookResourceRow
	err := pool.QueryRow(ctx, `
SELECT structure_item_id, provider, external_tool_id, metadata, updated_at
FROM course.module_textbook_resources
WHERE structure_item_id = $1
`, itemID).Scan(&r.StructureItemID, &r.Provider, &r.ExternalToolID, &r.Metadata, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func updateTextbookResource(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, meta json.RawMessage) error {
	_, err := pool.Exec(ctx, `
UPDATE course.module_textbook_resources
SET metadata   = $2,
    updated_at = NOW()
WHERE structure_item_id = $1
`, itemID, meta)
	return err
}

func getInclusiveAccess(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*inclusiveAccessRow, error) {
	var r inclusiveAccessRow
	err := pool.QueryRow(ctx, `
SELECT id, course_id, isbn, title, opt_out_url, provider, enabled, updated_at
FROM course.inclusive_access_courses
WHERE course_id = $1
`, courseID).Scan(&r.ID, &r.CourseID, &r.ISBN, &r.Title, &r.OptOutURL, &r.Provider, &r.Enabled, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func upsertInclusiveAccess(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, isbn, title, optOutURL, provider string, enabled bool) (*inclusiveAccessRow, error) {
	var r inclusiveAccessRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.inclusive_access_courses (course_id, isbn, title, opt_out_url, provider, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (course_id) DO UPDATE SET
    isbn        = EXCLUDED.isbn,
    title       = EXCLUDED.title,
    opt_out_url = EXCLUDED.opt_out_url,
    provider    = EXCLUDED.provider,
    enabled     = EXCLUDED.enabled,
    updated_at  = NOW()
RETURNING id, course_id, isbn, title, opt_out_url, provider, enabled, updated_at
`, courseID, isbn, title, optOutURL, provider, enabled).Scan(
		&r.ID, &r.CourseID, &r.ISBN, &r.Title, &r.OptOutURL, &r.Provider, &r.Enabled, &r.UpdatedAt,
	)
	return &r, err
}

func insertTextbookLaunchEvent(ctx context.Context, pool *pgxpool.Pool, itemID, courseID uuid.UUID, provider string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.textbook_launch_events (item_id, course_id, provider) VALUES ($1, $2, $3)
`, itemID, courseID, provider)
	return err
}

type textbookLaunchEventRow struct {
	ID         uuid.UUID
	ItemID     uuid.UUID
	Provider   string
	AccessedAt time.Time
}

func listTextbookLaunchEvents(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, limit int) ([]textbookLaunchEventRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, item_id, provider, accessed_at
FROM course.textbook_launch_events
WHERE course_id = $1
ORDER BY accessed_at DESC
LIMIT $2
`, courseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []textbookLaunchEventRow
	for rows.Next() {
		var e textbookLaunchEventRow
		if err := rows.Scan(&e.ID, &e.ItemID, &e.Provider, &e.AccessedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func getBookstoreConfig(ctx context.Context, pool *pgxpool.Pool) (*bookstoreConfigRow, error) {
	var r bookstoreConfigRow
	err := pool.QueryRow(ctx, `
SELECT id, default_provider, vitalsource_tool_id, redshelf_tool_id, updated_at
FROM settings.bookstore_config
ORDER BY id ASC
LIMIT 1
`).Scan(&r.ID, &r.DefaultProvider, &r.VitalSourceToolID, &r.RedShelfToolID, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func upsertBookstoreConfig(ctx context.Context, pool *pgxpool.Pool, defaultProvider string, vitalSourceToolID, redShelfToolID *uuid.UUID) (*bookstoreConfigRow, error) {
	var r bookstoreConfigRow
	err := pool.QueryRow(ctx, `
UPDATE settings.bookstore_config
SET default_provider    = $1,
    vitalsource_tool_id = $2,
    redshelf_tool_id    = $3,
    updated_at          = NOW()
WHERE id = (SELECT id FROM settings.bookstore_config ORDER BY id LIMIT 1)
RETURNING id, default_provider, vitalsource_tool_id, redshelf_tool_id, updated_at
`, defaultProvider, vitalSourceToolID, redShelfToolID).Scan(
		&r.ID, &r.DefaultProvider, &r.VitalSourceToolID, &r.RedShelfToolID, &r.UpdatedAt,
	)
	if err == nil {
		return &r, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}
	err = pool.QueryRow(ctx, `
INSERT INTO settings.bookstore_config (default_provider, vitalsource_tool_id, redshelf_tool_id)
VALUES ($1, $2, $3)
RETURNING id, default_provider, vitalsource_tool_id, redshelf_tool_id, updated_at
`, defaultProvider, vitalSourceToolID, redShelfToolID).Scan(
		&r.ID, &r.DefaultProvider, &r.VitalSourceToolID, &r.RedShelfToolID, &r.UpdatedAt,
	)
	return &r, err
}

// ─── JSON response types ─────────────────────────────────────────────────────────

type textbookResourceJSON struct {
	ItemID    string          `json:"itemId"`
	Provider  string          `json:"provider"`
	Metadata  json.RawMessage `json:"metadata"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

func textbookResourceToJSON(r *textbookResourceRow) textbookResourceJSON {
	meta := r.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}
	return textbookResourceJSON{
		ItemID:    r.StructureItemID.String(),
		Provider:  r.Provider,
		Metadata:  meta,
		UpdatedAt: r.UpdatedAt,
	}
}

type inclusiveAccessJSON struct {
	Enabled   bool   `json:"enabled"`
	ISBN      string `json:"isbn"`
	Title     string `json:"title"`
	OptOutURL string `json:"optOutUrl"`
	Provider  string `json:"provider"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func inclusiveAccessToJSON(r *inclusiveAccessRow) inclusiveAccessJSON {
	out := inclusiveAccessJSON{
		Enabled:   r.Enabled,
		ISBN:      r.ISBN,
		Title:     r.Title,
		OptOutURL: r.OptOutURL,
		Provider:  r.Provider,
	}
	if !r.UpdatedAt.IsZero() {
		out.UpdatedAt = r.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

type bookstoreConfigJSON struct {
	DefaultProvider   string  `json:"defaultProvider"`
	VitalSourceToolID *string `json:"vitalsourceToolId"`
	RedShelfToolID    *string `json:"redshelfToolId"`
	UpdatedAt         string  `json:"updatedAt,omitempty"`
}

func bookstoreConfigToJSON(r *bookstoreConfigRow) bookstoreConfigJSON {
	out := bookstoreConfigJSON{DefaultProvider: r.DefaultProvider}
	if r.VitalSourceToolID != nil {
		s := r.VitalSourceToolID.String()
		out.VitalSourceToolID = &s
	}
	if r.RedShelfToolID != nil {
		s := r.RedShelfToolID.String()
		out.RedShelfToolID = &s
	}
	if !r.UpdatedAt.IsZero() {
		out.UpdatedAt = r.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// ─── HTTP handlers ───────────────────────────────────────────────────────────────

// handleCreateModuleTextbookResource is POST .../modules/{module_id}/textbook-resources.
func (d Deps) handleCreateModuleTextbookResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireBookstoreEnabled(w) {
			return
		}
		_, _, cid, moduleID, ok := d.beginCreateUnderModule(w, r)
		if !ok {
			return
		}
		var body struct {
			Title          string                               `json:"title"`
			Provider       string                               `json:"provider"`
			ExternalToolID *string                              `json:"externalToolId"`
			Metadata       coursestructure.TextbookResourceMeta `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		provider := strings.ToLower(strings.TrimSpace(body.Provider))
		if provider == "" {
			provider = "vitalsource"
		}
		if !validBookstoreProvider(provider) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid provider. Allowed: vitalsource, redshelf.")
			return
		}
		var toolID *uuid.UUID
		if body.ExternalToolID != nil && strings.TrimSpace(*body.ExternalToolID) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.ExternalToolID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid externalToolId.")
				return
			}
			toolID = &id
		}
		row, err := coursestructure.InsertTextbookResourceUnderModule(
			r.Context(), d.Pool, cid, moduleID, body.Title, provider, toolID, body.Metadata,
		)
		if err != nil {
			if strings.Contains(err.Error(), "title is required") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
				return
			}
			if err == pgx.ErrNoRows {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Module not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create textbook resource.")
			return
		}
		d.writeCreatedStructureItem(w, r, cid, row)
	}
}

// handleGetModuleTextbookResource is GET /api/v1/courses/{course_code}/textbook-resources/{item_id}.
func (d Deps) handleGetModuleTextbookResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		_, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		res, err := getTextbookResource(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load textbook resource.")
			return
		}
		if res == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Textbook resource not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(textbookResourceToJSON(res))
	}
}

type patchTextbookResourceBody struct {
	Metadata json.RawMessage `json:"metadata"`
}

// handlePatchModuleTextbookResource is PATCH /api/v1/courses/{course_code}/textbook-resources/{item_id}.
func (d Deps) handlePatchModuleTextbookResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		var body patchTextbookResourceBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		meta := body.Metadata
		if len(meta) == 0 {
			meta = json.RawMessage("{}")
		}
		if err := updateTextbookResource(r.Context(), d.Pool, itemID, meta); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update textbook resource.")
			return
		}
		res, err := getTextbookResource(r.Context(), d.Pool, itemID)
		if err != nil || res == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload textbook resource.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(textbookResourceToJSON(res))
	}
}

// handlePostTextbookLaunchEvent is POST /api/v1/courses/{course_code}/textbook-resources/{item_id}/access.
// Records an anonymized COUNTER launch event (no student PII — AC-5).
func (d Deps) handlePostTextbookLaunchEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		_, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		cid, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		// Provider comes from the textbook resource row; default to vitalsource.
		provider := "vitalsource"
		if res, err := getTextbookResource(r.Context(), d.Pool, itemID); err == nil && res != nil {
			provider = res.Provider
		}
		if err := insertTextbookLaunchEvent(r.Context(), d.Pool, itemID, *cid, provider); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record launch event.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetTextbookLaunchEvents is GET /api/v1/courses/{course_code}/textbook-launch-events.
// Returns COUNTER-compatible usage stats for admin/bookstore reporting (no PII).
func (d Deps) handleGetTextbookLaunchEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		cid, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		events, err := listTextbookLaunchEvents(r.Context(), d.Pool, *cid, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load launch events.")
			return
		}
		out := make([]map[string]any, 0, len(events))
		for _, e := range events {
			out = append(out, map[string]any{
				"id":         e.ID.String(),
				"itemId":     e.ItemID.String(),
				"provider":   e.Provider,
				"accessedAt": e.AccessedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": out})
	}
}

// handleGetInclusiveAccess is GET /api/v1/courses/{course_code}/inclusive-access.
// Any enrolled course member may read the IA status to render the opt-out banner.
func (d Deps) handleGetInclusiveAccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		ia, err := getInclusiveAccess(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load inclusive access.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if ia == nil {
			_ = json.NewEncoder(w).Encode(inclusiveAccessJSON{Enabled: false})
			return
		}
		_ = json.NewEncoder(w).Encode(inclusiveAccessToJSON(ia))
	}
}

type postInclusiveAccessBody struct {
	ISBN      string `json:"isbn"`
	Title     string `json:"title"`
	OptOutURL string `json:"optOutUrl"`
	Provider  string `json:"provider"`
	Enabled   *bool  `json:"enabled"`
}

// handlePostInclusiveAccess is POST /api/v1/courses/{course_code}/inclusive-access.
// Instructor/admin configures the Inclusive Access title + opt-out link for a course.
func (d Deps) handlePostInclusiveAccess() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		cid, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var body postInclusiveAccessBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		isbn := strings.TrimSpace(body.ISBN)
		title := strings.TrimSpace(body.Title)
		optOut := strings.TrimSpace(body.OptOutURL)
		if isbn == "" || title == "" || optOut == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "isbn, title, and optOutUrl are required.")
			return
		}
		provider := strings.ToLower(strings.TrimSpace(body.Provider))
		if provider == "" {
			provider = "vitalsource"
		}
		if !validBookstoreProvider(provider) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid provider. Allowed: vitalsource, redshelf.")
			return
		}
		enabled := true
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		ia, err := upsertInclusiveAccess(r.Context(), d.Pool, *cid, isbn, title, optOut, provider, enabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save inclusive access.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(inclusiveAccessToJSON(ia))
	}
}

// handleGetAdminBookstoreConfig is GET /api/v1/admin/bookstore/config.
func (d Deps) handleGetAdminBookstoreConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		cfg, err := getBookstoreConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load bookstore config.")
			return
		}
		if cfg == nil {
			cfg = &bookstoreConfigRow{DefaultProvider: "vitalsource"}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(bookstoreConfigToJSON(cfg))
	}
}

type postAdminBookstoreConfigBody struct {
	DefaultProvider   string  `json:"defaultProvider"`
	VitalSourceToolID *string `json:"vitalsourceToolId"`
	RedShelfToolID    *string `json:"redshelfToolId"`
}

// handlePostAdminBookstoreConfig is POST /api/v1/admin/bookstore/config.
func (d Deps) handlePostAdminBookstoreConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireBookstoreEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		var body postAdminBookstoreConfigBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		provider := strings.ToLower(strings.TrimSpace(body.DefaultProvider))
		if provider == "" {
			provider = "vitalsource"
		}
		if !validBookstoreProvider(provider) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid defaultProvider. Allowed: vitalsource, redshelf.")
			return
		}
		parseToolID := func(s *string) (*uuid.UUID, bool) {
			if s == nil || strings.TrimSpace(*s) == "" {
				return nil, true
			}
			id, err := uuid.Parse(strings.TrimSpace(*s))
			if err != nil {
				return nil, false
			}
			return &id, true
		}
		vsID, okVS := parseToolID(body.VitalSourceToolID)
		if !okVS {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid vitalsourceToolId.")
			return
		}
		rsID, okRS := parseToolID(body.RedShelfToolID)
		if !okRS {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid redshelfToolId.")
			return
		}
		cfg, err := upsertBookstoreConfig(r.Context(), d.Pool, provider, vsID, rsID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save bookstore config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(bookstoreConfigToJSON(cfg))
	}
}

// ─── Route registration ──────────────────────────────────────────────────────────

func (d Deps) registerBookstoreRoutes(r chi.Router) {
	r.Get("/api/v1/admin/bookstore/config", d.handleGetAdminBookstoreConfig())
	r.Post("/api/v1/admin/bookstore/config", d.handlePostAdminBookstoreConfig())
}
