package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
)

// requireProctoringEnabled returns false and writes a 501 if the proctoring feature flag is off.
func (d Deps) requireProctoringEnabled(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFProctoringIntegration && !d.Config.FFProctoringIntegration {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Proctoring integration is not enabled.")
		return false
	}
	return true
}

// --- DB types ----------------------------------------------------------------

type proctoringConfigRow struct {
	ID             uuid.UUID
	StructureItemID uuid.UUID
	ExternalToolID uuid.UUID
	Vendor         string
	Required       bool
	Settings       json.RawMessage
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
}

type proctoringSessionRow struct {
	ID              uuid.UUID
	AttemptID       uuid.UUID
	Vendor          string
	VendorSessionID *string
	Status          string
	FlagCount       int
	ReviewURL       *string
	StartedAt       *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// --- DB helpers --------------------------------------------------------------

func getProctoringConfig(ctx context.Context, pool *pgxpool.Pool, structureItemID uuid.UUID) (*proctoringConfigRow, error) {
	var r proctoringConfigRow
	err := pool.QueryRow(ctx, `
SELECT id, structure_item_id, external_tool_id, vendor, required, settings, created_by, created_at
FROM course.quiz_proctoring_config
WHERE structure_item_id = $1
`, structureItemID).Scan(
		&r.ID, &r.StructureItemID, &r.ExternalToolID, &r.Vendor,
		&r.Required, &r.Settings, &r.CreatedBy, &r.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func upsertProctoringConfig(ctx context.Context, pool *pgxpool.Pool, structureItemID, externalToolID uuid.UUID, vendor string, required bool, settings json.RawMessage, createdBy *uuid.UUID) (*proctoringConfigRow, error) {
	var r proctoringConfigRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.quiz_proctoring_config
    (structure_item_id, external_tool_id, vendor, required, settings, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (structure_item_id) DO UPDATE SET
    external_tool_id = EXCLUDED.external_tool_id,
    vendor           = EXCLUDED.vendor,
    required         = EXCLUDED.required,
    settings         = EXCLUDED.settings
RETURNING id, structure_item_id, external_tool_id, vendor, required, settings, created_by, created_at
`, structureItemID, externalToolID, vendor, required, settings, createdBy).Scan(
		&r.ID, &r.StructureItemID, &r.ExternalToolID, &r.Vendor,
		&r.Required, &r.Settings, &r.CreatedBy, &r.CreatedAt,
	)
	return &r, err
}

func deleteProctoringConfig(ctx context.Context, pool *pgxpool.Pool, structureItemID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM course.quiz_proctoring_config WHERE structure_item_id = $1`, structureItemID)
	return err
}

func getProctoringSession(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) (*proctoringSessionRow, error) {
	var r proctoringSessionRow
	err := pool.QueryRow(ctx, `
SELECT id, attempt_id, vendor, vendor_session_id, status, flag_count, review_url, started_at, completed_at, created_at
FROM course.quiz_proctoring_sessions
WHERE attempt_id = $1
ORDER BY created_at DESC
LIMIT 1
`, attemptID).Scan(
		&r.ID, &r.AttemptID, &r.Vendor, &r.VendorSessionID, &r.Status,
		&r.FlagCount, &r.ReviewURL, &r.StartedAt, &r.CompletedAt, &r.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

func upsertProctoringSessionCallback(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID, vendor, vendorSessionID, status string, flagCount int, reviewURL *string, rawCallback json.RawMessage) error {
	now := time.Now().UTC()
	var completedAt *time.Time
	if status == "complete" || status == "flagged" {
		t := now
		completedAt = &t
	}
	// Update the existing session if one exists for this attempt+vendor; insert otherwise.
	tag, err := pool.Exec(ctx, `
UPDATE course.quiz_proctoring_sessions
SET vendor_session_id = $3,
    status            = $4,
    flag_count        = $5,
    review_url        = COALESCE($6, review_url),
    completed_at      = COALESCE($7, completed_at),
    raw_callback      = $8
WHERE attempt_id = $1 AND vendor = $2
`, attemptID, vendor, vendorSessionID, status, flagCount, reviewURL, completedAt, rawCallback)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = pool.Exec(ctx, `
INSERT INTO course.quiz_proctoring_sessions
    (attempt_id, vendor, vendor_session_id, status, flag_count, review_url, started_at, completed_at, raw_callback)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, attemptID, vendor, vendorSessionID, status, flagCount, reviewURL, now, completedAt, rawCallback)
	}
	return err
}

// --- JSON response types -----------------------------------------------------

type proctoringConfigJSON struct {
	ID             string          `json:"id"`
	QuizItemID     string          `json:"quizItemId"`
	ExternalToolID string          `json:"externalToolId"`
	Vendor         string          `json:"vendor"`
	Required       bool            `json:"required"`
	Settings       json.RawMessage `json:"settings"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type proctoringSessionJSON struct {
	ID              string     `json:"id"`
	AttemptID       string     `json:"attemptId"`
	Vendor          string     `json:"vendor"`
	VendorSessionID *string    `json:"vendorSessionId"`
	Status          string     `json:"status"`
	FlagCount       int        `json:"flagCount"`
	ReviewURL       *string    `json:"reviewUrl"`
	StartedAt       *time.Time `json:"startedAt"`
	CompletedAt     *time.Time `json:"completedAt"`
}

func proctoringConfigToJSON(r *proctoringConfigRow) proctoringConfigJSON {
	settings := r.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	return proctoringConfigJSON{
		ID:             r.ID.String(),
		QuizItemID:     r.StructureItemID.String(),
		ExternalToolID: r.ExternalToolID.String(),
		Vendor:         r.Vendor,
		Required:       r.Required,
		Settings:       settings,
		CreatedAt:      r.CreatedAt,
	}
}

func proctoringSessionToJSON(r *proctoringSessionRow) proctoringSessionJSON {
	return proctoringSessionJSON{
		ID:              r.ID.String(),
		AttemptID:       r.AttemptID.String(),
		Vendor:          r.Vendor,
		VendorSessionID: r.VendorSessionID,
		Status:          r.Status,
		FlagCount:       r.FlagCount,
		ReviewURL:       r.ReviewURL,
		StartedAt:       r.StartedAt,
		CompletedAt:     r.CompletedAt,
	}
}

// --- HTTP handlers -----------------------------------------------------------

// handleGetQuizProctoringConfig is GET /api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config
func (d Deps) handleGetQuizProctoringConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireProctoringEnabled(w) {
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
		cfg, err := getProctoringConfig(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load proctoring config.")
			return
		}
		if cfg == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(proctoringConfigToJSON(cfg))
	}
}

type postProctoringConfigBody struct {
	ExternalToolID string          `json:"externalToolId"`
	Vendor         string          `json:"vendor"`
	Required       bool            `json:"required"`
	Settings       json.RawMessage `json:"settings"`
}

// handlePostQuizProctoringConfig is POST /api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config
func (d Deps) handlePostQuizProctoringConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireProctoringEnabled(w) {
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
		var body postProctoringConfigBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		toolID, err := uuid.Parse(body.ExternalToolID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid externalToolId.")
			return
		}
		vendor := strings.ToLower(strings.TrimSpace(body.Vendor))
		switch vendor {
		case "honorlock", "respondus", "proctu", "examity":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid vendor. Allowed: honorlock, respondus, proctu, examity.")
			return
		}
		settings := body.Settings
		if len(settings) == 0 {
			settings = json.RawMessage("{}")
		}
		viewerID := viewer
		row, err := upsertProctoringConfig(r.Context(), d.Pool, itemID, toolID, vendor, body.Required, settings, &viewerID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save proctoring config.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(proctoringConfigToJSON(row))
	}
}

// handleDeleteQuizProctoringConfig is DELETE /api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config
func (d Deps) handleDeleteQuizProctoringConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireProctoringEnabled(w) {
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
		if err := deleteProctoringConfig(r.Context(), d.Pool, itemID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete proctoring config.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetQuizProctoringSession is GET /api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/proctoring-session
func (d Deps) handleGetQuizProctoringSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireProctoringEnabled(w) {
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
		attemptID, err := uuid.Parse(chi.URLParam(r, "attempt_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}
		session, err := getProctoringSession(r.Context(), d.Pool, attemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load proctoring session.")
			return
		}
		if session == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No proctoring session found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(proctoringSessionToJSON(session))
	}
}

// handleProctoringCallback is POST /api/v1/webhooks/proctoring-callback/{vendor}
// Verifies the HMAC-SHA256 signature from the vendor and stores the result.
func (d Deps) handleProctoringCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireProctoringEnabled(w) {
			return
		}
		vendor := strings.ToLower(chi.URLParam(r, "vendor"))
		switch vendor {
		case "honorlock", "respondus", "proctu", "examity":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown vendor.")
			return
		}

		// Decode raw JSON body (used for HMAC and for storage).
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		// Verify HMAC-SHA256 signature header if present.
		sig := r.Header.Get("X-Proctoring-Signature")
		hmacSecret := proctoringWebhookSecret(vendor)
		if sig != "" && hmacSecret != "" {
			mac := hmac.New(sha256.New, []byte(hmacSecret))
			mac.Write(raw)
			expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(sig), []byte(expected)) {
				apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeForbidden, "Invalid signature.")
				return
			}
		}

		// Parse the callback payload.
		var payload struct {
			AttemptID       string  `json:"attemptId"`
			VendorSessionID string  `json:"vendorSessionId"`
			Status          string  `json:"status"`
			FlagCount       int     `json:"flagCount"`
			ReviewURL       *string `json:"reviewUrl"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid payload.")
			return
		}
		attemptID, err := uuid.Parse(payload.AttemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attemptId.")
			return
		}
		status := strings.ToLower(strings.TrimSpace(payload.Status))
		switch status {
		case "pending", "active", "complete", "flagged":
		default:
			status = "active"
		}
		if err := upsertProctoringSessionCallback(r.Context(), d.Pool, attemptID, vendor, payload.VendorSessionID, status, payload.FlagCount, payload.ReviewURL, raw); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record proctoring session.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// proctoringWebhookSecret returns the HMAC secret for a given vendor from the environment.
// Variable names follow the pattern PROCTORING_HMAC_HONORLOCK, PROCTORING_HMAC_RESPONDUS, etc.
func proctoringWebhookSecret(vendor string) string {
	return strings.TrimSpace(os.Getenv("PROCTORING_HMAC_" + strings.ToUpper(vendor)))
}

// --- Route lookup helpers (used by getProctoringConfig for course validation) ---

// verifyQuizInCourse checks the structure item belongs to the given course.
func verifyQuizInCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string, itemID uuid.UUID) (bool, error) {
	cid, err := course.GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil || cid == nil {
		return false, err
	}
	var count int
	err = pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM course.course_structure_items csi
WHERE csi.id = $1 AND csi.course_id = $2 AND csi.kind = 'quiz'
`, itemID, *cid).Scan(&count)
	return count > 0, err
}
