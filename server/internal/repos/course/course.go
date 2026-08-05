// Package course is a minimal port of server/src/repos/course.rs (lookups by code / id).
package course

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetIDByCourseCode returns the course id or nil.
func GetIDByCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM course.courses WHERE course_code = $1`, courseCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetCourseCodeByID returns the course code or nil.
func GetCourseCodeByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*string, error) {
	var code string
	err := pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

// GetImportFlags returns question bank + QTI import flags for a course id.
func GetImportFlags(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (questionBankEnabled, qtiImportEnabled bool, err error) {
	e := pool.QueryRow(ctx, `
SELECT question_bank_enabled, qti_import_enabled FROM course.courses WHERE id = $1
`, courseID).Scan(&questionBankEnabled, &qtiImportEnabled)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, false, nil
	}
	return questionBankEnabled, qtiImportEnabled, e
}

// TitleAndLanguage returns a display title and optional catalog language for a course.
// On missing rows or errors, empty strings are returned (callers treat as soft context).
func TitleAndLanguage(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (title, lang string) {
	var t string
	var l *string
	err := pool.QueryRow(ctx, `
SELECT COALESCE(title, course_code), NULLIF(TRIM(catalog_language), '')
FROM course.courses WHERE id = $1
`, courseID).Scan(&t, &l)
	if err != nil {
		return "", ""
	}
	title = t
	if l != nil {
		lang = *l
	}
	return title, lang
}

// WelcomeDraftContext is course metadata safe to send to the welcome-draft AI path (CC.10).
type WelcomeDraftContext struct {
	Title       string
	Description string
	StartDate   string // YYYY-MM-DD or empty
	EndDate     string // YYYY-MM-DD or empty
	Language    string
}

// GetWelcomeDraftContext loads title, description, term dates, and language for AI welcome drafts.
// On missing rows or errors, a zero value is returned.
func GetWelcomeDraftContext(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) WelcomeDraftContext {
	var t string
	var description *string
	var language *string
	var startAt, endAt *time.Time
	err := pool.QueryRow(ctx, `
SELECT COALESCE(title, course_code), description, starts_at, ends_at, NULLIF(TRIM(catalog_language), '')
FROM course.courses WHERE id = $1
`, courseID).Scan(&t, &description, &startAt, &endAt, &language)
	if err != nil {
		return WelcomeDraftContext{}
	}
	out := WelcomeDraftContext{Title: t}
	if description != nil {
		out.Description = *description
	}
	if language != nil {
		out.Language = *language
	}
	if startAt != nil {
		out.StartDate = startAt.UTC().Format("2006-01-02")
	}
	if endAt != nil {
		out.EndDate = endAt.UTC().Format("2006-01-02")
	}
	return out
}
