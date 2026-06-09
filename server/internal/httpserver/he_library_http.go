package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// requireLibraryIntegrationEnabled returns false and writes 501 when the feature flag is off.
func (d Deps) requireLibraryIntegrationEnabled(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFLibraryIntegration && !d.Config.FFLibraryIntegration {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Library integration is not enabled.")
		return false
	}
	return true
}

// ─── DB types ─────────────────────────────────────────────────────────────────

type heLibraryConfigRow struct {
	ID               int
	EZProxyPrefix    string
	DomainPatterns   []string
	AlmaAPIBaseURL   string
	AlmaAPIKeyCipher string
	UpdatedAt        time.Time
}

type libraryResourceRow struct {
	StructureItemID uuid.UUID
	ResourceType    string
	ExternalToolID  *uuid.UUID
	AlmaMmsID       *string
	LegantoListID   *string
	Metadata        json.RawMessage
	EZProxyURL      *string
	UpdatedAt       time.Time
}

// ─── DB helpers ───────────────────────────────────────────────────────────────

func getHELibraryConfig(ctx context.Context, pool *pgxpool.Pool) (*heLibraryConfigRow, error) {
	var r heLibraryConfigRow
	err := pool.QueryRow(ctx, `
SELECT id, ezproxy_prefix, domain_patterns, alma_api_base_url, alma_api_key_cipher, updated_at
FROM settings.he_library_config
ORDER BY id ASC
LIMIT 1
`).Scan(&r.ID, &r.EZProxyPrefix, &r.DomainPatterns, &r.AlmaAPIBaseURL, &r.AlmaAPIKeyCipher, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func upsertHELibraryConfig(ctx context.Context, pool *pgxpool.Pool, ezproxyPrefix string, domainPatterns []string, almaAPIBaseURL, almaAPIKeyCipher string) (*heLibraryConfigRow, error) {
	// Try to update the single existing row first.
	var r heLibraryConfigRow
	keyCipherExpr := almaAPIKeyCipher
	var err error
	if almaAPIKeyCipher != "" {
		err = pool.QueryRow(ctx, `
UPDATE settings.he_library_config
SET ezproxy_prefix    = $1,
    domain_patterns   = $2,
    alma_api_base_url = $3,
    alma_api_key_cipher = $4,
    updated_at        = NOW()
WHERE id = (SELECT id FROM settings.he_library_config ORDER BY id LIMIT 1)
RETURNING id, ezproxy_prefix, domain_patterns, alma_api_base_url, alma_api_key_cipher, updated_at
`, ezproxyPrefix, domainPatterns, almaAPIBaseURL, keyCipherExpr).Scan(
			&r.ID, &r.EZProxyPrefix, &r.DomainPatterns, &r.AlmaAPIBaseURL, &r.AlmaAPIKeyCipher, &r.UpdatedAt,
		)
	} else {
		err = pool.QueryRow(ctx, `
UPDATE settings.he_library_config
SET ezproxy_prefix    = $1,
    domain_patterns   = $2,
    alma_api_base_url = $3,
    updated_at        = NOW()
WHERE id = (SELECT id FROM settings.he_library_config ORDER BY id LIMIT 1)
RETURNING id, ezproxy_prefix, domain_patterns, alma_api_base_url, alma_api_key_cipher, updated_at
`, ezproxyPrefix, domainPatterns, almaAPIBaseURL).Scan(
			&r.ID, &r.EZProxyPrefix, &r.DomainPatterns, &r.AlmaAPIBaseURL, &r.AlmaAPIKeyCipher, &r.UpdatedAt,
		)
	}
	if err == nil {
		return &r, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}
	// No row yet — insert the first one.
	cipher := almaAPIKeyCipher
	err = pool.QueryRow(ctx, `
INSERT INTO settings.he_library_config (ezproxy_prefix, domain_patterns, alma_api_base_url, alma_api_key_cipher)
VALUES ($1, $2, $3, $4)
RETURNING id, ezproxy_prefix, domain_patterns, alma_api_base_url, alma_api_key_cipher, updated_at
`, ezproxyPrefix, domainPatterns, almaAPIBaseURL, cipher).Scan(
		&r.ID, &r.EZProxyPrefix, &r.DomainPatterns, &r.AlmaAPIBaseURL, &r.AlmaAPIKeyCipher, &r.UpdatedAt,
	)
	return &r, err
}

func getLibraryResource(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*libraryResourceRow, error) {
	var r libraryResourceRow
	err := pool.QueryRow(ctx, `
SELECT structure_item_id, resource_type, external_tool_id, alma_mms_id, leganto_list_id, metadata, ezproxy_url, updated_at
FROM course.module_library_resources
WHERE structure_item_id = $1
`, itemID).Scan(
		&r.StructureItemID, &r.ResourceType, &r.ExternalToolID, &r.AlmaMmsID,
		&r.LegantoListID, &r.Metadata, &r.EZProxyURL, &r.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func updateLibraryResource(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, meta json.RawMessage, ezproxyURL *string) error {
	_, err := pool.Exec(ctx, `
UPDATE course.module_library_resources
SET metadata    = $2,
    ezproxy_url = $3,
    updated_at  = NOW()
WHERE structure_item_id = $1
`, itemID, meta, ezproxyURL)
	return err
}

func insertLibraryLinkEvent(ctx context.Context, pool *pgxpool.Pool, itemID, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.library_link_events (item_id, course_id) VALUES ($1, $2)
`, itemID, courseID)
	return err
}

func listLibraryLinkEvents(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, limit int) ([]struct {
	ID         uuid.UUID
	ItemID     uuid.UUID
	CourseID   uuid.UUID
	AccessedAt time.Time
}, error) {
	rows, err := pool.Query(ctx, `
SELECT id, item_id, course_id, accessed_at
FROM course.library_link_events
WHERE course_id = $1
ORDER BY accessed_at DESC
LIMIT $2
`, courseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID         uuid.UUID
		ItemID     uuid.UUID
		CourseID   uuid.UUID
		AccessedAt time.Time
	}
	for rows.Next() {
		var e struct {
			ID         uuid.UUID
			ItemID     uuid.UUID
			CourseID   uuid.UUID
			AccessedAt time.Time
		}
		if err := rows.Scan(&e.ID, &e.ItemID, &e.CourseID, &e.AccessedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─── EZProxy URL rewriting ─────────────────────────────────────────────────────

// rewriteEZProxy prepends the EZproxy login URL to a resource URL when a prefix
// is configured and the resource host matches one of the configured domain patterns.
// The prefix must include the scheme and host, e.g. "https://ezproxy.university.edu".
func rewriteEZProxy(prefix string, patterns []string, resourceURL string) string {
	prefix = strings.TrimSpace(prefix)
	resourceURL = strings.TrimSpace(resourceURL)
	if prefix == "" || resourceURL == "" {
		return resourceURL
	}
	u, err := url.Parse(resourceURL)
	if err != nil || u.Host == "" {
		return resourceURL
	}
	host := u.Hostname()
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if matchDomainPattern(p, host) {
			return prefix + "/login?url=" + resourceURL
		}
	}
	return resourceURL
}

func matchDomainPattern(pattern, host string) bool {
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return host == suffix[1:] || strings.HasSuffix(host, suffix)
	}
	return host == pattern
}

// ─── JSON response types ──────────────────────────────────────────────────────

type heLibraryConfigJSON struct {
	EZProxyPrefix  string   `json:"ezproxyPrefix"`
	DomainPatterns []string `json:"domainPatterns"`
	AlmaAPIBaseURL string   `json:"almaApiBaseUrl"`
	HasAlmaAPIKey  bool     `json:"hasAlmaApiKey"`
	UpdatedAt      string   `json:"updatedAt"`
}

type libraryResourceJSON struct {
	ItemID       string          `json:"itemId"`
	ResourceType string          `json:"resourceType"`
	Metadata     json.RawMessage `json:"metadata"`
	EZProxyURL   *string         `json:"ezproxyUrl"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

func libraryResourceToJSON(r *libraryResourceRow) libraryResourceJSON {
	meta := r.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}
	return libraryResourceJSON{
		ItemID:       r.StructureItemID.String(),
		ResourceType: r.ResourceType,
		Metadata:     meta,
		EZProxyURL:   r.EZProxyURL,
		UpdatedAt:    r.UpdatedAt,
	}
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

// handleCreateModuleLibraryResource is POST .../library-resources.
func (d Deps) handleCreateModuleLibraryResource() http.HandlerFunc {
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
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		_, _, cid, moduleID, ok := d.beginCreateUnderModule(w, r)
		if !ok {
			return
		}
		var body struct {
			Title          string                         `json:"title"`
			ResourceType   string                         `json:"resourceType"`
			ExternalToolID *string                        `json:"externalToolId"`
			Metadata       coursestructure.LibraryResourceMeta `json:"metadata"`
			SourceURL      string                         `json:"sourceUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
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
		// Apply EZproxy rewriting if configured.
		ezproxyURL := ""
		if body.SourceURL != "" {
			cfg, err := getHELibraryConfig(r.Context(), d.Pool)
			if err == nil && cfg != nil {
				ezproxyURL = rewriteEZProxy(cfg.EZProxyPrefix, cfg.DomainPatterns, body.SourceURL)
			}
		}
		row, err := coursestructure.InsertLibraryResourceUnderModule(
			r.Context(), d.Pool, cid, moduleID, body.Title,
			body.ResourceType, toolID, body.Metadata, ezproxyURL,
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
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create library resource.")
			return
		}
		d.writeCreatedStructureItem(w, r, cid, row)
	}
}

// handleGetModuleLibraryResource is GET /api/v1/courses/{course_code}/library-resources/{item_id}.
func (d Deps) handleGetModuleLibraryResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		_ = courseCode
		_ = viewer
		res, err := getLibraryResource(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load library resource.")
			return
		}
		if res == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Library resource not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(libraryResourceToJSON(res))
	}
}

type patchLibraryResourceBody struct {
	Metadata   json.RawMessage `json:"metadata"`
	EZProxyURL *string         `json:"ezproxyUrl"`
}

// handlePatchModuleLibraryResource is PATCH /api/v1/courses/{course_code}/library-resources/{item_id}.
func (d Deps) handlePatchModuleLibraryResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
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
		var body patchLibraryResourceBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		meta := body.Metadata
		if len(meta) == 0 {
			meta = json.RawMessage("{}")
		}
		if err := updateLibraryResource(r.Context(), d.Pool, itemID, meta, body.EZProxyURL); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update library resource.")
			return
		}
		res, err := getLibraryResource(r.Context(), d.Pool, itemID)
		if err != nil || res == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload library resource.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(libraryResourceToJSON(res))
	}
}

// handleLibrarySearch is GET /api/v1/library/search?q=&org_id= — proxies Alma catalog search.
func (d Deps) handleLibrarySearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "q is required.")
			return
		}
		cfg, err := getHELibraryConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load library config.")
			return
		}
		if cfg == nil || cfg.AlmaAPIBaseURL == "" || cfg.AlmaAPIKeyCipher == "" {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeNotImplemented, "Alma API is not configured.")
			return
		}
		results, err := almaSearch(r.Context(), cfg.AlmaAPIBaseURL, cfg.AlmaAPIKeyCipher, q)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Alma search failed.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}
}

// almaSearch calls the Ex Libris Alma brief search API and returns parsed results.
func almaSearch(ctx context.Context, baseURL, apiKey, query string) ([]map[string]any, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/almaws/v1/bibs?q=title~" + url.QueryEscape(query) +
		"&limit=10&offset=0&expand=None&apikey=" + url.QueryEscape(apiKey) + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Bib []struct {
			MmsID  string `json:"mms_id"`
			Title  string `json:"title"`
			Author string `json:"author"`
			ISSN   string `json:"issn"`
			ISBN   string `json:"isbn"`
		} `json:"bib"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(raw.Bib))
	for _, b := range raw.Bib {
		out = append(out, map[string]any{
			"mmsId":  b.MmsID,
			"title":  b.Title,
			"author": b.Author,
			"issn":   b.ISSN,
			"isbn":   b.ISBN,
		})
	}
	return out, nil
}

// handleGetAdminLibraryConfig is GET /api/v1/admin/library/config.
func (d Deps) handleGetAdminLibraryConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		cfg, err := getHELibraryConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load library config.")
			return
		}
		if cfg == nil {
			cfg = &heLibraryConfigRow{}
		}
		out := heLibraryConfigJSON{
			EZProxyPrefix:  cfg.EZProxyPrefix,
			DomainPatterns: cfg.DomainPatterns,
			AlmaAPIBaseURL: cfg.AlmaAPIBaseURL,
			HasAlmaAPIKey:  cfg.AlmaAPIKeyCipher != "",
		}
		if !cfg.UpdatedAt.IsZero() {
			out.UpdatedAt = cfg.UpdatedAt.UTC().Format(time.RFC3339)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type postAdminLibraryConfigBody struct {
	EZProxyPrefix  string   `json:"ezproxyPrefix"`
	DomainPatterns []string `json:"domainPatterns"`
	AlmaAPIBaseURL string   `json:"almaApiBaseUrl"`
	AlmaAPIKey     string   `json:"almaApiKey"`
}

const libraryAPIKeyPlaceholder = "••••••••••••"

// handlePostAdminLibraryConfig is POST /api/v1/admin/library/config.
func (d Deps) handlePostAdminLibraryConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		var body postAdminLibraryConfigBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		// Validate EZproxy prefix format.
		if body.EZProxyPrefix != "" {
			if _, err := url.ParseRequestURI(body.EZProxyPrefix); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "ezproxyPrefix must be a valid URL.")
				return
			}
		}
		patterns := body.DomainPatterns
		if patterns == nil {
			patterns = []string{}
		}
		// Don't re-encrypt if the client sent back the placeholder.
		apiKeyCipher := strings.TrimSpace(body.AlmaAPIKey)
		if apiKeyCipher == libraryAPIKeyPlaceholder {
			apiKeyCipher = ""
		}
		cfg, err := upsertHELibraryConfig(r.Context(), d.Pool,
			strings.TrimSpace(body.EZProxyPrefix),
			patterns,
			strings.TrimSpace(body.AlmaAPIBaseURL),
			apiKeyCipher,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save library config.")
			return
		}
		out := heLibraryConfigJSON{
			EZProxyPrefix:  cfg.EZProxyPrefix,
			DomainPatterns: cfg.DomainPatterns,
			AlmaAPIBaseURL: cfg.AlmaAPIBaseURL,
			HasAlmaAPIKey:  cfg.AlmaAPIKeyCipher != "",
			UpdatedAt:      cfg.UpdatedAt.UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostLibraryLinkEvent is POST /api/v1/courses/{course_code}/library-resources/{item_id}/access.
// Records a COUNTER link-resolve event without PII.
func (d Deps) handlePostLibraryLinkEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
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
		if err := insertLibraryLinkEvent(r.Context(), d.Pool, itemID, *cid); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record access event.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetLibraryLinkEvents is GET /api/v1/courses/{course_code}/library-link-events.
// Returns COUNTER usage stats for admin/librarian use (no PII).
func (d Deps) handleGetLibraryLinkEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLibraryIntegrationEnabled(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
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
		events, err := listLibraryLinkEvents(r.Context(), d.Pool, *cid, 500)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load link events.")
			return
		}
		out := make([]map[string]any, 0, len(events))
		for _, e := range events {
			out = append(out, map[string]any{
				"id":         e.ID.String(),
				"itemId":     e.ItemID.String(),
				"accessedAt": e.AccessedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": out})
	}
}

// ─── Route registration ────────────────────────────────────────────────────────

func (d Deps) registerHELibraryRoutes(r chi.Router) {
	r.Get("/api/v1/library/search", d.handleLibrarySearch())
	r.Get("/api/v1/admin/library/config", d.handleGetAdminLibraryConfig())
	r.Post("/api/v1/admin/library/config", d.handlePostAdminLibraryConfig())
}
